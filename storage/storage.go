package storage

type (
	Config struct {
		Path        string  `json:"path"`
		Concurrency int     `json:"concurrency"`
		Logger      *string `json:"logger"`
	}
)
