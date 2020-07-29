package feed

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	fluhttp "github.com/jfk9w-go/flu/http"
	metrics "github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/media/descriptor"
	"github.com/pkg/errors"
)

type Aggregator struct {
	Channel
	Storage
	*media.Tor
	Metrics metrics.Registry
	Timeout time.Duration
	Aliases map[telegram.Username]telegram.ID
	AdminID telegram.ID
	sources map[string]Source
	chats   map[telegram.ID]bool
	mu      sync.RWMutex
}

func (a *Aggregator) AddSource(source Source) *Aggregator {
	if a.sources == nil {
		a.sources = make(map[string]Source)
	}
	a.sources[source.ID()] = source
	return a
}

func (a *Aggregator) Init() *Aggregator {
	a.chats = make(map[telegram.ID]bool)
	for _, chatID := range a.Active() {
		a.RunFeed(chatID)
	}
	return a
}

func (a *Aggregator) runUpdater(chatID telegram.ID) {
	sub := a.Advance(chatID)
	if sub == nil {
		// no next item - subscriptions exhausted, stopping the updater
		a.mu.Lock()
		delete(a.chats, chatID)
		a.mu.Unlock()
		log.Printf("Stopped updater for %v", chatID)
		a.Metrics.Gauge("active_subscribers", nil).Dec()
		return
	}

	err := a.pullUpdates(chatID, *sub)
	if err != nil {
		if err == ErrNotFound {
			// the storage update has failed
			// meaning the subscription was suspended by external source
			// we don't need to do anything else
		} else {
			a.change(context.Background(), 0, sub.ID, Change{Error: err})
		}
	}

	// reschedule the updater
	time.AfterFunc(a.Timeout, func() { a.runUpdater(chatID) })
}

func (a *Aggregator) pullUpdates(chatID telegram.ID, sub Subscription) error {
	source, ok := a.sources[sub.ID.Source]
	if !ok {
		return errors.Errorf("no such source: %s", sub.ID.Source)
	}
	ctx, cancel := context.WithCancel(context.Background())
	pull := newUpdatePull(ctx, sub)
	go pull.run(source)
	hasUpdates := false
	for update := range pull.queue {
		ctx := context.Background()
		hasUpdates = true
		err := a.SendUpdate(context.Background(), chatID, update)
		if err != nil {
			return errors.Wrapf(err, "send update: %+v", update)
		}
		metricsLabels := metrics.Labels{
			"chat", sub.ID.ChatID.String(),
			"source", sub.ID.Source,
			"id", sub.ID.ID,
		}
		a.Metrics.Counter("updates", metricsLabels).Inc()
		a.Metrics.Counter("media", metricsLabels).Add(float64(len(update.Media)))
		err = a.change(ctx, 0, sub.ID, Change{RawData: update.RawData})
		if err != nil {
			cancel()
			return err
		}
	}
	if pull.err != nil {
		return errors.Wrap(pull.err, "pull updates")
	}
	if !hasUpdates {
		if err := a.change(context.Background(), 0, sub.ID, Change{RawData: sub.RawData.Bytes()}); err != nil {
			return err
		}
	}
	return nil
}

func (a *Aggregator) RunFeed(chatID telegram.ID) {
	// check that the feed does not exist yet
	// via double-checked locking
	a.mu.RLock()
	ok := a.chats[chatID]
	a.mu.RUnlock()
	if ok {
		return
	}
	a.mu.Lock()
	if a.chats[chatID] {
		a.mu.Unlock()
		return
	}
	a.chats[chatID] = true
	a.mu.Unlock()
	// run updater
	go a.runUpdater(chatID)
	a.Metrics.Gauge("active_subscribers", nil).Inc()
	log.Printf("Started updater for %v", chatID)
}

var ErrNotFound = errors.New("not found")

func (a *Aggregator) change(ctx context.Context, userID telegram.ID, id ID, change Change) error {
	changeContext := &changeContext{chatID: id.ChatID}
	if err := changeContext.checkAccess(ctx, a, userID); err != nil {
		return err
	}
	ok := a.Storage.Change(id, change)
	if !ok {
		return ErrNotFound
	}
	if change.RawData != nil {
		// if this is an offset update we don't need to notify admins
		log.Printf("Updated raw data for %s to %s", id, string(change.RawData))
		return nil
	} else {
		if change.Error == nil {
			log.Printf("Resumed %s", id)
		} else {
			log.Printf("Suspended %s (reason: '%s')", id, change.Error)
		}
	}

	if change.Error == nil {
		a.RunFeed(id.ChatID)
	}

	sub := a.Get(id)
	if sub == nil {
		return ErrNotFound
	}

	// notifications
	adminIDs, err := changeContext.getAdminIDs(ctx, a)
	if err != nil {
		return err
	}
	status := "OK 🔥"
	if change.Error != nil {
		status = "suspended"
	}

	title, _ := changeContext.getChatTitle(ctx, a)
	text := format.NewHTML(telegram.MaxMessageSize, 0, nil, nil).
		Text("Subscription " + status).NewLine().
		Text("Chat: " + title).NewLine().
		Text("Service: " + id.SourceName(a.sources)).NewLine().
		Text("Item: " + sub.Name)
	var button telegram.ReplyMarkup
	if change.Error != nil {
		button = telegram.InlineKeyboard(
			[][3]string{
				{"Delete", "d", id.String()},
				{"Resume", "r", id.String()},
			},
		)
		text.NewLine().
			Text("Reason: " + change.Error.Error())
	} else {
		button = telegram.InlineKeyboard(
			[][3]string{
				{"Suspend", "s", id.String()},
			},
		)
	}

	a.SendAlert(ctx, adminIDs, text.Format(), button)
	return nil
}

func (a *Aggregator) changeByUser(ctx context.Context, tg telegram.Client, c *telegram.Command, change Change) error {
	reply := "OK"
	id, err := ParseID(c.Payload)
	if err == nil {
		if err = a.change(ctx, c.User.ID, id, change); err != nil {
			reply = err.Error()
		}
	} else {
		reply = "failed to parse ID"
	}
	return c.Reply(ctx, tg, reply)
}

func (a *Aggregator) createChangeContext(ctx context.Context, c *telegram.Command, fields []string, chatIdx int) (*changeContext, error) {
	changeContext := new(changeContext)
	if len(fields) > chatIdx && fields[chatIdx] != "." {
		username := telegram.Username(fields[chatIdx])
		var chatID telegram.ChatID = username
		if unaliased, ok := a.Aliases[username]; ok {
			chatID = unaliased
		}
		changeContext.chatID = chatID
	} else {
		changeContext.chatID = c.Chat.ID
		changeContext.chat = c.Chat
	}
	return changeContext, changeContext.checkAccess(ctx, a, c.User.ID)
}

func (a *Aggregator) doCreate(ctx context.Context, c *telegram.Command) error {
	fields := strings.Fields(c.Payload)
	cmd := fields[0]
	changeContext, err := a.createChangeContext(ctx, c, fields, 1)
	if err != nil {
		return err
	}
	options := ""
	if len(fields) > 2 {
		options = fields[2]
	}
	rawData := NewRawData(nil)
	for sourceID, source := range a.sources {
		var draft *Draft
		draft, err = source.Draft(cmd, options, rawData)
		switch err {
		case ErrDraftFailed:
			continue
		case nil:
			id := ID{
				ID:     draft.ID,
				ChatID: changeContext.chat.ID,
				Source: sourceID,
			}
			if len(id.String()) > 62 {
				return errors.New("ID too long")
			}
			sub := &Subscription{
				ID:      id,
				Name:    draft.Name,
				RawData: rawData,
			}
			if a.Storage.Create(sub) {
				return a.change(ctx, 0, sub.ID, Change{})
			} else {
				return errors.New("exists")
			}
		default:
			return err
		}
	}
	return err
}

func (a *Aggregator) Create(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	if err := a.doCreate(ctx, c); err != nil {
		return c.Reply(ctx, tg, err.Error())
	} else {
		return nil
	}
}

func (a *Aggregator) Resume(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	return a.changeByUser(ctx, tg, c, Change{})
}

var ErrSuspendedByUser = errors.New("suspended by user")

func (a *Aggregator) Suspend(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	return a.changeByUser(ctx, tg, c, Change{Error: ErrSuspendedByUser})
}

func (a *Aggregator) Delete(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	id, err := ParseID(c.Payload)
	if err != nil {
		return c.Reply(ctx, tg, "failed to parse ID")
	}
	changeContext := &changeContext{chatID: id.ChatID}
	if err := changeContext.checkAccess(ctx, a, c.User.ID); err != nil {
		return c.Reply(ctx, tg, err.Error())
	}
	if a.Storage.Delete(id) {
		return c.Reply(ctx, tg, "OK")
	} else {
		return c.Reply(ctx, tg, "not found")
	}
}

func (a *Aggregator) Status(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	return c.Reply(ctx, tg, "OK")
}

func (a *Aggregator) YouTube(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	dtor, err := descriptor.From(fluhttp.NewClient(nil), c.Payload)
	if err != nil {
		return c.Reply(ctx, tg, err.Error())
	}
	if err = a.SendUpdate(ctx, c.Chat.ID, Update{
		Pages: []string{""},
		Media: []*media.Promise{a.Submit(c.Payload, dtor, media.Options{})},
	}); err != nil {
		err = c.Reply(ctx, tg, err.Error())
	}

	return err
}

func (a *Aggregator) List(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	fields := strings.Fields(c.Payload)
	changeContext, err := a.createChangeContext(ctx, c, fields, 0)
	if err != nil {
		return err
	}
	active := false
	command := "resume"
	if len(fields) > 1 && fields[1] != "r" {
		active = true
		command = "suspend"
	}
	subs := a.Storage.List(changeContext.chat.ID, active)
	if (len(fields) < 2 || fields[1] == "") && len(subs) == 0 {
		active = true
		command = "suspend"
		subs = a.Storage.List(changeContext.chat.ID, active)
	}
	keyboard := make([][][3]string, len(subs)*3)
	for i, sub := range subs {
		keyboard[i] = [][3]string{{
			"[" + sub.ID.SourceName(a.sources) + "] " + sub.Name,
			command[:1],
			sub.ID.String(),
		}}
	}
	title, _ := changeContext.getChatTitle(ctx, a)
	a.SendAlert(ctx,
		[]telegram.ID{c.Chat.ID},
		format.NewHTML(0, 0, nil, nil).
			Text("Chat: ").Text(title).
			NewLine().
			Text(strconv.Itoa(len(subs))).
			Text(" subscriptions eligible for ").
			Tag("b").Text(command).EndTag().
			Format(),
		telegram.InlineKeyboard(keyboard...))
	return nil
}

func (a *Aggregator) Clear(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	space := strings.Index(c.Payload, " ")
	if space < 0 || len(c.Payload) == space+1 {
		return c.Reply(ctx, tg, "this command requires two arguments")
	}
	fields := [2]string{c.Payload[:space], c.Payload[space+1:]}
	changeContext, err := a.createChangeContext(ctx, c, fields[:], 0)
	if err != nil {
		return err
	}
	cleared := a.Storage.Clear(changeContext.chat.ID, fields[1])
	title, _ := changeContext.getChatTitle(ctx, a)
	a.SendAlert(ctx,
		[]telegram.ID{c.Chat.ID},
		format.NewHTML(0, 0, nil, nil).
			Text("Chat: ").Text(title).
			NewLine().
			Text(strconv.Itoa(cleared)).
			Text(" subscriptions ").
			Tag("b").Text("cleared").EndTag().
			Format(),
		nil)
	return nil
}

func (a *Aggregator) Halt(ctx context.Context, tg telegram.Client, c *telegram.Command) error {
	if c.User.ID == a.AdminID {
		time.AfterFunc(1*time.Minute, func() { panic("halt") })
		return c.Reply(ctx, tg, "halt scheduled in 1 minute")
	} else {
		return c.Reply(ctx, tg, "forbidden")
	}
}

func (a *Aggregator) CommandListener(username string) *telegram.CommandListener {
	return telegram.NewCommandListener(username).
		HandleFunc("/sub", a.Create).
		HandleFunc("r", a.Resume).
		HandleFunc("s", a.Suspend).
		HandleFunc("d", a.Delete).
		HandleFunc("/status", a.Status).
		HandleFunc("/youtube", a.YouTube).
		HandleFunc("/list", a.List).
		HandleFunc("/clear", a.Clear).
		HandleFunc("/halt", a.Halt)
}
