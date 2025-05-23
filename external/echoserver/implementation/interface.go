package implementation

import (
	"github.com/google/uuid"
	"github.com/jaster-prj/canopenrest/entities"
)

// ICanopenRest interface represents use case handlers for the CanOpenRest UseCase
type ICanopenRest interface {
	ReadNmt(node int) (*string, error)
	WriteNmt(node int, state string) error
	ReadSDO(node int, index uint16, subindex uint8) ([]byte, error)
	WriteSDO(node int, index uint16, subindex uint8, data []byte) error
	CreateNode(id int, edsFile []byte) error
	FlashNode(id int, flashFile []byte, version *string) (*uuid.UUID, error)
	GetFlashState(id uuid.UUID) (*entities.FlashOrderState, error)
}
