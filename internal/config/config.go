package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var (
	DBURL        = ""
	CredPath     = ""
	SyncInterval = 500 * time.Millisecond
	Host         = "0.0.0.0"
	Port         = 2324
)

func init() {
	// Load .env file if present
	_ = godotenv.Load()

	if v := os.Getenv("FIREBASE_DB_URL"); v != "" {
		DBURL = v
	}
	if v := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); v != "" {
		CredPath = v
	} else if v := os.Getenv("FIREBASE_CREDENTIALS_JSON"); v != "" {
		// Optional: Support passing JSON content directly (needs manual parsing, out of scope for now)
		// Or assume user sets GOOGLE_APPLICATION_CREDENTIALS path
	}

	if v := os.Getenv("HOST"); v != "" {
		Host = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			Port = p
		}
	}
}
