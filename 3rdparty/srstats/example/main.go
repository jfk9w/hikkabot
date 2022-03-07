package main

import (
	"context"
	"hikkabot/3rdparty/srstats"

	"github.com/sirupsen/logrus"
)

func main() {
	client := new(srstats.Client)
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
