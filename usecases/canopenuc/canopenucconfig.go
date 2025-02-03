package canopenuc

import (
	can "github.com/angelodlfrtr/go-can"
	transports "github.com/angelodlfrtr/go-can/transports"
	"github.com/jaster-prj/canopenrest/external/persistence"
	canopen "github.com/jormenjanssen/go-canopen"
)

type CanOpenUCConfig struct {
	Persistence persistence.IPersistence
	CanPort     string
}

func (cc *CanOpenUCConfig) CreateCanOpenUC() (*CanOpenUC, error) {
	// Configure transport
	tr := &transports.SocketCan{
		Interface: cc.CanPort,
	}

	// Open bus
	bus := can.NewBus(tr)

	if err := bus.Open(); err != nil {
		return nil, err
	}
	network, err := canopen.NewNetwork(*bus)
	if err != nil {
		return nil, err
	}
	err = network.Run()
	if err != nil {
		return nil, err
	}
	return &CanOpenUC{
		persistence: cc.Persistence,
		network:     network,
		nodes:       map[int]*canopen.Node{},
	}, nil
}

func NewCanOpenUC(canPort string) (*CanOpenUC, error) {
	config := CanOpenUCConfig{CanPort: canPort}
	return config.CreateCanOpenUC()
}
