package keeper

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/unit"
)

type Json interface {
	json.Marshaler
	json.Unmarshaler

	T
}

type FileSync struct {
	unit.Aux
	ptr    Json
	period time.Duration
	path   string
	log    logrus.Logger
	mu     sync.Mutex
}

func RunFileSync(ptr Json, config Config) (*FileSync, error) {
	var (
		syncTimeout = 60000
		logger      = "keeper"
	)

	if config.SyncTimeout != nil {
		syncTimeout = *config.SyncTimeout
	}
	if config.Logger != nil {
		logger = *config.Logger
	}

	path, err := misc.Expand(config.DBPath)
	if err != nil {
		return nil, err
	}

	fs := &FileSync{unit.NewAux(), ptr, time.Duration(syncTimeout) * time.Millisecond, path, logrus.GetLogger(logger), sync.Mutex{}}
	if err := fs.load(); err != nil {
		if os.IsNotExist(err) {
			fs.log.Infof("File %s does not exist, starting from scratch", path)
		} else {
			return nil, err
		}
	}

	// test save
	if err := fs.Save(); err != nil {
		fs.log.Errorf("Failed to write to file %s: %s", path, err)
		return nil, err
	}

	go fs.sync()
	fs.log.Infof("Started file sync to %s", path)

	return fs, nil
}

func (fs *FileSync) Save() (err error) {
	fs.mu.Lock()
	err = misc.WriteJSON(fs.path, fs.ptr)
	if err == nil {
		err = misc.WriteJSON(fs.path+".copy", fs.ptr)
	}

	if err != nil {
		fs.log.Errorf("Failed to write data: %s", err)
	} else {
		fs.log.Infof("Data sync OK")
	}

	fs.mu.Unlock()
	return
}

func (fs *FileSync) load() (err error) {
	fs.mu.Lock()
	err = misc.ReadJSON(fs.path, fs.ptr)
	fs.mu.Unlock()
	return
}

func (fs *FileSync) sync() {
	for {
		if err := fs.Exec(func() {
			time.Sleep(fs.period)
		}); err == unit.ErrInterrupted {
			return
		}

		if err := fs.Save(); err != nil {
			fs.log.Fatalf("Failed to background sync: %s", err)
		}
	}
}
