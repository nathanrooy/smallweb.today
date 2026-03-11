package models

import "time"

type FeedState struct {
	BaseURL      string
	FeedURL      string
	LastChecked  time.Time
	LastModified string
	ETag         string
}
