package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eng-theme-generator/api/internal/theme"
)

type server struct {
	service *theme.Service
	static  http.Handler
}

func main() {
	srv := &server{
		service: theme.NewService(),
		static:  newStaticHandler(env("WEB_DIST_DIR", "web/dist")),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", srv.handleHealth)
	mux.HandleFunc("/api/theme", srv.handleTheme)
	mux.HandleFunc("/api/advice", srv.handleAdvice)
	if srv.static != nil {
		mux.Handle("/", srv.static)
	}

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

func (s *server) handleAdvice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var request struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 32*1024)).Decode(&request); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"advice": s.service.ReviewEnglish(r.Context(), request.Text),
		"meta": map[string]int{
			"characters": len([]rune(strings.TrimSpace(request.Text))),
			"words":      len(strings.Fields(strings.TrimSpace(request.Text))),
		},
	})
}

func newStaticHandler(dir string) http.Handler {
	indexPath := filepath.Join(dir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("static assets unavailable: %v", err)
		}
		return nil
	}

	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		if r.URL.Path == "/" || !strings.Contains(filepath.Base(r.URL.Path), ".") {
			http.ServeFile(w, r, indexPath)
			return
		}

		fileServer.ServeHTTP(w, r)
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
