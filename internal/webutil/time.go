package webutil

import (
	"fmt"
	"time"
)

func RelativeTime(t time.Time) string {
	duration := time.Since(t)

	// if it's within 15 minutes
	if duration.Seconds() < 15*60 {
		return " just now"
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	var timeStr = " "
	if hours == 1 {
		timeStr += " 1 hour"
	} else if hours > 1 {
		timeStr += fmt.Sprintf(" %d hours", hours)
	}

	if hours > 0 && minutes > 0 {
		timeStr += " and"
	}

	if minutes == 1 {
		timeStr += " 1 minute"
	} else if minutes > 1 {
		timeStr += fmt.Sprintf(" %d minutes", minutes)
	}

	return timeStr + " ago"
}
