package admin

import (
	"bytes"
	"log"
	"net/http"
	"slices"
	"text/template"

	"smallweb.today/internal/db"
	"smallweb.today/internal/models"
	"smallweb.today/internal/webutil"
)

type AdminPageData struct {
	Posts       []models.AnnotatedPost
	Definitions []models.AnnotationDefinition
}

func RenderAdminDashboard(w http.ResponseWriter, r *http.Request, store *db.DB) {

	// get the annotation definitions
	defs, err := store.GetAnnotationDefinitions(r.Context())
	if err != nil {
		http.Error(w, "Error fetching annotation definitions", 500)
		return
	}

	// get the data
	posts, err := store.GetAnnotatedPosts(r.Context())
	if err != nil {
		log.Println(err)
		http.Error(w, "Error fetching posts", 500)
		return
	}

	// define some helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"formatTime": webutil.RelativeTime,

		// check if the map has the key and if the value matches
		"isSelected": func(annotations map[string][]string, attr string, val string) bool {
			return slices.Contains(annotations[attr], val)
		},
	}

	// munge the data
	var buf bytes.Buffer
	pageData := AdminPageData{
		Posts:       posts,
		Definitions: defs,
	}

	// render
	tmpl, err := template.New("admin_dashboard.gohtml").Funcs(funcMap).ParseFiles("templates/admin_dashboard.gohtml")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}

	if err := tmpl.Execute(&buf, pageData); err != nil {
		log.Printf("Admin Error (Execute): %v", err)
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)

}
