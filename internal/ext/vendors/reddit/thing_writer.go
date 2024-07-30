package reddit

import (
	"context"

	"github.com/jfk9w/hikkabot/v4/internal/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/v4/internal/core"
	"github.com/jfk9w/hikkabot/v4/internal/feed"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
)

type thingWriter[C core.MediatorContext] struct {
	mediator feed.Mediator
}

func (w thingWriter[C]) String() string {
	return "vendors.reddit.thing-writer"
}

func (w *thingWriter[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var mediator core.Mediator[C]
	if err := app.Use(ctx, &mediator, false); err != nil {
		return err
	}

	w.mediator = mediator
	return nil
}

func (w *thingWriter[C]) writeHTML(ctx context.Context, feedID feed.ID, layout ThingLayout, thing reddit.ThingData) feed.WriteHTML {
	var mediaRef receiver.MediaRef
	if !thing.IsSelf && !layout.HideMedia {
		var dedupKey *feed.ID
		if !layout.ShowText {
			dedupKey = &feedID
		}

		mediaRef = w.mediaRef(ctx, thing, dedupKey)
	}

	return layout.WriteHTML(feedID, thing, mediaRef)
}

func (w *thingWriter[C]) mediaRef(ctx context.Context, thing reddit.ThingData, dedupKey *feed.ID) receiver.MediaRef {
	url := thing.URL.String
	if thing.Domain == "v.redd.it" {
		url = thing.MediaContainer.FallbackURL()
		if url == "" {
			for _, mc := range thing.CrosspostParentList {
				url = mc.FallbackURL()
				if url != "" {
					break
				}
			}
		}

		if url == "" {
			return syncf.Val[*receiver.Media]{E: errors.New("unable to find url")}
		}
	}

	return w.mediator.Mediate(ctx, url, dedupKey)
}
