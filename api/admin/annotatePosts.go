package admin

import (
	"bytes"
	"log"
	"net/http"
	"slices"
	"strings"
	"text/template"

	"smallweb.today/internal/db"
	"smallweb.today/internal/models"
	"smallweb.today/internal/webutil"
)

type AdminPageData struct {
	Posts       []models.AnnotatedPost
	Definitions []models.AnnotationDefinition
	Stats       map[string]int
}

func RenderPostAnnotationDashboard(w http.ResponseWriter, r *http.Request, store *db.DB) {

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

	// munge the data
	var buf bytes.Buffer
	pageData := AdminPageData{
		Posts:       posts,
		Definitions: defs,
		Stats:       make(map[string]int),
	}

	// get some basic annotation stats
	for _, post := range posts {
		pageData.Stats["Total"]++
		if post.Verified {
			pageData.Stats["Verified"]++
		}
		if post.VerifiedAdmin {
			pageData.Stats["VerifiedAdmin"]++
		}
	}

	// define some helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"pcnt": func(a, b int) float64 {
			if b == 0 {
				return 0
			}
			return 100 * float64(a) / float64(b)
		},
		"formatTime": webutil.RelativeTime,

		// check if the map has the key and if the value matches
		"isSelected": func(annotations map[string][]string, attr string, val string) bool {
			return slices.Contains(annotations[attr], val)
		},
	}

	// load the template
	tmpl, err := template.New("admin_annotate_posts.gohtml").Funcs(funcMap).ParseFiles("templates/admin_annotate_posts.gohtml")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}

	// render
	if err := tmpl.Execute(&buf, pageData); err != nil {
		log.Printf("Admin Error (Execute): %v", err)
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

func SavePostAnnotations(w http.ResponseWriter, r *http.Request, store *db.DB) {

	// only allow post requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// parse the form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// map: [PostURL] -> [Attribute] -> []Values
	updates := make(map[string]map[string][]string)

	// map: [PostURL] -> BaseURL
	baseURLs := make(map[string]string)

	for key, values := range r.PostForm {

		if strings.HasPrefix(key, "ann--") {
			parts := strings.Split(key, "--")
			if len(parts) == 3 {
				postURL, attr := parts[1], parts[2]
				if _, ok := updates[postURL]; !ok {
					updates[postURL] = make(map[string][]string)
				}
				updates[postURL][attr] = values
			}
		}

		// handle baseURL lookup
		if strings.HasPrefix(key, "base--") {
			parts := strings.Split(key, "--")
			if len(parts) == 2 {
				postURL := parts[1]
				baseURLs[postURL] = values[0]
			}
		}
	}

	// save new annotations
	err := store.SaveAnnotations(r.Context(), updates, baseURLs)
	if err != nil {
		log.Printf("CRITICAL DATABASE ERROR: %v", err)
		http.Error(w, "Failed to save new annotations", http.StatusInternalServerError)
		return
	}

	// refresh the materalized view feeds_filtered
	if err := store.RefreshViews(r.Context()); err != nil {
		log.Printf("View Refresh Error: %v", err)
	}

	// redirect back to the dashboard
	http.Redirect(w, r, "/admin/annotate-posts?success=true", http.StatusSeeOther)
}
