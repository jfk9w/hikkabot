package red

type Name = string

type ThingData struct {
	Title      string  `json:"title"`
	Subreddit  string  `json:"subreddit"`
	Name       Name    `json:"name"`
	Domain     string  `json:"domain"`
	URL        string  `json:"url"`
	CreatedUTC float32 `json:"created_utc"`
	Ups        int     `json:"ups"`
}
