package redis

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddr        = "127.0.0.1:6379"
	defaultDialTimeout = 3 * time.Second
)

// Config contains the minimal Redis wiring for coordination-only usage.
type Config struct {
	Addr        string
	Username    string
	Password    string
	DB          int
	DialTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		Addr:        defaultAddr,
		DB:          0,
		DialTimeout: defaultDialTimeout,
	}
}

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Client struct {
	cfg    Config
	dialer Dialer
}

func NewClient(cfg Config) *Client {
	dialer := &net.Dialer{Timeout: cfg.DialTimeout}
	if cfg.Addr == "" {
		cfg.Addr = defaultAddr
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = defaultDialTimeout
	}
	return &Client{cfg: cfg, dialer: dialer}
}

func (c *Client) Config() Config {
	return c.cfg
}

func (c *Client) Ping(ctx context.Context) error {
	conn, err := c.dialer.DialContext(ctx, "tcp", c.cfg.Addr)
	if err != nil {
		return fmt.Errorf("dial redis: %w", err)
	}
	defer conn.Close()

	if err := c.writeCommand(conn, "PING"); err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read redis ping response: %w", err)
	}
	if !strings.HasPrefix(line, "+PONG") {
		return fmt.Errorf("unexpected redis ping response: %s", strings.TrimSpace(line))
	}
	return nil
}

func (c *Client) writeCommand(conn net.Conn, parts ...string) error {
	if _, err := fmt.Fprintf(conn, "*%d\r\n", len(parts)); err != nil {
		return fmt.Errorf("write redis command prefix: %w", err)
	}
	for _, part := range parts {
		if _, err := fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(part), part); err != nil {
			return fmt.Errorf("write redis command part: %w", err)
		}
	}
	return nil
}

type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) error
}

type Queue interface {
	Enqueue(ctx context.Context, queue string, payload []byte) error
}

type EphemeralState interface {
	Put(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type Backend struct {
	client *Client
	locks  Locker
	queues Queue
	state  EphemeralState
}

func NewBackend(client *Client) *Backend {
	return &Backend{
		client: client,
		locks:  &redisLocker{},
		queues: &redisQueue{},
		state:  &redisEphemeralState{},
	}
}

func (b *Backend) Client() *Client {
	return b.client
}

func (b *Backend) Locks() Locker {
	return b.locks
}

func (b *Backend) Queues() Queue {
	return b.queues
}

func (b *Backend) Ephemeral() EphemeralState {
	return b.state
}

type redisLocker struct{}

func (r *redisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) error {
	return fmt.Errorf("redis lock acquire not implemented in baseline")
}

type redisQueue struct{}

func (r *redisQueue) Enqueue(ctx context.Context, queue string, payload []byte) error {
	return fmt.Errorf("redis queue enqueue not implemented in baseline")
}

type redisEphemeralState struct{}

func (r *redisEphemeralState) Put(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return fmt.Errorf("redis ephemeral state put not implemented in baseline")
}

func ParseDB(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	db, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("redis db must be numeric")
	}
	if db < 0 {
		return 0, fmt.Errorf("redis db must be zero or positive")
	}
	return db, nil
}
