package httpx

import (
	"encoding/json"
	"io"
	"net/url"
	"os"

	"io/ioutil"

	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
)

// Input encapsulates an input (query parameters, files, etc.).
type Input interface {
	write(io.Writer) error
}

// POST encapsulates an Input with a content type.
type POST interface {
	Input
	contentType() string
}

// Output encapsulates an output (JSON, files).
type Output interface {
	read(io.Reader) error
}

// JSON is a request or response "body".
type JSON struct {
	Value interface{}
}

func (j *JSON) read(reader io.Reader) error {
	var data, err = ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, j.Value)
	if err != nil {
		return InvalidFormat{data, err}
	}

	return nil
}

func (j *JSON) write(writer io.Writer) error {
	data, err := json.Marshal(j.Value)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

func (j JSON) contentType() string {
	return "application/json"
}

// File is an input or output file.
type File struct {
	Path string
	Size int64
}

func (f *File) read(reader io.Reader) error {
	err := fsx.EnsureParent(f.Path)
	if err != nil {
		return err
	}

	file, err := os.Create(f.Path)
	if err != nil {
		return err
	}

	f.Size, err = io.Copy(file, reader)
	file.Close()
	return err
}

func (f *File) write(writer io.Writer) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}

	f.Size, err = io.Copy(writer, file)
	file.Close()
	return err
}

// Delete removes the file from the file system.
func (f *File) Delete() error {
	return os.Remove(f.Path)
}

// Params is query parameters.
type Params url.Values

// Set calls url.Values and returns the Params.
func (p Params) Set(key, value string) Params {
	url.Values(p).Set(key, value)
	return p
}

// Add calls url.Values and returns the Params.
func (p Params) Add(key string, values ...string) Params {
	for _, value := range values {
		url.Values(p).Add(key, value)
	}

	return p
}

// Encode calls url.Values.
func (p Params) Encode() string {
	return url.Values(p).Encode()
}

func (p Params) write(writer io.Writer) error {
	_, err := writer.Write([]byte(p.Encode()))
	return err
}

func (p Params) contentType() string {
	return "application/x-www-form-urlencoded"
}

// Multipart encapsulates files for multipart/form-data requests.
type Multipart map[string]*File
