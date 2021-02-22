package resolvers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type Float64 float64

func (u Float64) ImplementsGraphQLType(name string) bool {
	return name == "Float64"
}

func (u *Float64) Native() float64 {
	if u == nil {
		return 0
	}
	return float64(*u)
}

func (u *Float64) UnmarshalGraphQL(input interface{}) error {
	switch input := input.(type) {
	case string:
		res, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return fmt.Errorf("invalid float64 string value %q: %w", input, err)
		}
		*u = Float64(res)
	case float64:
		*u = Float64(input)
	case float32:
		*u = Float64(input)
	case int64:
		*u = Float64(input)
	case uint64:
		*u = Float64(input)
	case uint32:
		*u = Float64(input)
	case int32:
		*u = Float64(input)
	default:
		return fmt.Errorf("invalid input type %T", input)
	}
	return nil
}

func (u Float64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatFloat(u.Native(), 'f', 18, 64) + `"`), nil
}

func (i *Float64) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("empty value")
	}

	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		val, err := strconv.ParseFloat(s, 10)
		if err != nil {
			return err
		}

		*i = Float64(val)

		return nil
	}

	var v float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = Float64(v)

	return nil
}
