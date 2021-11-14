package thread

import (
	"github.com/jfk9w-go/flu/me3x"
)

type Data struct {
	Board     string `json:"board"`
	Num       int    `json:"num"`
	MediaOnly bool   `json:"media_only,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Tag       string `json:"tag"`
}

func (d *Data) Labels() me3x.Labels {
	return me3x.Labels{}
}
