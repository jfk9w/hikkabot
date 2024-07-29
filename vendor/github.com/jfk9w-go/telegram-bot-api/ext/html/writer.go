package html

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"golang.org/x/net/html"
)

type Writer struct {
	Out        Output
	Tags       TagConverter
	Anchors    AnchorFormat
	ctx        context.Context
	currTag    *Tag
	currAnchor *anchor
	err        error
}

func (w *Writer) Context() context.Context {
	if w.ctx == nil {
		return context.Background()
	}

	return w.ctx
}

func (w *Writer) WithContext(ctx context.Context) *Writer {
	value := *w
	value.ctx = ctx
	return &value
}

func (w *Writer) StartTag(name string, attrs []html.Attribute) *Writer {
	if w.err != nil || w.Out.IsOverflown() {
		return w
	}

	switch name {
	case "br":
		w.err = w.Out.WriteBreakable(w.Context(), "\n")

	case "a":
		w.currAnchor = &anchor{
			attrs:  attrs,
			parent: w.currTag,
		}

	default:
		if tag, ok := w.getTagConverter().Get(name, attrs); ok {
			if len(tag.Open)+len(tag.Close)+3 >= w.Out.PageCapacity(w.Context()) {
				if err := w.Out.BreakPage(w.Context()); err != nil {
					w.err = err
					return w
				}
			}

			if w.currAnchor != nil {
				w.currAnchor.text += tag.Open
			} else {
				w.Out.Write(tag.Open)
				w.Out.UpdatePrefix(func(prefix string) string { return prefix + tag.Open })
				w.Out.UpdateSuffix(func(suffix string) string { return tag.Close + suffix })
			}

			tag.parent = w.currTag
			w.currTag = &tag
			return w
		} else {
			w.currTag = &Tag{parent: w.currTag}
		}
	}

	return w
}

func (w *Writer) Text(text string, args ...interface{}) *Writer {
	if w.err != nil || w.Out.IsOverflown() {
		return w
	}

	if len(args) > 0 {
		text = fmt.Sprintf(text, args...)
	}

	text = html.EscapeString(text)
	if w.currAnchor != nil {
		w.currAnchor.text += html.EscapeString(text)
	} else {
		w.err = w.Out.WriteBreakable(w.Context(), text)
	}

	return w
}

func (w *Writer) EndTag() *Writer {
	if w.err != nil || w.Out.IsOverflown() {
		return w
	}

	switch {
	case w.currAnchor != nil && w.currAnchor.parent == w.currTag:
		str := w.getAnchorFormat().Format(w.currAnchor.text, w.currAnchor.attrs)
		if err := w.Out.WriteUnbreakable(w.Context(), str); err != nil {
			w.err = err
		} else {
			w.currAnchor = nil
		}

	case w.currTag != nil:
		if w.currAnchor != nil {
			w.currAnchor.text += w.currTag.Close
		} else {
			w.Out.Write(w.currTag.Close)
			w.Out.UpdatePrefix(func(prefix string) string { return prefix[:len(prefix)-len(w.currTag.Open)] })
			w.Out.UpdateSuffix(func(suffix string) string { return suffix[len(w.currTag.Close):] })
		}

		w.currTag = w.currTag.parent
	}

	return w
}

func (w *Writer) Bold(text string, args ...interface{}) *Writer {
	return w.StartTag("b", nil).Text(text, args...).EndTag()
}

func (w *Writer) Italic(text string, args ...interface{}) *Writer {
	return w.StartTag("i", nil).Text(text, args...).EndTag()
}

func (w *Writer) Code(text string, args ...interface{}) *Writer {
	return w.StartTag("code", nil).Text(text, args...).EndTag()
}

func (w *Writer) Pre(text string, args ...interface{}) *Writer {
	return w.StartTag("pre", nil).Text(text, args...).EndTag()
}

func (w *Writer) Link(text, href string) *Writer {
	if w.err != nil || w.Out.IsOverflown() {
		return w
	}
	w.err = w.Out.WriteUnbreakable(w.Context(), Anchor(text, href))
	return w
}

func (w *Writer) Markup(reader io.Reader) *Writer {
	if w.err != nil || w.Out.IsOverflown() {
		return w
	}
	tokenizer := html.NewTokenizer(reader)
	for {
		if w.err != nil || w.Out.IsOverflown() {
			return w
		}
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return w
		}
		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			w.StartTag(token.Data, token.Attr)
		case html.TextToken:
			w.Text(token.Data)
		case html.EndTagToken:
			w.EndTag()
		}
	}
}

func (w *Writer) MarkupString(markup string) *Writer {
	return w.Markup(strings.NewReader(markup))
}

func (w *Writer) Media(url string, ref syncf.Ref[*receiver.Media], collapsible bool, anchored bool) *Writer {
	if w.err != nil || w.Out.IsOverflown() {
		return w
	}

	anchor := ""
	if anchored {
		anchor = Anchor("[media]", url)
	}

	w.err = w.Out.AddMedia(w.Context(), ref, anchor, collapsible)
	return w
}

func (w *Writer) Flush() error {
	if w.err != nil || w.Out.IsOverflown() {
		return w.err
	}
	for w.currAnchor != nil || w.currTag != nil {
		w.EndTag()
	}

	return w.Out.Flush(w.Context())
}

func (w *Writer) getTagConverter() TagConverter {
	if w.Tags != nil {
		return w.Tags
	}

	return DefaultTagConverter
}

func (w *Writer) getAnchorFormat() AnchorFormat {
	if w.Anchors != nil {
		return w.Anchors
	}

	return DefaultAnchorFormat
}
