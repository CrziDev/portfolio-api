// Command portfolio-api runs the portfolio's API (see internal/api).
//
// Configuration comes from the environment:
//
//	ADDR             listen address (default ":8001")
//	DATA_DIR         holds portfolio.db and uploads/ (default "./data")
//	ADMIN_PASSWORD   password for /api/login; editing is disabled when unset
//	ALLOWED_ORIGINS  comma-separated origins the pages are served from
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"portfolio-api/internal/api"
	"portfolio-api/internal/store"
)

func main() {
	addr := env("ADDR", ":8001")
	dataDir := env("DATA_DIR", "./data")
	password := os.Getenv("ADMIN_PASSWORD")
	origins := splitList(os.Getenv("ALLOWED_ORIGINS"))

	if password == "" {
		log.Print("ADMIN_PASSWORD is not set; editing is disabled")
	}
	if len(origins) == 0 {
		log.Print("ALLOWED_ORIGINS is not set; browsers on other sites can't call the API")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	uploadDir := filepath.Join(dataDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatal(err)
	}
	// The seed's screenshots, which seed.json refers to as /uploads/<project>-….webp.
	if err := store.CopySeedUploads(uploadDir); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(ctx, filepath.Join(dataDir, "portfolio.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	server := &http.Server{
		Addr:              addr,
		Handler:           api.New(api.Config{Store: st, Password: password, UploadDir: uploadDir, Origins: origins}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	failed := make(chan error, 1)
	go func() { failed <- server.ListenAndServe() }()
	log.Printf("listening on %s", addr)

	select {
	case err := <-failed:
		log.Fatal(err)
	case <-ctx.Done():
	}

	// Let requests in flight finish before the database closes.
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		log.Print(err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// splitList splits a comma-separated value, dropping blanks and trailing slashes,
// since an Origin header never ends in one.
func splitList(value string) []string {
	var list []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimRight(strings.TrimSpace(item), "/"); item != "" {
			list = append(list, item)
		}
	}
	return list
}
