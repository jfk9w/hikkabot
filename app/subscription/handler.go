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
	ctrl     *controller
}

func NewHandler(bot *telegram.Bot, ctx Context, storage Storage, interval time.Duration, services []Service) *Handler {
	ctrl := newController(bot, ctx, storage, interval)
	ctrl.init()
	return &Handler{bot, services, ctrl}
}

func (h *Handler) Sub(c *telegram.Command) error {
	parts := strings.Split(c.Payload, " ")
	cmd := parts[0]
	var chatRef, opts string
	if len(parts) > 1 {
		chatRef = parts[1]
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
			auth.fill(h.bot, c, telegram.Username(chatRef))
			err := auth.check(h.bot)
			if err != nil {
				return err
			}

			if h.ctrl.create(item, auth) {
				c.Reply("OK")
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
		return errors.New("absent")
	}

	auth := &auth{chatID: item.ChatID, userID: c.User.ID}
	err := auth.check(h.bot)
	if err != nil {
		return err
	}

	if h.ctrl.suspend(item.PrimaryID, auth, errors.New("suspended by user")) {
		c.Reply("OK")
	}

	return nil
}

func (h *Handler) Resume(c *telegram.Command) error {
	item, ok := h.ctrl.get(c.Payload)
	if !ok {
		return errors.New("absent")
	}

	auth := &auth{chatID: item.ChatID, userID: c.User.ID}
	err := auth.check(h.bot)
	if err != nil {
		return err
	}

	if h.ctrl.resume(item.PrimaryID, auth) {
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
		HandleFunc("suspend", h.Suspend).
		HandleFunc("resume", h.Resume).
		HandleFunc("/status", h.Status)
}
