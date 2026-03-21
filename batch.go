package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"smallweb.today/internal/db"
	"smallweb.today/internal/models"
	"smallweb.today/internal/worker"
)

const (
	// number of feeds to process in one batch.
	batchSize = 5000

	// number feed workers
	workerCount = 4

	// wait times for different states
	sleepOnEmpty = 10 * time.Minute
	sleepOnError = 30 * time.Second
)

// update the db with the lastest discoveries
func commitBatch(ctx context.Context, store *db.DB, states []*models.FeedState, posts []*models.Post) error {
	return store.WithTx(ctx, func(tx *sql.Tx) error {

		// update feed states
		if err := store.UpdateFeedStates(ctx, tx, states); err != nil {
			log.Printf("Error updating states: %v", err)
			return err

		}

		// update new posts
		if err := store.SavePosts(ctx, tx, posts); err != nil {
			log.Printf("Error saving posts: %v", err)
			return err
		}

		// commit new inserts/updates
		return nil
	})
}

// periodically fetches and processes a batch of stale feeds.
func runBatch(ctx context.Context, store *db.DB, landingPageHTML *atomic.Value) {
	w := worker.New(store)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Println("Batcher: Fetching stale feeds...")

			tStart := time.Now()

			// pull stale feeds
			feeds, err := store.GetFeedStates(ctx, batchSize)
			if err != nil {
				log.Printf("Batcher error: %v. Retrying in %v...", err, sleepOnError)
				time.Sleep(sleepOnError)
				continue
			}

			// no stale feeds
			if len(feeds) == 0 {
				log.Printf("Batcher: No stale feeds. Sleeping for %v...", sleepOnEmpty)
				time.Sleep(sleepOnEmpty)
				continue
			}

			// delegate to helper for cleaner flow
			updatedStates, allNewPosts := processFeeds(ctx, w, feeds)

			if err := commitBatch(ctx, store, updatedStates, allNewPosts); err != nil {
				log.Printf("Batcher Error: Save failed: %v. Skipping...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Println(strings.Repeat("-", 80))
			log.Printf("> batch summary :: duration = %0.2fm :: new posts = %d", time.Since(tStart).Minutes(), len(allNewPosts))
			log.Println(strings.Repeat("-", 80))

			refreshLandingPage(store, landingPageHTML)
			log.Println("Batcher: Regenerating landing page...")
		}
	}
}

// handles the concurrent fetching of feeds
func processFeeds(ctx context.Context, w *worker.Worker, feeds []*models.FeedState) ([]*models.FeedState, []*models.Post) {

	// create and fill the channel
	jobs := make(chan *models.FeedState, len(feeds))
	results := make(chan worker.WorkerResult, len(feeds))
	for _, f := range feeds {
		jobs <- f
	}
	close(jobs)

	// run the worker pool
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Run(ctx, jobs, results)
		}()
	}

	// create a separate goroutine to close the results
	go func() {
		wg.Wait()
		close(results)
	}()

	// collect the results
	updatedStates := make([]*models.FeedState, 0, len(feeds))
	var allNewPosts []*models.Post
	for r := range results {
		updatedStates = append(updatedStates, r.State)
		if r.Posts != nil {
			allNewPosts = append(allNewPosts, r.Posts...)
		}
	}

	return updatedStates, allNewPosts
}
