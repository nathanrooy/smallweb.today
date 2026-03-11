package main

import (
	"log"
	"net/http"
	"sync/atomic"

	"smallweb.today/api/admin"
	"smallweb.today/internal/db"
)

func Router(store *db.DB, landingPageHTML *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s [%s]", r.Method, r.URL.Path, r.RemoteAddr)

		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(landingPageHTML.Load().(string)))

		case "/admin", "/admin/":
			admin.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
				admin.RenderAdminDashboard(w, r, store)
			})(w, r)

		case "/admin/dashboard":
			admin.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
				admin.RenderAdminDashboard(w, r, store)
			})(w, r)

		case "/admin/save-annotations":
			admin.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
				admin.SaveAnnotations(w, r, store)
			})(w, r)

		case "/admin/login":
			admin.LoginHandler(w, r)

		case "/admin/logout":
			admin.AuthMiddleware(admin.LogoutHandler)(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}
