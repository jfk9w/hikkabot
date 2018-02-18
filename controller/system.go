package controller

import (
	"sync"
	"time"

	dv "github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/screen"
	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	"github.com/jfk9w/hikkabot/webm"
)

type (
	user struct {
		Q chan storage.ThreadID
		H util.Handle
	}

	users = map[storage.AccountID]user

	system struct {
		dvach *dv.API
		bot   telegram.BotAPI
		conv  chan<- webm.Request
		db    storage.T
		subs  users
		mu    *sync.Mutex
	}
)

func (sys *system) subscribe(acc AccountID, thr ThreadID) {
	sys.mu.Lock()
	defer sys.mu.Unlock()

	if _, ok := sys.subs[acc]; !ok {
		q := make(chan storage.ThreadID, 20)
		h := util.NewHandle()
		go func(acc storage.AccountID,
			q chan storage.ThreadID, h util.Handle) {

			r := 0
			qr := make(map[storage.ThreadID]int)
			t := time.NewTicker(10 * time.Second)
			defer func() {
				t.Stop()
				h.Reply()
			}()

			for {
				select {
				case <-h.C:
					return

				case <-t.C:
					select {
					case thr := <-q:
						offset, err := sys.db.GetOffset(acc, thr)
						if err != nil || offset == -1 {
							continue
						}

						switch sys.feed(acc, thr, offset, h) {
						case eok:
							r = 0
							qr[thr] = 0

						case ethread:
							if _, ok := qr[thr]; !ok {
								qr[thr] = 0
							}

							qr[thr] += 1
							if qr[thr] >= 10 {
								sys.db.Suspend(acc, thr)
							}

						case echat:
							r += 1
							if r >= 3 {
								sys.db.SuspendAll(acc)
								return
							}

						case einterrupt:
							return
						}

					default:
					}
				}
			}
		}(acc)
	}

	sys.subs[acc].Q <- thr
}

type ferror uint8

const (
	eok ferror = iota
	ethread
	echat
	einterrupt
)

func (sys *system) feed(acc storage.AccountID,
	thr storage.ThreadID, offset int, h util.Handle) ferror {

	chat := storage.ReadAccountID(acc)
	board, thread := storage.ReadThreadID(thr)

	if offset == 0 {
		preview, err := sys.dvach.GetPost(board, thread)
		if err != nil || len(preview) == 0 {
			return ethread
		}

		if resp, err := sys.bot.SendMessageSync(telegram.SendMessageRequest{
			Chat: chat,
			Text: fmt.Sprintf(
				"#thread %s %s",
				preview[0].Subject, dv.FormatThreadURL(board, thread)),
		}, true); err != nil || !resp.Ok {
			return echat
		}
	}

	posts, err := sys.dvach.GetThread(board, thread, offset)
	if err != nil {
		return ethread
	}

	reqs := make(map[string]chan string)
	for _, post := range posts {
		webms := dv.GetWebms(post)
		for _, w := range webms {
			req := webm.NewRequest(w)
			sys.conv <- req
			reqs[w] = req.C
		}
	}

	for _, post := range posts {
		select {
		case <-h.C:
			return einterrupt

		default:
		}

		msgs := screen.Parse(board, post, webms)
		for _, msg := range msgs {
			if resp, err := sys.bot.SendMessageSync(telegram.SendMessageRequest{
				Chat:                chat,
				ParseMode:           telegram.HTML,
				Text:                msg,
				DisableNotification: true,
			}, false); err != nil || !resp.Ok {
				return echat
			}
		}

		sys.db.Update(acc, thr, post.NumInt()+1)
	}

	return eok
}
