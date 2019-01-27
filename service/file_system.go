package service

import (
	"os"
	"path/filepath"

	"github.com/jfk9w-go/flu"
	"github.com/segmentio/ksuid"
)

type FileSystemService string

func FileSystem(tmpDir string) FileSystemService {
	return FileSystemService(tmpDir)
}

func (svc FileSystemService) NewTempResource() flu.FileSystemResource {
	path := filepath.Join(string(svc), ksuid.New().String())
	_ = os.RemoveAll(path)
	return flu.NewFileSystemResource(path)
}
