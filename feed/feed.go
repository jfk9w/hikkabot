package feed

import (
	"github.com/jfk9w-go/hikkabot/common/aconvert-api"
	"github.com/jfk9w-go/hikkabot/common/dvach-api"
	"github.com/jfk9w-go/hikkabot/common/logx"
	"github.com/jfk9w-go/hikkabot/common/reddit-api"
)

type (
	Dvach    = *dvach.API
	Aconvert = *aconvert.Balancer
	Red      = *red.API
)

var log = logx.Get("feed")
