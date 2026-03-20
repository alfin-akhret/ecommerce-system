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
