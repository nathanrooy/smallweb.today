package admin

import (
	"log"
	"net/http"
	"strings"

	"smallweb.today/internal/db"
)

func SaveAnnotations(w http.ResponseWriter, r *http.Request, store *db.DB) {

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
	http.Redirect(w, r, "/admin/dashboard?success=true", http.StatusSeeOther)
}
