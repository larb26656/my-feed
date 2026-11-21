// Package cursor encodes and decodes JSON payloads for URL-safe cursor tokens.
package cursor

import (
	"encoding/base64"
	"encoding/json"
)

func Encode(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func Decode(token string, dst any) error {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}
