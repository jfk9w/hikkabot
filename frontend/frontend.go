package frontend

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/fsx"
	"github.com/jfk9w-go/hikkabot/content"
	"github.com/jfk9w-go/hikkabot/engine"
	"github.com/jfk9w-go/hikkabot/feed"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type Engine = *engine.Engine

var (
	SendOptsHTML = &telegram.SendOpts{
		ParseMode:           telegram.HTML,
		DisableNotification: true,
	}

	MessageOptsHTML = &telegram.MessageOpts{
		SendOpts: SendOptsHTML,
	}

	MessageOptsHTMLNoPreview = &telegram.MessageOpts{
		SendOpts:              SendOptsHTML,
		DisableWebPagePreview: true,
	}
)

type Frontend struct {
	Engine
	ctx    *engine.Context
	config Config
}

func Init(engine Engine, ctx *engine.Context, config Config) *Frontend {
	var frontend = &Frontend{engine, ctx, config}
	go frontend.Run()
	return frontend
}

func (frontend *Frontend) Run() {
	for command := range frontend.ctx.CommandChannel() {
		go frontend.OnCommand(command)
	}
}

func (frontend *Frontend) ParseChat(command telegram.Command, idx int) (target telegram.ChatID, err error) {
	var value = command.Arg(idx, ".")
	if value == "." {
		target = command.Chat
	} else {
		var ref telegram.Ref
		ref, err = telegram.ParseRef(value)
		if err != nil {
			return
		}

		var chat *telegram.Chat
		chat, err = frontend.ctx.GetChat(ref)
		if err != nil {
			return
		}

		target = chat.ID
	}

	return
}

var RedRegexp = regexp.MustCompile(`^(/r/[a-zA-Z0-9_]+)/(hot|new)$`)

func ParseRed(value string) (string, string, bool) {
	var groups = RedRegexp.FindStringSubmatch(value)
	if len(groups) == 3 {
		return groups[1], groups[2], true
	}

	return "", "", false
}

var DvachWatchRegexp = regexp.MustCompile(`^/watch/([a-z]+)/(.*)$`)

func ParseDvachWatch(value string) (dvach.Board, []string, bool) {
	var groups = DvachWatchRegexp.FindStringSubmatch(value)
	if len(groups) == 3 {
		return dvach.Board(groups[1]), strings.Split(groups[2], ","), true
	}

	return "", nil, false
}

func (frontend *Frontend) ParseState(
	command telegram.Command) (
	chat telegram.ChatID, state *feed.State, err error) {

	state = new(feed.State)

	var dref dvach.Ref
	dref, err = dvach.ParseUrl(command.Arg(0, ""))
	if err == nil {
		state.Type = feed.DvachType
		state.ID = dref.Board + "/" + dref.NumString

		var thread *dvach.Thread
		thread, err = frontend.ctx.Thread(dref)
		if err != nil {
			return
		}

		var meta feed.DvachMeta
		meta.Title = content.FormatDvachThreadTag(thread)
		meta.Mode = command.Arg(2, feed.FullDvachMode)

		state.Meta, err = json.Marshal(&meta)
	} else if subreddit, mode, ok := ParseRed(command.Arg(0, "")); ok {
		state.Type = feed.RedType
		state.ID = subreddit

		var ups int
		ups, err = strconv.Atoi(command.Arg(2, "0"))

		var meta feed.RedMeta
		meta.Mode = mode
		meta.Ups = ups

		state.Meta, err = json.Marshal(&meta)
	} else if board, query, ok := ParseDvachWatch(command.Arg(0, "")); ok {
		state.Type = feed.DvachWatchType
		state.ID = board + "/" + strings.Join(query, ",")

		var meta feed.DvachWatchMeta
		meta.Board = board
		meta.Query = query

		state.Meta, err = json.Marshal(&meta)
	} else {
		err = errors.New("invalid url")
	}

	if err != nil {
		return
	}

	chat, err = frontend.ParseChat(command, 1)
	return
}

//language=SQL
const resumeQuery = `update feed set error = ''
where error like 'invalid%'
   or error like 'Get%'
   or error like 'Post%'`

func (frontend *Frontend) OnCommand(command telegram.Command) {
	switch command.Command {
	case "status":
		frontend.ctx.SendMessage(command.Chat, "alive", MessageOptsHTML)

	case "sub":
		var chat, state, err = frontend.ParseState(command)
		if err == nil {
			if !frontend.Start(chat, state) {
				err = errors.New("exists")
			}
		}

		frontend.CheckError(command, err)

	case "unsub":
		var chat, err = frontend.ParseChat(command, 0)
		if err == nil {
			if !frontend.Suspend(chat) {
				err = errors.New("absent")
			}
		}

		frontend.CheckError(command, err)

	case "force":
		var chat, err = frontend.ParseChat(command, 0)
		if !frontend.CheckError(command, err) {
			return
		}

		frontend.Schedule(chat)

	case "front", "search":
		frontend.CheckError(command, frontend.Search(command))

	case "catalog":
		frontend.CheckError(command, frontend.Catalog(command))

	case "exec":
		frontend.CheckError(command, frontend.Exec(command, strings.Join(command.Args, " ")))

	case "query":
		frontend.CheckError(command, frontend.Query(command))

	case "resume":
		if frontend.CheckError(command, frontend.Exec(command, resumeQuery)) {
			var active = frontend.LoadActiveAccounts()
			for _, chat := range active {
				frontend.Schedule(chat)
			}
		}
	}
}

func (frontend *Frontend) Exec(command telegram.Command, query string) (err error) {
	err = frontend.CheckSuperuser(command.User)
	if err != nil {
		return
	}

	var updated int64
	updated, err = frontend.DB.Exec(query)
	if !frontend.CheckError(command, err) {
		return
	}

	var text = fmt.Sprintf("updated %d rows", updated)
	_, err = frontend.ctx.SendMessage(command.Chat, text, nil)
	return
}

func (frontend *Frontend) Query(command telegram.Command) (err error) {
	err = frontend.CheckSuperuser(command.User)
	if err != nil {
		return
	}

	var report [][]string
	report, err = frontend.DB.Query(strings.Join(command.Args, " "))
	if err != nil {
		return
	}

	var path = fsx.TempFile(frontend.config.TempStorage)
	err = fsx.EnsureParent(path)
	if err != nil {
		return
	}

	var file = &httpx.File{Path: path}
	defer file.Delete()

	realFile, err := os.Create(path)
	if err != nil {
		return
	}

	defer realFile.Close()

	err = csv.NewWriter(realFile).WriteAll(report)
	if err != nil {
		return
	}

	_, err = frontend.ctx.SendDocument(command.Chat, file, &telegram.MediaOpts{})
	return
}

func (frontend *Frontend) Catalog(command telegram.Command) (err error) {
	var board = command.Arg(0, "")
	if board == "" {
		err = errors.New("invalid command")
		return
	}

	var catalog *dvach.Catalog
	catalog, err = frontend.ctx.Catalog(board)
	if err != nil {
		return
	}

	var (
		query  = command.Arg(1, "")
		tokens []string

		limitString = command.Arg(2, "30")
		limit       int
	)

	if query != "" {
		tokens = strings.Split(query, " ")
	}

	limit, err = strconv.Atoi(limitString)
	if err != nil {
		return
	}

	var parts = content.FormatDvachCatalog(
		content.SearchDvachCatalog(catalog.Threads, content.DvachUnsorted, tokens, limit))
	for _, part := range parts {
		_, err = frontend.ctx.SendMessage(command.Chat, part, MessageOptsHTMLNoPreview)
		if err != nil {
			return
		}
	}

	return
}

func (frontend *Frontend) Search(command telegram.Command) (err error) {
	var board = command.Arg(0, "")
	if board == "" {
		err = errors.New("invalid command")
		return
	}

	var catalog *dvach.Catalog
	catalog, err = frontend.ctx.Catalog(board)
	if err != nil {
		return
	}

	var (
		query  = command.Arg(1, "")
		tokens []string

		limitString = command.Arg(2, "30")
		limit       int
	)

	if query != "" {
		tokens = strings.Split(query, " ")
	}

	limit, err = strconv.Atoi(limitString)
	if err != nil {
		return
	}

	var parts = content.FormatDvachCatalog(
		content.SearchDvachCatalog(catalog.Threads, content.DvachSortByPace, tokens, limit))
	for _, part := range parts {
		_, err = frontend.ctx.SendMessage(command.Chat, part, MessageOptsHTMLNoPreview)
		if err != nil {
			return
		}
	}

	return
}

func (frontend *Frontend) CheckError(command telegram.Command, err error) bool {
	if err != nil {
		go frontend.ctx.SendMessage(command.Chat, err.Error(), MessageOptsHTML)
		return false
	}

	return true
}

func (frontend *Frontend) CheckSuperuser(user telegram.ChatID) error {
	for _, superuser := range frontend.config.Superusers {
		if user == superuser {
			return nil
		}
	}

	return errors.New("forbidden")
}

func (frontend *Frontend) Authorize(user, chat telegram.ChatID) error {
	var enriched, err = frontend.ctx.EnrichChat(chat)
	if err != nil {
		return err
	}

	for _, admin := range enriched.Administrators {
		if admin == user {
			return nil
		}
	}

	return errors.New("forbidden")
}
