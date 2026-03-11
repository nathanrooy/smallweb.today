package admin

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	sessionToken  string
	sessionExpiry time.Time
	sessionMutex  sync.RWMutex
)

func createToken() string {
	b := make([]byte, 64)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		RenderLoginTemplate(w, "")
		return

	} else if r.Method == http.MethodPost {

		time.Sleep(500 * time.Millisecond)

		// validate the password
		err := bcrypt.CompareHashAndPassword(
			[]byte(os.Getenv("ADMIN_HASH")),
			[]byte(r.FormValue("password")),
		)
		if err != nil {
			log.Printf("Failed login attempt from: %s", r.RemoteAddr)
			w.WriteHeader(http.StatusUnauthorized)
			RenderLoginTemplate(w, "Invalid credentials")
			return
		}

		// create new session in memory
		token := createToken()
		sessionMutex.Lock()
		sessionToken = token
		sessionExpiry = time.Now().Add(24 * time.Hour)
		sessionMutex.Unlock()

		// set the secure cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "admin_session",
			Value:    token,
			Path:     "/",  // available to all admin routes
			HttpOnly: true, // prevents JS access (XSS protection)
			Secure:   true, // only sent over HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400, // 24 hours in seconds
		})

		// redirect to the dashboard on success
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {

	// invalidate the current session
	sessionMutex.Lock()
	sessionToken = ""
	sessionMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:   "admin_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1, // tells browser to delete the cookie
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func RenderLoginTemplate(w http.ResponseWriter, errorMsg string) {
	tmpl, err := template.ParseFiles("templates/login.gohtml")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// wrap the error in a simple map so the template can read it
	data := map[string]string{
		"Error": errorMsg,
	}

	tmpl.Execute(w, data)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("verifying session token...")
		cookie, err := r.Cookie("admin_session")

		sessionMutex.RLock()
		isValidToken := err == nil &&
			sessionToken != "" &&
			cookie.Value == sessionToken &&
			time.Now().Before(sessionExpiry)
		sessionMutex.RUnlock()

		// if no cookie or token is invalid, redirect to login
		if !isValidToken {
			log.Println("session token is invalid...")
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		// successful login
		log.Println("session token is good...")
		next(w, r)
	}
}
