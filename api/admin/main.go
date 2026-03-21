package admin

import (
	"bytes"
	"log"
	"net/http"
	"text/template"

	"smallweb.today/internal/db"
)

func RenderAdminMain(w http.ResponseWriter, r *http.Request, store *db.DB) {

	tmpl, err := template.New("admin_main.gohtml").ParseFiles("templates/admin_main.gohtml")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		log.Printf("Admin Error (Execute): %v", err)
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}
