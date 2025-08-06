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

func isUpdateTimeNoTimeout(update time.Time, timeout time.Duration) bool {
	if update.IsZero() {
		return false
	}

	currentTime := time.Now()
	duration := currentTime.Sub(update)

	return duration <= timeout
}
