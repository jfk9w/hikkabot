package output

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/jfk9w-go/flu/syncf"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"golang.org/x/exp/utf8string"
)

type Paged struct {
	Receiver       receiver.Interface
	overflown      bool
	prefix, suffix string
	curr           strings.Builder
	currSize       int
	currCount      int
}

func (o *Paged) IsOverflown() bool {
	return o.overflown
}

func (o *Paged) UpdatePrefix(update func(prefix string) string) {
	o.prefix = update(o.prefix)
}

func (o *Paged) UpdateSuffix(update func(suffix string) string) {
	o.suffix = update(o.suffix)
}

func (o *Paged) Write(text string) {
	if o.overflown {
		return
	}

	o.curr.WriteString(text)
	o.currSize += utf8.RuneCountInString(text)
}

func (o *Paged) WriteBreakable(ctx context.Context, text string) error {
	if o.overflown {
		return nil
	}

	utext := utf8string.NewString(text)
	length := utext.RuneCount()
	offset := 0
	capacity := o.PageCapacity(ctx)
	end := offset + capacity
	for end < length {
		nextOffset := end
	search:
		for i := end; i >= 0; i-- {
			switch utext.At(i) {
			case '\n', ' ', '\t', '\v':
				end, nextOffset = i, i+1
				break search
			case ',', '.', ':', ';':
				end, nextOffset = i+1, i+1
				break search
			default:
				continue
			}
		}

		o.Write(trim(utext.Slice(offset, end)))
		if err := o.BreakPage(ctx); err != nil {
			return err
		}

		if o.overflown {
			return nil
		}

		offset = nextOffset
		capacity = o.PageCapacity(ctx)
		end = offset + capacity
	}

	o.Write(utext.Slice(offset, length))
	return nil
}

func (o *Paged) WriteUnbreakable(ctx context.Context, text string) error {
	if o.overflown {
		return nil
	}

	length := utf8.RuneCountInString(text)
	if length > o.PageCapacity(ctx) {
		if err := o.BreakPage(ctx); err != nil {
			return err
		}

		if length > o.PageCapacity(ctx) {
			return o.WriteBreakable(ctx, "BROKEN")
		}
	}

	o.Write(text)
	return nil
}

func (o *Paged) AddMedia(ctx context.Context, ref syncf.Ref[*receiver.Media], anchor string, collapsible bool) error {
	if o.overflown {
		return nil
	}

	caption := anchor
	captionSize := utf8.RuneCountInString(anchor)
	if collapsible && o.currCount == 0 && o.currSize+captionSize+1 <= telegram.MaxCaptionSize {
		if o.currSize > utf8.RuneCountInString(o.suffix) {
			caption += "\n" + o.curr.String()
			o.reset()
		}
	} else {
		if err := o.Flush(ctx); err != nil {
			return err
		}
	}

	return o.Receiver.SendMedia(ctx, ref, caption)
}

func (o *Paged) BreakPage(ctx context.Context) error {
	if o.overflown {
		return nil
	}

	maxPages, ok := maxPages(ctx)
	if !ok || maxPages <= 0 {
		maxPages = math.MaxInt32
	}

	if o.currSize > utf8.RuneCountInString(o.suffix) {
		o.Write(o.suffix)
		if err := o.Receiver.SendText(ctx, trim(o.curr.String())); err != nil {
			return err
		}

		o.reset()
		o.currCount++
		if o.currCount >= maxPages {
			o.overflown = true
		}

		if o.overflown {
			return nil
		}

		o.Write(o.prefix)
	}

	return nil
}

func (o *Paged) Flush(ctx context.Context) error {
	if err := o.BreakPage(ctx); err != nil {
		return err
	}

	o.currCount = 0
	return nil
}

func (o *Paged) PageCapacity(ctx context.Context) int {
	pageSize, ok := pageSize(ctx)
	if !ok {
		return math.MaxInt32
	}

	return pageSize - o.currSize - utf8.RuneCountInString(o.suffix)
}

func (o *Paged) reset() {
	o.curr.Reset()
	o.currSize = 0
}

func trim(text string) string {
	return strings.Trim(text, " \t\n\v")
}
