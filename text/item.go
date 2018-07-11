package text

import (
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/common"
	"golang.org/x/net/html"
)

func format(item dvach.Item, chunkSize int) []string {
	tokenizer := prepare(item.Comment)
	builder := newHtmlBuilder(chunkSize)
	skip := false

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		data := token.Data
		switch tokenType {
		case html.StartTagToken:
			if !isAllowed(token.Data) {
				continue
			}

			switch data {
			case "a":
				if datanum, ok := attr(token, "data-num"); ok {
					var ref, err = dvach.ToRef(item.Board, datanum)
					if err == nil {
						builder.write(common.RefTag(ref) + " ")
						skip = true
					}

					continue
				}

				if link, ok := attr(token, "href"); ok {
					builder.writeLink(link)
					skip = true
					continue
				}

			default:
				builder.writeStartTag(token.String())
			}

		case html.TextToken:
			if !skip {
				builder.writeText(data)
			}

		case html.EndTagToken:
			if !isAllowed(token.Data) {
				continue
			}

			if !skip {
				builder.writeEndTag()
			}

			skip = false
		}
	}

	return builder.get()
}
