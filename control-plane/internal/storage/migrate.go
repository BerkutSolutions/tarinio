package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func MigrateLegacyDocument(backend Backend, key string, legacyPath string) error {
	exists, err := HasDocument(backend, key)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	content, err := os.ReadFile(strings.TrimSpace(legacyPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return backend.SaveDocument(key, content)
}

func MigrateLegacyBlob(backend Backend, key string, legacyPath string) error {
	exists, err := HasBlob(backend, key)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	content, err := os.ReadFile(strings.TrimSpace(legacyPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return backend.SaveBlob(key, content)
}

func MigrateLegacyBlobDir(backend Backend, keyPrefix string, legacyDir string) error {
	info, err := os.Stat(strings.TrimSpace(legacyDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(legacyDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "files" {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(legacyDir, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(filepath.Join(keyPrefix, relative))
		return MigrateLegacyBlob(backend, key, path)
	})
}

func MigrateLegacyDocumentDir(backend Backend, keyPrefix string, legacyDir string) error {
	info, err := os.Stat(strings.TrimSpace(legacyDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	keys := make([]string, 0)
	fileByKey := make(map[string]string)
	err = filepath.WalkDir(legacyDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "files" {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(legacyDir, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(filepath.Join(keyPrefix, relative))
		keys = append(keys, key)
		fileByKey[key] = path
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := MigrateLegacyDocument(backend, key, fileByKey[key]); err != nil {
			return err
		}
	}
	return nil
}
