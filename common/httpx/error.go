package httpx

import "encoding/json"

// InvalidFormat is a parse error.
type InvalidFormat struct {
	Data  []byte
	Cause error
}

func (e InvalidFormat) Error() string {
	return e.Cause.Error() + ": " + string(e.Data)
}

// UnmarshalJSON allows to unmarshal the (corrupt) JSON into value.
func (e InvalidFormat) UnmarshalJSON(value interface{}) error {
	return json.Unmarshal(e.Data, value)
}
