# hikkabot

A Telegram subscription service for [2ch.hk](https://2ch.hk).
Available at [@h1kkabot](https://t.me/h1kkabot).


## Features

* Subscribe to any thread.
* Manage group and channel subscriptions. Caller must be either a `creator` or an `administrator` who `can_post_messages`. The bot will try to alert all chat administrators about subscription changes.
* Images and other post attachments are sent as links (marked as `[A]`) to leverage Telegram link previews.
* Reply navigation using hashtags.

### Available commands

| Command | Description |
|---------|-------------|
| /subscribe [thread\_link] | Subscribe this chat to a thread. If a `thread_link` is not provided, it will be requested. |
| /subscribe thread\_link channel\_name | Subscribe a channel to a thread. A `channel_name` must start with a `@`. |
| /unsubscribe | Unsubscribe this chat from all threads. |
| /unsubscribe channel\_name | Unsubscribe a channel from all threads. A `channel_name` must start with a `@`. |
| /status | Check if the bot is alive. |

For `/subscribe` and `/unsubscribe` commands shortcuts are available: `/sub` and `/unsub` respectively.

### Navigation and hashtags

Navigation is built entirely upon hashtags. Every detected post number will be replaced by a similar hashtag. You can click on any post hashtag and use standard Telegram search features.

At the beginning of each thread a message containing a hashtag `#thread`, a thread summary and the URL will be sent to the subscribed chat.

All service messages begin with a hashtag `#info`.


## Usage

You can use `go build` to build the bot.

To run type `./hikkabot -config=your_config_file` (see [sample.conf](https://github.com/jfk9w/hikkabot/blob/master/sample.conf))

### <span>bot.sh</span>

This is a utility script provided for easy build and deploy.

```bash
cd hikkabot

# build
./bot.sh build

# config skeleton
cp sample.conf build/app.conf

# to run in foreground
# logs are printed to stdout
./bot.sh run

# to run in background
# logs are printed to build/log
./bot.sh start

# tail build/log
./bot.sh log

# to stop background bot process
./bot.sh stop
```
