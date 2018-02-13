package telegram

import (
	"time"

	"github.com/jfk9w/hikkabot/util"
)

func incoming(ctx *context, req GetUpdatesRequest) (chan Update, util.Handle) {
	c := make(chan Update, 20)
	h := util.NewHandle()
	go func() {
		defer func() {
			close(c)
			h.Reply()
		}()

		for {
			select {
			case <-h.C:
				return

			default:
				resp, err := ctx.request(req)
				if err != nil || !resp.Ok {
					time.Sleep(3 * time.Second)
					continue
				}

				updates := make([]Update, 0)
				err = resp.Parse(&updates)
				if err != nil {
					// logging
					continue
				}

				for _, update := range updates {
					c <- update
					offset := update.ID + 1
					if req.Offset < offset {
						req.Offset = offset
					}
				}
			}
		}
	}()

	return c, h
}
