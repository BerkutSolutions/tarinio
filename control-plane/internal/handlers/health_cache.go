package handlers

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

type healthResponseCache struct {
	mu       sync.Mutex
	ttl      time.Duration
	written  time.Time
	status   int
	header   http.Header
	body     []byte
}

func newHealthResponseCache(ttl time.Duration) *healthResponseCache {
	return &healthResponseCache{ttl: ttl}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cache := h.responseCache
	if cache == nil {
		h.serveFresh(w, r)
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.status != 0 && time.Since(cache.written) < cache.ttl {
		writeCachedHealthResponse(w, cache.status, cache.header, cache.body)
		return
	}
	recorder := httptest.NewRecorder()
	h.serveFresh(recorder, r)
	cache.status = recorder.Code
	cache.header = recorder.Header().Clone()
	cache.body = append(cache.body[:0], recorder.Body.Bytes()...)
	cache.written = time.Now()
	writeCachedHealthResponse(w, cache.status, cache.header, cache.body)
}

func writeCachedHealthResponse(w http.ResponseWriter, status int, header http.Header, body []byte) {
	for key, values := range header {
		w.Header()[key] = append([]string(nil), values...)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
