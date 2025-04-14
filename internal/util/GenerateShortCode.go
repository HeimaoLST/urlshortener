package util

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateShortCode(length int) (string, error) {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length < 6 || length > 10 {
		return "", fmt.Errorf("length must be between 6 and 10")
	}
	shortCode := make([]byte, length)
	for i := range shortCode {
		shortCode[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortCode), nil

}
