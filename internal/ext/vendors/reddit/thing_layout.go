package reddit

import (
	"fmt"

	"github.com/jfk9w/hikkabot/internal/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/internal/feed"
	"github.com/jfk9w/hikkabot/internal/util"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
)

type ThingLayout struct {
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

func (l *ThingLayout) WriteHTML(feedID feed.ID, thing reddit.ThingData, mediaRef receiver.MediaRef) feed.WriteHTML {
	return func(html *html.Writer) error {
		var buttons []telegram.Button
		ctx := html.Context()
		if l.ShowPaywall {
			buttons = []telegram.Button{PaywallButton(feedID, thing.Subreddit, thing.ID)}
			ctx = output.With(ctx, telegram.MaxCaptionSize, 1)
		}

		if !l.ShowText && !l.HideMedia {
			ctx = receiver.SkipOnMediaError(ctx)
		}

		if l.ShowPreference {
			buttons = append(buttons,
				PreferenceButtons(thing.Subreddit, thing.ID, 0, 0, l.ShowPaywall)...)
		}

		if len(buttons) > 0 {
			ctx = receiver.ReplyMarkup(ctx, telegram.InlineKeyboard(buttons))
		}

		html = html.WithContext(ctx)
		if !l.HideSubreddit {
			html.Text(getSubredditName(thing.Subreddit))
		}

		if !l.HideLink {
			html.Text(" ").Link("💬", thing.PermalinkURL())
		}

		if l.ShowAuthor {
			html.Text("\n").Text(`u/`).Text(util.Hashtag(thing.Author))
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

func PaywallButton(feedID feed.ID, subreddit, thingID string) telegram.Button {
	return (&telegram.Command{
		Key:  "sr_c",
		Args: []string{feedID.String(), subreddit, thingID},
	}).Button("ℹ️")
}

func PreferenceButtons(subreddit, thingID string, likes, dislikes int64, paywall bool) []telegram.Button {
	args := []string{subreddit, thingID}
	if paywall {
		args = append(args, "p")
	}

	return []telegram.Button{
		(&telegram.Command{
			Key:  "sr_l",
			Args: args,
		}).Button(fmt.Sprintf("👍 %d", likes)),
		(&telegram.Command{
			Key:  "src_dl",
			Args: args,
		}).Button(fmt.Sprintf("👎 %d", dislikes)),
	}
}
