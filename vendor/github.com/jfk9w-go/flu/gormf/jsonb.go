package gormf

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

// JSONB provides PostgreSQL jsonb type support for gorm.
type JSONB json.RawMessage

// ToJSONB converts the specified value to jsonb.
func ToJSONB(value any) (JSONB, error) {
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

func (j JSONB) As(value any) error {
	return json.Unmarshal(j, value)
}

func (j *JSONB) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.Errorf("failed to scan JSONB value: %s", value)
	}

	*j = bytes
	return nil
}

func (j JSONB) String() string {
	return string(j)
}
