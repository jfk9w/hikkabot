package thread

import "github.com/sirupsen/logrus"

type Data struct {
	Board     string `json:"board"`
	Num       int    `json:"num"`
	MediaOnly bool   `json:"media_only,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Tag       string `json:"tag"`
}

func (d *Data) Fields() logrus.Fields {
	return logrus.Fields{
		"board": d.Board,
		"num":   d.Num,
	}
}
