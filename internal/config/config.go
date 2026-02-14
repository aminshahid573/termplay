package config

import (
	"os"
	"strconv"
	"time"
)

var (
	DBURL        = "https://YOUR-FIREBASE-DB-URL.firebasedatabase.app"
	CredPath     = "serviceAccount.json"
	SyncInterval = 500 * time.Millisecond
	Host         = "localhost"
	Port         = 2324
)

func init() {
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
