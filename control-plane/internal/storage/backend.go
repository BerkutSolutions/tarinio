package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrNotFound = errors.New("storage object not found")

type Backend interface {
	LoadDocument(key string) ([]byte, error)
	SaveDocument(key string, content []byte) error
	DeleteDocument(key string) error
	LoadBlob(key string) ([]byte, error)
	SaveBlob(key string, content []byte) error
	DeleteBlob(key string) error
	DeleteBlobsByPrefix(prefix string) error
	ListBlobKeys(prefix string) ([]string, error)
}

func IsNilBackend(backend Backend) bool {
	if backend == nil {
		return true
	}
	value := reflect.ValueOf(backend)
	switch value.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func:
		return value.IsNil()
	default:
		return false
	}
}

func HasDocument(backend Backend, key string) (bool, error) {
	if backend == nil {
		return false, errors.New("backend is not configured")
	}
	_, err := backend.LoadDocument(key)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}

func HasBlob(backend Backend, key string) (bool, error) {
	if backend == nil {
		return false, errors.New("backend is not configured")
	}
	_, err := backend.LoadBlob(key)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}

type FileBackend struct{}

func NewFileBackend() *FileBackend {
	return &FileBackend{}
}

func (b *FileBackend) LoadDocument(key string) ([]byte, error) {
	content, err := os.ReadFile(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (b *FileBackend) SaveDocument(key string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(key), 0o755); err != nil {
		return err
	}
	return os.WriteFile(key, content, 0o644)
}

func (b *FileBackend) DeleteDocument(key string) error {
	if err := os.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (b *FileBackend) LoadBlob(key string) ([]byte, error) {
	content, err := os.ReadFile(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (b *FileBackend) SaveBlob(key string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(key), 0o755); err != nil {
		return err
	}
	return os.WriteFile(key, content, 0o600)
}

func (b *FileBackend) DeleteBlob(key string) error {
	if err := os.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (b *FileBackend) DeleteBlobsByPrefix(prefix string) error {
	if strings.TrimSpace(prefix) == "" {
		return nil
	}
	if err := os.RemoveAll(prefix); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (b *FileBackend) ListBlobKeys(prefix string) ([]string, error) {
	info, err := os.Stat(prefix)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{prefix}, nil
	}
	out := make([]string, 0)
	err = filepath.WalkDir(prefix, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

type PostgresBackend struct {
	db *sql.DB
}

func NewPostgresBackend(dsn string) (*PostgresBackend, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("postgres dsn is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxIdleConns(4)
	db.SetMaxOpenConns(16)
	return &PostgresBackend{db: db}, nil
}

func (b *PostgresBackend) DB() *sql.DB {
	return b.db
}

func (b *PostgresBackend) Ping() error {
	if b == nil || b.db == nil {
		return errors.New("postgres backend is not initialized")
	}
	return b.db.Ping()
}

func (b *PostgresBackend) LoadDocument(key string) ([]byte, error) {
	return b.load("state_documents", key)
}

func (b *PostgresBackend) SaveDocument(key string, content []byte) error {
	return b.save("state_documents", key, content)
}

func (b *PostgresBackend) DeleteDocument(key string) error {
	return b.del("state_documents", key)
}

func (b *PostgresBackend) LoadBlob(key string) ([]byte, error) {
	return b.load("state_blobs", key)
}

func (b *PostgresBackend) SaveBlob(key string, content []byte) error {
	return b.save("state_blobs", key, content)
}

func (b *PostgresBackend) DeleteBlob(key string) error {
	return b.del("state_blobs", key)
}

func (b *PostgresBackend) DeleteBlobsByPrefix(prefix string) error {
	_, err := b.db.Exec(`DELETE FROM state_blobs WHERE key LIKE $1`, strings.TrimSpace(prefix)+"%")
	return err
}

func (b *PostgresBackend) ListBlobKeys(prefix string) ([]string, error) {
	rows, err := b.db.Query(`SELECT key FROM state_blobs WHERE key LIKE $1 ORDER BY key`, strings.TrimSpace(prefix)+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (b *PostgresBackend) load(table string, key string) ([]byte, error) {
	var content []byte
	err := b.db.QueryRow(`SELECT content FROM `+table+` WHERE key = $1`, strings.TrimSpace(key)).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (b *PostgresBackend) save(table string, key string, content []byte) error {
	_, err := b.db.Exec(
		`INSERT INTO `+table+` (key, content, updated_at) VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET content = EXCLUDED.content, updated_at = EXCLUDED.updated_at`,
		strings.TrimSpace(key),
		content,
	)
	return err
}

func (b *PostgresBackend) del(table string, key string) error {
	_, err := b.db.Exec(`DELETE FROM `+table+` WHERE key = $1`, strings.TrimSpace(key))
	return err
}
