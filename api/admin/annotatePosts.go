package admin

import (
	"bytes"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"smallweb.today/internal/db"
	"smallweb.today/internal/models"
	"smallweb.today/internal/webutil"
)

type AdminPageData struct {
	Posts          []models.AnnotatedPost
	Definitions    []models.AnnotationDefinition
	Stats          map[string]int
	ShowVerified   bool
	ShowUnverified bool
}

func RenderPostAnnotationDashboard(w http.ResponseWriter, r *http.Request, store *db.DB) {

	// munge the data
	var buf bytes.Buffer
	pageData := AdminPageData{
		Stats:          make(map[string]int),
		ShowVerified:   false,
		ShowUnverified: false,
	}

	// parse the verified/unverified display url params
	query := r.URL.Query()
	if query.Has("show-verified") {
		pageData.ShowVerified, _ = strconv.ParseBool(query.Get("show-verified"))
	}
	if query.Has("show-unverified") {
		pageData.ShowUnverified, _ = strconv.ParseBool(query.Get("show-unverified"))
	}

	// if no display params have been set, assign the default
	if pageData.ShowVerified == false && pageData.ShowUnverified == false {
		pageData.ShowVerified = true
	}

	// get the annotation definitions
	var err error
	pageData.Definitions, err = store.GetAnnotationDefinitions(r.Context())
	if err != nil {
		http.Error(w, "Error fetching annotation definitions", 500)
		return
	}

	// get the data
	pageData.Posts, err = store.GetAnnotatedPosts(r.Context(), pageData.ShowVerified, pageData.ShowUnverified)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error fetching posts", 500)
		return
	}

	// get some basic annotation stats
	pageData.Stats["Verified"] = 0
	pageData.Stats["VerifiedAdmin"] = 0
	pageData.Stats["Total"] = 0
	for _, post := range pageData.Posts {
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

	// munge the form data into annotation records
	updates := []models.AnnotationRecord{}
	for key, values := range r.PostForm {
		if strings.HasPrefix(key, "ann---") {
			parts := strings.Split(key, "---")
			if len(parts) == 6 && len(values) == 1 {
				updates = append(updates, models.AnnotationRecord{
					BaseURL:         parts[1],
					Target:          parts[2],
					TargetURL:       parts[3],
					Annotator:       parts[4],
					AnnotationType:  parts[5],
					AnnotationValue: values[0],
				})
			}
		}
	}

	log.Println(">>>", updates)

	// save new annotations
	err := store.SaveAnnotations(r.Context(), updates)
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
