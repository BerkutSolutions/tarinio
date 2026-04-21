package storage

import (
	"errors"
	"os"
	"strings"
)

type JSONState struct {
	backend    Backend
	key        string
	legacyPath string
}

func NewFileJSONState(path string) *JSONState {
	return &JSONState{
		backend:    NewFileBackend(),
		key:        path,
		legacyPath: path,
	}
}

func NewBackendJSONState(backend Backend, key string, legacyPath string) *JSONState {
	return &JSONState{
		backend:    backend,
		key:        strings.TrimSpace(key),
		legacyPath: strings.TrimSpace(legacyPath),
	}
}

func (s *JSONState) Load() ([]byte, error) {
	if s == nil || s.backend == nil {
		return nil, errors.New("json state backend is not configured")
	}
	content, err := s.backend.LoadDocument(s.key)
	if err == nil {
		return content, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if strings.TrimSpace(s.legacyPath) == "" || s.legacyPath == s.key {
		return nil, ErrNotFound
	}
	content, readErr := os.ReadFile(s.legacyPath)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, readErr
	}
	if saveErr := s.backend.SaveDocument(s.key, content); saveErr != nil {
		return nil, saveErr
	}
	return content, nil
}

func (s *JSONState) Save(content []byte) error {
	if s == nil || s.backend == nil {
		return errors.New("json state backend is not configured")
	}
	return s.backend.SaveDocument(s.key, content)
}

func (s *JSONState) Delete() error {
	if s == nil || s.backend == nil {
		return errors.New("json state backend is not configured")
	}
	return s.backend.DeleteDocument(s.key)
}
