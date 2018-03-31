package dvach

import (
	"errors"
	"fmt"
	"regexp"
)

const Endpoint = "https://2ch.hk"

type (
	API interface {
		GetThread(string, string, int) ([]Post, error)
		GetPost(string, string) ([]Post, error)
	}
)

const webmType = 6

func GetWebms(post Post) []string {
	webms := make([]string, 0)
	for _, file := range post.Files {
		if file.Type == webmType {
			webms = append(webms, file.URL())
		}
	}

	return webms
}

var ErrInvalidThreadLink = errors.New("invalid thread link")

var threadLinkRegexp = regexp.MustCompile(`((http|https):\/\/){0,1}2ch\.hk\/([a-z]+)\/res\/([0-9]+)\.html`)

// FormatThreadURL composes thread URL from board code and thread ID
func FormatThreadURL(board string, threadID string) string {
	return fmt.Sprintf("%s/%s/res/%s.html", Endpoint, board, threadID)
}

// ParseThreadURL extracts board code and thread ID from thread URL
func ParseThreadURL(url string) (string, string, error) {
	groups := threadLinkRegexp.FindSubmatch([]byte(url))
	if len(groups) == 5 {
		return string(groups[3]), string(groups[4]), nil
	}

	return "", "", ErrInvalidThreadLink
}
