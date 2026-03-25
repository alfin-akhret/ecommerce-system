package helper

import "time"

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
