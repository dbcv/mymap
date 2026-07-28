package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

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

func fetchLocationsFromSupabase(token, userID string) ([]Location, error) {

	req, err := http.NewRequest("GET", SupabaseURL+"/rest/v1/locations?user_id=eq."+userID+"&select=*", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var locations []Location
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, err
	}
	return locations, nil
}

func createLocationInSupabase(token string, loc Location) error {
	body, _ := json.Marshal(loc)
	req, err := http.NewRequest("POST", SupabaseURL+"/rest/v1/locations", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Supabase save failed: %d", resp.StatusCode)
		return fmt.Errorf("supabase save failed: %d", resp.StatusCode)
	}
	return nil
}
