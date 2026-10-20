package service

import (
	"bytes"
	"encoding/json"
)

// OptionalDate distinguishes an omitted PATCH field from an explicit JSON null.
type OptionalDate struct {
	Set   bool
	Value *string
}

func (d *OptionalDate) UnmarshalJSON(data []byte) error {
	d.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		d.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	d.Value = &value
	return nil
}

func validPlanDate(value *string) bool { return value == nil || (*value != "" && validDueDate(value)) }
