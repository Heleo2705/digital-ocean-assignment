package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func CanonicalJSON(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func ComputeIdempotencyHash(value interface{}) (string, error) {
	payload, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}
