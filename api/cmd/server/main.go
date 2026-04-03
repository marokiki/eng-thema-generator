package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"eng-theme-generator/api/internal/theme"
)

type server struct {
	service *theme.Service
}

func main() {
	srv := &server{service: theme.NewService()}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", srv.handleHealth)
	mux.HandleFunc("/api/theme", srv.handleTheme)

	addr := env("ADDR", ":8080")
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *server) handleTheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	prompt := s.service.Pick(
		r.Context(),
		query.Get("category"),
		query.Get("energy"),
		query.Get("mode"),
		time.Now(),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"prompt": prompt,
		"meta": map[string]string{
			"mode":     "random",
			"category": fallback(query.Get("category"), "any"),
			"energy":   fallback(query.Get("energy"), "any"),
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func env(key, fallbackValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallbackValue
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
