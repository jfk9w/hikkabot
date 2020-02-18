package descriptor

import (
	"fmt"
	_url "net/url"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/media"
)

type Factory func(*flu.Client, *_url.URL) media.Descriptor

func (f Factory) RegisterFor(domains ...string) {
	for _, domain := range domains {
		if _, ok := SupportedDomains[domain]; ok {
			panic(fmt.Errorf("domain %s is already registered", domain))
		}

		SupportedDomains[domain] = f
	}
}

func init() {
	Factory(func(client *flu.Client, url *_url.URL) media.Descriptor {
		return &Gfycat{Client: client, URL: url.String()}
	}).RegisterFor("gfycat.com", "www.gfycat.com")
	Factory(func(client *flu.Client, url *_url.URL) media.Descriptor {
		return &Imgur{Client: client, URL: url.String()}
	}).RegisterFor("imgur.com", "www.imgur.com")
	Factory(func(client *flu.Client, url *_url.URL) media.Descriptor {
		rawurl := url.String()
		if strings.Contains(rawurl, ".gifv") {
			rawurl = strings.Replace(rawurl, ".gifv", ".mp4", 1)
		}
		return &media.URLDescriptor{
			Client: client,
			URL:    rawurl,
		}
	}).RegisterFor("i.imgur.com")
	Factory(func(client *flu.Client, url *_url.URL) media.Descriptor {
		id := url.Query().Get("v")
		return &Youtube{Client: client, ID: id}
	}).RegisterFor("youtube.com", "www.youtube.com")
	Factory(func(client *flu.Client, url *_url.URL) media.Descriptor {
		id := strings.Trim(url.Path, "/")
		return &Youtube{Client: client, ID: id}
	}).RegisterFor("youtu.be")
}

var SupportedDomains = make(map[string]Factory)

func From(client *flu.Client, rawurl string) (media.Descriptor, error) {
	url, err := _url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	if factory, ok := SupportedDomains[url.Host]; ok {
		return factory(client, url), nil
	} else {
		return media.URLDescriptor{Client: client, URL: rawurl}, nil
	}
}
