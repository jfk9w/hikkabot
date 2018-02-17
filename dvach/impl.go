package dvach

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

type defaultImpl http.Client

func New(client *http.Client) API {
	return (*defaultImpl)(client)
}

func (c *defaultImpl) GetThread(board string,
	thread string, offset int) ([]Post, error) {

	if offset <= 0 {
		offset, _ = strconv.Atoi(thread)
	}

	url := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_thread&board=%s&thread=%s&num=%d",
		Endpoint, board, thread, offset)

	resp := make([]Post, 0)
	r, err := (*http.Client)(c).Get(url)
	if err != nil {
		log.WithFields(log.Fields{
			"url": url,
		}).Debug("DVCH GetThread", err)

		return nil, err
	}

	if err = util.ReadResponse(r, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *defaultImpl) GetPost(board string, post string) ([]Post, error) {
	url := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_post&board=%s&post=%s",
		Endpoint, board, post)

	resp := make([]Post, 0)
	r, err := (*http.Client)(c).Get(url)
	if err != nil {
		log.WithFields(log.Fields{
			"url": url,
		}).Debug("DVCH GetPost", err)

		return nil, err
	}

	if err = util.ReadResponse(r, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}
