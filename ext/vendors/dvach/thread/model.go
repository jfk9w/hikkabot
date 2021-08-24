package thread

import (
	"github.com/jfk9w-go/flu/metrics"
)

type Data struct {
	Board     string `json:"board"`
	Num       int    `json:"num"`
	MediaOnly bool   `json:"media_only,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Tag       string `json:"tag"`
}

func (d *Data) Labels() metrics.Labels {
	return metrics.Labels{}
}
