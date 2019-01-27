// Package jsonx provides various JSON readers and writers
package jsonx

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
)

func Read(reader io.Reader, value interface{}) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}

func ReadFile(path string, value interface{}) error {
	var err error
	path, err = fsx.Path(path)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()
	return Read(file, value)
}

func Write(writer io.Writer, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

func WriteFile(path string, value interface{}) error {
	var err error
	path, err = fsx.Path(path)
	if err != nil {
		return err
	}

	err = fsx.EnsureParent(path)
	if err != nil && !os.IsExist(err) {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()
	return Write(file, value)
}
