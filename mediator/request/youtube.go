package request

import (
	"io"
	"net/http"
	_url "net/url"
	"strconv"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/pkg/errors"
)

type youtubePlayerResponse struct {
	StreamingData struct {
		Formats []youtubeStreamingDataFormat `json:"formats"`
	} `json:"streamingData"`
}

type youtubeStreamingDataFormat struct {
	ContentLength string `json:"contentLength"`
	MimeType      string `json:"mimeType"`
	URL           string `json:"url"`
	Cipher        string `json:"cipher"`
}

func (f youtubeStreamingDataFormat) parseSize() (int64, error) {
	return strconv.ParseInt(f.ContentLength, 10, 64)
}

func (f youtubeStreamingDataFormat) parseFormat() (string, error) {
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

func (f youtubeStreamingDataFormat) parseURL() (string, error) {
	if f.URL != "" {
		return f.URL, nil
	}
	cipher, err := _url.ParseQuery(f.Cipher)
	if err != nil {
		return "", errors.Wrap(err, "parse query")
	}
	url := cipher.Get("url")
	if url == "" {
		return "", errors.Errorf("no URL in cipher: %v", cipher)
	}
	return url, nil
}

type youtubeVideoInfo struct {
	formats []youtubeStreamingDataFormat
}

func (vi *youtubeVideoInfo) ReadFrom(r io.Reader) error {
	resp := flu.PlainText("")
	if err := resp.ReadFrom(r); err != nil {
		return errors.Wrap(err, "read response")
	}
	info, err := _url.ParseQuery(resp.Value)
	if err != nil {
		return errors.Wrap(err, "parse query")
	}
	playerResponse := new(youtubePlayerResponse)
	err = flu.JSON(playerResponse).ReadFrom(strings.NewReader(info.Get("player_response")))
	if err != nil {
		return errors.Errorf("no player_response in info: %v", info)
	}
	vi.formats = playerResponse.StreamingData.Formats
	return nil
}

type Youtube struct {
	URL     string
	MaxSize int64
	realURL string
	body    io.Reader
}

func (r *Youtube) Metadata() (*mediator.Metadata, error) {
	url, err := _url.Parse(r.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parse URL")
	}
	var id string
	switch {
	case strings.Contains(url.Host, "youtube.com"):
		id = url.Query().Get("v")
	case strings.Contains(url.Host, "youtu.be"):
		id = strings.Trim(url.Path, "/")
	}
	if id == "" {
		return nil, errors.New("failed to find id in URL")
	}
	info := new(youtubeVideoInfo)
	err = flu.DefaultClient.
		GET("http://youtube.com/get_video_info?video_id=" + id).
		Execute().
		CheckStatusCode(http.StatusOK).
		Read(info).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "get_video_info")
	}
	var maxSize int64 = -1
	var bestFormat youtubeStreamingDataFormat
	for _, sdf := range info.formats {
		size, err := sdf.parseSize()
		if err != nil {
			continue
		}
		if size > 0 && size > maxSize && size < r.MaxSize {
			maxSize = size
			bestFormat = sdf
		}
	}
	if maxSize < 0 {
		return nil, errors.Errorf("failed to find suitable video in: %+v", info.formats)
	}
	format, err := bestFormat.parseFormat()
	if err != nil {
		return nil, errors.Wrap(err, "parse best streaming format")
	}
	realURL, err := bestFormat.parseURL()
	if err != nil {
		return nil, errors.Wrap(err, "parse best streaming format URL")
	}
	r.realURL = realURL
	return &mediator.Metadata{
		URL:    realURL,
		Format: format,
		Size:   maxSize,
	}, nil
}

func (r *Youtube) Reader() (io.Reader, error) {
	return flu.DefaultClient.
		GET(r.realURL).
		Execute().
		CheckStatusCode(http.StatusOK).
		Reader()
}
