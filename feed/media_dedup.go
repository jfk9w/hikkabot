package feed

import (
	"context"
	"crypto/md5"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"

	"github.com/corona10/goimagehash"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
)

type ReadImageFunc func(io.Reader) (image.Image, error)

type DefaultMediaDedup struct {
	Hashes Hashes
}

func (d DefaultMediaDedup) Check(ctx context.Context, feedID ID, url, mimeType string, blob format.Blob) error {
	var readImage ReadImageFunc
	switch mimeType {
	case "image/jpeg":
		readImage = jpeg.Decode
	case "image/png":
		readImage = png.Decode
	case "image/bmp":
		readImage = bmp.Decode
	}

	reader, err := blob.Reader()
	if err != nil {
		return errors.Wrap(err, "read")
	}

	defer flu.ReaderCloser{Reader: reader}.Close()

	var hashStr string
	if readImage != nil {
		img, err := readImage(reader)
		if err != nil {
			return errors.Wrap(err, "read image")
		}

		hash, err := goimagehash.ExtAverageHash(img, 16, 16)
		if err == nil {
			hashStr = hash.ToString()
		} else {
			log.Printf("[dedup %d] failed to compute image hash for %s: %s", feedID, url, err)
		}
	}

	if hashStr == "" {
		hash := md5.New()
		if err := flu.Copy(blob, flu.IO{W: hash}); err != nil {
			return errors.Wrap(err, "hash")
		}

		hashStr = fmt.Sprintf("%x", hash.Sum(nil))
	}

	return d.Hashes.Check(ctx, feedID, url, hashStr)
}
