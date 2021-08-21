package feed

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/jfk9w-go/telegram-bot-api/ext/blob"

	"github.com/corona10/goimagehash"
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
)

type ReadImageFunc func(io.Reader) (image.Image, error)

type DefaultMediaDedup struct {
	BlobStorage Storage
	Clock       flu.Clock
}

func (d DefaultMediaDedup) Check(ctx context.Context, feedID telegram.ID, url, mimeType string, blob blob.Blob) error {
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
		if _, err := flu.Copy(blob, flu.IO{W: md5Hash}); err != nil {
			return errors.Wrap(err, "md5 hash")
		}

		hash = md5Hash.Sum(nil)
	}

	now := d.Clock.Now()
	return d.BlobStorage.Check(ctx, &BlobHash{
		FeedID:    feedID,
		URL:       url,
		Type:      hashType,
		Hash:      fmt.Sprintf("%x", hash),
		FirstSeen: now,
		LastSeen:  now,
	})
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
