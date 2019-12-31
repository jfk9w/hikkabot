package reddit

import (
	"html"
	"log"
	"net/url"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type YoutubeMediaResolver struct{}

func (y YoutubeMediaResolver) Resolve(http *flu.Client, thing *Thing) (*ResolvedMedia, error) {
	return y.ResolveURL(http, html.UnescapeString(thing.Data.URL))
}

func (y YoutubeMediaResolver) ResolveURL(http *flu.Client, rawurl string) (*ResolvedMedia, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}
	id := u.Query().Get("v")
	if id == "" {
		return nil, errors.New("failed to find id in url")
	}
	resp := new(flu.PlainTextBody)
	err = http.NewRequest().
		Resource("http://youtube.com/get_video_info?video_id=" + id).
		GET().
		Send().
		Read(resp).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "get video info")
	}
	info, err := url.ParseQuery(resp.Value)
	if err != nil {
		return nil, errors.Wrap(err, "parse info")
	}
	fmtStreamMap, err := url.ParseQuery(info.Get("url_encoded_fmt_stream_map"))
	if err != nil {
		return nil, errors.Wrap(err, "parse url_encoded_fmt_stream_map")
	}
	log.Printf("\n%v", fmtStreamMap)
	return nil, nil
}
