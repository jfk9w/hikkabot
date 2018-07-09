package text

import (
	"fmt"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/telegram/api"
)

type Post struct {
	*dvach.Post
	Hashtag Hashtag
}

func FormatPost(post Post) []string {
	chunks := format(post.Item, api.MaxMessageSize*4/5)
	if len(chunks) == 0 {
		chunks = []string{""}
	}

	if len(chunks) > 0 {
		chunks[0] = fmt.Sprintf("%s\n%s\n---\n%s", post.Hashtag, FormatRef(post.Ref), chunks[0])
		if post.Parent == 0 {
			chunks[0] = "#THREAD\n" + chunks[0]
		}
	}

	return chunks
}
