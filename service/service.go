package service

import (
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/schedx"
	"github.com/jfk9w-go/hikkabot/common"
	"github.com/jfk9w-go/logx"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type T struct {
	*Context
	Scheduler
	*DB
}

func Init(ctx *Context, interval time.Duration, filename string) *T {
	scheduler := schedx.New(interval)
	db := OpenDB(filename).InitSchema()
	svc := &T{ctx, scheduler, db}
	return svc.initScheduler()
}

func (svc *T) CreateSubscription(chat telegram.ChatID, ref dvach.Ref, lastPost int, feedType FeedType) error {
	var info, err = svc.ThreadInfo(ref)
	if err != nil {
		return err
	}

	if !svc.DB.CreateSubscription(chat, FeedItem{
		Ref:      info.Ref,
		LastPost: lastPost,
		Type:     feedType,
		Header:   info.Header,
	}) {
		return errors.New("exists")
	}

	svc.Schedule(chat)
	return nil
}

func (svc *T) SuspendSubscription(chat telegram.ChatID, ref dvach.Ref) error {
	if !svc.DB.SuspendSubscription(chat, ref, SuspendedByUser) {
		return errors.New("already suspended")
	}

	return nil
}

func (svc *T) SuspendAccount(chat telegram.ChatID) error {
	if svc.DB.SuspendAccount(chat, SuspendedByUser) == 0 {
		return errors.New("not subscribed")
	}

	svc.Cancel(chat)
	return nil
}

func (svc *T) work(any interface{}) {
	var (
		chat   *telegram.Chat
		item   FeedItem
		offset int
		posts  []*dvach.Post
		err    error
	)

	chat, err = svc.GetChat(any.(telegram.ChatID))
	if err != nil {
		svc.DB.SuspendAccount(chat.ID, err)
		return
	}

	item = svc.Feed(chat.ID)
	if !item.Exists {
		svc.Cancel(any)
		return
	}

	offset = item.LastPost
	if offset > 0 {
		offset++
	}

	posts, err = svc.Posts(item.Ref, offset)
	if svc.pause(chat, item, err) {
		return
	}

	for _, post := range posts {
		err = svc.SendPost(chat, item.Header, post, item.Type)
		if svc.pause(chat, item, err) {
			return
		}

		item.LastPost = post.Num
		if !svc.DB.UpdateSubscription(chat.ID, item) {
			break
		}
	}

	if posts == nil {
		svc.DB.UpdateSubscription(chat.ID, item)
	}
}

func (svc *T) pause(chat *telegram.Chat, item FeedItem, err error) bool {
	if err != nil {
		svc.DB.SuspendSubscription(chat.ID, item.Ref, err)
		go svc.NotifyAdministrators(chat.ID, `#info
Subscription paused.
Chat: `+common.ChatTitle(chat)+`
Thread: `+item.Header+`
Reason: `+err.Error())

		return true
	}

	return false
}

func (svc *T) initScheduler() *T {
	svc.Scheduler.Init(svc.work)
	var active = svc.LoadActiveAccounts()
	log.Debugf("Loading active accounts: %v", active)
	for _, chat := range active {
		svc.Scheduler.Schedule(chat)
	}

	return svc
}

var log = logx.Get("service")
