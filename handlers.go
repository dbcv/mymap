package main

import (
	"html/template"
	"net/http"
	"time"
)

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if tokenCookie, err := r.Cookie("access_token"); err == nil && tokenCookie.Value != "" {
		if _, err := fetchSupabaseUser(tokenCookie.Value); err == nil {
			http.Redirect(w, r, "/hello", http.StatusSeeOther)
			return
		}
	}

	tmpl := template.Must(template.ParseFiles("templates/login.html"))

	if r.Method == http.MethodGet {
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")

		token, err := authenticateWithSupabase(email, password)
		if err != nil {
			tmpl.Execute(w, PageData{Error: err.Error()})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "access_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   3600 * 24, // 1日
		})

		http.Redirect(w, r, "/hello", http.StatusSeeOther)
	}
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("access_token")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := fetchSupabaseUser(cookie.Value)
	if err != nil {
		clearAuthCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/hello.html"))
	tmpl.Execute(w, PageData{
		Email:  user.Email,
		UserID: user.ID,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
