package api

type (
	ParseMode string
)

const (
	None     ParseMode = ""
	Markdown ParseMode = "Markdown"
	HTML     ParseMode = "HTML"

	MaxMessageSize = 4096
	MaxCaptionSize = 200
)

type UpdatesOpts struct {
	Offset         int
	Limit          int
	Timeout        int
	AllowedUpdates []string
}

func (opts *UpdatesOpts) params() params {
	return params{}.Add(
		"offset", ints(opts.Offset)...).Add(
		"limit", ints(opts.Limit)...).Add(
		"timeout", ints(opts.Timeout)...).Add(
		"allowed_updates", opts.AllowedUpdates...)
}

type SendOpts struct {
	ParseMode           ParseMode
	DisableNotification bool
	ReplyToMessageID    MessageID
}

func (opts *SendOpts) params() params {
	if opts == nil {
		return params{}
	}

	return params{}.Add(
		"parse_mode", strs(string(opts.ParseMode))...).Add(
		"disable_notification", bools(opts.DisableNotification)...).Add(
		"reply_to_message_id", ints(opts.ReplyToMessageID)...)
}

type MessageOpts struct {
	*SendOpts
	DisableWebPagePreview bool
}

func (opts *MessageOpts) params() params {
	if opts == nil {
		return params{}
	}

	return opts.SendOpts.params().Add(
		"disable_web_page_preview", bools(opts.DisableWebPagePreview)...)
}

type MediaOpts struct {
	*SendOpts
	Caption string
}

func (opts *MediaOpts) caption() []string {
	if len(opts.Caption) == 0 {
		return arr()
	}

	return arr(opts.Caption)
}

func (opts *MediaOpts) params() params {
	if opts == nil {
		return params{}
	}

	return opts.SendOpts.params().Add(
		"caption", strs(opts.Caption)...)
}

type VideoOpts struct {
	*MediaOpts
	Duration, Width, Height int
}

func (opts *VideoOpts) params() params {
	if opts == nil {
		return params{}
	}

	return opts.MediaOpts.params().Add(
		"duration", ints(opts.Duration)...).Add(
		"width", ints(opts.Width)...).Add(
		"height", ints(opts.Height)...)
}
