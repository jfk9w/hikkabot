package dvach

import (
	"context"
	"net/http"
)

type Interface interface {
	GetCatalog(ctx context.Context, board string) (*Catalog, error)
	GetThread(ctx context.Context, board string, num int, offset int) ([]Post, error)
	GetPost(ctx context.Context, board string, num int) (*Post, error)
	GetBoards(ctx context.Context) ([]Board, error)
	GetBoard(ctx context.Context, id string) (*Board, error)
}

func cookies(usercode string, path string) []*http.Cookie {
	return []*http.Cookie{
		{
			Name:   "usercode_auth",
			Value:  usercode,
			Domain: Domain,
			Path:   path,
		},
		{
			Name:   "ageallow",
			Value:  "1",
			Domain: Domain,
			Path:   path,
		},
	}
}
