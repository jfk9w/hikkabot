package util

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type UnitType = struct{}

var Unit UnitType

func MinInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func ReadResponse(resp *http.Response, r interface{}, validStatusCodes ...int) error {
	if len(validStatusCodes) == 0 {
		validStatusCodes = []int{http.StatusOK}
	}

	ok := false
	for _, code := range validStatusCodes {
		if resp.StatusCode == code {
			ok = true
			break
		}
	}

	if !ok {
		return errors.Errorf("invalid HTTP status: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, r)
}
