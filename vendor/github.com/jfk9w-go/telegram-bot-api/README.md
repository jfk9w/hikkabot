## telegram-bot-api

**telegram-bot-api** is a Telegram Bot API client and bot implementation.

### Disclaimer

Not all API methods, options and types are implemented at the moment.
Godoc is also pretty poor, I am working on it. Test coverage is absent (haha, classic).
API is more or less stable in the root package, **ext** subpackage is to be refactored heavily
and is mainly for my personal use at the moment.

### Installation
Simply install the package via go get:
```bash
go get -u github.com/jfk9w-go/telegram-bot-api
```

### Example

```go
package main

import (
	"context"
	"os"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
)

type Handler struct{}

func (h Handler) Ping(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	return cmd.Reply(ctx, client, "pong")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// read token from command line arguments
	token := os.Args[1]

	// instantiate Handler
	// available commands will be resolved automatically by reflection
	// you can also simply create a CommandRegistry and fill it manually
	var handler Handler
	
	bot := telegram.
		// create bot instance
		NewBot(syncf.DefaultClock, nil, token).
		// start command listener
		CommandListener(handler)

	defer flu.CloseQuietly(bot)

	// wait for signal
	syncf.AwaitSignal(ctx)
}
```
