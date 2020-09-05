module github.com/jfk9w/hikkabot

require (
	github.com/doug-martin/goqu/v9 v9.9.0
	github.com/jfk9w-go/aconvert-api v0.9.10-0.20200413135913-298c9e9364dc
	github.com/jfk9w-go/flu v0.9.15
	github.com/jfk9w-go/telegram-bot-api v0.9.6
	github.com/martinlindhe/base36 v1.1.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pkg/errors v0.9.1
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/stretchr/testify v1.6.0
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	golang.org/x/exp v0.0.0-20200821190819-94841d0725da
	golang.org/x/net v0.0.0-20200822124328-c89045814202
)

go 1.13

replace (
	github.com/jfk9w-go/aconvert-api => ../jfk9w-go/aconvert-api
	github.com/jfk9w-go/telegram-bot-api => ../jfk9w-go/telegram-bot-api
)
