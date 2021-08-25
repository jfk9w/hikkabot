package media

import (
	"context"
	"crypto/md5"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/corona10/goimagehash"
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/core/feed"
	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
)

type readImageFunc func(io.Reader) (image.Image, error)

var imageTypes = map[string]readImageFunc{
	"image/jpeg": jpeg.Decode,
	"image/png":  png.Decode,
	"image/bmp":  bmp.Decode,
}

type Deduplicator struct {
	flu.Clock
	HashStorage
}

func (d *Deduplicator) Check(ctx context.Context,
	feedID telegram.ID, url, mimeType string, blob feed.Blob) (
	bool, error) {

	if d == nil {
		return true, nil
	}

	now := d.Now()
	hash := &Hash{
		FeedID:    feedID,
		URL:       url,
		FirstSeen: now,
		LastSeen:  now,
	}

	var err error
	if readImage, ok := imageTypes[mimeType]; ok {
		err = hashImage(blob, hash, readImage)
	} else {
		err = hashAny(blob, hash)
	}

	if err != nil {
		return false, err
	}

	return d.HashStorage.Check(ctx, hash)
}

func hashImage(blob feed.Blob, hash *Hash, readImage readImageFunc) error {
	reader, err := blob.Reader()
	if err != nil {
		return errors.Wrap(err, "open blob")
	}

	defer flu.Close(reader)
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

func hashAny(blob feed.Blob, hash *Hash) error {
	md5 := md5.New()
	if _, err := flu.Copy(blob, flu.IO{W: md5}); err != nil {
		return errors.Wrap(err, "get md5 hash")
	}

	hash.Type = "md5"
	hash.Value = fmt.Sprintf("%x", md5.Sum(nil))
	return nil
}
