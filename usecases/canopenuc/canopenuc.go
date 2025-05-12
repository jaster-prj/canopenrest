package canopenuc

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaster-prj/canopenrest/common"
	"github.com/jaster-prj/canopenrest/entities"
	"github.com/jaster-prj/canopenrest/external/persistence"
	canopen "github.com/jaster-prj/go-canopen"
	"github.com/rs/zerolog/log"
)

const (
	ERROR_REGISTER                = 4097 //0x1001
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
	mu          sync.Mutex
	flashOrders chan entities.FlashOrder
	persistence persistence.IPersistence
	network     *canopen.Network
	nodes       map[int]*canopen.Node
}

func (c *CanOpenUC) RunFlashTask() {
	go func() {
		for {
			select {
			case flashOrder, ok := <-c.flashOrders:
				if !ok {
					break
				}
				c.flashNode(flashOrder)
			}
		}
	}()
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
	c.mu.Lock()
	defer c.mu.Unlock()
	status := node.NMTMaster.GetStateString()
	return &status, nil
}
func (c *CanOpenUC) WriteNmt(id int, state string) error {
	node, err := c.getNode(id)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return node.NMTMaster.SetState(state)
}
func (c *CanOpenUC) ReadSDO(id int, index uint16, subindex uint8) ([]byte, error) {
	node, err := c.getNode(id)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
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
	c.mu.Lock()
	defer c.mu.Unlock()
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

func (c *CanOpenUC) FlashNode(id int, flashFile []byte, version *string) (*uuid.UUID, error) {
	order, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	flashOrder := entities.FlashOrder{
		FlashOrderId: order,
		Id:           id,
		FlashFile:    flashFile,
		Version:      version,
	}
	log.Debug().Str("Function", "FlashNode").Msgf("SetFlashState %s", order.String())
	err = c.persistence.SetFlashState(order, entities.FlashRequested, nil)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("Function", "FlashNode").Msgf("Add to Channel %s", order.String())
	c.flashOrders <- flashOrder
	log.Debug().Str("Function", "FlashNode").Msgf("Finished %s", order.String())
	return common.POINTER(order), nil
}

func (c *CanOpenUC) GetFlashState(id uuid.UUID) (*entities.FlashOrderState, error) {
	return c.persistence.GetFlashState(id)
}

func (c *CanOpenUC) flashNode(flashOrder entities.FlashOrder) {
	node, err := c.getNode(flashOrder.Id)
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(err))
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashPreOperational, nil)
	err = node.NMTMaster.SetState("PRE-OPERATIONAL")
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("Set PRE-OPERATIONAL failed: %v", err)))
		return
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramStopBefore, nil)
	err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_STOP)})
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("PROGRAM_CONTROL_STOP failed: %v", err)))
		return
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramClear, nil)
	err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_CLEAR)})
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("PROGRAM_CONTROL_CLEAR failed: %v", err)))
		return
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramWriteData, nil)
	err = node.SDOClient.Write(PROGRAM_DATA, 1, false, flashOrder.FlashFile)
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("PROGRAM_DATA failed: %v", err)))
		return
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramWriteFinish, nil)
	flashStatus, err := node.SDOClient.Read(FLASH_STATUS_IDENT, 1)
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("Read FLASH_STATUS_IDENT failed: %v", err)))
		node.NMTMaster.SetState("RESET")
		return
	}
	if int(flashStatus[0]) != 0 {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("FlashStatus failed: %d", int(flashStatus[0]))))
		node.NMTMaster.SetState("RESET")
		return
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramStopAfter, nil)
	err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_STOP)})
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("PROGRAM_CONTROL_STOP2 failed: %v", err)))
		node.NMTMaster.SetState("RESET")
		return
	}
	time.Sleep(time.Second * 1)
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramStart, nil)
	err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_START)})
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("PROGRAM_CONTROL_START failed: %v", err)))
		node.NMTMaster.SetState("RESET")
		return
	}
	time.Sleep(time.Second * 10)
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramCheckError, nil)
	response, err := node.SDOClient.Read(ERROR_REGISTER, 0)
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("Read ERROR_REGISTER failed: %v", err)))
		node.NMTMaster.SetState("RESET")
		return
	}
	if response[0] != 0 {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("Read ERROR_REGISTER not 0: %d", response[0])))
		node.NMTMaster.SetState("RESET")
		return
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramCheckVersion, nil)
	response, err = node.SDOClient.Read(MANUFACTURER_SOFTWARE_VERSION, 0)
	if err != nil {
		c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("Read MANUFACTURER_SOFTWARE_VERSION failed: %v", err)))
		node.NMTMaster.SetState("RESET")
		return
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramAck, nil)
	log.Debug().Str("Function", "flashNode").Msgf("New Version: %s", string(response))
	if flashOrder.Version != nil {
		if string(response) == *flashOrder.Version {
			err = node.SDOClient.Write(PROGRAM_CONTROL, 1, false, []byte{byte(PROGRAM_CONTROL_ACK)})
			if err != nil {
				c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramError, common.POINTER(fmt.Errorf("PROGRAM_CONTROL_ACK failed: %v", err)))
				node.NMTMaster.SetState("RESET")
				return
			}
		}
	}
	c.persistence.SetFlashState(flashOrder.FlashOrderId, entities.FlashProgramFinish, nil)
	log.Debug().Str("Function", "flashNode").Msgf("Flash Success")
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
