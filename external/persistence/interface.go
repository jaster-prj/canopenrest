package persistence

import (
	"github.com/google/uuid"
	"github.com/jaster-prj/canopenrest/entities"
)

type IPersistence interface {
	SafeNode(id int, odsFile []byte) error
	GetNodes() ([]int, error)
	GetObjDict(id int) ([]byte, error)
	GetFlashState(id uuid.UUID) (*entities.FlashOrderState, error)
	SetFlashState(id uuid.UUID, state entities.FlashState, errState *error) error
}
