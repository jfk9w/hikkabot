package service

import (
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/schedx"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type T struct {
	*Context
	Scheduler
	*DB
}

func Init(ctx *Context, filename string) *T {
	scheduler := schedx.New(10 * time.Second)
	db := OpenDB(filename).InitSchema()
	svc := &T{ctx, scheduler, db}
	return svc.initScheduler()
}

func (svc *T) CreateSubscription(chat telegram.ChatID, ref dvach.Ref, lastPost int, feedType FeedType) error {
	var err error
	ref, err = svc.ParentThread(ref)
	if err != nil {
		return err
	}

	var thread *dvach.Thread
	thread, err = svc.Dvach.Thread(ref)
	if err != nil {
		return err
	}

	var outline = text.FormatSubject(thread.Subject)
	if !svc.DB.CreateSubscription(chat, FeedItem{
		Ref:      ref,
		LastPost: lastPost,
		Type:     feedType,
		Outline:  outline,
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
	if err != nil {
		svc.DB.SuspendSubscription(chat.ID, item.Ref, err)
		go svc.NotifyAdministrators(chat, `#info
Subscription paused.
Chat: `+chat.Title+`
Thread: `+text.FormatRef(item.Ref)+`
Reason: `+err.Error())
		return
	}

	for _, post := range posts {
		err = svc.SendPost(chat, item.Outline, post, item.Type)
		if err != nil {
			svc.DB.SuspendSubscription(chat.ID, item.Ref, err)
			go svc.NotifyAdministrators(chat, `#info
Subscription paused.
Chat: `+chat.Title+`
Thread: `+text.FormatRef(item.Ref)+`
Reason: `+err.Error())
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

func (svc *T) initScheduler() *T {
	svc.Scheduler.Init(svc.work)
	for _, chat := range svc.LoadActiveAccounts() {
		svc.Scheduler.Schedule(chat)
	}

	return svc
}
