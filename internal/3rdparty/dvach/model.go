package dvach

import (
	"github.com/pkg/errors"
)

const (
	Domain = "2ch.hk"
	Host   = "https://" + Domain
)

var (
	ErrNotFound = errors.New("not found")
)

const (
	JPEG FileType = 1
	PNG  FileType = 2
	GIF  FileType = 4
	WebM FileType = 6
	MP4  FileType = 10
)

var Type2MIMEType = map[FileType]string{
	JPEG: "image/jpeg",
	PNG:  "image/png",
	GIF:  "image/gif",
	WebM: "video/webm",
	MP4:  "video/mp4",
}

type FileType int

func (ft FileType) MIMEType() string {
	return Type2MIMEType[ft]
}
