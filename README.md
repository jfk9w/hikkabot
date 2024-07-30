# hikkabot

[![Go Reference](https://pkg.go.dev/badge/github.com/jfk9w/hikkabot.svg)](https://pkg.go.dev/github.com/jfk9w/hikkabot)
[![Go Report](https://goreportcard.com/badge/github.com/jfk9w/hikkabot)](https://goreportcard.com/report/github.com/jfk9w/hikkabot)
[![Go Coverage](https://github.com/jfk9w/hikkabot/wiki/coverage.svg)](https://raw.githack.com/wiki/jfk9w/hikkabot/coverage.html)
[![CodeQL](https://github.com/jfk9w/hikkabot/workflows/CodeQL/badge.svg)](https://github.com/jfk9w/hikkabot/actions?query=workflow%3ACodeQL)

Telegram bot which allows relaying third-party feed updates to Telegram chats.

### Installation and execution

Install using Go package manager:

```bash
$ go install github.com/jfk9w/hikkabot@latest
$ hikkabot --config.schema=yaml > config.schema.yaml # this way you can get configuration JSON schema
$ hikkabot --config.file=config.yml # pass your configuration file
```

Alternatively, you can use our Docker image:

```bash
$ docker run ghcr.io/jfk9w/hikkabot --telegram.token=<your_telegram_bot_api_token>
$ hikkabot_telegram_token=<your_telegram_bot_api_token> docker run ghcr.io/jfk9w/hikkabot # you can also pass configuration options as environment variables
```

`--help` is also available.

## Features

* Aggregator relays updates from various pluggable content feed providers ("vendors").
* Supports PostgreSQL and SQLite3 as aggregator backends (including in-memory with no strings attached).
* Automatically extracts direct media links from reddit submissions.
* Converts webm to mp4 in order to leverage Telegram built-in video player.
* Supports navigation/filtration via hashtags (use Telegram X on mobiles for best experience).
* Detects media duplicates and filters them out when applicable.

### Vendors

Vendor is a content feed provider. It is responsible for parsing subscription options and loading, parsing and formatting feed updates and media attachments.

A user interacts with a vendor via subscription command. It looks like this:
`/sub SUB [CHAT_REF] [OPTIONS]`, where:

* `SUB` is the desired subscription. It can be in the form of URL or ID, whatever is required by the vendor.
* `[CHAT_REF]` is the reference to the chat you wish to add the subscription to. It should either be an alias, a channel username without the leading `@`, or `.` for the current
  chat. Defaults to `.`
* `[OPTIONS]` is a string of subscription options. These are vendor-specific.

#### 2ch/catalog

###### Features

* Watches new threads on the specified board of [2ch.hk](https://2ch.hk).
* Can filter threads based on a regular expression applied to the OP text.
* Can render a "subscribe" button in order to quickly subscribe a given chat to new threads via `2ch/thread` vendor.

###### Options

A regular expression can be passed in order to filter new threads based on their contents.

`auto` option enables thread subscription button rendering.
`auto` is followed by `[CHAT_REF] [OPTIONS]` which are passed directly to the subscription command when pressing the rendered button.

###### Examples

* `/sub /b .` will subscribe the current chat to all new thread updates in /b/.
* `/sub /mobi channel_a (привет|пока)` will subscribe @channel_a to all new threads in /mobi/ where a content substring matches `(привет|пока)` regular expression.
* `/sub /pr channel_a привет auto channel_b !m` will subscribe @channel_a to all new threads in /pr/ where a content substring matches `привет` regular expression with thread
  subscription button targeted at @channel_b with an `!m` option.

#### 2ch/thread

###### Features

* Watch for post updates in any given thread on [2ch.hk](https://2ch.hk).
* Relay both text and media updates with preserved formatting.
* Relay only images and videos from new posts with automatic media deduplication.
* Reply and thread navigation based on hashtags.
* Automatic webm to mp4 conversion.

###### Options

`m` option can be passed in order to relay only media updates.

`#hashtag_text` can be passed in order to insert
`#hashtag_text` in every thread post instead of a hashtag inferred from thread title text. May be useful for thread grouping based on a common subject.

###### Examples

* `/sub https://2ch.hk/b/res/123456.html .` will subscribe the current chat to all post updates in https://2ch.hk/b/res/123456.html.
* `/sub https://2ch.hk/b/res/123456.html channel_a m` will subscribe @channel_a to all media updates in https://2ch.hk/b/res/123456.html.

#### subreddit

###### Features

* Watch for new posts updates in any given subreddit on [reddit](https://reddit.com).
* Filter out unpopular posts.
* Relay both text and media updates with preserved formatting.
* Relay only images and videos from new posts with automatic media deduplication.
* Reply and thread navigation based on hashtags.
* Automatic image & video direct link extraction and embedding.

###### Options

`!m` option can be passed in order to relay both text and media updates. By default, only media updates are relayed.

A floating number between `0` and `1` can be passed in order to specify the ratio of best posts which will be relayed. By default `0.3`, this means that only top 30% of all posts
will make it into updates.

###### Examples

* `/sub /r/meirl .` will subscribe the current chat to media updates from `/r/meirl`.
* `/sub /r/meirl channel_a !m 0.5` will relay to `@channel_a` top 50% of both text and media posts of all posts from `/r/meirl`.

###### Post samples

<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/subreddit-image.png" height="400px"></img>
<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/subreddit-text.png" height="300px"></img>

### Subscription management

All notifications about subscription changes will be sent to `supervisor_id`. These will contain buttons to help you manage the subscription during its lifecycle. Note that some
emoji-coding is used: fire emoji means "started" or "resumed", stop sign means "suspended", and wastebasket means "removed".

### Available commands

In addition to button control there are also commands which you can enter manually. Apart from `/sub` mentioned earlier there are also:

###### /status

Returns `OK` for all users enriched with some debug information only for the supervisor.

###### /clear PATTERN [CHAT_REF]

Removes all subscriptions with errors like `PATTERN`.

`PATTERN` is the value which will be passed to SQL "like" query. So something like `%404%` will match `Error code 404: page not found`.

`CHAT_REF` is optional and is the same as in `/sub` command.

###### /list [CHAT_REF] [r]

Lists subscriptions with buttons for suspending/resuming.

`CHAT_REF` is optional and is the same as in `/sub` command.

`r` is the option you can pass in order to list only active subscriptions. By default `/list` will return only suspended subscriptions if there are any, otherwise it will return
all active subscriptions (so all subscriptions basically).

#### Example (with pictures)

We start with a fresh channel. Below you can see that `/list` returns zero active subscriptions which means there are no subscriptions at all. Please ignore message time
inconsistencies.

<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/list-0-subs.png" height="150px"></img>

Let's subscribe our chat to `meirl` subreddit. `0.05` means that only top 5% of the posts should make it to our channel. In response we receive a notification which contains the
button allowing to suspend the subscription:

<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/sub-ok.png" height="150px"></img>

Check that `/list` returns the new subscription now. Note that `/list` outputs a button for each subscription. Button action depends on the context: if subscriptions are suspended,
the button will resume the chosen one and vice versa.

<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/list-1-sub.png" height="150px"></img>

We press the button and receive the notification below. Note that suspend and resume notifications are always sent only to the supervisor.

<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/sub-suspended.png" height="150px"></img>

The suspend notification contains buttons to resume and delete the suspended subscription. We could use the latter, but instead (just to show off) let's use the `/clear` command
(note how we infer the pattern from the error message above):

<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/clear-1-sub.png" height="150px"></img>

That's it! Our channel is as good as new. Sorry for using the same pic.

<img src="https://github.com/jfk9w/hikkabot/raw/master/assets/list-0-subs.png" height="150px"></img>
