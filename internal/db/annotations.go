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
		attribute,
		array_agg(value order by value asc) as options
	from public.annotation_definitions
	group by attribute
	order by attribute asc;`

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

		// IMPORTANT: Postgres text[] requires pq.Array() to scan into a Go slice
		err := rows.Scan(&def.Attribute, pq.Array(&def.Options))
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
			&p.BaseURL, &p.PostURL, &p.PostTitle, &p.DiscoveredUTC, &p.PublishedUTC,
			pq.Array(&p.Sources), &p.Verified, &p.VerifiedAdmin, &p.VerifiedCount, &p.Hide, &p.Annotations); err != nil {
			return nil, err
		}
		annotatedPosts = append(annotatedPosts, p)
	}

	return annotatedPosts, nil
}

func (d *DB) SaveAnnotations(ctx context.Context, updates map[string]map[string][]string, baseURLs map[string]string) error {

	validEntries := []models.AnnotationRecord{}
	affectedPosts := make(map[string]bool)
	var pURLs, aTypes, aVals []string

	// munge the form data into records
	for postURL, attrs := range updates {
		affectedPosts[postURL] = true
		for attr, values := range attrs {
			for _, val := range values {
				if val == "" {
					continue
				}

				rec := models.AnnotationRecord{
					BaseURL:         baseURLs[postURL],
					PostURL:         postURL,
					AnnotationType:  attr,
					AnnotationValue: val,
				}

				validEntries = append(validEntries, rec)
				pURLs = append(pURLs, rec.PostURL)
				aTypes = append(aTypes, rec.AnnotationType)
				aVals = append(aVals, rec.AnnotationValue)

			}
		}
	}

	// cleanup
	urls := make([]string, 0, len(affectedPosts))
	for url := range affectedPosts {
		urls = append(urls, url)
	}

	return d.WithTx(ctx, func(tx *sql.Tx) error {

		// bulk upsert.
		for _, rec := range validEntries {
			_, err := tx.ExecContext(ctx, `
            INSERT INTO public.post_annotations (base_url, post_url, annotator, annotation_type, annotation_value)
            VALUES ($1, $2, 'admin', $3, $4)
            ON CONFLICT (post_url, annotation_type, annotation_value) DO NOTHING`,
				rec.BaseURL, rec.PostURL, rec.AnnotationType, rec.AnnotationValue,
			)
			if err != nil {
				log.Printf("DB ERROR during INSERT: %v", err)
				return err
			}
		}

		// cleanup
		if len(urls) > 0 {
			// We delete anything for these posts that was NOT just inserted/preserved.
			// We use a temporary values table (unnest) to compare against our list.
			cleanupSQL := `
            DELETE FROM public.post_annotations
            WHERE post_url = ANY($1)
            AND (post_url, annotation_type, annotation_value) NOT IN (
                SELECT post_url, annotation_type, annotation_value 
                FROM UNNEST($2::text[], $3::text[], $4::text[]) 
                AS t(post_url, annotation_type, annotation_value)
            )`

			// perform the cleanup
			_, err := tx.ExecContext(ctx, cleanupSQL, pq.Array(urls), pq.Array(pURLs), pq.Array(aTypes), pq.Array(aVals))
			if err != nil {
				return fmt.Errorf("bulk cleanup failed: %w", err)
			}

		}

		return nil
	})
}
