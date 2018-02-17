package service

import (
	"errors"
	"fmt"
	"regexp"
	"sync"

	dv "github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/screen"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	"github.com/phemmer/sawmill"
)

const (
	maxGetThreadAttempts   = 5
	maxSendMessageAttempts = 2
)

var (
	errGetThread    = errors.New("GetThread failed")
	errSendMessage  = errors.New("SendMessage failed")
	errAccessDenied = errors.New("access denied")

	_mutex   = new(sync.RWMutex)
	_subs    = make(map[SubscriberKey]*Subscriber)
	_runtime serviceRT
)

// Init service instance
func Init(bot telegram.BotAPI, dvach dv.API, filename string) {
	var persister *persister
	if len(filename) > 0 {
		persister = newPersister(filename)
		persister.init()
	}

	_runtime = serviceRT{
		bot:                 bot,
		dvach:               dvach,
		persister:           persister,
		attemptsGetThread:   make(map[ThreadKey]int),
		attemptsSendMessage: make(map[SubscriberKey]int),
		mutex:               new(sync.Mutex),
	}
}

// Start service instance
func Start() {
	if _runtime.persister != nil {
		_runtime.persister.start()
	}

	for key, sub := range _subs {
		chat := ParseSubscriberKey(key)
		sub.start(func(board string, threadID string, offset int) (int, error) {
			newOffset, err := onEvent(chat, board, threadID, offset)
			if err != nil {
				go onAlertAdministrators(chat,
					"#info\nAn error has occured. Subscription suspended.\nChat: %s\nThread: %s\nError: %s",
					chat.Key(), dv.FormatThreadURL(board, threadID), err.Error())

				return 0, err
			}

			return newOffset, nil
		})
	}

	sawmill.Notice("service started")
}

// Stop service instance
func Stop() {
	p := _runtime.persister
	if p != nil {
		p.stop()
	}

	for _, sub := range _subs {
		sub.stop()
	}

	sawmill.Notice("service stopped")

	if p != nil {
		p.persist()
	}
}

// Subscribe to a thread
func Subscribe(chat telegram.ChatRef, board string, threadID string) {
	threadURL := dv.FormatThreadURL(board, threadID)

	sub := subscriber(chat)
	err := sub.newActiveThread(board, threadID)
	if err != nil {
		go onAlertAdministrators(chat,
			"#info\nSubscription failed.\nChat: %s\nThread: %s\nError: %s",
			chat.Key(), threadURL, err.Error())

		return
	}

	go func() {
		onAlertAdministrators(chat,
			"#info\nSubscription OK.\nChat: %s\nThread: %s",
			chat.Key(), dv.FormatThreadURL(board, threadID))

		preview, err := _runtime.dvach.GetPost(board, threadID)
		if err != nil {
			sawmill.Warning("error getting preview", sawmill.Fields{
				"error": err,
			})

			return
		}

		text := ""
		if len(preview) > 0 {
			text = fmt.Sprintf(
				"#thread %s %s",
				preview[0].Subject, threadURL)
		} else {
			text = fmt.Sprintf("#thread %s", threadURL)
		}

		_runtime.bot.SendMessage(telegram.SendMessageRequest{
			Chat: chat,
			Text: text,
		}, true, nil)
	}()
}

// Unsubscribe chat from all threads
func Unsubscribe(chat telegram.ChatRef) {
	key := FormatSubscriberKey(chat)

	_mutex.Lock()
	defer _mutex.Unlock()

	if sub, ok := _subs[key]; ok {
		sub.deleteAllActiveThreads()
		go onAlertAdministrators(chat,
			"#info\nSubscriptions cleared.\nChat: %s",
			chat.Key())
	}
}

// CheckAccess for operation
func CheckAccess(caller telegram.UserID, chat telegram.ChatRef) error {
	// Special case - private chat
	if !chat.IsChannel() && int64(chat.ID) == int64(caller) {
		return nil
	}

	admins, err := _runtime.bot.GetChatAdministrators(chat)
	if err != nil {
		return err
	}

	for _, admin := range admins {
		if admin.User.ID == caller &&
			(admin.Status == "creator" ||
				admin.Status == "administrator" && admin.CanPostMessages) {
			return nil
		}
	}

	return errAccessDenied
}

func subscriber(chat telegram.ChatRef) *Subscriber {
	key := FormatSubscriberKey(chat)

	_mutex.Lock()
	defer _mutex.Unlock()

	if sub, ok := _subs[key]; !ok {
		sub = newSubscriber()
		sub.init()
		sub.start(func(board string, threadID string, offset int) (int, error) {
			newOffset, err := onEvent(chat, board, threadID, offset)
			if err != nil {
				go onAlertAdministrators(chat,
					"An error has occured. Subscription suspended.\nChat: %s\nThread: %s\nError: %s",
					chat.Key(), dv.FormatThreadURL(board, threadID), err.Error())

				return 0, err
			}

			return newOffset, nil
		})

		_subs[key] = sub
	}

	return _subs[key]
}

var space = regexp.MustCompile(`\s`)

func onEvent(chat telegram.ChatRef, board string, threadID string, offset int) (int, error) {
	key := FormatThreadKey(board, threadID)
	posts, err := _runtime.dvach.GetThread(board, threadID, offset)
	if err != nil {
		if !registerGetThreadAttempt(key) {
			return 0, err
		}

		return offset, nil
	}

	resetGetThreadAttempts(key)

	newOffset := offset
	limit := util.MinInt(dv.BatchPostCount, len(posts))
	for i := 0; i < limit; i++ {
		post := posts[i]
		webms := _runtime.dvach.GetFiles(post)
		sawmill.Debug("sending post", sawmill.Fields{
			"post":     post,
			"chat":     chat.Key(),
			"board":    board,
			"threadID": threadID,
		})

		msgs, err := screen.Parse(board, post, webms)
		if err != nil {
			go onAlertAdministrators(chat, "Parsing post failed, skipping.\n%s", err.Error())
			newOffset = post.NumInt() + 1
			continue
		}

		for _, msg := range msgs {
			if len(space.ReplaceAllString(msg, ``)) == 0 {
				continue
			}

			_, err := _runtime.bot.SendMessageSync(telegram.SendMessageRequest{
				Chat:                chat,
				Text:                msg,
				ParseMode:           telegram.HTML,
				DisableNotification: true,
			}, false)

			key := FormatSubscriberKey(chat)
			if err != nil {
				if !registerSendMessageAttempt(key) {
					files := make([]string, len(post.Files))
					for i, file := range post.Files {
						files[i] = file.URL()
					}

					sawmill.Error("post sending error", sawmill.Fields{
						"comment": post.Comment,
						"files":   files,
					})

					return 0, err
				}

				return post.NumInt(), nil
			}

			resetSendMessageAttempts(key)
		}

		newOffset = post.NumInt() + 1
	}

	return newOffset, nil
}

func onAlertAdministrators(chat telegram.ChatRef, pattern string, args ...interface{}) {
	text := fmt.Sprintf(pattern, args...)

	admins, err0 := _runtime.bot.GetChatAdministrators(chat)
	if err0 != nil {
		sawmill.Error("GetChatAdministrators: "+err0.Error(), sawmill.Fields{
			"chat": chat.Key(),
		})

		return
	}

	if admins == nil {
		notify(chat, text)
		return
	}

	for _, admin := range admins {
		if !admin.User.IsBot &&
			(admin.Status == "creator" ||
				admin.Status == "administrator" && admin.CanPostMessages) {
			notify(telegram.ChatRef{
				ID: telegram.ChatID(admin.User.ID),
			}, text)
		}
	}
}

func notify(chat telegram.ChatRef, text string) {
	go func() {
		resp, err := _runtime.bot.SendMessageSync(telegram.SendMessageRequest{
			Chat: chat,
			Text: text,
		}, true)

		if err != nil {
			sawmill.Error("notify failed: "+err.Error(), sawmill.Fields{
				"user": chat.Key(),
			})

			return
		}

		if !resp.Ok {
			sawmill.Error("notify failed", sawmill.Fields{
				"user":        chat.Key(),
				"errorCode":   resp.ErrorCode,
				"description": resp.Description,
			})
		}
	}()
}

func registerGetThreadAttempt(key ThreadKey) bool {
	_runtime.mutex.Lock()
	defer _runtime.mutex.Unlock()

	attempts := _runtime.attemptsGetThread[key]
	attempts++
	if attempts > maxGetThreadAttempts {
		delete(_runtime.attemptsGetThread, key)
		return false
	}

	_runtime.attemptsGetThread[key] = attempts

	return true
}

func resetGetThreadAttempts(key ThreadKey) {
	_runtime.mutex.Lock()
	defer _runtime.mutex.Unlock()

	delete(_runtime.attemptsGetThread, key)
}

func registerSendMessageAttempt(key SubscriberKey) bool {
	_runtime.mutex.Lock()
	defer _runtime.mutex.Unlock()

	attempts := _runtime.attemptsSendMessage[key]
	attempts++
	if attempts > maxSendMessageAttempts {
		delete(_runtime.attemptsSendMessage, key)
		return false
	}

	_runtime.attemptsSendMessage[key] = attempts

	return true
}

func resetSendMessageAttempts(key SubscriberKey) {
	_runtime.mutex.Lock()
	defer _runtime.mutex.Unlock()

	delete(_runtime.attemptsSendMessage, key)
}

type serviceRT struct {
	bot       telegram.BotAPI
	dvach     dv.API
	persister *persister

	attemptsGetThread   map[ThreadKey]int
	attemptsSendMessage map[SubscriberKey]int
	mutex               *sync.Mutex
}
