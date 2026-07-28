package main

import (
	"encoding/json"
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

func handleMap(w http.ResponseWriter, r *http.Request) {
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

	tmpl := template.Must(template.ParseFiles("templates/map.html"))
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

func handleLocations(w http.ResponseWriter, r *http.Request) {
	// 認証チェック
	cookie, err := r.Cookie("access_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	accessToken := cookie.Value

	// ユーザー情報取得 (既存関数を活用)
	user, err := fetchSupabaseUser(accessToken)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// 地点一覧取得
		locations, err := fetchLocationsFromSupabase(accessToken, user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(locations)

	case http.MethodPost:
		// 新規地点追加
		var loc Location
		if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		loc.UserID = user.ID // ログインユーザーのIDを設定

		if err := createLocationInSupabase(accessToken, loc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
