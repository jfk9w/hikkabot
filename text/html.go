package text

import (
	"golang.org/x/net/html"
)

func attr(token html.Token, key string) (string, bool) {
	for _, attr := range token.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}

	return "", false
}

var allowedTags = []string{"a", "b", "i"}

func isAllowed(tag string) bool {
	//tag = strings.ToLower(tag)
	for _, t := range allowedTags {
		if t == tag {
			return true
		}
	}

	return false
}
