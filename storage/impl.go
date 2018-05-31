package storage

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/unit"
	"github.com/pkg/errors"
)

type T struct {
	http httpx.Client
	path string
	sem  chan unit.T
	log  logrus.Logger
}

func Configure(config Config, httpConfig *httpx.Config) *T {
	path, err := misc.Expand(config.Path)
	if err != nil {
		panic(err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		panic(err)
	}

	logger := "dvach"
	if config.Logger != nil {
		logger = *config.Logger
	}

	return &T{
		http: httpx.Configure(httpConfig),
		path: path,
		sem:  make(chan unit.T, config.Concurrency),
		log:  logrus.GetLogger(logger),
	}
}

func (def *T) Download(ustr string) error {
	path, err := def.locate(ustr)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		def.log.Debugf("File %s found, skipping download")
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	part := path + ".part"
	os.Remove(part)
	out, err := os.Create(part)
	if err != nil {
		return err
	}

	def.sem <- unit.Value

	resp, err := def.http.Client.Get(ustr)
	if err != nil {
		def.log.Warningf("Unable to download %s: %s", ustr, err)
		out.Close()
		os.Remove(part)
		return err
	}

	<-def.sem

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out.Close()
		os.Remove(part)
		return errors.Errorf("status code %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		out.Close()
		os.Remove(part)
		return err
	}

	out.Close()
	return os.Rename(part, path)
}

func (def *T) Remove(ustr string) error {
	path, err := def.locate(ustr)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

func (def *T) Path(ustr string) (string, error) {
	path, err := def.locate(ustr)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", errors.New("absent")
}

func (def *T) locate(ustr string) (string, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return "", err
	}

	host := strings.Replace(u.Host, ":", "_", -1)
	path := strings.Replace(u.Path, ":", "_", -1)
	return filepath.Join(def.path, host, path), nil
}
