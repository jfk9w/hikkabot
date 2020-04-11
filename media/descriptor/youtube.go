package descriptor

import (
	"io"
	"net/http"
	_url "net/url"
	"strconv"
	"strings"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/media"
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

func (vi *YoutubeVideoInfo) DecodeFrom(r io.Reader) error {
	resp := &flu.PlainText{""}
	if err := resp.DecodeFrom(r); err != nil {
		return errors.Wrap(err, "read response")
	}

	info, err := _url.ParseQuery(resp.Value)
	if err != nil {
		return errors.Wrap(err, "parse query")
	}

	playerResponse := new(YoutubePlayerResponse)
	err = flu.JSON{playerResponse}.DecodeFrom(strings.NewReader(info.Get("player_response")))
	if err != nil {
		return errors.Errorf("no player_response in info: %v", info)
	}

	vi.formats = playerResponse.StreamingData.Formats
	return nil
}

type Youtube struct {
	Client fluhttp.Client
	ID     string
	URL    string
}

func (d *Youtube) Metadata(maxSize int64) (*media.Metadata, error) {
	info := new(YoutubeVideoInfo)
	if err := d.Client.
		GET("http://youtube.com/get_video_info?video_id=" + d.ID).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBody(info).
		Error; err != nil {
		return nil, errors.Wrap(err, "get_video_info")
	}

	var (
		bestSize   int64 = -1
		bestFormat YoutubeStreamingDataFormat
	)

	for _, format := range info.formats {
		size, err := strconv.ParseInt(format.ContentLength, 10, 64)
		if err != nil {
			continue
		}

		if size > 0 && size > bestSize && (maxSize <= 0 || size < maxSize) {
			bestSize = size
			bestFormat = format
		}
	}

	if maxSize < 0 {
		return nil, errors.Errorf("failed to find suitable video in: %+v", info.formats)
	}

	url, err := bestFormat.GetURL()
	if err != nil {
		return nil, errors.Wrap(err, "parse best streaming format URL")
	}

	d.URL = url
	return &media.Metadata{
		URL:      url,
		Size:     bestSize,
		MIMEType: bestFormat.MIMEType,
	}, nil
}

func (d *Youtube) Reader() (io.Reader, error) {
	return d.Client.GET(d.URL).Execute().
		CheckStatus(http.StatusOK).
		Reader()
}
