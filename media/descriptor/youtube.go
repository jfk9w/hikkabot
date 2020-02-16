package descriptor

import (
	"io"
	"net/http"
	_url "net/url"
	"strconv"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/pkg/errors"
)

type YoutubePlayerResponse struct {
	StreamingData struct {
		Formats []YoutubeStreamingDataFormat `json:"formats"`
	} `json:"streamingData"`
}

type YoutubeStreamingDataFormat struct {
	ContentLength string `json:"contentLength"`
	MIMEType      string `json:"mimeType"`
	URL           string `json:"url"`
	Cipher        string `json:"cipher"`
}

func (f YoutubeStreamingDataFormat) GetURL() (string, error) {
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

type YoutubeVideoInfo struct {
	formats []YoutubeStreamingDataFormat
}

func (vi *YoutubeVideoInfo) ReadFrom(r io.Reader) error {
	resp := flu.PlainText("")
	if err := resp.ReadFrom(r); err != nil {
		return errors.Wrap(err, "read response")
	}

	info, err := _url.ParseQuery(resp.Value)
	if err != nil {
		return errors.Wrap(err, "parse query")
	}

	playerResponse := new(YoutubePlayerResponse)
	err = flu.JSON(playerResponse).ReadFrom(strings.NewReader(info.Get("player_response")))
	if err != nil {
		return errors.Errorf("no player_response in info: %v", info)
	}

	vi.formats = playerResponse.StreamingData.Formats
	return nil
}

type Youtube struct {
	Client  *flu.Client
	URL     string
	MaxSize int64
}

func (d *Youtube) Metadata() (*media.Metadata, error) {
	url, err := _url.Parse(d.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parse URL")
	}

	var id string
	switch {
	case strings.Contains(url.Host, "youtube.com"):
		id = url.Query().Get("v")
	case strings.Contains(url.Host, "youtu.be"):
		id = strings.Trim(url.Path, "/")
	default:
		return media.URLDescriptor{
			Client: d.Client,
			URL:    d.URL,
		}.Metadata()
	}

	if id == "" {
		return nil, errors.New("failed to find id in URL")
	}

	info := new(YoutubeVideoInfo)
	if err = mediator.CommonClient.
		GET("http://youtube.com/get_video_info?video_id=" + id).
		Execute().
		CheckStatusCode(http.StatusOK).
		Read(info).
		Error; err != nil {
		return nil, errors.Wrap(err, "get_video_info")
	}

	var (
		maxSize    int64 = -1
		bestFormat YoutubeStreamingDataFormat
	)

	for _, format := range info.formats {
		size, err := strconv.ParseInt(format.ContentLength, 10, 64)
		if err != nil {
			continue
		}

		if size > 0 && size > maxSize && (d.MaxSize <= 0 || size < d.MaxSize) {
			maxSize = size
			bestFormat = format
		}
	}

	if maxSize < 0 {
		return nil, errors.Errorf("failed to find suitable video in: %+v", info.formats)
	}

	realURL, err := bestFormat.GetURL()
	if err != nil {
		return nil, errors.Wrap(err, "parse best streaming format URL")
	}

	d.URL = realURL
	return &media.Metadata{
		URL:      realURL,
		MIMEType: bestFormat.MIMEType,
		Size:     maxSize,
	}, nil
}

func (d *Youtube) Reader() (io.Reader, error) {
	return d.Client.GET(d.URL).Execute().
		CheckStatusCode(http.StatusOK).
		Reader()
}
