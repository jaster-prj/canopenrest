package canopenuc

import (
	"time"

	"github.com/jaster-prj/canopenrest/external/persistence"
	canopen "github.com/jaster-prj/go-canopen"
	"github.com/rs/zerolog/log"
)

const (
	MANUFACTURER_SOFTWARE_VERSION = 4106 //0x100A
	PROGRAM_DATA                  = 8016 //0x1F50
	PROGRAM_CONTROL               = 8017 //0x1F51
	PROGRAM_SOFTWARE_IDENT        = 8022
	FLASH_STATUS_IDENT            = 8023
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
	timestamp := node.NMTMaster.Timestamp
	timeout := time.Now().Add(time.Second * 3)
	for {
		if time.Now().After(timeout) {
			break
		}
		if timestamp != node.NMTMaster.Timestamp {
			break
		}
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
		log.Debug().Str("Function", "ReadSDO").Msgf("data: %v", data)
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

func (c *CanOpenUC) FlashNode(id int, flashFile []byte, version *string) error {
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
		err = node.SDOClient.Write(PROGRAM_DATA, 1, false, flashFile)
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_DATA failed: %v", err)
			return
		}
		flashStatus, err := node.SDOClient.Read(FLASH_STATUS_IDENT, 1)
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("Read FLASH_STATUS_IDENT failed: %v", err)
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
		response, err := node.SDOClient.Read(MANUFACTURER_SOFTWARE_VERSION, 0)
		if err != nil {
			log.Debug().Str("Function", "FlashNode").Msgf("Read MANUFACTURER_SOFTWARE_VERSION failed: %v", err)
			return
		}
		log.Debug().Str("Function", "FlashNode").Msgf("New Version: %s", string(response))
		if version != nil {
			if string(response) == *version {
				err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_ACK)})
				if err != nil {
					log.Debug().Str("Function", "FlashNode").Msgf("PROGRAM_CONTROL_ACK failed: %v", err)
					return
				}
			}
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
