package catalog

import (
	"github.com/jfk9w/hikkabot/vendors"
	"github.com/sirupsen/logrus"
)

type Data struct {
	Board  string         `json:"board"`
	Query  *vendors.Query `json:"query"`
	Offset int            `json:"offset,omitempty"`
	Auto   []string       `json:"auto,omitempty"`
}

func (d *Data) Fields() logrus.Fields {
	return logrus.Fields{
		"board": d.Board,
		"query": d.Query,
		"auto":  d.Auto,
	}
}
