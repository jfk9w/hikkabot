package dvach

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

var unit struct{}

func httpGetJSON(client *http.Client, url string, result interface{}) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(strconv.Itoa(resp.StatusCode))
	}

	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return err
	}

	return nil
}
