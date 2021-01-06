package feed

import (
	"context"
	"crypto/md5"
	"encoding/binary"
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
	BlobStorage BlobStorage
}

func (d DefaultMediaDedup) Check(ctx context.Context, feedID ID, url, mimeType string, blob format.Blob) error {
	reader, err := blob.Reader()
	if err != nil {
		return errors.Wrap(err, "read")
	}

	defer flu.Close(reader)

	var (
		hashType  string
		hash      []byte
		readImage ReadImageFunc
	)

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

	return d.BlobStorage.CheckBlob(ctx, feedID, url, hashType, hash)
}

func (d DefaultMediaDedup) hashImage(readImage ReadImageFunc, reader io.Reader) (string, []byte, error) {
	img, err := readImage(reader)
	if err != nil {
		return "", nil, errors.Wrap(err, "read image")
	}

	hash, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return "", nil, errors.Wrap(err, "compute image hash")
	}

	buf := make([]byte, hash.Bits()/8)
	binary.LittleEndian.PutUint64(buf, hash.GetHash())
	return "dhash", buf, nil
}
