package srstats

import (
	"context"
	"net/http"
	"sort"

	"github.com/jfk9w-go/flu"
	httpf "github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

var BaseURL = "https://subredditstats.com/api"

type Client http.Client

func (c *Client) Unmask() *http.Client {
	return (*http.Client)(c)
}

func (c *Client) GetGlobalHistogram(ctx context.Context) (map[string]float64, error) {
	var m map[string]float64
	return m, httpf.GET(BaseURL+"/globalSubredditsIdHist").
		Exchange(ctx, c.Unmask()).
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&m)).
		Error()
}

func (c *Client) GetHistogram(ctx context.Context, subreddit string) (map[string]float64, error) {
	var m map[string]float64
	return m, httpf.GET(BaseURL+"/subredditNameToSubredditsHist").
		Query("subredditName", subreddit).
		Exchange(ctx, c.Unmask()).
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&m)).
		Error()
}

func (c *Client) GetSubredditNames(ctx context.Context, ids []string) ([]string, error) {
	body := new(flu.ByteBuffer)
	if err := flu.EncodeTo(flu.JSON(map[string][]string{"subredditIds": ids}), body); err != nil {
		return nil, err
	}

	names := make([]string, 0)
	return names, httpf.POST(BaseURL+"/specificSubredditIdsToNames", &flu.Text{Value: string(body.Bytes())}).
		Exchange(ctx, c.Unmask()).
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&names)).
		Error()
}

func (c *Client) GetSuggestions(ctx context.Context, subreddits map[string]float64) (Suggestions, error) {
	global, err := c.GetGlobalHistogram(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get global histogram")
	}

	normalize(global)
	m := make(map[string]float64)
	for subreddit, multiplier := range subreddits {
		histogram, err := c.GetHistogram(ctx, subreddit)
		if err != nil {
			return nil, errors.Wrap(err, "get subreddit histogram")
		}

		normalize(histogram)
		for subreddit, score := range histogram {
			globalScore := global[subreddit]
			if globalScore < 0.0001 {
				continue
			}

			m[subreddit] += multiplier * score / globalScore
		}
	}

	//normalize(m)
	suggestions := make(Suggestions, len(m))
	ids := make([]string, len(m))
	i := 0
	for subreddit, score := range m {
		suggestions[i] = Suggestion{Subreddit: subreddit, Score: score}
		ids[i] = subreddit
		i++
	}

	names, err := c.GetSubredditNames(ctx, ids)
	if err != nil {
		return nil, errors.Wrap(err, "get subreddit names")
	}

	for i := range suggestions {
		if len(names) <= i {
			suggestions = suggestions[:len(names)]
			break
		}

		suggestions[i].Subreddit = names[i]
	}

	sort.Sort(suggestions)
	return suggestions, nil
}

func normalize(histogram map[string]float64) {
	sum := 0.
	for _, value := range histogram {
		sum += value
	}

	for key, value := range histogram {
		histogram[key] = value / sum
	}
}
