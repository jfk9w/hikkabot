package blobs

import (
	"context"
	"os"
	"sync"
	"time"

	"hikkabot/feed/media"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w-go/flu/syncf"

	"github.com/gofrs/uuid"
)

type Files struct {
	Clock      syncf.Clock
	TTL        time.Duration
	Dir        string
	SizeBounds [2]media.Size
	files      map[flu.File]time.Time
	once       sync.Once
	mu         syncf.RWMutex
}

func (fs *Files) String() string {
	return ServiceID
}

func (fs *Files) Buffer(mimeType string, ref media.Ref) media.MetaRef {
	fs.once.Do(func() { fs.files = make(map[flu.File]time.Time) })
	return &fileRef{
		fs:   fs,
		meta: media.Meta{MIMEType: mimeType},
		ref:  ref,
	}
}

func (fs *Files) alloc(ctx context.Context) (flu.File, error) {
	ctx, cancel := fs.mu.Lock(ctx)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	defer cancel()

	now := fs.Clock.Now()
	for file, createdAt := range fs.files {
		if now.Sub(createdAt) > fs.TTL {
			err := file.Remove()
			logf.Get(fs).Resultf(ctx, logf.Debug, logf.Warn, "remove blob file [%s]: %v", file, err)
		}
	}

	file := flu.File(fs.Dir + "/" + uuid.Must(uuid.NewV4()).String())
	fs.files[file] = now
	logf.Get(fs).Debugf(ctx, "allocated new file blob [%s]", file)
	return file, nil
}

func (fs *Files) Close() error {
	return os.RemoveAll(fs.Dir)
}

type fileRef struct {
	fs   *Files
	meta media.Meta
	ref  media.Ref
	file flu.File
	err  error
	once sync.Once
}

func (r *fileRef) GetMeta(ctx context.Context) (*media.Meta, error) {
	r.once.Do(func() { r.get(ctx) })
	return &r.meta, r.err
}

func (r *fileRef) Get(ctx context.Context) (flu.Input, error) {
	r.once.Do(func() { r.get(ctx) })
	return r.file, r.err
}

func (r *fileRef) get(ctx context.Context) {
	input, err := r.ref.Get(ctx)
	if err != nil {
		r.err = err
		return
	}

	file, ok := input.(flu.File)
	if !ok {
		file, err = r.fs.alloc(ctx)
		if err != nil {
			r.err = err
			return
		}

		if _, err := flu.Copy(input, file); err != nil {
			r.err = err
			return
		}
	}

	stat, err := os.Stat(file.String())
	if err != nil {
		r.err = err
		return
	}

	r.file = file

	if skipSizeCheck(ctx) {
		return
	}

	size := media.Size(stat.Size())
	if size > 0 {
		switch {
		case size < r.fs.SizeBounds[0]:
			r.err = errors.Errorf("size %s too low", size)
			return
		case size >= r.fs.SizeBounds[1]:
			r.err = errors.Errorf("size %s too large", size)
			return
		}
	}

	r.meta.Size = media.Size(stat.Size())
}
