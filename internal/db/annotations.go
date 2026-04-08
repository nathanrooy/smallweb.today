package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
	"smallweb.today/internal/models"
)

func (d *DB) GetAnnotationDefinitions(ctx context.Context) ([]models.AnnotationDefinition, error) {

	// define the query
	sql := `
	select
		target,
		attribute,
		array_agg(value order by value asc) as options
	from public.annotation_definitions
	group by attribute, target
	order by attribute, target asc;`

	// perform the query
	rows, err := d.conn.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	// iterate through the results
	var definitions []models.AnnotationDefinition
	for rows.Next() {
		var def models.AnnotationDefinition
		err := rows.Scan(&def.Target, &def.Attribute, pq.Array(&def.Options))
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		definitions = append(definitions, def)
	}

	// check for errors
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return definitions, nil
}

func (d *DB) GetAnnotatedPosts(ctx context.Context, verified bool, unverified bool) ([]models.AnnotatedPost, error) {

	// construct the query
	query := `
	SELECT
		base_url,
		feed_url,
		post_url,
		post_title,
		utc_discovered,
		utc_published,
		sources,
		verified,
		verified_admin,
		verified_count,
		hide,
		annotations
	from public.posts_latest_annotated`

	// add visability filters
	if verified == true && unverified == false {
		query += ` where verified = true`
	} else if verified == false && unverified == true {
		query += ` where verified = false`
	}

	// finish
	query += ` order by utc_discovered desc limit 2500;`

	// get the data
	rows, err := d.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// iterate through the results
	var annotatedPosts []models.AnnotatedPost
	for rows.Next() {
		var p models.AnnotatedPost
		if err := rows.Scan(
			&p.BaseURL, &p.FeedURL, &p.PostURL, &p.PostTitle, &p.DiscoveredUTC, &p.PublishedUTC,
			pq.Array(&p.Sources), &p.Verified, &p.VerifiedAdmin, &p.VerifiedCount, &p.Hide, &p.Annotations); err != nil {
			return nil, err
		}
		annotatedPosts = append(annotatedPosts, p)
	}

	return annotatedPosts, nil
}

func (d *DB) SaveAnnotations(ctx context.Context, updates []models.AnnotationRecord) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {

		// bulk upsert
		urls := make(map[string]struct{})
		for _, annotation := range updates {
			_, err := tx.ExecContext(ctx, `
            INSERT INTO public.annotations (base_url, target, target_url, annotator, annotation_type, annotation_value)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (base_url, target, target_url, annotator, annotation_type, annotation_value) DO NOTHING`,
				annotation.BaseURL,
				annotation.Target,
				annotation.TargetURL,
				annotation.Annotator,
				annotation.AnnotationType,
				annotation.AnnotationValue,
			)
			if err != nil {
				log.Printf("DB ERROR during INSERT: %v", err)
				return err
			} else {
				urls[annotation.TargetURL] = struct{}{}
			}
		}

		return nil
	})
}
