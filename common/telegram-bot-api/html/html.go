package html

import (
	"strings"

	"golang.org/x/net/html"
)

type (
	Token        = html.Token
	IsTagAllowed = func(string) bool
	ProcessLink  = func(Token, *Output)
	Format       struct {
		MaxChunks    int
		ChunkSize    int
		IsTagAllowed IsTagAllowed
		ProcessLink  ProcessLink
	}
)

func NewFormat(maxChunks, chunkSize int, isTagAllowed IsTagAllowed, processLink ProcessLink) *Format {
	return &Format{
		MaxChunks:    maxChunks,
		ChunkSize:    chunkSize,
		IsTagAllowed: isTagAllowed,
		ProcessLink:  processLink,
	}
}

var (
	AllowedTags = map[string]struct{}{
		"a":      {},
		"b":      {},
		"strong": {},
		"i":      {},
		"em":     {},
	}

	DefaultIsTagAllowed = func(data string) bool {
		var _, ok = AllowedTags[data]
		return ok
	}

	DefaultProcessLink = func(token Token, output *Output) {
		if link, ok := TokenAttribute(token, "href"); ok {
			output.AppendLink(link)
		}
	}
)

func TokenAttribute(token Token, key string) (string, bool) {
	for _, attr := range token.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}

	return "", false
}

func (format *Format) SetDefaults() *Format {
	if format.IsTagAllowed == nil {
		format.IsTagAllowed = DefaultIsTagAllowed
	}

	if format.ProcessLink == nil {
		format.ProcessLink = DefaultProcessLink
	}

	return format
}

// Translates HTML into acceptable by the API
func (format *Format) Format(text string) []string {
	var (
		tokenizer = html.NewTokenizer(strings.NewReader(text))
		output    = NewOutput(format.MaxChunks, format.ChunkSize)
		skipDepth = 0
	)

	for {
		var tokenType = tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		var (
			token = tokenizer.Token()
			data  = token.Data
		)

		switch tokenType {
		case html.StartTagToken:
			if skipDepth > 0 {
				skipDepth++
				continue
			}

			if !format.IsTagAllowed(data) {
				continue
			}

			switch data {
			case "a":
				format.ProcessLink(token, output)
				skipDepth += 1

			default:
				output.AppendStartTag(data)
			}

		case html.TextToken:
			if skipDepth == 0 {
				output.AppendText(data)
			}

		case html.EndTagToken:
			if skipDepth > 0 {
				skipDepth--
				continue
			}

			if !format.IsTagAllowed(data) {
				continue
			}

			if skipDepth == 0 {
				output.AppendEndTag()
			}
		}

		if output.End() {
			return output.Chunks()
		}
	}

	return output.Flush()
}
