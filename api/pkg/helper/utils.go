package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

func FormatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	str := utc.Format(time.RFC3339)
	return &str
}

// format for money
func ToCents(price float64) int64 {
	return int64(price * 100)
}

func ToFloat(price int64) float64 {
	return float64(price) / 100
}

func GenerateHMAC(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyHMAC(message, secret, givenSignature string) bool {
	expectedHMAC := hmac.New(sha256.New, []byte(secret))
	expectedHMAC.Write([]byte(message))
	expectedSignature := expectedHMAC.Sum(nil)

	decodedGiven, err := hex.DecodeString(givenSignature)
	if err != nil {
		return false
	}

	return hmac.Equal(expectedSignature, decodedGiven)
}
