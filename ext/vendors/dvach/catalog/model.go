package catalog

import (
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w/hikkabot/ext/vendors"
)

type Data struct {
	Board  string         `json:"board"`
	Query  *vendors.Query `json:"query"`
	Offset int            `json:"offset,omitempty"`
	Auto   []string       `json:"auto,omitempty"`
}

func (d *Data) Labels() metrics.Labels {
	return metrics.Labels{}
}
