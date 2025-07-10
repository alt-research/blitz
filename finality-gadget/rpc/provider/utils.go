package provider

import "time"

func isTimeNotGreaterThan(inputTime time.Time) bool {
	if inputTime.IsZero() {
		return false
	}

	currentTime := time.Now()

	duration := currentTime.Sub(inputTime)

	return duration <= 240*time.Second
}
