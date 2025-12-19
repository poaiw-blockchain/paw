package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type response struct {
	Status string      `json:"status,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/health":
		json.NewEncoder(w).Encode(response{Status: "ok"})
	default:
		json.NewEncoder(w).Encode(response{Data: []interface{}{}})
	}
}

func main() {
	addr := ":8080"
	if v := os.Getenv("STUB_PORT"); v != "" {
		addr = v
	}
	http.HandleFunc("/", handler)
	log.Printf("Starting stub indexer on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
