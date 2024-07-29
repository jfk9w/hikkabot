package receiver

import (
	"context"

	"github.com/pkg/errors"
)

type MediaCaption struct {
	Ref     MediaRef
	Caption string
}

type Buffer struct {
	Pages []string
	Media []MediaCaption
}

func NewBuffer() *Buffer {
	return &Buffer{
		Pages: make([]string, 0),
		Media: make([]MediaCaption, 0),
	}
}

func (b *Buffer) SendText(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}

	b.Pages = append(b.Pages, text)
	return nil
}

func (b *Buffer) SendMedia(ctx context.Context, ref MediaRef, caption string) error {
	b.Media = append(b.Media, MediaCaption{ref, caption})
	return nil
}

func (b *Buffer) Flush(ctx context.Context, receiver Interface) error {
	for _, page := range b.Pages {
		if err := receiver.SendText(ctx, page); err != nil {
			return errors.Wrap(err, "send text")
		}
	}

	for _, media := range b.Media {
		if err := receiver.SendMedia(ctx, media.Ref, media.Caption); err != nil {
			return errors.Wrap(err, "send media")
		}
	}

	return nil
}
