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

type Client httpf.Client

func (c *Client) Unmask() *httpf.Client {
	return (*httpf.Client)(c)
}

func (c *Client) GetGlobalHistogram(ctx context.Context) (map[string]float64, error) {
	m := make(map[string]float64)
	return m, c.Unmask().GET(BaseURL + "/globalSubredditsIdHist").
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&m)).
		Error
}

func (c *Client) GetHistogram(ctx context.Context, subreddit string) (map[string]float64, error) {
	m := make(map[string]float64)
	return m, c.Unmask().GET(BaseURL+"/subredditNameToSubredditsHist").
		QueryParam("subredditName", subreddit).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&m)).
		Error
}

func (c *Client) GetSubredditNames(ctx context.Context, ids []string) ([]string, error) {
	body := new(flu.ByteBuffer)
	if err := flu.EncodeTo(flu.JSON(map[string][]string{"subredditIds": ids}), body); err != nil {
		return nil, err
	}

	names := make([]string, 0)
	return names, c.Unmask().POST(BaseURL + "/specificSubredditIdsToNames").
		BodyEncoder(&flu.Text{Value: string(body.Bytes())}).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&names)).
		Error
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
