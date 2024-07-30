package mediator

import (
	"crypto/md5"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/jfk9w/hikkabot/v4/internal/feed"

	"github.com/corona10/goimagehash"
	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
)

type readImageFunc func(io.Reader) (image.Image, error)

var imageTypes = map[string]readImageFunc{
	"image/jpeg": jpeg.Decode,
	"image/png":  png.Decode,
	"image/bmp":  bmp.Decode,
}

func hashImage(blob flu.Input, hash *feed.MediaHash, readImage readImageFunc) error {
	reader, err := blob.Reader()
	if err != nil {
		return errors.Wrap(err, "open blob")
	}

	defer flu.CloseQuietly(reader)
	img, err := readImage(reader)
	if err != nil {
		return errors.Wrap(err, "read image")
	}

	dhash, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return errors.Wrap(err, "get diff hash")
	}

	hash.Type = "dhash"
	hash.Value = fmt.Sprintf("%x", dhash.GetHash())
	return nil
}

func hashAny(blob flu.Input, hash *feed.MediaHash) error {
	md5 := md5.New()
	if _, err := flu.Copy(blob, flu.IO{W: md5}); err != nil {
		return errors.Wrap(err, "get md5 hash")
	}

	hash.Type = "md5"
	hash.Value = fmt.Sprintf("%x", md5.Sum(nil))
	return nil
}
