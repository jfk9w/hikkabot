package media

import (
	"os"
	"path/filepath"

	"github.com/jfk9w-go/flu"
)

type Resource interface {
	flu.Input
	Pull(flu.Input) error
	Cleanup() error
}

type FileResource struct {
	flu.File
}

func NewFileResource(path ...string) Resource {
	return &FileResource{flu.File(filepath.Join(path...))}
}

func (r *FileResource) Pull(in flu.Input) error {
	if file, ok := in.(*FileResource); ok {
		r.File = file.File
		return nil
	}

	return flu.Copy(in, r)
}

func (r *FileResource) Cleanup() error {
	return os.RemoveAll(r.Path())
}

type MemoryResource struct {
	flu.Buffer
}

func NewMemoryResource(size int) Resource {
	buf := flu.NewBuffer()
	if size > 0 {
		buf.Grow(size)
	}

	return &MemoryResource{buf}
}

func (r *MemoryResource) Pull(in flu.Input) error {
	if buf, ok := in.(*MemoryResource); ok {
		r.Buffer = buf.Buffer
		return nil
	}

	return flu.Copy(in, r)
}

func (r *MemoryResource) Cleanup() error {
	return nil
}

type VolatileResource struct {
	flu.Input
}

func (r *VolatileResource) Pull(in flu.Input) error {
	r.Input = in
	return nil
}

func (r *VolatileResource) Cleanup() error {
	return nil
}
