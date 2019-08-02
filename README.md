# hikkabot

A Telegram subscription service for [2ch.hk](https://2ch.hk) and [reddit.com](https://reddit.com).
Available at [@h1kkabot](https://t.me/h1kkabot).


## Features

* Subscribe to a 2ch thread/board or a subreddit.
* Manage group and channel subscriptions.
* Receive alerts to all group/channel administrators about subscription changes.
* Automatic WebM-to-MP4 conversion to leverage Telegram built-in video player.

### Available commands

#### sub

Usage: `/sub ITEM [CHAT_REF] [OPTIONS]`

| Parameter | Description |
|-----------|-------------|
| `ITEM` | See available services and their items description below. |
| `CHAT_REF` | Chat to be subscribed to this item. Should be either empty or `.` for this chat and a username for any channel.  |
| `OPTIONS` | Each item has its own options, check below. |


#### suspend & resume

Usage: `/suspend ITEM_ID` OR `/resume ITEM_ID`

Suspend an active subscription OR Resume an inactive subscription.

`ITEM_ID` is a primary subscription item ID and is generally not exposed to an end-user. 
As such, manual execution of these commands is not possible.
Instead, under every subscription state change message sent to all chat administrators there is a Suspend or Resume button.

#### status

Usage: `/status`

Simply returns an `OK` string.

### Available services

| Service | Item | Item examples | Options |
|---|---|---|---|
| Dvach/Thread | Thread URL | `https://2ch.hk/b/res/12345678.html` | `m` for streaming only media files. |
| Dvach/Catalog | Board URL or code (end slashes are optional) | `https://2ch.hk/b[/]` <br><br> `/b[/]` | Regexp for filtering threads. Defaults to `.*`. |
| Reddit | Subreddit URL or code with optional sort | `https://reddit.com/r/meirl[/hot]` <br><br> `/r/meirl[/hot]` | Minimum amount of ups. Defaults to -1. |


## Usage

### Prerequisites

An SQL database server should be installed (PostgreSQL recommended).

### Configuration

Hikkabot requires a JSON configuration file. The path must be provided as the first command-line argument.

You can use [this skeleton](https://github.com/jfk9w/hikkabot/blob/master/config.json) to build the configuration upon.

### Installation and execution

Install using Go package manager:

```bash
$ go install github.com/jfk9w/hikkabot
$ hikkabot config.json
```