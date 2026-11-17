package canopenuc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/jaster-prj/canopenrest/entities"
	"github.com/jaster-prj/canopenrest/external/persistence"
	can "github.com/jaster-prj/go-can"
	canopen "github.com/jaster-prj/go-canopen"
)

const (
	udpCANFrameSize = 72
	udpCANDataSize  = 64
)

// UDPTransport uses the native_sim UDP-CAN wire format to communicate with
// a UDP CAN gateway. The gateway is responsible for forwarding frames to the
// simulator nodes and returning their responses to this socket.
type UDPTransport struct {
	RemoteAddr string
	LocalAddr  string

	conn   *net.UDPConn
	remote *net.UDPAddr
	frames chan *can.Frame
}

// NewCanOpenUCWithUDP creates a CANopen service using a UDP-CAN gateway.
func NewCanOpenUCWithUDP(store persistence.IPersistence, remoteAddr, localAddr string) (*CanOpenUC, error) {
	tr := &UDPTransport{RemoteAddr: remoteAddr, LocalAddr: localAddr}
	bus := can.NewBus(tr)
	if err := bus.Open(); err != nil {
		return nil, err
	}
	network, err := canopen.NewNetwork(*bus)
	if err != nil {
		_ = bus.Close()
		return nil, err
	}
	if err := network.Run(); err != nil {
		_ = bus.Close()
		return nil, err
	}
	service := &CanOpenUC{
		mu:          sync.Mutex{},
		flashOrders: make(chan entities.FlashOrder, 20),
		persistence: store,
		network:     network,
		nodes:       map[int]*canopen.Node{},
	}
	service.RunFlashTask()
	return service, nil
}

func (t *UDPTransport) Open() error {
	remote, err := net.ResolveUDPAddr("udp", t.RemoteAddr)
	if err != nil {
		return fmt.Errorf("resolve UDP CAN gateway address: %w", err)
	}
	local := &net.UDPAddr{}
	if t.LocalAddr != "" {
		local, err = net.ResolveUDPAddr("udp", t.LocalAddr)
		if err != nil {
			return fmt.Errorf("resolve UDP CAN local address: %w", err)
		}
	}
	t.conn, err = net.ListenUDP("udp", local)
	if err != nil {
		return fmt.Errorf("listen for UDP CAN frames: %w", err)
	}
	t.remote = remote
	t.frames = make(chan *can.Frame, 64)
	go t.receive()
	return nil
}

func (t *UDPTransport) Close() error {
	if t.conn == nil {
		return nil
	}
	return t.conn.Close()
}

func (t *UDPTransport) Write(frame *can.Frame) error {
	if t.conn == nil || t.remote == nil {
		return errors.New("UDP CAN transport is not open")
	}
	if frame.DLC > 8 {
		return fmt.Errorf("CAN frame DLC %d exceeds classic CAN limit", frame.DLC)
	}
	wire := make([]byte, udpCANFrameSize)
	binary.LittleEndian.PutUint32(wire[0:4], frame.ArbitrationID)
	wire[4] = frame.DLC
	copy(wire[8:16], frame.Data[:])
	if _, err := t.conn.WriteToUDP(wire, t.remote); err != nil {
		return fmt.Errorf("send UDP CAN frame: %w", err)
	}
	return nil
}

func (t *UDPTransport) ReadChan() chan *can.Frame {
	return t.frames
}

func (t *UDPTransport) receive() {
	defer close(t.frames)
	wire := make([]byte, 2048)
	for {
		n, _, err := t.conn.ReadFromUDP(wire)
		if err != nil {
			return
		}
		if n != udpCANFrameSize || wire[4] > 8 {
			continue
		}
		frame := &can.Frame{
			ArbitrationID: binary.LittleEndian.Uint32(wire[0:4]),
			DLC:           wire[4],
		}
		copy(frame.Data[:], wire[8:16])
		t.frames <- frame
	}
}
