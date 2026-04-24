package source

import "errors"

var ErrRedisBackendNotImplemented = errors.New("redis sentinel source backend is reserved for HA mode and is not implemented yet")

type RedisBackend struct{}

func NewRedisBackend() *RedisBackend {
	return &RedisBackend{}
}

func (b *RedisBackend) Read(offset int64) ([]Event, int64, error) {
	return nil, offset, ErrRedisBackendNotImplemented
}
