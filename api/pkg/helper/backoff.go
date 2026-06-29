package helper

import "time"

func CalculateBackoff(retry int) time.Duration {
	switch {
	case retry <= 1:
		return time.Second
	case retry == 2:
		return 2 * time.Second
	case retry == 3:
		return 4 * time.Second
	case retry == 4:
		return 8 * time.Second
	default:
		return 30 * time.Second
	}
}
