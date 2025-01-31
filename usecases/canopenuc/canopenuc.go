package canopenuc

import (
	"time"

	canopen "github.com/angelodlfrtr/go-canopen"
	"github.com/jaster-prj/canopenrest/external/persistence"
	"github.com/rs/zerolog/log"
)

const (
	PROGRAM_DATA           = 8016
	PROGRAM_CONTROL        = 8017
	PROGRAM_SOFTWARE_IDENT = 8022
	FLASH_STATUS_IDENT     = 8023
)

type ProgramControlState int

const (
	PROGRAM_CONTROL_STOP  ProgramControlState = 0
	PROGRAM_CONTROL_START ProgramControlState = 1
	PROGRAM_CONTROL_RESET ProgramControlState = 2
	PROGRAM_CONTROL_CLEAR ProgramControlState = 3
	PROGRAM_CONTROL_ACK   ProgramControlState = 128
)

type CanOpenUC struct {
	persistence persistence.IPersistence
	network     *canopen.Network
	nodes       map[int]*canopen.Node
}

func (c *CanOpenUC) ReadNmt(id int) (*string, error) {
	node, err := c.getNode(id)
	if err != nil {
		return nil, err
	}
	status := node.NMTMaster.GetStateString()
	return &status, nil
}
func (c *CanOpenUC) WriteNmt(id int, state string) error {
	node, err := c.getNode(id)
	if err != nil {
		return err
	}
	return node.NMTMaster.SetState(state)
}
func (c *CanOpenUC) ReadSDO(id int, index uint16, subindex uint8) ([]byte, error) {
	node, err := c.getNode(id)
	if err != nil {
		return nil, err
	}
	data, err := node.SDOClient.Read(index, subindex)
	if err == nil {
		log.Debug().Str("Function", "WriteSDO").Msgf("data: %v", data)
	}
	return data, err
}
func (c *CanOpenUC) WriteSDO(id int, index uint16, subindex uint8, data []byte) error {
	node, err := c.getNode(id)
	if err != nil {
		return err
	}
	log.Debug().Str("Function", "WriteSDO").Msgf("data: %v", data)
	return node.SDOClient.Write(index, subindex, false, data)
}

func (c *CanOpenUC) CreateNode(id int, edsFile []byte) error {
	err := c.persistence.SafeNode(id, edsFile)
	if err != nil {
		return err
	}
	_, err = c.getNode(id)
	return err
}

func (c *CanOpenUC) FlashNode(id int, flashFile []byte) error {
	node, err := c.getNode(id)
	if err != nil {
		return err
	}
	go func() {
		err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_STOP)})
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_CONTROL_STOP failed: %v", err)
			return
		}
		err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_CLEAR)})
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_CONTROL_CLEAR failed: %v", err)
			return
		}
		err = node.SDOClient.Write(PROGRAM_DATA, 1, true, flashFile)
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_DATA failed: %v", err)
			return
		}
		flashStatus, err := node.SDOClient.Read(FLASH_STATUS_IDENT, 1)
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_DATA failed: %v", err)
			return
		}
		if int(flashStatus[0]) != 0 {
			log.Debug().Str("Function", "FlashNode").Msgf("FlashStatus failed: %d", int(flashStatus[0]))
			return
		}
		err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_STOP)})
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_CONTROL_STOP2 failed: %v", err)
			return
		}
		time.Sleep(time.Second * 5)
		err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_START)})
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_CONTROL_START failed: %v", err)
			return
		}
		log.Debug().Str("Function", "FlashNode").Msgf("Flash Success")
	}()
	return nil
}

func (c *CanOpenUC) getNode(id int) (*canopen.Node, error) {
	var node *canopen.Node
	if _, ok := c.nodes[id]; !ok {
		odsFile, err := c.persistence.GetObjDict(id)
		if err != nil {
			return nil, err
		}
		config := NodeConfig{
			c.network,
			id,
			odsFile,
		}
		node, err = config.CreateNode()
		if err != nil {
			return nil, err
		}
		c.nodes[id] = node
	} else {
		node = c.nodes[id]
	}
	return node, nil
}
