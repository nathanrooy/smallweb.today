package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"smallweb.today/internal/models"
)

func (d *DB) UpdateFeedStates(ctx context.Context, tx *sql.Tx, states []*models.FeedState) error {
	// nothing to save
	if len(states) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("INSERT INTO feed_states (feed_url, last_checked, last_modified, etag) VALUES ")

	vals := make([]interface{}, 0, len(states)*4)
	for i, s := range states {
		if i > 0 {
			b.WriteString(",")
		}
		n := i * 4
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4)
		vals = append(vals, s.FeedURL, s.LastChecked, s.LastModified, s.ETag)
	}

	b.WriteString(`
		ON CONFLICT (feed_url) DO UPDATE SET
			last_checked = EXCLUDED.last_checked,
			last_modified = EXCLUDED.last_modified,
			etag = EXCLUDED.etag`)

	_, err := tx.ExecContext(ctx, b.String(), vals...)
	return err
}

func (d *DB) GetFeedStates(ctx context.Context, limit int) ([]*models.FeedState, error) {
	query := `
		select base_url, feed_url, last_checked, last_modified, etag
		from feeds_queue
		order by last_checked asc
		limit $1`
	rows, err := d.conn.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.FeedState
	for rows.Next() {
		f := &models.FeedState{}
		if err := rows.Scan(&f.BaseURL, &f.FeedURL, &f.LastChecked, &f.LastModified, &f.ETag); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}

	return feeds, nil
}
