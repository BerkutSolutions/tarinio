package redis

import (
	"context"
	"net"
	"testing"
	"time"
)

type fakeDialer struct {
	conn net.Conn
	err  error
}

func (f *fakeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.conn, nil
}

type fakeConn struct {
	response []byte
	readDone bool
	written  []byte
}

func (f *fakeConn) Read(p []byte) (int, error) {
	if f.readDone {
		return 0, net.ErrClosed
	}
	n := copy(p, f.response)
	f.readDone = true
	return n, nil
}

func (f *fakeConn) Write(p []byte) (int, error) {
	f.written = append(f.written, p...)
	return len(p), nil
}

func (f *fakeConn) Close() error                       { return nil }
func (f *fakeConn) LocalAddr() net.Addr                { return fakeAddr("local") }
func (f *fakeConn) RemoteAddr() net.Addr               { return fakeAddr("remote") }
func (f *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (f *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (f *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

type fakeAddr string

func (f fakeAddr) Network() string { return "tcp" }
func (f fakeAddr) String() string  { return string(f) }

func TestClientPing(t *testing.T) {
	conn := &fakeConn{response: []byte("+PONG\r\n")}

	redisClient := NewClient(DefaultConfig())
	redisClient.dialer = &fakeDialer{conn: conn}

	if err := redisClient.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if len(conn.written) == 0 {
		t.Fatal("expected ping command to be written")
	}
}

func TestParseDB(t *testing.T) {
	db, err := ParseDB("2")
	if err != nil {
		t.Fatalf("parse db failed: %v", err)
	}
	if db != 2 {
		t.Fatalf("unexpected db: %d", db)
	}
}
