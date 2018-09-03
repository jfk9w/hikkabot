package feed

import (
	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/logx"
	"github.com/jfk9w-go/red"
)

type (
	Dvach    = *dvach.API
	Aconvert = *aconvert.Balancer
	Red      = *red.API
)

var log = logx.Get("feed")
