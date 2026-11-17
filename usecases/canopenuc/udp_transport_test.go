package canopenuc

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	can "github.com/jaster-prj/go-can"
)

func TestUDPTransportExchangesNativeSimFrame(t *testing.T) {
	gateway, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()

	transport := &UDPTransport{RemoteAddr: gateway.LocalAddr().String()}
	if err := transport.Open(); err != nil {
		t.Fatal(err)
	}
	defer transport.Close()

	frame := &can.Frame{ArbitrationID: 0x581, DLC: 2, Data: [8]byte{0x40, 0x11}}
	if err := transport.Write(frame); err != nil {
		t.Fatal(err)
	}
	wire := make([]byte, udpCANFrameSize)
	_ = gateway.SetReadDeadline(time.Now().Add(time.Second))
	n, client, err := gateway.ReadFromUDP(wire)
	if err != nil {
		t.Fatal(err)
	}
	if n != udpCANFrameSize || binary.LittleEndian.Uint32(wire[:4]) != frame.ArbitrationID ||
		wire[4] != frame.DLC || wire[8] != frame.Data[0] {
		t.Fatalf("unexpected request frame: size=%d bytes=%x", n, wire[:n])
	}

	binary.LittleEndian.PutUint32(wire[:4], 0x601)
	wire[4] = 2
	wire[8], wire[9] = 0x4b, 0x11
	if _, err := gateway.WriteToUDP(wire, client); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-transport.ReadChan():
		if got.ArbitrationID != 0x601 || got.DLC != 2 || got.Data[0] != 0x4b {
			t.Fatalf("unexpected response frame: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for UDP CAN response")
	}
}
