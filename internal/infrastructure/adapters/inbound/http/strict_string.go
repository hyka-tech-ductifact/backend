package http

import (
	"encoding/json"
	"errors"
)

// ErrNullString is returned when a JSON field is explicitly set to null
// but the schema requires a string value (not nullable).
var ErrNullString = errors.New("field must be a string, not null")

// StrictString is a string type that rejects JSON null during unmarshalling.
// Use it in request DTOs for fields that are optional (omitempty) but must be
// a valid string if present — never null.
//
// JSON behaviour:
//   - "value"  → StrictString("value")
//   - absent   → StrictString("") (zero value, skipped by omitempty)
//   - null     → UnmarshalJSON returns ErrNullString → binding fails with 400
type StrictString string

// String returns the underlying string value.
func (s StrictString) String() string {
	return string(s)
}

// UnmarshalJSON implements json.Unmarshaler.
// It rejects null and only accepts JSON string tokens.
func (s *StrictString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return ErrNullString
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = StrictString(raw)
	return nil
}
