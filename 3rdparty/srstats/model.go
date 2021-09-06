package srstats

type Suggestion struct {
	Subreddit string
	Score     float64
}

type Suggestions []Suggestion

func (s Suggestions) Len() int {
	return len(s)
}

func (s Suggestions) Less(i, j int) bool {
	return s[i].Score > s[j].Score
}

func (s Suggestions) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
