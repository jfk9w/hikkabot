package telegram

import (
	"encoding/json"
	"strconv"

	"github.com/jfk9w-go/flu/httpf"

	"github.com/jfk9w-go/flu"
)

type sendable interface {
	kind() string
	body(*httpf.Form) (flu.EncoderTo, error)
}

type Sendable interface {
	sendable
	self() Sendable
}

type Text struct {
	Text                  string    `url:"text"`
	ParseMode             ParseMode `url:"parse_mode,omitempty"`
	DisableWebPagePreview bool      `url:"disable_web_page_preview,omitempty"`
}

func (t Text) kind() string {
	return "message"
}

func (t Text) body(form *httpf.Form) (flu.EncoderTo, error) {
	return form, nil
}

func (t Text) self() Sendable {
	return t
}

type MediaType string

const (
	Photo     MediaType = "photo"
	Animation MediaType = "animation"
	Video     MediaType = "video"
	Document  MediaType = "document"
	Audio     MediaType = "audio"
	Sticker   MediaType = "sticker"
	Voice     MediaType = "voice"
)

func (mt MediaType) RemoteMaxSize() int64 {
	if mt == Photo {
		return 5 << 20
	} else {
		return 20 << 20
	}
}

func (mt MediaType) AttachMaxSize() int64 {
	if mt == Photo {
		return 10 << 20
	} else {
		return 50 << 20
	}
}

var (
	DefaultMediaType   = Document
	MIMEType2MediaType = map[string]MediaType{
		"image/jpeg":               Photo,
		"image/png":                Photo,
		"image/bmp":                Photo,
		"image/gif":                Animation,
		"video/mp4":                Video,
		"application/pdf":          Document,
		"application/octet-stream": Document,
		"audio/mpeg":               Audio,
		"audio/ogg":                Voice,
		"image/webp":               Sticker,
	}
)

func MediaTypeByMIMEType(mimeType string) MediaType {
	if mediaType, ok := MIMEType2MediaType[mimeType]; ok {
		return mediaType
	} else {
		return DefaultMediaType
	}
}

type Media struct {
	Type      MediaType `url:"-" json:"type"`
	Input     flu.Input `url:"-" json:"-"`
	Filename  string    `url:"-" json:"-"`
	Caption   string    `url:"caption,omitempty" json:"caption,omitempty"`
	ParseMode ParseMode `url:"parse_mode,omitempty" json:"parse_mode,omitempty"`
}

func (m Media) filename() string {
	if m.Filename != "" {
		return m.Filename
	}

	var suffix string
	switch m.Type {
	case Animation:
		suffix = ".gif"
	case Video:
		suffix = ".mp4"
	case Audio:
		suffix = ".mp3"
	}

	return string(m.Type) + suffix
}

func (m Media) kind() string {
	return string(m.Type)
}

func (m Media) body(form *httpf.Form) (flu.EncoderTo, error) {
	switch r := m.Input.(type) {
	case flu.URL:
		return form.Set(string(m.Type), r.String()), nil
	default:
		return form.Multipart().File(string(m.Type), m.filename(), m.Input), nil
	}
}

func (m Media) self() Sendable {
	return m
}

type mediaJSON struct {
	Media
	MediaURL string `json:"media"`
}

type MediaGroup []Media

func (mg MediaGroup) kind() string {
	return "mediaGroup"
}

func (mg MediaGroup) body(form *httpf.Form) (flu.EncoderTo, error) {
	var multipart *httpf.MultipartForm
	multiparted := false
	media := make([]mediaJSON, len(mg))
	for i, m := range mg {
		m := mediaJSON{m, ""}
		switch r := m.Input.(type) {
		case flu.URL:
			m.MediaURL = r.String()
		default:
			if !multiparted {
				multipart = form.Multipart()
				multiparted = true
			}

			id := "media" + strconv.Itoa(i)
			multipart = multipart.File(id, m.filename(), m.Input)
			m.MediaURL = "attach://" + id
		}

		media[i] = m
	}

	bytes, err := json.Marshal(media)
	if err != nil {
		return nil, err
	}
	form = form.Set("media", string(bytes))
	if multiparted {
		return multipart, nil
	}
	return form, nil
}
