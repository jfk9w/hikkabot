package service

import (
	"path/filepath"

	"github.com/jfk9w-go/flu"
	"github.com/segmentio/ksuid"
)

type FileSystemService struct {
	tmpDir string
}

func FileSystem(tmpDir string) FileSystemService {
	return FileSystemService{tmpDir}
}

func (svc FileSystemService) newTempResource() flu.FileSystemResource {
	return flu.NewFileSystemResource(filepath.Join(svc.tmpDir, ksuid.New().String()))
}
