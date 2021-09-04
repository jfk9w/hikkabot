package subreddit

import (
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	tgmedia "github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/core/feed"
	"github.com/jfk9w/hikkabot/ext/vendors"
)

type Layout struct {
	HideSubreddit  bool `json:"hide_subreddit,omitempty"`
	HideLink       bool `json:"hide_link,omitempty"`
	ShowAuthor     bool `json:"show_author,omitempty"`
	HideTitle      bool `json:"hide_title,omitempty"`
	ShowText       bool `json:"show_text,omitempty"`
	HideMedia      bool `json:"hide_media,omitempty"`
	HideMediaLink  bool `json:"hide_media_link,omitempty"`
	ShowPaywall    bool `json:"show_paywall,omitempty"`
	ShowPreference bool `json:"show_preference,omitempty"`
}

func (l Layout) WriteHTML(thing *reddit.ThingData, mediaRef tgmedia.Ref) feed.WriteHTML {
	return func(html *html.Writer) error {
		if out, ok := html.Out.(*output.Paged); ok {
			if chat, ok := out.Receiver.(*receiver.Chat); ok {
				var buttons []telegram.Button
				if l.ShowPaywall {
					buttons = []telegram.Button{
						(&telegram.Command{
							Key:  clickCommandKey,
							Args: []string{thing.Subreddit, thing.ID},
						}).Button("Get info"),
					}

					out.PageCount = 1
					out.PageSize = telegram.MaxCaptionSize
				}

				if l.ShowPreference {
					buttons = append(buttons,
						(&telegram.Command{
							Key:  likeCommandKey,
							Args: []string{thing.Subreddit, thing.ID},
						}).Button("👍"),
						(&telegram.Command{
							Key:  dislikeCommandKey,
							Args: []string{thing.Subreddit, thing.ID},
						}).Button("👎"),
					)
				}

				chat.ReplyMarkup = telegram.InlineKeyboard(buttons)
			}
		}

		if !l.HideSubreddit {
			html.Text(getSubredditName(thing.Subreddit))
		}

		if !l.HideLink {
			html.Text(" ").Link("💬", thing.PermalinkURL())
		}

		if l.ShowAuthor {
			html.Text("\n").Text(`u/`).Text(vendors.Hashtag(thing.Author))
		}

		if !l.HideTitle {
			html.Text("\n")
			if thing.IsSelf {
				html.Bold(thing.Title)
			} else {
				html.Text(thing.Title)
			}
		}

		if l.ShowText {
			html.Text("\n").MarkupString(thing.SelfTextHTML)
		}

		if !l.HideMedia {
			html.Media(thing.URL.String, mediaRef, true, !l.HideMediaLink)
		}

		return nil
	}
}
