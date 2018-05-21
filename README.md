# hikkabot

A Telegram subscription service for [2ch.hk](https://2ch.hk).


## Features

* Manage thread subscriptions for private chats, groups and public channels.
* Basic navigation.
* Automatic webm conversion using [aconvert](https://www.aconvert.com) for better Telegram experience.


## Commands

| Command | Shortcuts | Parameters | Description | Usage |
|---|---|---|---|---|
| `/subscribe` | `/sub` | THREAD_KEY [CHANNEL_NAME] | Subscribe to a thread. | `/sub https://2ch.hk/abu/res/42375.html`<br>`/sub #ABU42375`<br>`/sub #ABU49947`<br>`/sub #ABU42375 @channel` |
| `/unsubscribe` | `/unsub` | THREAD_KEY [CHANNEL_NAME] | Unsubscribe from a thread. | `/unsub https://2ch.hk/abu/res/42375.html`<br>`/unsub #ABU42375`<br>`/sub #ABU49947`<br>`/unsub #ABU42375 @channel` |
| `/clear` | | [CHANNEL_NAME] | Clear active subscriptions. | `/clear`<br>`/clear @channel` |
| `/dump` | | [CHANNEL_NAME] | Print out active subscriptions. | `/dump`<br>`/dump @channel` |
| `/search` | | BOARD [SEARCH_QUERY] | Print out the board's fastest threads. If SEARCH_QUERY is specified, then only the threads containing the specified words will be printed out. The number of printed threads is limit by 30. | `/search abu`<br>`/search abu поиск` |

`THREAD_KEY` can be specified as one of the following:

* A thread or post URL.
* A thread or post hashtag (see [Layout](#layout)). Also works without the leading `#`.

Please note that if a post URL or hashtag is specified the subscription will start with the post **right after** the specified.


## Layout

### IDs

Thread and post IDs are formatted as hashtags with the following schema: `#<BOARD><NUM>`.
Any post references are transformed into corresponding hashtags.
This allows for a basic navigation and easier thread management.

### Posts

Posts are printed as follows:

The first group of messages are messages with the post text content. 
The first message will have a header containing a thread subject hashtag and the post hashtag.
If this is a start of a thread, then an additional `#THREAD` hashtag will be printed.

Example:
```
#THREAD
#БУГУРТ_ДВАЧЕРОВ_ИРЛ_ТРЕД_И
#B176310796
---
#B176307024
+50 к переносимому весу
```

After the text content messages containing attached files are sent.
 
All photos and videos (except for webm) are sent as is as a respective media with a caption containing the original file URL.
Webms are initially converted to mp4. If an error during sending occures, then the bot will try to send only the file URL.

### Threads

A message may contain up to 10 threads. 
The sorting is done by speed. 

Text contents are cut to 275 symbols in length.
The header contains the creation date and the thread hashtag.
The post count and the speed are printed out additionally.
No attached files are sent.
 
Example:
```
21/05/18 Пнд 13:09:57
#B176308783
5 / 2.26/hr
---
Привет
---
```
