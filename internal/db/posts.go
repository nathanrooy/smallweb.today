package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"smallweb.today/internal/models"
)

func dedupePosts(posts []*models.Post) []*models.Post {
	uniquePostsMap := make(map[string]*models.Post)
	for _, post := range posts {
		uniquePostsMap[post.PostURL] = post
	}

	// convert batck to a slice
	var dedupedPosts []*models.Post
	for _, post := range uniquePostsMap {
		dedupedPosts = append(dedupedPosts, post)
	}

	return dedupedPosts
}

func (d *DB) SavePosts(ctx context.Context, tx *sql.Tx, posts []*models.Post) error {
	// nothing to save
	if len(posts) == 0 {
		return nil
	}

	// query header
	var b strings.Builder
	b.WriteString("UPSERT INTO posts (base_url, feed_url, post_url, post_title, utc_published, utc_discovered) VALUES ")

	// munge the posts for postgres
	vals := make([]interface{}, 0, len(posts)*6)
	for i, p := range dedupePosts(posts) {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 6
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5, n+6)
		vals = append(vals, p.BaseURL, p.FeedURL, p.PostURL, p.PostTitle, p.PublishedUTC, p.DiscoveredUTC)
	}

	_, err := tx.ExecContext(ctx, b.String(), vals...)
	return err
}

// Get the most recent posts from the last 24 hours
func (d *DB) GetPosts() ([]models.Post, error) {

	// query the db
	sql := `
        select base_url, post_url, post_title, utc_discovered, utc_published
        from public.posts_latest_filtered
        order by utc_discovered desc
        limit 5000;`
	rows, err := d.conn.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// iterate through the results
	var posts []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.BaseURL, &p.PostURL, &p.PostTitle, &p.DiscoveredUTC, &p.PublishedUTC); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, nil
}
