// Package fsx provides various filesystem-related utilities.
package fsx

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/segmentio/ksuid"
)

// Substitutes all environmental variables and returns an absolute path.
func Path(path string) (string, error) {
	return filepath.Abs(os.ExpandEnv(path))
}

// Ensures the parent directory and intermediates exist.
func EnsureParent(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, os.ModePerm)
}

func Read(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	return ioutil.ReadAll(file)
}

func Join(elems ...string) string {
	return filepath.Join(elems...)
}

func Base(path string) string {
	return filepath.Base(path)
}

// File is not created, you need to do this yourself
func TempFile(dir string) string {
	return Join(dir, ksuid.New().String())
}
