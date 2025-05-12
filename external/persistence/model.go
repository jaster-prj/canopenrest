package persistence

import (
	"time"

	"github.com/jaster-prj/canopenrest/entities"
)

type FlashPersistence struct {
	Requested time.Time           `yaml:"requested"`
	Start     *time.Time          `yaml:"start,omitempty"`
	Finish    *time.Time          `yaml:"finish,omitempty"`
	State     entities.FlashState `yaml:"state"`
	Error     *string             `yaml:"error,omitempty"`
}
