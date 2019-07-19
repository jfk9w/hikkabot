package services

import (
	"github.com/jfk9w/hikkabot/app/services/dvach"
	"github.com/jfk9w/hikkabot/app/services/reddit"
	"github.com/jfk9w/hikkabot/app/subscription"
)

var (
	Reddit       = reddit.Factory
	DvachCatalog = dvach.CatalogFactory
	DvachThread  = dvach.ThreadFactory

	All = []subscription.Service{
		Reddit,
		DvachCatalog,
		DvachThread,
	}

	Map = make(map[string]subscription.Service)
)

func init() {
	for _, service := range All {
		Map[service().Service()] = service
	}
}
