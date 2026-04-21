package redis

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

var ErrLockNotAcquired = errors.New("redis lock not acquired")

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
	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := c.writeCommand(conn, "PING"); err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	resp, err := readRESP(reader)
	if err != nil {
		return fmt.Errorf("read redis ping response: %w", err)
	}
	line, ok := resp.(string)
	if !ok || !strings.EqualFold(strings.TrimSpace(line), "PONG") {
		return fmt.Errorf("unexpected redis ping response: %v", resp)
	}
	return nil
}

func (c *Client) connect(ctx context.Context) (net.Conn, error) {
	conn, err := c.dialer.DialContext(ctx, "tcp", c.cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("dial redis: %w", err)
	}
	reader := bufio.NewReader(conn)
	if strings.TrimSpace(c.cfg.Password) != "" {
		args := []string{"AUTH"}
		if strings.TrimSpace(c.cfg.Username) != "" {
			args = append(args, c.cfg.Username)
		}
		args = append(args, c.cfg.Password)
		if _, err := c.execWithReader(conn, reader, args...); err != nil {
			conn.Close()
			return nil, err
		}
	}
	if c.cfg.DB > 0 {
		if _, err := c.execWithReader(conn, reader, "SELECT", strconv.Itoa(c.cfg.DB)); err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
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

func (c *Client) exec(ctx context.Context, parts ...string) (any, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	return c.execWithReader(conn, reader, parts...)
}

func (c *Client) execWithReader(conn net.Conn, reader *bufio.Reader, parts ...string) (any, error) {
	if err := c.writeCommand(conn, parts...); err != nil {
		return nil, err
	}
	resp, err := readRESP(reader)
	if err != nil {
		return nil, err
	}
	if redisErr, ok := resp.(redisError); ok {
		return nil, redisErr
	}
	return resp, nil
}

type redisError string

func (e redisError) Error() string { return string(e) }

func readRESP(reader *bufio.Reader) (any, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+':
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
	case '-':
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		return redisError(strings.TrimSpace(line)), nil
	case ':':
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		value, convErr := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
		if convErr != nil {
			return nil, convErr
		}
		return value, nil
	case '$':
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr != nil {
			return nil, convErr
		}
		if size < 0 {
			return nil, nil
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return nil, err
		}
		return string(buf[:size]), nil
	case '*':
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		count, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr != nil {
			return nil, convErr
		}
		if count < 0 {
			return nil, nil
		}
		items := make([]any, 0, count)
		for i := 0; i < count; i++ {
			item, err := readRESP(reader)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported redis response prefix %q", string(prefix))
	}
}

type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (HeldLock, error)
}

type HeldLock interface {
	Key() string
	Token() string
	Refresh(ctx context.Context, ttl time.Duration) error
	Release(ctx context.Context) error
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
		locks:  &redisLocker{client: client},
		queues: &redisQueue{client: client},
		state:  &redisEphemeralState{client: client},
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

type redisHeldLock struct {
	client *Client
	key    string
	token  string
}

func (l *redisHeldLock) Key() string {
	return l.key
}

func (l *redisHeldLock) Token() string {
	return l.token
}

func (l *redisHeldLock) Refresh(ctx context.Context, ttl time.Duration) error {
	if l == nil || l.client == nil {
		return fmt.Errorf("redis lock unavailable")
	}
	script := "if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('PEXPIRE', KEYS[1], ARGV[2]) else return 0 end"
	resp, err := l.client.exec(ctx, "EVAL", script, "1", l.key, l.token, strconv.FormatInt(ttl.Milliseconds(), 10))
	if err != nil {
		return err
	}
	value, ok := resp.(int64)
	if !ok || value != 1 {
		return ErrLockNotAcquired
	}
	return nil
}

func (l *redisHeldLock) Release(ctx context.Context) error {
	if l == nil || l.client == nil {
		return fmt.Errorf("redis lock unavailable")
	}
	script := "if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) else return 0 end"
	resp, err := l.client.exec(ctx, "EVAL", script, "1", l.key, l.token)
	if err != nil {
		return err
	}
	value, ok := resp.(int64)
	if !ok || value != 1 {
		return ErrLockNotAcquired
	}
	return nil
}

type redisLocker struct {
	client *Client
}

func (r *redisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (HeldLock, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("redis client unavailable")
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	resp, err := r.client.exec(ctx, "SET", strings.TrimSpace(key), token, "NX", "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, ErrLockNotAcquired
	}
	line, ok := resp.(string)
	if !ok || !strings.EqualFold(strings.TrimSpace(line), "OK") {
		return nil, ErrLockNotAcquired
	}
	return &redisHeldLock{client: r.client, key: strings.TrimSpace(key), token: token}, nil
}

type redisQueue struct {
	client *Client
}

func (r *redisQueue) Enqueue(ctx context.Context, queue string, payload []byte) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client unavailable")
	}
	_, err := r.client.exec(ctx, "RPUSH", strings.TrimSpace(queue), string(payload))
	return err
}

type redisEphemeralState struct {
	client *Client
}

func (r *redisEphemeralState) Put(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client unavailable")
	}
	resp, err := r.client.exec(ctx, "SET", strings.TrimSpace(key), string(value), "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	if err != nil {
		return err
	}
	line, ok := resp.(string)
	if !ok || !strings.EqualFold(strings.TrimSpace(line), "OK") {
		return fmt.Errorf("unexpected redis response: %v", resp)
	}
	return nil
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

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate redis lock token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
