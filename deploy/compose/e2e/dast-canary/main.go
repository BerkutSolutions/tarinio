package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type record struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query"`
	At     string `json:"at"`
}

type recorder struct {
	mu      sync.Mutex
	records []record
}

func (r *recorder) reset(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	r.records = nil
	r.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (r *recorder) list(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	items := append([]record(nil), r.records...)
	r.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"count": len(items), "records": items})
}

func (r *recorder) serve(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	r.records = append(r.records, record{
		Method: req.Method, Path: req.URL.Path, Query: req.URL.RawQuery, At: time.Now().UTC().Format(time.RFC3339Nano),
	})
	r.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "reached-canary"})
}

func main() {
	r := &recorder{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__dast/records", r.list)
	mux.HandleFunc("POST /__dast/reset", r.reset)
	mux.HandleFunc("/", r.serve)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
