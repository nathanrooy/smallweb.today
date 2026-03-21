package main

import (
	"cmp"
	"context"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"smallweb.today/internal/db"
)

func main() {

	log.Println("start")

	// check for db connection string
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	// initialize the db
	store, err := db.Open(os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	// setup the landing page
	var landingPageHTML atomic.Value
	landingPageHTML.Store("<h1>Feed checker warming up...</h1>")

	// setup the background feed processor and shared state
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the background feed processor
	go runBatch(ctx, store, &landingPageHTML)

	// define web routes
	router := Router(store, &landingPageHTML)
	http.HandleFunc("/", router)

	// static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// this call hangs forever until the program is killed
	port := cmp.Or(os.Getenv("PORT"), "8080")
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
