package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
	"smallweb.today/internal/db"
	"smallweb.today/internal/models"
)

type Worker struct {
	db     *db.DB
	client *http.Client
	parser *gofeed.Parser
}

type WorkerResult struct {
	State      *models.FeedState
	Posts      []*models.Post
	Err        error
	StatusCode int
}

// creates a worker instance with a few shared utilities
func New(database *db.DB) *Worker {
	return &Worker{
		db:     database,
		parser: gofeed.NewParser(),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (w *Worker) Run(ctx context.Context, jobs <-chan *models.FeedState, results chan<- WorkerResult) {
	for f := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			workerResult := w.processFeed(ctx, f)
			workerResult.State.LastChecked = time.Now().UTC()
			results <- workerResult
		}
	}
}

func (w *Worker) processFeed(ctx context.Context, f *models.FeedState) WorkerResult {

	// build the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.FeedURL, nil)
	if err != nil {
		return WorkerResult{State: f, Posts: nil, Err: nil}
	}

	// add the conditional get headers (if we have them)
	if f.ETag != "" {
		req.Header.Set("If-None-Match", f.ETag)
	}
	if f.LastModified != "" {
		req.Header.Set("If-Modified-Since", f.LastModified)
	}

	// make the request
	resp, err := w.client.Do(req)
	if err != nil {
		return WorkerResult{State: f, Posts: nil, Err: nil, StatusCode: 0}
	}
	defer resp.Body.Close()

	// handle "304" - not modified
	if resp.StatusCode == http.StatusNotModified {
		return WorkerResult{State: f, Posts: nil, Err: nil, StatusCode: resp.StatusCode}
	}

	// handle 200 - parse the feed
	if resp.StatusCode == http.StatusOK {

		// parse the response body directly
		feed, err := w.parser.Parse(resp.Body)
		if err != nil {
			return WorkerResult{State: f, Posts: nil, Err: nil, StatusCode: resp.StatusCode}
		}

		// cycle through the feed items
		var newPosts []*models.Post
		cutoff := time.Now().UTC().Add(-24 * time.Hour)
		for _, item := range feed.Items {

			// make sure the post has a publication date and title
			if item.PublishedParsed != nil && item.Title != "" &&

				// make sure the item has been posted within the last 24 hours and not from the future
				item.PublishedParsed.After(cutoff) && item.PublishedParsed.Before(time.Now().UTC()) {

				newPosts = append(newPosts, &models.Post{
					BaseURL:       f.BaseURL,
					FeedURL:       f.FeedURL,
					PostURL:       item.Link,
					PostTitle:     item.Title,
					DiscoveredUTC: time.Now().UTC(),
					PublishedUTC:  *item.PublishedParsed,
				})
			}
		}

		f.LastModified = resp.Header.Get("Last-Modified")
		f.ETag = resp.Header.Get("ETag")
		return WorkerResult{State: f, Posts: newPosts, Err: nil, StatusCode: resp.StatusCode}
	}

	return WorkerResult{State: f, Posts: nil, Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode), StatusCode: resp.StatusCode}
}
