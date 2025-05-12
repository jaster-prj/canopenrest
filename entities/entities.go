package entities

import (
	"time"

	"github.com/google/uuid"
)

type FlashOrder struct {
	FlashOrderId uuid.UUID
	Id           int
	FlashFile    []byte
	Version      *string
}

type FlashOrderState struct {
	Requested time.Time
	Start     *time.Time
	Finish    *time.Time
	State     FlashState
	Error     *string
}

type FlashState int

const (
	FlashRequested = iota
	FlashPreOperational
	FlashProgramStopBefore
	FlashProgramClear
	FlashProgramWriteData
	FlashProgramWriteFinish
	FlashProgramStopAfter
	FlashProgramStart
	FlashProgramCheckError
	FlashProgramCheckVersion
	FlashProgramAck
	FlashProgramFinish
	FlashProgramError
)

var flashStateNames = map[FlashState]string{
	FlashRequested:           "Flash requested",
	FlashPreOperational:      "Flash set PRE-OPERATIONAL",
	FlashProgramStopBefore:   "Flash Program stopping before flashing",
	FlashProgramClear:        "Flash Program data clearing",
	FlashProgramWriteData:    "Flash Program data writing",
	FlashProgramWriteFinish:  "Flash Program check data writing",
	FlashProgramStopAfter:    "Flash Program stopping after flashing",
	FlashProgramStart:        "Flash Program starting application",
	FlashProgramCheckError:   "Flash Program check errors",
	FlashProgramCheckVersion: "Flash Program check version",
	FlashProgramAck:          "Flash Program acknowledging application",
	FlashProgramFinish:       "Flash Program finished",
	FlashProgramError:        "Flash Program finished with error",
}

func (fs FlashState) String() string {
	return flashStateNames[fs]
}
