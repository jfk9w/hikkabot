package service

import (
	"os"
	"path/filepath"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
)

type MediaService struct {
	tmp      string
	aconvert *aconvert.Client
}

func NewMediaService(tmp string, aconvert *aconvert.Client) *MediaService {
	return &MediaService{tmp, aconvert}
}

func (svc *MediaService) NewTempResource() flu.FileSystemResource {
	path := filepath.Join(string(svc.tmp), ksuid.New().String())
	_ = os.RemoveAll(path)
	return flu.NewFileSystemResource(path)
}

type MediaType uint8

const (
	Photo MediaType = iota
	Video
	WebM
)

func (t MediaType) MaxSize() int64 {
	switch t {
	case Photo:
		return MaxPhotoSize
	default:
		return MaxVideoSize
	}
}

type MediaFunc func(flu.FileSystemResource) (MediaType, error)

type MediaRequest struct {
	Func    MediaFunc
	Href    string
	MinSize int64
}

type Media struct {
	Resource flu.FileSystemResource
	Href     string
	Type     MediaType
}

type MediaResponse struct {
	Media
	Err error
}

func (svc *MediaService) Download(out chan<- MediaResponse, reqs ...MediaRequest) {
	var prev, curr chan struct{}
	for i, req := range reqs {
		if i > 0 {
			prev = curr
		}

		curr = make(chan struct{})
		go svc.download(out, req, prev, curr)
	}

	<-curr
	close(out)
}

func (svc *MediaService) Download1(req MediaRequest) (Media, error) {
	r := svc.NewTempResource()
	mediaType, err := req.Func(r)
	m := Media{r, req.Href, mediaType}
	if err != nil {
		_ = os.RemoveAll(r.Path())
		return m, errors.Wrap(err, "download failed")
	}

	if mediaType == WebM {
		resp, err := svc.aconvert.Convert(r, aconvert.NewOpts().
			TargetFormat("mp4").
			Code(81000).
			VideoOptionSize(0))
		if err != nil {
			_ = os.RemoveAll(r.Path())
			return m, errors.Wrap(err, "WebM conversion failed")
		}

		err = svc.aconvert.Download(resp, r)
		if err != nil {
			_ = os.RemoveAll(r.Path())
			return m, errors.Wrap(err, "MP4 download failed")
		}
	}

	stat, err := os.Stat(r.Path())
	if err != nil {
		_ = os.RemoveAll(r.Path())
		return m, errors.Wrap(err, "stat failed")
	} else {
		size := stat.Size()
		maxSize := mediaType.MaxSize()
		if size > maxSize {
			_ = os.RemoveAll(r.Path())
			return m, errors.Wrapf(err, "size (%d MB) exceeds limit (%d MB)",
				stat.Size()>>20, maxSize>>20)
		}

		if size < req.MinSize {
			_ = os.RemoveAll(r.Path())
			return m, errors.Wrapf(err, "size (%d KB) is below threshold (%d KB)",
				stat.Size()>>10, req.MinSize>>10)
		}
	}

	return m, nil
}

func (svc *MediaService) download(out chan<- MediaResponse, req MediaRequest, prev, curr chan struct{}) {
	media, err := svc.Download1(req)
	resp := MediaResponse{media, err}

	if prev != nil {
		<-prev
	}

	out <- resp

	if curr != nil {
		curr <- struct{}{}
	}
}
