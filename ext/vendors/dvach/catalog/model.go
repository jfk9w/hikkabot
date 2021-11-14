package catalog

import (
	"github.com/jfk9w-go/flu/me3x"

	"hikkabot/ext/vendors"
)

type Data struct {
	Board  string         `json:"board"`
	Query  *vendors.Query `json:"query"`
	Offset int            `json:"offset,omitempty"`
	Auto   []string       `json:"auto,omitempty"`
}

func (d *Data) Labels() me3x.Labels {
	return me3x.Labels{}
}
