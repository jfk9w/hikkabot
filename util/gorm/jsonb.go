package gorm

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

type Jsonb json.RawMessage

func ToJsonb(value interface{}) (Jsonb, error) {
	return json.Marshal(value)
}

func (j Jsonb) GormDataType() string {
	return "jsonb"
}

func (j Jsonb) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.RawMessage(j).MarshalJSON()
}

func (j Jsonb) Unmarshal(value interface{}) error {
	return json.Unmarshal(j, value)
}

func (j *Jsonb) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.Errorf("failed to unmarshal JSONB value: %s", value)
	}

	*j = bytes
	return nil
}

func (j Jsonb) String() string {
	return string(j)
}
