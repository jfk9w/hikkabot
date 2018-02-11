package dvach

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

func parseResponseJSON(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(strconv.Itoa(resp.StatusCode))
	}

	err := json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return err
	}

	return nil
}
