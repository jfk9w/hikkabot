package services

import (
	"github.com/jfk9w/hikkabot/services/dvach"
	"github.com/jfk9w/hikkabot/services/reddit"
	"github.com/jfk9w/hikkabot/subscription"
)

var (
	Reddit       = reddit.Service
	DvachCatalog = dvach.CatalogService
	DvachThread  = dvach.ThreadService

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
