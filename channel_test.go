package t1k

import (
	"errors"
	"net"
	"testing"
	"time"
)

// 创建一个模拟TcpFactory
type MockFactory struct{}

func (f *MockFactory) Factory() (interface{}, error) {
	// 创建一对内存中相互连接的连接
	client, _ := net.Pipe()
	return client, nil
}

func (f *MockFactory) Close(v interface{}) error {
	return nil
}

func (f *MockFactory) Ping(v interface{}) error {
	return nil
}

type failDialFactory struct {
	MockFactory
}

func (f *failDialFactory) Factory() (interface{}, error) {
	return nil, errors.New("dial failed")
}

type failPingFactory struct {
	MockFactory
}

func (f *failPingFactory) Ping(v interface{}) error {
	return errors.New("ping failed")
}

func newTestPool(t *testing.T, cfg PoolConfig) *ChannelPool {
	t.Helper()
	pool, err := NewChannelPool(&cfg)
	if err != nil {
		t.Fatalf("NewChannelPool: %v", err)
	}
	return pool
}

func TestPoolStatsInitial(t *testing.T) {
	pool := newTestPool(t, PoolConfig{
		InitialCap:  2,
		MaxIdle:     4,
		MaxCap:      8,
		Factory:     &MockFactory{},
		IdleTimeout: 30 * time.Second,
	})
	defer pool.Release()

	stats := pool.Stats()
	if stats.IdleConns != 2 {
		t.Errorf("IdleConns = %d, want 2", stats.IdleConns)
	}
	if stats.ActiveConns != 2 {
		t.Errorf("ActiveConns = %d, want 2", stats.ActiveConns)
	}
	if stats.MaxActive != 8 {
		t.Errorf("MaxActive = %d, want 8", stats.MaxActive)
	}
	if stats.WaitingReqs != 0 {
		t.Errorf("WaitingReqs = %d, want 0", stats.WaitingReqs)
	}
}

func TestPoolStatsDialFailed(t *testing.T) {
	pool := newTestPool(t, PoolConfig{
		InitialCap:  0,
		MaxIdle:     1,
		MaxCap:      1,
		Factory:     &failDialFactory{},
		IdleTimeout: 30 * time.Second,
	})
	defer pool.Release()

	_, err := pool.Get()
	if err == nil {
		t.Fatal("expected dial error")
	}
	if pool.Stats().DialFailed != 1 {
		t.Errorf("DialFailed = %d, want 1", pool.Stats().DialFailed)
	}
}

func TestPoolStatsIdleExpired(t *testing.T) {
	pool := newTestPool(t, PoolConfig{
		InitialCap:  1,
		MaxIdle:     1,
		MaxCap:      1,
		Factory:     &MockFactory{},
		IdleTimeout: time.Millisecond,
	})
	defer pool.Release()

	time.Sleep(2 * time.Millisecond)

	_, err := pool.Get()
	if err != nil {
		t.Fatalf("Get after idle expiry: %v", err)
	}
	if pool.Stats().IdleExpired != 1 {
		t.Errorf("IdleExpired = %d, want 1", pool.Stats().IdleExpired)
	}
}

func TestPoolStatsPingFailed(t *testing.T) {
	pool := newTestPool(t, PoolConfig{
		InitialCap:  1,
		MaxIdle:     1,
		MaxCap:      1,
		Factory:     &failPingFactory{},
		IdleTimeout: 30 * time.Second,
	})
	defer pool.Release()

	_, err := pool.Get()
	if err != nil {
		t.Fatalf("Get after ping failure: %v", err)
	}
	if pool.Stats().PingFailed != 1 {
		t.Errorf("PingFailed = %d, want 1", pool.Stats().PingFailed)
	}
}

func TestPoolStatsPoolFullClose(t *testing.T) {
	pool := newTestPool(t, PoolConfig{
		InitialCap:  1,
		MaxIdle:     1,
		MaxCap:      2,
		Factory:     &MockFactory{},
		IdleTimeout: 30 * time.Second,
	})
	defer pool.Release()

	conn1, err := pool.Get()
	if err != nil {
		t.Fatalf("Get conn1: %v", err)
	}
	conn2, err := pool.Get()
	if err != nil {
		t.Fatalf("Get conn2: %v", err)
	}

	if err := pool.Put(conn1); err != nil {
		t.Fatalf("Put conn1: %v", err)
	}
	if err := pool.Put(conn2); err != nil {
		t.Fatalf("Put conn2: %v", err)
	}
	if pool.Stats().PoolFullClose != 1 {
		t.Errorf("PoolFullClose = %d, want 1", pool.Stats().PoolFullClose)
	}
}

func TestPutAfterRelease(t *testing.T) {
	// 创建连接池
	pool, _ := NewChannelPool(&PoolConfig{
		InitialCap:  1,
		MaxIdle:     16,
		MaxCap:      32,
		Factory:     &MockFactory{},
		IdleTimeout: 30 * time.Second},
	)

	// 获取一个连接
	conn, _ := pool.Get()

	// 释放连接池
	pool.Release()

	// 尝试归还连接，应该不会panic，而是返回一个错误
	err := pool.Put(conn)
	if err == nil {
		t.Error("Expected error when putting connection after release")
	}
}
