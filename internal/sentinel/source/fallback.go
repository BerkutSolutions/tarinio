package source

import "errors"

// FallbackBackend reads from a prioritized backend list and falls back when
// a backend returns a temporary/unavailable signal.
type FallbackBackend struct {
	backends []Backend
}

func NewFallbackBackend(backends ...Backend) *FallbackBackend {
	filtered := make([]Backend, 0, len(backends))
	for _, backend := range backends {
		if backend == nil {
			continue
		}
		filtered = append(filtered, backend)
	}
	return &FallbackBackend{backends: filtered}
}

func (b *FallbackBackend) Read(offset int64) ([]Event, int64, error) {
	if b == nil || len(b.backends) == 0 {
		return nil, offset, nil
	}
	var lastErr error
	for _, backend := range b.backends {
		items, nextOffset, err := backend.Read(offset)
		if err == nil {
			return items, nextOffset, nil
		}
		if errors.Is(err, ErrRedisBackendNotImplemented) {
			lastErr = err
			continue
		}
		return nil, offset, err
	}
	return nil, offset, lastErr
}
