package dvach

import (
	"fmt"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/service"
)

const (
	maxMessageSize = telegram.MaxMessageSize * 5 / 7
	maxCaptionSize = telegram.MaxCaptionSize * 5 / 7
)

type Service struct {
	agg   *service.Aggregator
	media *service.MediaService
	dvach *dvach.Client
}

func NewService(agg *service.Aggregator, media *service.MediaService, dvach *dvach.Client) *Service {
	return &Service{agg, media, dvach}
}

func (s *Service) Catalog() *CatalogService {
	return (*CatalogService)(s)
}

func (s *Service) Thread() *ThreadService {
	return (*ThreadService)(s)
}

func (s *Service) download(files ...*dvach.File) <-chan service.MediaResponse {
	out := make(chan service.MediaResponse)
	reqs := make([]service.MediaRequest, len(files))
	for i, file := range files {
		reqs[i] = s.mediaRequest(file)
	}

	go s.media.Download(out, reqs...)
	return out
}

func (s *Service) mediaRequest(file *dvach.File) service.MediaRequest {
	return service.MediaRequest{
		Func:    s.mediaFunc(file),
		Href:    fmt.Sprintf(dvach.Host + file.Path),
		Type:    mediaType(file),
		MinSize: 10 << 10,
	}
}

func (s *Service) mediaFunc(file *dvach.File) service.MediaFunc {
	return func(resource flu.FileSystemResource) error {
		return s.dvach.DownloadFile(file, resource)
	}
}

func mediaType(file *dvach.File) service.MediaType {
	switch file.Type {
	case dvach.WebM:
		return service.WebM
	case dvach.MP4, dvach.GIF:
		return service.Video
	default:
		return service.Photo
	}
}
