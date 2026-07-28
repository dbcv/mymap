package main

// Supabase Auth API
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

// Template Data
type PageData struct {
	Error  string
	Email  string
	UserID string
}

type Location struct {
	ID     string  `json:"id,omitempty"`
	UserID string  `json:"user_id,omitempty"`
	Name   string  `json:"name"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	Memo   string  `json:"memo"`
}
