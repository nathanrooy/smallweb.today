package api

// var landingPageHTML atomic.Value

// func Handler(w http.ResponseWriter, r *http.Request) {

// 	// set the edge caching: 5 minute refresh
// 	// w.Header().Set("Cache-Control", "public, s-maxage=300, stale-while-revalidate=86400")

// 	// router
// 	switch r.URL.Path {
// 	case "/":
// 		renderIndex(w, r)
// 	case "/admin", "/admin/":
// 		admin.AuthMiddleware(admin.RenderAdminDashboard)(w, r)
// 	case "/admin/login":
// 		admin.LoginHandler(w, r)
// 	case "/admin/logout":
// 		admin.AuthMiddleware(admin.LogoutHandler)(w, r)
// 	case "/admin/dashboard":
// 		admin.AuthMiddleware(admin.RenderAdminDashboard)(w, r)
// 	case "/admin/save-annotations":
// 		admin.AuthMiddleware(admin.SaveAnnotations)(w, r)
// 	default:
// 		http.NotFound(w, r)
// 	}
// }

// func renderIndex(w http.ResponseWriter, r *http.Request) {

// 	// get the data
// 	posts, err := db.GetPosts()
// 	if err != nil {
// 		http.Error(w, "Error fetching posts", 500)
// 		return
// 	}

// 	// define some helper functions
// 	funcMap := template.FuncMap{
// 		"add": func(a, b int) int {
// 			return a + b
// 		},
// 		"formatTime": webutil.RelativeTime,
// 	}

// 	// render
// 	tmpl, err := template.New("index.gohtml").Funcs(funcMap).ParseFiles("internal/templates/index.gohtml")
// 	if err != nil {
// 		http.Error(w, "Template error: "+err.Error(), 500)
// 		return
// 	}
// 	tmpl.Execute(w, posts)
// 	log.Println("rendered new landing page")
// }
