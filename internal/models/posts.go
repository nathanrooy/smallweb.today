package models

import "time"

type Post struct {
	BaseURL       string
	FeedURL       string
	PostURL       string
	PostTitle     string
	DiscoveredUTC time.Time
	PublishedUTC  time.Time
}
