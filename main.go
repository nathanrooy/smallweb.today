package main

import (
	"bytes"
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"smallweb.today/internal/db"
	"smallweb.today/internal/models"
	"smallweb.today/internal/webutil"
	"smallweb.today/internal/worker"
)

func main() {

	log.Println("start")

	// check for db connection string
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// initialize the db
	store, err := db.Open(os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	// setup the landing page
	var landingPageHTML atomic.Value
	landingPageHTML.Store("<h1>Crawler warming up...</h1>")

	// setup the worker and shared state
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// batch controller
	go runBatch(ctx, store, &landingPageHTML)

	// define web routes
	router := Router(store, &landingPageHTML)
	http.HandleFunc("/", router)

	// static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// This call hangs forever until the program is killed
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

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

// run through 1k feeds + update the landing page
func runBatch(ctx context.Context, store *db.DB, landingPageHTML *atomic.Value) {
	w := worker.New(store)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Println("Batcher: Fetching 1000 stale feeds...")

			// pull stale feeds
			feeds, err := store.GetFeedStates(ctx, 100)
			if err != nil {
				log.Printf("Batcher error: %v. Retrying in 30s...", err)
				time.Sleep(30 * time.Second)
				continue
			}

			// no stale feeds
			if len(feeds) == 0 {
				log.Println("Batcher: No stale feeds. Sleeping for 10m...")
				time.Sleep(10 * time.Minute)
				continue
			}

			//  create and fill the cahnnel
			jobs := make(chan *models.FeedState, len(feeds))
			results := make(chan worker.WorkerResult, len(feeds))
			for _, f := range feeds {
				jobs <- f
			}
			close(jobs)

			// run the worker pool
			totalFeeds := len(feeds)
			var currentCount int64 = 0
			var wg sync.WaitGroup
			for i := 0; i < 10; i++ {
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

			// collect the results into a single array
			var updatedStates []*models.FeedState
			var allNewPosts []*models.Post
			for r := range results {

				// print out some stats
				currentCount++
				log.Printf("[%d/%d] :: status code = %d :: %s :: items = %d",
					currentCount,
					totalFeeds,
					r.StatusCode,
					r.State.BaseURL,
					len(r.Posts),
				)

				// munge the results
				updatedStates = append(updatedStates, r.State)
				if r.Posts != nil {
					allNewPosts = append(allNewPosts, r.Posts...)
				}
			}

			if err := commitBatch(ctx, store, updatedStates, allNewPosts); err != nil {
				log.Printf("Batcher Error: Save failed: %v. Skipping...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// print some stats
			log.Println(strings.Repeat("-", 80))
			log.Printf("> batch summary :: duration = :: new posts = %d", len(allNewPosts))
			log.Println(strings.Repeat("-", 80))

			// regenerate the landing page after the current batch finishes
			refreshLandingPage(store, landingPageHTML)
			log.Println("Batcher: Regenerating landing page...")
		}
	}
}

func refreshLandingPage(store *db.DB, landingPageHTML *atomic.Value) {

	log.Println("refreshing landing page...")

	// get the data
	posts, err := store.GetPosts()
	if err != nil {
		log.Printf("Refresh Error: Failed to fetch posts: %v", err)
		return
	}

	// define some helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"formatTime": webutil.RelativeTime,
	}

	// parse the template
	// TODO -> parse this once at startup
	tmpl, err := template.New("index.gohtml").Funcs(funcMap).ParseFiles("templates/index.gohtml")
	if err != nil {
		log.Printf("Refresh Error: Template execution failure: %v", err)
		return
	}

	// render
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, posts); err != nil {
		log.Printf("Refresh Error: Template execution failure: %v", err)
		return
	}

	// update the html
	landingPageHTML.Store(buf.String())
	log.Printf("Success: Landing page refreshed with %d posts", len(posts))
}
