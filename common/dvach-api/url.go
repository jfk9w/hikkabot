package dvach

import (
	"regexp"

	"github.com/pkg/errors"
)

var urlRegex = regexp.MustCompile(`^((http|https)://)?2ch\.hk/([a-z]+)/res/(\d+)\.html(#(\d+))?$`)

func ParseUrl(url string) (Ref, error) {
	var groups = urlRegex.FindStringSubmatch(url)
	if groups == nil {
		return Ref{}, errors.Errorf("invalid thread url: %s", url)
	}

	var (
		board = groups[3]
		num   = groups[4]
	)

	if groups[5] != "" {
		num = groups[6]
	}

	return ToRef(board, num)
}
