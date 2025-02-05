package filestorage

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
)

type Filestorage struct {
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
