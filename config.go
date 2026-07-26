package main

import (
	"os"
)

var (
	SupabaseURL     string
	SupabaseAnonKey string
)

func initConfig() {
	SupabaseURL = getEnv("SUPABASE_URL", "http://127.0.0.1:8000")
	SupabaseAnonKey = getEnv("SUPABASE_ANON_KEY", "YOUR_SUPABASE_ANON_KEY")
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
