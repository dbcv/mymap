package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var (
	SupabaseURL     string
	SupabaseAnonKey string
)

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

// Data structures for Supabase Auth API
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error_description"`
	Msg         string `json:"msg"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type PageData struct {
	Error  string
	Email  string
	UserID string
}

func main() {
	_ = godotenv.Load()

	SupabaseURL = getEnv("SUPABASE_URL", "http://127.0.0.1:8000")
	SupabaseAnonKey = getEnv("SUPABASE_ANON_KEY", "YOUR_SUPABASE_ANON_KEY")

	log.Printf("Loaded Supabase URL: %s", SupabaseURL)

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/hello", handleHello)
	http.HandleFunc("/logout", handleLogout)

	log.Println("Server running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

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

func authenticateWithSupabase(email, password string) (string, error) {
	reqBody, _ := json.Marshal(LoginRequest{Email: email, Password: password})
	apiURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", SupabaseURL)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", SupabaseAnonKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Supabaseへの接続に失敗しました: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var authResp AuthResponse
	json.Unmarshal(body, &authResp)

	if resp.StatusCode != http.StatusOK {
		if authResp.Error != "" {
			return "", fmt.Errorf("ログイン失敗: %s", authResp.Error)
		}
		if authResp.Msg != "" {
			return "", fmt.Errorf("ログイン失敗: %s", authResp.Msg)
		}
		return "", fmt.Errorf("メールアドレスまたはパスワードが正しくありません")
	}

	return authResp.AccessToken, nil
}

func fetchSupabaseUser(accessToken string) (*UserResponse, error) {
	apiURL := fmt.Sprintf("%s/auth/v1/user", SupabaseURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", SupabaseAnonKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token")
	}

	body, _ := io.ReadAll(resp.Body)
	var user UserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
