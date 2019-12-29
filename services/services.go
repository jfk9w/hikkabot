package services

import (
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/services/dvach"
	"github.com/jfk9w/hikkabot/services/reddit"
)

var (
	Reddit       = reddit.Service
	DvachCatalog = dvach.CatalogService
	DvachThread  = dvach.ThreadService

	All = []feed.Service{
		Reddit,
		DvachCatalog,
		DvachThread,
	}

	Map = make(map[string]feed.Service)
)

func init() {
	for _, service := range All {
		Map[service().Service()] = service
	}
}
