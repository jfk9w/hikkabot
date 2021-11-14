package main

import (
	"context"

	httpf "github.com/jfk9w-go/flu/httpf"
	"github.com/sirupsen/logrus"

	"hikkabot/3rdparty/srstats"
)

func main() {
	client := (*srstats.Client)(httpf.NewClient(nil))
	ctx := context.Background()
	subreddits, err := client.GetSuggestions(ctx, map[string]float64{
		"Whatcouldgowrong":     3,
		"WatchPeopleDieInside": 2,
		"4chan":                2,
		"reverseanimalrescue":  2,
	})

	if err != nil {
		panic(err)
	}

	for _, row := range subreddits {
		logrus.Info(row)
	}
}
