package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

//go:embed static/*
var staticFS embed.FS

type statusResponse struct {
	Overall    string            `json:"overall"`
	Components []ComponentStatus `json:"components"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Interval   string            `json:"interval"`
}

func main() {
	cfg := loadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	monitor := NewMonitor(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go monitor.Start(ctx)

	mux := http.NewServeMux()
	mux.Handle("/api/status", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		overall, components, updatedAt := monitor.Snapshot()
		writeJSON(w, statusResponse{
			Overall:    overall,
			Components: components,
			UpdatedAt:  updatedAt,
			Interval:   cfg.CheckInterval.String(),
		})
	}))
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("static assets error: %v", err)
	}
	fileServer := http.FileServer(http.FS(staticRoot))
	mux.Handle("/", fileServer)

	server := &http.Server{
		Addr:         ":" + itoa(cfg.Port),
		Handler:      logMiddleware(mux),
		ReadTimeout:  cfg.HTTPTimeout + 2*time.Second,
		WriteTimeout: cfg.HTTPTimeout + 2*time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Status page listening on :%d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
