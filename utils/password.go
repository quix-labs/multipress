package utils

import (
	"crypto/rand"
	"math/big"
	mrand "math/rand"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-#"

func GenerateSecurePassword(length int) string {
	password := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))
	for i := range password {
		index, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			index = big.NewInt(int64(mrand.Intn(len(charset))))
		}
		password[i] = charset[index.Int64()]
	}

	return string(password)
}
