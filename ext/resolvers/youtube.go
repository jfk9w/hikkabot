package resolvers

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jfk9w/hikkabot/core/media"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

type YouTube media.Ref

func (r *YouTube) Unmask() *media.Ref {
	return (*media.Ref)(r)
}

func (r *YouTube) GetClient(defaultClient *fluhttp.Client) *fluhttp.Client {
	return defaultClient
}

func (r *YouTube) Resolve(ctx context.Context, client *fluhttp.Client, rawURL string, maxSize int64) (string, error) {
	url, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.Wrapf(err, "parse url: %s", rawURL)
	}

	var id string
	switch url.Host {
	case "youtube.com", "www.youtube.com":
		id = url.Query().Get("v")
	case "youtu.be":
		id = strings.Trim(url.Path, "/")
	}

	info := new(videoInfo)
	if err := client.
		GET("https://youtube.com/get_video_info?video_id=" + id).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBody(info).
		Error; err != nil {
		return "", errors.Wrap(err, "get_video_info")
	}

	var (
		bestSize   int64 = -1
		bestFormat streamingDataFormat
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
		return "", errors.Errorf("failed to find suitable video in: %+v", info.formats)
	}

	rawURL, err = bestFormat.GetURL()
	if err != nil {
		return "", errors.Wrap(err, "parse best streaming format URL")
	}

	r.Unmask().Size = bestSize
	r.Unmask().MIMEType = strings.Split(bestFormat.MIMEType, ";")[0]

	return rawURL, nil
}

type streamingDataFormat struct {
	ContentLength string `json:"contentLength"`
	MIMEType      string `json:"mimeType"`
	URL           string `json:"url"`
	Cipher        string `json:"cipher"`
}

func (f streamingDataFormat) GetURL() (string, error) {
	if f.URL != "" {
		return f.URL, nil
	}

	cipher, err := url.ParseQuery(f.Cipher)
	if err != nil {
		return "", errors.Wrap(err, "parse query")
	}

	url := cipher.Get("url")
	if url == "" {
		return "", errors.Errorf("no URL in cipher: %v", cipher)
	}

	return url, nil
}

type videoInfo struct {
	formats []streamingDataFormat
}

func (vi *videoInfo) DecodeFrom(r io.Reader) error {
	resp := &flu.PlainText{Value: ""}
	if err := resp.DecodeFrom(r); err != nil {
		return errors.Wrap(err, "read response")
	}

	info, err := url.ParseQuery(resp.Value)
	if err != nil {
		return errors.Wrap(err, "parse query")
	}

	presp := new(struct {
		StreamingData struct {
			Formats []streamingDataFormat `json:"formats"`
		} `json:"streamingData"`
	})

	err = flu.JSON{Value: presp}.DecodeFrom(strings.NewReader(info.Get("player_response")))
	if err != nil {
		return errors.Errorf("no player_response in info: %v", info)
	}

	vi.formats = presp.StreamingData.Formats
	return nil
}
