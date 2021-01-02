package feed

import (
	"context"
	"crypto/md5"
	"image"
	"image/jpeg"
	"image/png"
	"io"

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
	reader, err := blob.Reader()
	if err != nil {
		return errors.Wrap(err, "read")
	}

	defer flu.ReaderCloser{Reader: reader}.Close()

	var (
		hashType string
		hash     []byte
	)

	var readImage ReadImageFunc
	switch mimeType {
	case "image/jpeg":
		readImage = jpeg.Decode
	case "image/png":
		readImage = png.Decode
	case "image/bmp":
		readImage = bmp.Decode
	}

	if readImage != nil {
		hashType, hash, err = d.hashImage(readImage, reader)
		if err != nil {
			return err
		}
	} else {
		hashType = "md5"
		md5Hash := md5.New()
		if err := flu.Copy(blob, flu.IO{W: md5Hash}); err != nil {
			return errors.Wrap(err, "md5 hash")
		}

		hash = md5Hash.Sum(nil)
	}

	return d.Hashes.Check(ctx, feedID, url, hashType, hash)
}

func (d DefaultMediaDedup) hashImage(readImage func(io.Reader) (image.Image, error), reader io.Reader) (string, []byte, error) {
	img, err := readImage(reader)
	if err != nil {
		return "", nil, errors.Wrap(err, "read image")
	}

	hash, err := goimagehash.ExtAverageHash(img, 16, 16)
	if err != nil {
		return "", nil, errors.Wrap(err, "compute image hash")
	}

	buf := flu.NewBuffer()
	if err := hash.Dump(buf); err != nil {
		return "", nil, errors.Wrap(err, "image hash dump")
	}

	return "ahash", buf.Bytes(), nil
}
