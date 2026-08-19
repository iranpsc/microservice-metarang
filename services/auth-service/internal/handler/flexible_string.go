package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"metarang/shared/pkg/helpers"
)

// flexibleString unmarshals JSON numbers or strings (including Persian digits).
type flexibleString string

func (f *flexibleString) UnmarshalJSON(data []byte) error {
	data = bytesTrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*f = ""
		return nil
	}

	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = flexibleString(strings.TrimSpace(helpers.NormalizePersianNumbers(s)))
		return nil
	}

	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexibleString(n.String())
		return nil
	}

	var f64 float64
	if err := json.Unmarshal(data, &f64); err != nil {
		return fmt.Errorf("invalid string value: %w", err)
	}
	*f = flexibleString(strconv.FormatInt(int64(f64), 10))
	return nil
}

func (f flexibleString) String() string {
	return string(f)
}
