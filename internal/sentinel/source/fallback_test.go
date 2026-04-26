package source

import (
	"errors"
	"testing"
)

type fakeBackend struct {
	events []Event
	offset int64
	err    error
}

func (f fakeBackend) Read(offset int64) ([]Event, int64, error) {
	if f.err != nil {
		return nil, offset, f.err
	}
	return append([]Event(nil), f.events...), f.offset, nil
}

func TestFallbackBackendFallsBackFromRedisStub(t *testing.T) {
	backend := NewFallbackBackend(
		NewRedisBackend(),
		fakeBackend{events: []Event{{IP: "203.0.113.10"}}, offset: 42},
	)
	items, nextOffset, err := backend.Read(0)
	if err != nil {
		t.Fatalf("read via fallback: %v", err)
	}
	if nextOffset != 42 {
		t.Fatalf("expected offset 42, got %d", nextOffset)
	}
	if len(items) != 1 || items[0].IP != "203.0.113.10" {
		t.Fatalf("unexpected events: %+v", items)
	}
}

func TestFallbackBackendReturnsErrorWhenNoBackendCanRead(t *testing.T) {
	backend := NewFallbackBackend(NewRedisBackend())
	_, _, err := backend.Read(0)
	if !errors.Is(err, ErrRedisBackendNotImplemented) {
		t.Fatalf("expected redis not implemented error, got %v", err)
	}
}
