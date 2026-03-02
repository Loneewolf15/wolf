package stdlib

import (
	"encoding/json"
)

// WolfJSON implements the Wolf\JSON standard library.
type WolfJSON struct{}

// Encode converts a value to a JSON string.
func (WolfJSON) Encode(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// Decode parses a JSON string into a map.
func (WolfJSON) Decode(s string) interface{} {
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}

// DecodeMap parses JSON into a string-keyed map.
func (WolfJSON) DecodeMap(s string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}

// DecodeArray parses JSON into a slice.
func (WolfJSON) DecodeArray(s string) []interface{} {
	var result []interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}

// EncodePretty converts a value to a pretty-printed JSON string.
func (WolfJSON) EncodePretty(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

// Valid checks if a string is valid JSON.
func (WolfJSON) Valid(s string) bool {
	return json.Valid([]byte(s))
}
