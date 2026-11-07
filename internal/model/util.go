package model

import "mihomo-tui/pkg/mihomo"

// FormatBytes formats byte counts into human-readable strings (e.g. 1.25 MB).
func FormatBytes(bytes int64) string {
	return mihomo.FormatBytes(bytes)
}

// FormatSpeed formats byte rate into human-readable transfer rate (e.g. 2.40 MB/s).
func FormatSpeed(bytesPerSec int64) string {
	return mihomo.FormatSpeed(bytesPerSec)
}

// FormatDelay formats latency in ms with readable format.
func FormatDelay(delay int) string {
	return mihomo.FormatDelay(delay)
}
