package filestorage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaster-prj/canopenrest/common"
	"github.com/jaster-prj/canopenrest/entities"
	"github.com/jaster-prj/canopenrest/external/persistence"
	"gopkg.in/yaml.v3"
)

type Filestorage struct {
	mu        sync.Mutex
	configDir string
}

func NewFilestorage() (*Filestorage, error) {

	var err error
	basePath := os.Getenv("CANOPEN_STORAGE")
	if basePath == "" {
		basePath, err = os.UserConfigDir()
		if err != nil {
			return nil, err
		}
	}
	configDir := path.Join(basePath, "CanOpenRest")
	_, err = os.Stat(configDir)
	if errors.Is(err, fs.ErrNotExist) {
		err = os.Mkdir(configDir, 0700)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &Filestorage{
		configDir: configDir,
	}, nil
}

func (f *Filestorage) SafeNode(id int, odsFile []byte) error {
	nodeDir := path.Join(f.configDir, strconv.Itoa(id))
	_, err := os.Stat(nodeDir)
	if errors.Is(err, fs.ErrNotExist) {
		os.Mkdir(nodeDir, 0700)
	} else if err != nil {
		return err
	}
	nodeFile := path.Join(nodeDir, "objdict.eds")
	file, err := os.OpenFile(nodeFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the bytes to the file
	if _, err := file.Write(odsFile); err != nil {
		return err
	}
	return nil
}

func (f *Filestorage) GetNodes() ([]int, error) {
	nodes := []int{}
	entries, err := os.ReadDir(f.configDir)
	if err != nil {
		return nodes, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			nodeId, err := strconv.Atoi(entry.Name())
			if err != nil {
				continue
			}
			nodes = append(nodes, nodeId)
		}
	}
	return nodes, nil
}

func (f *Filestorage) GetObjDict(id int) ([]byte, error) {
	nodeDir := path.Join(f.configDir, strconv.Itoa(id))
	_, err := os.Stat(nodeDir)
	if err != nil {
		return []byte{}, err
	}
	nodeFile := path.Join(nodeDir, "objdict.eds")
	file, err := os.Open(nodeFile)
	if err != nil {
		return []byte{}, err
	}
	defer file.Close()
	objdict, err := io.ReadAll(file)
	if err != nil {
		return []byte{}, err
	}
	return objdict, nil
}

func (f *Filestorage) GetFlashState(id uuid.UUID) (*entities.FlashOrderState, error) {
	flashDir := path.Join(f.configDir, "flash")
	_, err := os.Stat(flashDir)
	if err != nil {
		return nil, err
	}
	var flash persistence.FlashPersistence
	filePath := path.Join(flashDir, id.String())
	_, err = os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &flash)
	if err != nil {
		return nil, err
	}
	return &entities.FlashOrderState{
		Requested: flash.Requested,
		Start:     flash.Start,
		Finish:    flash.Finish,
		State:     flash.State,
		Error:     flash.Error,
	}, nil
}

func (f *Filestorage) SetFlashState(id uuid.UUID, state entities.FlashState, errState *error) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	flashDir := path.Join(f.configDir, "flash")
	_, err := os.Stat(flashDir)
	if errors.Is(err, fs.ErrNotExist) {
		os.Mkdir(flashDir, 0700)
	} else if err != nil {
		return err
	}
	var flash persistence.FlashPersistence
	filePath := path.Join(flashDir, id.String())
	_, err = os.Stat(filePath)
	if err == nil {
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		data, err := io.ReadAll(file)
		if err != nil {
			file.Close()
			return err
		}
		file.Close()
		err = yaml.Unmarshal(data, &flash)
		if err != nil {
			return err
		}
	}

	switch state {
	case entities.FlashRequested:
		flash.Requested = time.Now()
	case entities.FlashProgramStopBefore:
		flash.Start = common.POINTER(time.Now())
	case entities.FlashProgramFinish:
		flash.Finish = common.POINTER(time.Now())
	case entities.FlashProgramError:
		flash.Finish = common.POINTER(time.Now())
	}
	flash.State = state
	if errState != nil {
		flash.Error = common.POINTER(fmt.Sprintf("%w", *errState))
	}
	data, err := yaml.Marshal(flash)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the bytes to the file
	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}
