package main

import (
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
)

var (
	realmTable = []byte("realm")
)

type Context interface {
	Bot() *telegram.BotAPI
	Dvach() *dvach.API
	DB() *bolt.DB
}

type Controller struct {
	bot   *telegram.BotAPI
	dvach *dvach.API
	db    *bolt.DB

	stop chan struct{}
	wg   *sync.WaitGroup
}

func SetUp(cfg Config) *Controller {
	bot := telegram.NewBotAPI(HttpClient, cfg.Token)
	dvachClient := dvach.NewAPI(HttpClient, dvach.APIConfig{
		ThreadFeedTimeout: time.Minute,
	})

	db, err := bolt.Open(cfg.DbFilename, 0600, nil)
	if err != nil {
		panic(err)
	}

	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucket(realmTable)
		return nil
	})

	return &Controller{
		bot:   bot,
		dvach: dvachClient,
		db:    db,
		stop:  make(chan struct{}, 1),
	}
}

func (svc *Controller) Start() {
	svc.bot.Start(&telegram.GetUpdatesRequest{
		Timeout:        60,
		AllowedUpdates: []string{"message"},
	})

	svc.wg = new(sync.WaitGroup)
	go func() {
		defer svc.wg.Done()
		for {
			select {
			case <-svc.stop:
				return

			case u := <-svc.bot.Updates.C:
				if u.Message != nil {
					tokens := svc.ParseCommand(*u.Message)
				}
			}
		}
	}()
}

func (svc *Controller) Stop() {
	svc.bot.Stop(false)
	svc.db.Close()
}

func (svc *Controller) ParseCommand(msg telegram.Message) []string {
	for _, entity := range msg.Entities {
		if entity.Type == "bot_command" {
			end := entity.Offset + entity.Length

			cmd := msg.Text[entity.Offset:end]
			cmd = strings.Replace(cmd, "@"+svc.bot.Me.Username, "", 1)

			params := strings.Split(msg.Text[end:], " ")

			tokens := make([]string, 1)
			tokens[0] = cmd
			for _, param := range params {
				if len(param) > 0 {
					tokens = append(tokens, param)
				}
			}

			return tokens
		}
	}

	return nil
}
