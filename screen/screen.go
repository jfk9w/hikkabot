package screen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/util"
	"github.com/phemmer/sawmill"
	"golang.org/x/net/html"
)

func Parse(board string, post dvach.Post) ([]string, error) {
	var (
		tokenizer = html.NewTokenizer(strings.NewReader(post.Comment))
		ctx       = newContext()
	)

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			ctx.start(token)
			break

		case html.TextToken:
			ctx.text(board, token)
			break

		case html.EndTagToken:
			ctx.end(token)
		}
	}

	ctx.dump()
	messages := ctx.messages
	attach := parseAttachments(post)

	messagesLength := len(messages)
	attachLength := len(attach)
	l := util.MinInt(messagesLength, attachLength)
	for i := 0; i < l; i++ {
		messages[i] = attach[i] + "\n" + messages[i]
	}

	for i := messagesLength; i < attachLength; i++ {
		messages = append(messages, attach[i])
	}

	if len(messages) > 0 {
		id := "#" + strings.ToUpper(board) + post.Num + " /"
		if len(attach) > 0 {
			id += " "
		} else {
			id += "\n"
		}

		messages[0] = id + messages[0]
	}

	return messages, nil
}

func parseAttachments(post dvach.Post) []string {
	if len(post.Files) == 0 {
		return nil
	}

	wg := new(sync.WaitGroup)
	urls := make([]string, len(post.Files))
	for i, file := range post.Files {
		u := file.URL()
		n := i
		if strings.HasSuffix(strings.ToLower(u), ".webm") {
			wg.Add(1)
			go func() {
				defer wg.Done()

				resp, err := http.PostForm(
					"https://s17.aconvert.com/convert/convert-batch.php",
					url.Values{
						"file":              {u},
						"targetformat":      {"mp4"},
						"videooptiontype":   {"0"},
						"videosizetype":     {"0"},
						"customvideowidth":  {},
						"customvideoheight": {},
						"videobitratetype":  {"0"},
						"custombitrate":     {},
						"frameratetype":     {"0"},
						"customframerate":   {},
						"videoaspect":       {"0"},
						"code":              {"81000"},
						"filelocation":      {"online"},
					})

				if err != nil {
					sawmill.Error(err.Error(), sawmill.Fields{"url": u})
					return
				}

				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					sawmill.Error("webm error", sawmill.Fields{
						"url":    u,
						"status": resp.StatusCode,
					})
					return
				}

				result := new(WebMResult)
				err = json.NewDecoder(resp.Body).Decode(result)
				if err != nil {
					sawmill.Error("webm json "+err.Error(), sawmill.Fields{"url": u})
					return
				}

				sawmill.Info("webm response", sawmill.Fields{
					"resp": result,
				})

				urls[n] = fmt.Sprintf(
					"https://s%s.aconvert.com/convert/p3r68-cdx67/%s",
					result.Server, result.Filename)
			}()
		}
	}

	wg.Wait()

	for i, url := range urls {
		urls[i] = `<a href="` + escape(url) + `">[A]</a>`
	}

	return urls
}

type WebMResult struct {
	Filename string `json:"filename"`
	Ext      string `json:"ext"`
	Server   string `json:"server"`
	State    string `json:"state"`
}
