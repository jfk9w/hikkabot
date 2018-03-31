package service

import (
	"fmt"
	"sync"
	"time"

	dv "github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/screen"
	tg "github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	"github.com/jfk9w/hikkabot/webm"
	log "github.com/sirupsen/logrus"
)

type feed struct {
	Q chan ThreadID
	H util.Handle
}

type T struct {
	dvach dv.API
	bot   tg.BotAPI
	conv  chan<- webm.Request
	db    Storage
	feeds map[AccountID]feed
	mu    *sync.Mutex
}

type Caller struct {
	Chat   tg.ChatRef
	UserID tg.UserID
}

func New(dvach dv.API, bot tg.BotAPI, conv chan<- webm.Request, db Storage) *T {
	return &T{
		dvach: dvach,
		bot:   bot,
		conv:  conv,
		db:    db,
		feeds: make(map[AccountID]feed),
		mu:    new(sync.Mutex),
	}
}

func (x *T) Init(state State) {
	for acc, thrs := range state {
		f := x.ensure(ReadAccountID(acc))
		for _, thr := range thrs {
			f.Q <- thr
			log.WithFields(log.Fields{
				"acc": acc,
				"thr": thr,
			}).Debug("SRVC init")
		}
	}
}

func (x *T) Subscribe(caller Caller, chat tg.ChatRef, url string) {
	if !x.access(caller, chat) {
		return
	}

	board, thread, err := dv.ParseThreadURL(url)
	if err != nil {
		x.bot.SendMessage(tg.SendMessageRequest{
			Chat:      caller.Chat,
			ParseMode: tg.Markdown,
			Text:      "Usage: `/subscribe THREAD_URL`",
		}, true, nil)

		return
	}

	acc := GetAccountID(chat)
	thr := GetThreadID(board, thread)
	if x.db.InsertThread(acc, thr) {
		f := x.ensure(chat)
		f.Q <- thr
		x.notifyAdmins(chat,
			"Chat: %s\nThread: %s\nSubscription OK.",
			chat.Key(), url)
	} else {
		x.bot.SendMessage(tg.SendMessageRequest{
			Chat: caller.Chat,
			Text: "Already subscribed",
		}, true, nil)
	}
}

func (x *T) Unsubscribe(caller Caller, chat tg.ChatRef) {
	if !x.access(caller, chat) {
		return
	}

	x.db.DeleteAccount(GetAccountID(chat))
	x.notifyAdmins(chat,
		"Chat: %s\nSubscriptions cleared.",
		chat.Key())
}

func (x *T) Status(caller Caller) {
	x.bot.SendMessage(tg.SendMessageRequest{
		Chat: caller.Chat,
		Text: "While you're dying I'll be still alive\nAnd when you're dead I will be still alive\nStill alive\nS T I L L A L I V E",
	}, true, nil)
}

func (x *T) Front(caller Caller, board string, limit int) {
	c, err := x.dvach.GetFront(board)
	if err != nil {
		x.bot.SendMessage(tg.SendMessageRequest{
			Chat: caller.Chat,
			Text: fmt.Sprintf(
				"Unable to get the front page of /%s: %s",
				board, err.Error()),
		}, true, nil)

		return
	}

	limit = util.MinInt(limit, len(c.Threads))
	for i, post := range c.Threads[0:limit] {
		message, preview := screen.Thread(board, post)
		x.bot.SendMessageSync(tg.SendMessageRequest{
			Chat:                  caller.Chat,
			Text:                  fmt.Sprintf("<b>%d</b> / %s", i+1, message),
			ParseMode:             tg.HTML,
			DisableWebPagePreview: !preview,
		}, true)
	}
}

func (x *T) Stop() {
	x.mu.Lock()
	defer x.mu.Unlock()

	for _, v := range x.feeds {
		v.H.Ping()
	}
}

func (x *T) notify(chat tg.ChatRef, text string, args ...interface{}) {
	x.bot.SendMessage(tg.SendMessageRequest{
		Chat: chat,
		Text: fmt.Sprintf(text, args...),
	}, true, nil)
}

func (x *T) notifyAdmins(chat tg.ChatRef, text string, args ...interface{}) {
	text = "#info\n" + text
	if !chat.IsChannel() {
		x.notify(chat, text, args...)
		return
	}

	admins, err := x.bot.GetChatAdministrators(chat)
	if err != nil {
		log.WithFields(log.Fields{
			"sub": chat.Key(),
		}).Error("SRVC notify failed: ", err)

		return
	}

	for _, admin := range admins {
		x.notify(tg.ChatRef{
			ID: tg.ChatID(admin.User.ID),
		}, text, args...)
	}
}

func (x *T) access(caller Caller, chat tg.ChatRef) bool {
	if int64(chat.ID) == int64(caller.UserID) {
		return true
	}

	admins, err := x.bot.GetChatAdministrators(chat)
	if err == nil {
		for _, admin := range admins {
			if admin.User.ID == caller.UserID &&
				(admin.Status == "creator" ||
					admin.Status == "administrator" && admin.CanPostMessages) {
				return true
			}
		}
	}

	x.bot.SendMessage(tg.SendMessageRequest{
		Chat:      caller.Chat,
		ParseMode: tg.Markdown,
		Text: `#info
		Operation forbidden.`,
	}, true, nil)

	return false
}

type ferror uint8

const (
	eok ferror = iota
	ethread
	echat
	einterrupt
)

func (x *T) process(acc AccountID, thr ThreadID, offset int, h util.Handle) ferror {
	chat := ReadAccountID(acc)
	board, thread := ReadThreadID(thr)

	if offset == 0 {
		preview, err := x.dvach.GetPost(board, thread)
		if err != nil || len(preview) == 0 {
			return ethread
		}

		if resp, err := x.bot.SendMessageSync(tg.SendMessageRequest{
			Chat: chat,
			Text: fmt.Sprintf(
				"#thread %s %s",
				preview[0].Subject, dv.FormatThreadURL(board, thread)),
		}, true); err != nil || !resp.Ok {
			return echat
		}
	}

	posts, err := x.dvach.GetThread(board, thread, offset)
	if err != nil {
		return ethread
	}

	reqs := make(map[string]chan string)
	for _, post := range posts {
		webms := dv.GetWebms(post)
		for _, w := range webms {
			req := webm.NewRequest(w)
			x.conv <- req
			reqs[w] = req.C
		}
	}

	for _, post := range posts {
		select {
		case <-h.C:
			return einterrupt

		default:
		}

		msgs, err := screen.Post(board, post, reqs)
		if err != nil {
			return ethread
		}

		for _, msg := range msgs {
			if resp, err := x.bot.SendMessageSync(tg.SendMessageRequest{
				Chat:                chat,
				ParseMode:           tg.HTML,
				Text:                msg,
				DisableNotification: true,
			}, false); err != nil || !resp.Ok {
				return echat
			}
		}

		if !x.db.UpdateOffset(acc, thr, post.NumInt()+1) {
			break
		}
	}

	return eok
}

func (x *T) ensure(chat tg.ChatRef) feed {
	x.mu.Lock()
	defer x.mu.Unlock()

	acc := GetAccountID(chat)
	f, ok := x.feeds[acc]
	if !ok {
		f = feed{
			Q: make(chan ThreadID, 20),
			H: util.NewHandle(),
		}

		x.feeds[acc] = f

		l := log.WithFields(log.Fields{"acc": acc})

		go func() {
			r := 0
			qr := make(map[ThreadID]int)
			ticker := time.NewTicker(10 * time.Second)
			defer func() {
				l.Debug("SRVC feed exit")
				ticker.Stop()
				f.H.Reply()
			}()

			l.Debug("SRVC feed start")
			for {
				select {
				case <-f.H.C:
					return

				case thr := <-f.Q:
					<-ticker.C
					offset := x.db.GetOffset(acc, thr)
					if offset == -1 {
						continue
					}

					switch x.process(acc, thr, offset, f.H) {
					case eok:
						r = 0
						qr[thr] = 0
						f.Q <- thr

					case ethread:
						if _, ok := qr[thr]; !ok {
							qr[thr] = 0
						}

						qr[thr] += 1
						if qr[thr] >= 3 {
							x.db.DeleteThread(acc, thr)

							board, thread := ReadThreadID(thr)
							x.notifyAdmins(ReadAccountID(acc), `#info
							Chat: `+chat.Key()+`
							Thread: `+dv.FormatThreadURL(board, thread)+`
							An error has occured. Subscription suspended.`)
						}

					case echat:
						r += 1
						if r >= 3 {
							x.db.DeleteAccount(acc)

							x.notifyAdmins(chat, `#info
							Chat: `+chat.Key()+`
							Unable to send messages to the chat. All subscriptions suspended.`)
							return
						}

					case einterrupt:
						return
					}
				}
			}
		}()
	}

	return f
}
