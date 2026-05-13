package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRandomToken generates a secure random token of given length
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
