package telegram

import (
	"github.com/jfk9w-go/hikkabot/common/gox/closer"
	"github.com/jfk9w-go/hikkabot/common/gox/unit"
)

var DefaultUpdatesOpts = &UpdatesOpts{
	Timeout:        60,
	AllowedUpdates: []string{"message"},
}

type Updater struct {
	closer.I
	Updates chan Update
}

func RunUpdater(api *API, opts *UpdatesOpts) *Updater {
	if opts == nil {
		opts = DefaultUpdatesOpts
	}

	var (
		mon     = unit.NewChan()
		updates = make(chan Update)
	)

	go func() {
		defer close(updates)
		for {
			select {
			case <-mon.Out():
				return

			default:
				var (
					resp []Update
					err  error
				)

				if !mon.Exec(func() {
					resp, err = api.GetUpdates(opts)
				}) {
					return
				}

				if err != nil {
					continue
				}

				for _, update := range resp {
					updates <- update
					opts.Offset = update.ID + 1
				}
			}
		}
	}()

	return &Updater{mon, updates}
}
