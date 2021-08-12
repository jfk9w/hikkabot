package feed

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

const (
	ContentTypeHeader   = "Content-Type"
	ContentLengthHeader = "Content-Length"
)

type MediaMetadata struct {
	Size     int64
	MIMEType string
}

func (m *MediaMetadata) Handle(resp *http.Response) error {
	return m.Fill(resp.Header.Get(ContentTypeHeader), resp.Header.Get(ContentLengthHeader))
}

func (m *MediaMetadata) Fill(contentType, contentLength string) error {
	var err error
	m.MIMEType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return errors.Wrapf(err, "invalid %s: %s", ContentTypeHeader, contentType)
	}

	m.Size, err = strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		m.Size = UnknownSize
	}

	return nil
}

type MediaClient interface {
	Metadata(ctx context.Context, url string) (*MediaMetadata, error)
	Contents(ctx context.Context, url string, out flu.Output) error
}

type DefaultMediaClient struct {
	main, fallback MediaClient
	retries        int
}

func NewMediaClient(client *fluhttp.Client, curl string, retries int) DefaultMediaClient {
	main := StdLibClient{client}
	var fallback MediaClient
	if curl != "" {
		fallback = CURL{curl}
	} else {
		fallback = main
	}
	return DefaultMediaClient{main, fallback, retries}
}

func (c DefaultMediaClient) Metadata(ctx context.Context, url string) (*MediaMetadata, error) {
	var metadata *MediaMetadata
	return metadata, c.retry(ctx, url, "head", func(client MediaClient) error {
		var err error
		metadata, err = client.Metadata(ctx, url)
		return err
	})
}

func (c DefaultMediaClient) Contents(ctx context.Context, url string, out flu.Output) error {
	return c.retry(ctx, url, "download", func(client MediaClient) error {
		return client.Contents(ctx, url, out)
	})
}

func (c DefaultMediaClient) retry(ctx context.Context, url string, op string, body func(MediaClient) error) error {
	var client MediaClient = c.main
	if err := body(client); err != nil {
		for i := 0; i < c.retries; i++ {
			logrus.WithField("media", url).Warnf("%s (retry %d): %s", op, i, err)
			if !IsNetworkError(err) || i == 3 {
				client = c.fallback
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(3*i*i) * time.Second):
			}

			if err = body(client); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
			} else {
				return nil
			}
		}

		return err
	}

	return nil
}

func IsNetworkError(err error) bool {
	for {
		if _, ok := err.(*net.OpError); ok {
			return true
		} else if wrapped, ok := err.(interface{ Unwrap() error }); ok {
			err = wrapped.Unwrap()
		} else {
			return false
		}
	}
}

type StdLibClient struct {
	*fluhttp.Client
}

func (s StdLibClient) Metadata(ctx context.Context, url string) (*MediaMetadata, error) {
	m := new(MediaMetadata)
	return m, s.HEAD(url).Context(ctx).Execute().
		CheckStatus(http.StatusOK).
		HandleResponse(m).
		Error
}

func (s StdLibClient) Contents(ctx context.Context, url string, out flu.Output) error {
	return s.GET(url).Context(ctx).Execute().
		CheckStatus(http.StatusOK).
		DecodeBodyTo(out).
		Error
}

type CURL struct {
	Binary string
}

func (c CURL) Metadata(ctx context.Context, url string) (*MediaMetadata, error) {
	stderr := new(bytes.Buffer)
	if err := c.executeAndCheckStatus(ctx, url, stderr,
		"-I",              // HEAD
		"-o", "/dev/null", // redirect output to /dev/null
		"-D", "/dev/stderr", // dump headers to stderr
	); err != nil {
		return nil, err
	}

	m := new(MediaMetadata)
	contentType, contentLength := "", ""
	scanner := bufio.NewScanner(stderr)
scan:
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, ContentTypeHeader):
			contentType = line[len(ContentTypeHeader)+2:]
			if contentLength != "" {
				break scan
			}
		case strings.HasPrefix(line, ContentLengthHeader):
			contentLength = line[len(ContentLengthHeader)+2:]
			if contentType != "" {
				break scan
			}
		}
	}

	return m, m.Fill(contentType, contentLength)
}

func (c CURL) Contents(ctx context.Context, url string, out flu.Output) error {
	w, err := out.Writer()
	if err != nil {
		return errors.Wrap(err, "write")
	}

	defer flu.Close(w)
	err = c.executeAndCheckStatus(ctx, url, w,
		"-o", "/dev/stderr", // redirect output to stderr
	)

	return err
}

func (c CURL) executeAndCheckStatus(ctx context.Context, url string, stderr io.Writer, args ...string) error {
	args = append(args,
		"-s",                          // silent
		"-S",                          // show error on fails
		"-L",                          // follow redirects
		"--write-out", "%{http_code}", // write response status code to stdout)
		url,
	)

	cmd := exec.CommandContext(ctx, c.Binary, args...)
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "run process")
	}

	code, err := ioutil.ReadAll(stdout)
	if err != nil {
		return errors.Wrap(err, "read stdout")
	}

	if string(code) != "200" {
		return fluhttp.StatusCodeError{ResponseBody: code}
	}

	return nil
}
