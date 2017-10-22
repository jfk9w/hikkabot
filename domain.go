package main

import (
	"fmt"

	"encoding/json"
	"io/ioutil"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/html2md"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/phemmer/sawmill"
)

const maxGetThreadRetries = 3

type Thread struct {
	Offset int `json:"offset"`

	stop0 chan struct{}
	done  chan struct{}
}

func (t *Thread) run(
	bot *telegram.BotAPI, client *dvach.API,
	mgmt telegram.ChatRef, target telegram.ChatRef,
	board string, threadId int,
	snooze func(board string, threadId int)) {

	t.stop0 = make(chan struct{}, 1)
	t.done = make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer func() {
			t.done <- unit
			ticker.Stop()
		}()

		retry := 0
		for {
			select {
			case <-t.stop0:
				sawmill.Info("Thread.stop", sawmill.Fields{
					"mgmt":     mgmt.Key(),
					"target":   target.Key(),
					"board":    board,
					"threadId": threadId,
				})

				return

			case <-ticker.C:
				posts, err := client.GetThread(board, threadId, t.Offset)
				if err != nil {
					sawmill.Debug("Thread.run", sawmill.Fields{
						"mgmt":     mgmt.Key(),
						"target":   target.Key(),
						"board":    board,
						"threadId": threadId,
						"err":      err,
						"retry":    retry,
					})

					retry++
					if retry >= maxGetThreadRetries {
						bot.SendMessage(telegram.SendMessageRequest{
							Chat: mgmt,
							Text: fmt.Sprintf(
								"Ошибка при обновлении %s для %s: %s",
								dvach.FormatThreadURL(board, threadId),
								target.Key(), err.Error()),
							DisableWebPagePreview: true,
						}, nil, true)

						// this can cause deadlock without goroutine
						go snooze(board, threadId)
						return
					}
				} else {
					for _, post := range posts {
						select {
						case <-t.stop0:
							return

						default:
							done := make(chan struct{}, 1)
							if post.NumInt() >= t.Offset {
								parts := html2md.Parse(post)
								for _, part := range parts {
									var sent bool
									bot.SendMessage(telegram.SendMessageRequest{
										Chat:                target,
										Text:                part,
										ParseMode:           telegram.Markdown,
										DisableNotification: true,
									}, func(resp *telegram.Response, err error) {
										if err != nil || !resp.Ok {
											var description string
											if err != nil {
												description = err.Error()
											} else {
												description = resp.Description
											}

											sawmill.Error("Thread.send", sawmill.Fields{
												"Thread.Offset": t.Offset,
												"mgmt":          mgmt.Key(),
												"target":        target.Key(),
												"board":         board,
												"threadId":      threadId,
												"description":   description,
											})

											bot.SendMessage(telegram.SendMessageRequest{
												Chat: mgmt,
												Text: fmt.Sprintf(
													"Ошибка при отправке сообщения в %s: %s",
													target.Key(), description),
											}, nil, true)
										} else {
											sent = true
										}

										done <- unit
									}, false)

									<-done
									if !sent {
										go snooze(board, threadId)
										return
									}
								}

								t.Offset = post.NumInt() + 1

								sawmill.Debug("Thread.send", sawmill.Fields{
									"Thread.Offset": t.Offset,
									"mgmt":          mgmt.Key(),
									"target":        target.Key(),
									"board":         board,
									"threadId":      threadId,
								})
							}
						}
					}
				}
			}
		}
	}()

	sawmill.Info("Thread.run", sawmill.Fields{
		"Thread.Offset": t.Offset,
		"mgmt":          mgmt.Key(),
		"target":        target.Key(),
		"board":         board,
		"threadId":      threadId,
	})
}

func (t *Thread) stop() <-chan struct{} {
	t.stop0 <- unit
	return t.done
}

type ThreadKey string

var threadKeyRegexp = regexp.MustCompile(`([a-z]+)\/([0-9]+)`)

func getThreadKey(board string, threadId int) ThreadKey {
	return ThreadKey(fmt.Sprintf("%s/%d", board, threadId))
}

func parseThreadKey(key ThreadKey) (string, int) {
	groups := threadKeyRegexp.FindSubmatch([]byte(string(key)))
	board := string(groups[1])
	threadId, _ := strconv.Atoi(string(groups[2]))
	return board, threadId
}

type Subs struct {
	Active   map[ThreadKey]*Thread `json:"active"`
	Inactive map[ThreadKey]*Thread `json:"inactive"`

	mgmt   telegram.ChatRef
	target telegram.ChatRef
	bot    *telegram.BotAPI
	client *dvach.API
	mutex  *sync.Mutex
	snooze func(string, int)
}

func NewSubs() *Subs {
	var (
		active   = make(map[ThreadKey]*Thread)
		inactive = make(map[ThreadKey]*Thread)
	)

	return &Subs{
		Active:   active,
		Inactive: inactive,
	}
}

func (s *Subs) init(
	bot *telegram.BotAPI, client *dvach.API,
	mgmt telegram.ChatRef, target telegram.ChatRef,
	mutex *sync.Mutex) {

	s.mgmt = mgmt
	s.target = target
	s.bot = bot
	s.client = client
	s.mutex = mutex
	s.snooze = func(board string, threadId int) {
		mutex.Lock()
		defer mutex.Unlock()

		threadKey := getThreadKey(board, threadId)
		if t, ok := s.Active[threadKey]; ok {
			s.Inactive[threadKey] = t
			delete(s.Active, threadKey)
		}
	}
}

func (s *Subs) runAll() {
	for threadKey, t := range s.Active {
		board, threadId := parseThreadKey(threadKey)
		t.run(s.bot, s.client, s.mgmt, s.target, board, threadId, s.snooze)
	}
}

func (s *Subs) subscribe(board string, threadId int) {
	threadKey := getThreadKey(board, threadId)

	if _, ok := s.Active[threadKey]; ok {
		s.bot.SendMessage(telegram.SendMessageRequest{
			Chat: s.mgmt,
			Text: fmt.Sprintf("%s уже подписан на этот тред.", s.target.Key()),
		}, nil, true)

		return
	}

	if _, ok := s.Inactive[threadKey]; ok {
		s.ensure(board, threadId, "...", func() {
			s.mutex.Lock()
			defer s.mutex.Unlock()

			if inactive, ok := s.Inactive[threadKey]; ok {
				thread := inactive
				delete(s.Inactive, threadKey)
				thread.run(s.bot, s.client, s.mgmt, s.target, board, threadId, s.snooze)
				s.Active[threadKey] = thread
			}
		})
	} else {
		s.ensure(board, threadId, fmt.Sprintf(
			"#thread %s", dvach.FormatThreadURL(board, threadId)), func() {

			s.mutex.Lock()
			defer s.mutex.Unlock()

			if _, ok := s.Active[threadKey]; !ok {
				if _, ok := s.Inactive[threadKey]; !ok {
					thread := &Thread{}
					thread.run(s.bot, s.client, s.mgmt, s.target, board, threadId, s.snooze)
					s.Active[threadKey] = thread
				}
			}
		})
	}
}

func (s *Subs) ensure(board string, threadId int, message string, callback func()) {
	preview, err := s.client.GetPost(board, threadId)
	if err != nil {
		s.bot.SendMessage(telegram.SendMessageRequest{
			Chat: s.mgmt,
			Text: fmt.Sprintf("Ошибка при запросе треда: %s", err.Error()),
		}, nil, true)

		return
	}

	s.bot.SendMessage(telegram.SendMessageRequest{
		Chat: s.target,
		Text: message,
	}, func(resp *telegram.Response, err error) {
		if err != nil || !resp.Ok {
			var description string
			if err != nil {
				description = err.Error()
			} else {
				description = resp.Description
			}

			s.bot.SendMessage(telegram.SendMessageRequest{
				Chat: s.mgmt,
				Text: fmt.Sprintf(
					"Ошибка при отправке сообщения в %s: %s",
					s.target.Key(), description),
			}, nil, true)
		} else {
			callback()
			if len(preview) > 0 {
				s.bot.SetChatTitle(telegram.SetChatTitleRequest{
					Chat:  s.target,
					Title: preview[0].Subject,
				})
			}
		}
	}, true)
}

func (s *Subs) unsubscribe() {
	for threadKey, t := range s.Active {
		<-t.stop()
		s.Inactive[threadKey] = t
		delete(s.Active, threadKey)
	}

	sawmill.Debug("Subs.unsubscribe", sawmill.Fields{
		"Subs.mgmt":   s.mgmt.Key(),
		"Subs.target": s.target.Key(),
	})

	s.bot.SendMessage(telegram.SendMessageRequest{
		Chat: s.mgmt,
		Text: "Подписки очищены.",
	}, nil, true)
}

func (s *Subs) stop() {
	for _, t := range s.Active {
		<-t.stop()
	}

	sawmill.Debug("Subs.stop", sawmill.Fields{
		"Subs.mgmt":   s.mgmt.Key(),
		"Subs.target": s.target.Key(),
	})
}

type DomainKey string

func getDomainKey(mgmt telegram.ChatRef) DomainKey {
	return DomainKey(telegram.FormatChatID(mgmt.ID))
}

func parseDomainKey(key DomainKey) telegram.ChatRef {
	return telegram.ChatRef{ID: telegram.ParseChatID(string(key))}
}

type Domain struct {
	Self    *Subs            `json:"_self"`
	Managed map[string]*Subs `json:"managed"`

	mgmt   telegram.ChatRef
	bot    *telegram.BotAPI
	client *dvach.API
	mutex  *sync.Mutex
}

func NewDomain() *Domain {
	return &Domain{
		Self:    NewSubs(),
		Managed: make(map[string]*Subs),
	}
}

func (d *Domain) init(
	bot *telegram.BotAPI, client *dvach.API,
	domainKey DomainKey,
	mutex *sync.Mutex) {

	mgmt := parseDomainKey(domainKey)

	d.mgmt = mgmt
	d.bot = bot
	d.client = client
	d.mutex = mutex

	d.Self.init(bot, client, mgmt, mgmt, mutex)
	for channel, subs := range d.Managed {
		subs.init(bot, client, mgmt, telegram.ChatRef{Username: channel}, mutex)
	}
}

func (d *Domain) runAll() {
	d.Self.runAll()
	for _, subs := range d.Managed {
		subs.runAll()
	}
}

func (d *Domain) subscribe(channel *string, board string, threadId int) {
	var subs *Subs
	if channel == nil {
		subs = d.Self
	} else {
		target := telegram.ChatRef{Username: *channel}
		if subs0, ok := d.Managed[*channel]; !ok {
			subs = NewSubs()
			subs.init(d.bot, d.client, d.mgmt, target, d.mutex)
			d.Managed[*channel] = subs
		} else {
			subs = subs0
		}
	}

	subs.subscribe(board, threadId)
}

func (d *Domain) unsubscribe(channel *string) {
	if channel == nil {
		d.Self.unsubscribe()
	} else {
		if subs, ok := d.Managed[*channel]; ok {
			subs.unsubscribe()
		} else {
			d.bot.SendMessage(telegram.SendMessageRequest{
				Chat: d.mgmt,
				Text: "Нет активных подписок.",
			}, nil, true)
		}
	}
}

func (d *Domain) stop() {
	d.Self.stop()
	for _, subs := range d.Managed {
		subs.stop()
	}

	sawmill.Debug("Domain.stop", sawmill.Fields{
		"Domain.mgmt": d.mgmt.Key(),
	})
}

type Domains struct {
	domains  map[DomainKey]*Domain
	bot      *telegram.BotAPI
	client   *dvach.API
	filename *string
	mutex    *sync.Mutex
	stop     chan struct{}
	done     chan struct{}
}

func NewDomains(filename *string) *Domains {
	return &Domains{
		domains:  make(map[DomainKey]*Domain),
		filename: filename,
	}
}

func (d *Domains) Init(bot *telegram.BotAPI, client *dvach.API) {
	d.mutex = new(sync.Mutex)

	d.bot = bot
	d.client = client

	for domainKey, domain := range d.domains {
		domain.init(bot, client, domainKey, d.mutex)
	}
}

func (d *Domains) RunAll() {
	if d.filename != nil {
		ticker := time.NewTicker(10 * time.Minute)
		d.stop = make(chan struct{}, 1)
		d.done = make(chan struct{}, 1)

		go func() {
			defer func() {
				sawmill.Info("Domains.save unscheduled")
				d.done <- unit
				ticker.Stop()
			}()

			for {
				select {
				case <-d.stop:
					return

				case <-ticker.C:
					d.save()
				}
			}
		}()

		sawmill.Info("Domains.save scheduled")
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	for _, domain := range d.domains {
		domain.runAll()
	}
}

func (d *Domains) Subscribe(mgmt telegram.ChatRef, channel *string, board string, threadId int) {
	domainKey := getDomainKey(mgmt)

	d.mutex.Lock()
	defer d.mutex.Unlock()

	var domain *Domain
	if domain0, ok := d.domains[domainKey]; !ok {
		domain = NewDomain()
		domain.init(d.bot, d.client, domainKey, d.mutex)
		d.domains[domainKey] = domain
	} else {
		domain = domain0
	}

	domain.subscribe(channel, board, threadId)
}

func (d *Domains) Unsubscribe(mgmt telegram.ChatRef, channel *string) {
	domainKey := getDomainKey(mgmt)

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if domain, ok := d.domains[domainKey]; ok {
		domain.unsubscribe(channel)
	} else {
		d.bot.SendMessage(telegram.SendMessageRequest{
			Chat: mgmt,
			Text: "Нет активных управляемых подписок.",
		}, nil, true)
	}
}

func (d *Domains) Stop() {
	if d.filename != nil {
		d.stop <- unit
		<-d.done
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	for _, domain := range d.domains {
		domain.stop()
	}

	if d.filename != nil {
		d.save0()
	}

	sawmill.Debug("Domains.stop")
}

func (d *Domains) save() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.save0()
}

func (d *Domains) save0() {
	data, err := json.Marshal(d.domains)
	if err != nil {
		sawmill.Error("Domains.save0", sawmill.Fields{
			"err": err,
		})
		return
	}

	err = ioutil.WriteFile(*d.filename, data, 0600)
	if err != nil {
		sawmill.Error("Domains.save0", sawmill.Fields{
			"err": err,
		})
	}

	sawmill.Info("Domains.save0")
}
