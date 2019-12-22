package media

import (
	"os"
	"path/filepath"

	"github.com/jfk9w-go/flu"
	"github.com/segmentio/ksuid"
)

type Storage interface {
	Init() error
	NewResource() ReadWrite
	Cleanup()
}

type FileStorage struct {
	Dir string
}

func (fs FileStorage) Init() error {
	fs.Cleanup()
	return os.MkdirAll(fs.Dir, os.ModeDir|os.ModePerm)
}

func (fs FileStorage) NewResource() ReadWrite {
	path := filepath.Join(fs.Dir, ksuid.New().String())
	_ = os.RemoveAll(path)
	return File{flu.File(path)}
}

func (fs FileStorage) Cleanup() {
	os.RemoveAll(fs.Dir)
}

type ReadOnly interface {
	flu.Readable
	Size() (int64, error)
	Cleanup()
}

type ReadWrite interface {
	ReadOnly
	flu.Writable
}

type File struct {
	flu.File
}

func (f File) Size() (int64, error) {
	stat, err := os.Stat(f.Path())
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}

func (f File) Cleanup() {
	os.RemoveAll(f.Path())
}
