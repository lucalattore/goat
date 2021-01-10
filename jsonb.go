package goat

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONB describes a structure for JSONB fields
type JSONB map[string]interface{}

// Value is required by the driver
func (a JSONB) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan is requird by the driver
func (a *JSONB) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}
