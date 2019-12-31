package reddit

import (
	"html"
	"net/url"
	"strconv"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type YoutubeMediaResolver struct{}

func (r YoutubeMediaResolver) Resolve(http *flu.Client, thing *Thing) (*ResolvedMedia, error) {
	return r.ResolveURL(http, html.UnescapeString(thing.Data.URL))
}

func (r YoutubeMediaResolver) ResolveURL(http *flu.Client, rawurl string) (*ResolvedMedia, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}
	var id string
	if strings.Contains(u.Host, "youtube.com") {
		id = u.Query().Get("v")
	} else {
		id = strings.Trim(u.Path, "/")
	}
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
	playerResponse := new(youtubePlayerResponse)
	err = flu.Read(flu.PipeOut(flu.PlainText(info.Get("player_response"))), flu.JSON(playerResponse))
	if err != nil {
		return nil, errors.Wrap(err, "parse player_response")
	}
	var maxSize int64 = -1
	var bestFormat youtubeStreamingDataFormat
	for _, format := range playerResponse.StreamingData.Formats {
		contentLength, err := format.contentLength()
		if err != nil {
			continue
		}
		if contentLength > 0 && contentLength > maxSize && contentLength < 50<<20 {
			maxSize = contentLength
			bestFormat = format
		}
	}
	if maxSize < 0 {
		return nil, errors.Errorf("failed to find suitable video")
	}
	extension, err := bestFormat.extension()
	if err != nil {
		return nil, errors.Wrap(err, "parse extension")
	}
	return &ResolvedMedia{bestFormat.URL, extension}, nil
}

type youtubePlayerResponse struct {
	StreamingData struct {
		Formats []youtubeStreamingDataFormat `json:"formats"`
	} `json:"streamingData"`
}

type youtubeStreamingDataFormat struct {
	ContentLength string `json:"contentLength"`
	MimeType      string `json:"mimeType"`
	URL           string `json:"url"`
}

func (f youtubeStreamingDataFormat) contentLength() (int64, error) {
	return strconv.ParseInt(f.ContentLength, 10, 64)
}

func (f youtubeStreamingDataFormat) extension() (string, error) {
	slash := strings.Index(f.MimeType, "/")
	if slash > 0 {
		semicolon := strings.Index(f.MimeType, ";")
		if semicolon == 0 {
			semicolon = len(f.MimeType)
		}
		return f.MimeType[slash+1 : semicolon], nil
	} else {
		return "", errors.Errorf("failed to parse mimeType: %s", f.MimeType)
	}
}
