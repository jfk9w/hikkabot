package gorm

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

type JSONB json.RawMessage

func ToJSONB(value interface{}) (JSONB, error) {
	return json.Marshal(value)
}

func (j JSONB) GormDataType() string {
	return "jsonb"
}

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.RawMessage(j).MarshalJSON()
}

func (j JSONB) Unmarshal(value interface{}) error {
	return json.Unmarshal(j, value)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.Errorf("failed to unmarshal JSONB value: %s", value)
	}

	*j = bytes
	return nil
}

func (j JSONB) String() string {
	return string(j)
}
