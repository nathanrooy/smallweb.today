package main

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"sync/atomic"

	"smallweb.today/internal/db"
	"smallweb.today/internal/webutil"
)

//go:embed templates/*
var TemplateFS embed.FS

var indexTmpl *template.Template

func init() {

	// define some helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"formatTime": webutil.RelativeTime,
	}

	var err error
	indexTmpl, err = template.New("index.gohtml").Funcs(funcMap).ParseFS(TemplateFS, "templates/index.gohtml")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
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

	// render
	var buf bytes.Buffer
	if err := indexTmpl.Execute(&buf, posts); err != nil {
		log.Printf("Refresh Error: Template execution failure: %v", err)
		return
	}

	// update the html
	landingPageHTML.Store(buf.String())
	log.Printf("Success: Landing page refreshed with %d posts", len(posts))
}
