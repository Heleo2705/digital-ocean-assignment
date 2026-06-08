package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

func CanonicalJSON(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case json.RawMessage:
		if len(v) == 0 {
			return []byte("null"), nil
		}
		var decoded interface{}
		if err := json.Unmarshal(v, &decoded); err != nil {
			return nil, err
		}
		return CanonicalJSON(decoded)
	case map[string]interface{}:
		return canonicalizeMap(v)
	case []interface{}:
		return canonicalizeArray(v)
	default:
		return json.Marshal(v)
	}
}

func canonicalizeMap(m map[string]interface{}) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := &bytes.Buffer{}
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')

		valueBytes, err := CanonicalJSON(m[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valueBytes)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func canonicalizeArray(arr []interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteByte('[')
	for i, item := range arr {
		if i > 0 {
			buf.WriteByte(',')
		}
		itemBytes, err := CanonicalJSON(item)
		if err != nil {
			return nil, err
		}
		buf.Write(itemBytes)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func ComputeIdempotencyHash(value interface{}) (string, error) {
	payload, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}
