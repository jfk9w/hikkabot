package html

import "golang.org/x/net/html"

func Get(attrs []html.Attribute, key string) string {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Val
		}
	}

	return ""
}
