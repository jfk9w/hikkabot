package subscription

import (
	"strings"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Handler struct {
	bot      *telegram.Bot
	services []Service
	aliases  map[telegram.Username]telegram.ID
	ctrl     *controller
}

func NewHandler(bot *telegram.Bot, ctx Context, storage Storage, interval time.Duration, services []Service,
	aliases map[telegram.Username]telegram.ID) *Handler {
	ctrl := newController(bot, ctx, storage, interval)
	ctrl.init()
	return &Handler{bot, services, aliases, ctrl}
}

func (h *Handler) Sub(c *telegram.Command) error {
	parts := strings.Split(c.Payload, " ")
	cmd := parts[0]
	var (
		username telegram.Username
		opts     string
	)
	if len(parts) > 1 {
		username = telegram.Username(parts[1])
	}
	if len(parts) > 2 {
		opts = parts[2]
	}

	for _, service := range h.services {
		item := service()
		err := item.Parse(h.ctrl.ctx, cmd, opts)
		switch err {
		case ErrParseFailed:
			continue
		case nil:
			auth := &auth{userID: c.User.ID}
			var chatID telegram.ChatID
			chatID, ok := h.aliases[username]
			if !ok {
				chatID = username
			}

			auth.fill(h.bot, c, chatID)
			err := auth.check(h.bot)
			if err != nil {
				return err
			}

			if !h.ctrl.create(item, auth) {
				return errors.New("exists")
			}

			return nil
		default:
			return err
		}
	}

	return ErrParseFailed
}

func (h *Handler) Suspend(c *telegram.Command) error {
	item, ok := h.ctrl.get(c.Payload)
	if !ok {
		return errors.New("not found")
	}

	auth := &auth{chatID: item.ChatID, userID: c.User.ID}
	err := auth.check(h.bot)
	if err != nil {
		return err
	}

	if h.ctrl.suspend(item, auth, errors.New("suspended by user")) {
		c.Reply("OK")
	}

	return nil
}

func (h *Handler) Resume(c *telegram.Command) error {
	item, ok := h.ctrl.get(c.Payload)
	if !ok {
		return errors.New("not found")
	}

	auth := &auth{chatID: item.ChatID, userID: c.User.ID}
	err := auth.check(h.bot)
	if err != nil {
		return err
	}

	if h.ctrl.resume(item, auth) {
		c.Reply("OK")
	}

	return nil
}

func (h *Handler) Status(c *telegram.Command) error {
	c.Reply("OK")
	return nil
}

func (h *Handler) CommandListener() *telegram.CommandListener {
	return telegram.NewCommandListener(h.bot).
		HandleFunc("/sub", h.Sub).
		HandleFunc("/suspend", h.Suspend).
		HandleFunc("/resume", h.Resume).
		HandleFunc("/status", h.Status)
}
