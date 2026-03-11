package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type JSONMap map[string][]string

// Value makes JSONMap implement the driver.Valuer interface (for saving to DB)
func (m JSONMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan makes JSONMap implement the sql.Scanner interface (for reading from DB)
func (m *JSONMap) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &m)
}

type AnnotationDefinition struct {
	Attribute string   `db:"attribute"`
	Options   []string `db:"options"`
}

type AnnotatedPost struct {
	BaseURL       string    `db:"base_url"`
	PostURL       string    `db:"post_url"`
	PostTitle     string    `db:"post_title"`
	DiscoveredUTC time.Time `db:"utc_discovered"`
	PublishedUTC  time.Time `db:"utc_published"`
	Sources       []string  `db:"sources"`
	Verified      bool      `db:"verified"`
	VerifiedCount int       `db:"verified_count"`
	Hide          bool      `db:"hide"`
	Annotations   JSONMap   `db:"annotations"`
}

type AnnotationRecord struct {
	BaseURL         string
	PostURL         string
	Annotator       string
	AnnotationType  string
	AnnotationValue string
	AnnotationUTC   time.Time
}
