package cache

import "encoding/json"

func Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func Unmarshal[T any](data []byte) (T, error) {
	var value T
	err := json.Unmarshal(data, &value)
	return value, err
}
