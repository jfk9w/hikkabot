package receiver

import (
	"context"

	"github.com/jfk9w-go/flu/syncf"

	"github.com/jfk9w-go/flu"
)

type Media struct {
	MIMEType string
	Input    flu.Input
}

type MediaRef = syncf.Ref[*Media]

type MediaFunc func(ctx context.Context) (*Media, error)

func (f MediaFunc) Media(ctx context.Context) (*Media, error) {
	return f(ctx)
}

type MediaError struct {
	E error
}

func (e MediaError) Get(ctx context.Context) (media *Media, err error) {
	err = e.E
	return
}

func (e MediaError) Error() string {
	return e.E.Error()
}

func (e MediaError) Cause() error {
	return e.E
}

func (e MediaError) Unwrap() error {
	return e.E
}

type Interface interface {
	SendText(ctx context.Context, text string) error
	SendMedia(ctx context.Context, ref MediaRef, caption string) error
}
