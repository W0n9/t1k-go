//go:build integration

package t1k

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/chaitin/t1k-go/detection"
)

func getTestAddr(t *testing.T) string {
	addr := os.Getenv("T1K_ADDR")
	if addr == "" {
		addr = "100.104.130.108:8000"
	}
	return addr
}

func makeRequest(t *testing.T, raw string) *http.Request {
	t.Helper()
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBufferString(raw)))
	if err != nil {
		t.Fatalf("failed to parse request: %v", err)
	}
	req.RemoteAddr = "10.0.0.1:12345"
	return req
}

// --- Server pool tests ---

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := NewWithPoolSizeWithTimeout(getTestAddr(t), 2, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestIntegrationServerNormalRequest(t *testing.T) {
	s := newTestServer(t)

	req := makeRequest(t, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := s.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	if !result.Passed() {
		t.Errorf("expected normal request to pass, got blocked (head=%c)", result.Head)
	}
}

func TestIntegrationServerSQLInjection(t *testing.T) {
	s := newTestServer(t)

	req := makeRequest(t, "GET /search?id=1'+OR+'1'='1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := s.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	if !result.Blocked() {
		t.Errorf("expected SQL injection to be blocked")
	}
	if result.EventID() == "" {
		t.Errorf("expected non-empty EventID for blocked request")
	}
	t.Logf("blocked: status=%d event_id=%s", result.StatusCode(), result.EventID())
}

func TestIntegrationServerXSS(t *testing.T) {
	s := newTestServer(t)

	req := makeRequest(t, "GET /page?q=<script>alert(1)</script> HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := s.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	if !result.Blocked() {
		t.Errorf("expected XSS to be blocked")
	}
	t.Logf("blocked: status=%d event_id=%s", result.StatusCode(), result.EventID())
}

func TestIntegrationServerPathTraversal(t *testing.T) {
	s := newTestServer(t)

	req := makeRequest(t, "GET /../../etc/passwd HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := s.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	if !result.Blocked() {
		t.Errorf("expected path traversal to be blocked")
	}
	t.Logf("blocked: status=%d event_id=%s", result.StatusCode(), result.EventID())
}

func TestIntegrationServerPostBody(t *testing.T) {
	s := newTestServer(t)

	body := "user=admin&pass=1' OR '1'='1"
	raw := "POST /login HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/x-www-form-urlencoded\r\n" +
		fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body)) +
		body
	req := makeRequest(t, raw)
	result, err := s.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	if !result.Blocked() {
		t.Errorf("expected SQL injection in POST body to be blocked")
	}
	t.Logf("blocked: status=%d event_id=%s", result.StatusCode(), result.EventID())
}

func TestIntegrationServerDetectContext(t *testing.T) {
	s := newTestServer(t)

	raw := "GET /safe HTTP/1.1\r\nHost: example.com\r\n\r\n"
	req := makeRequest(t, raw)

	dc, err := detection.MakeContextWithRequest(req)
	if err != nil {
		t.Fatalf("MakeContextWithRequest error: %v", err)
	}

	result, err := s.DetectRequestInCtx(dc)
	if err != nil {
		t.Fatalf("DetectRequestInCtx error: %v", err)
	}
	if !result.Passed() {
		t.Errorf("expected safe request to pass")
	}
	if dc.T1KContext == nil {
		t.Logf("note: T1KContext is nil (engine may not return context for passed requests)")
	}
}

func TestIntegrationServerMultipleRequests(t *testing.T) {
	s := newTestServer(t)

	for i := 0; i < 10; i++ {
		req := makeRequest(t, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
		result, err := s.DetectHttpRequest(req)
		if err != nil {
			t.Fatalf("request %d: DetectHttpRequest error: %v", i, err)
		}
		if !result.Passed() {
			t.Errorf("request %d: expected pass", i)
		}
	}
}

// --- ChannelPool tests ---

func newTestChannelPool(t *testing.T) *ChannelPool {
	t.Helper()
	pool, err := NewChannelPool(&PoolConfig{
		InitialCap:  1,
		MaxIdle:     4,
		MaxCap:      8,
		Factory:     &TcpFactory{Addr: getTestAddr(t)},
		IdleTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create channel pool: %v", err)
	}
	t.Cleanup(func() { pool.Release() })
	return pool
}

func TestIntegrationChannelPoolNormalRequest(t *testing.T) {
	pool := newTestChannelPool(t)

	req := makeRequest(t, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := pool.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	if !result.Passed() {
		t.Errorf("expected normal request to pass, got blocked (head=%c)", result.Head)
	}
}

func TestIntegrationChannelPoolSQLInjection(t *testing.T) {
	pool := newTestChannelPool(t)

	req := makeRequest(t, "GET /search?id=1'+OR+'1'='1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := pool.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	if !result.Blocked() {
		t.Errorf("expected SQL injection to be blocked")
	}
	if result.EventID() == "" {
		t.Errorf("expected non-empty EventID for blocked request")
	}
	t.Logf("blocked: status=%d event_id=%s", result.StatusCode(), result.EventID())
}

func TestIntegrationChannelPoolConcurrent(t *testing.T) {
	pool := newTestChannelPool(t)

	done := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() {
			req := makeRequest(t, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
			result, err := pool.DetectHttpRequest(req)
			if err != nil {
				done <- err
				return
			}
			if !result.Passed() {
				done <- fmt.Errorf("expected normal request to pass, got blocked (head=%c)", result.Head)
				return
			}
			done <- nil
		}()
	}
	for i := 0; i < 20; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent request error: %v", err)
		}
	}
}

func TestIntegrationChannelPoolWebshell(t *testing.T) {
	pool := newTestChannelPool(t)

	req := makeRequest(t, "GET /uploads/shell.php?cmd=whoami HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := pool.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	t.Logf("webshell result: passed=%v head=%c status=%d event_id=%s",
		result.Passed(), result.Head, result.StatusCode(), result.EventID())
}

func TestIntegrationKnownBadIP(t *testing.T) {
	s := newTestServer(t)

	req := makeRequest(t, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	req.RemoteAddr = "9.0.0.1:12345"

	dc, err := detection.MakeContextWithRequest(req)
	if err != nil {
		t.Fatalf("MakeContextWithRequest error: %v", err)
	}

	result, err := s.DetectRequestInCtx(dc)
	if err != nil {
		t.Fatalf("DetectRequestInCtx error: %v", err)
	}

	t.Logf("known bad IP result: passed=%v head=%c status=%d event_id=%s",
		result.Passed(), result.Head, result.StatusCode(), result.EventID())
	t.Logf("  ExtraBody=%q", string(result.ExtraBody))
	t.Logf("  BotDetected=%v BotQuery=%q BotBody=%q",
		result.BotDetected(), string(result.BotQuery), string(result.BotBody))

	if result.Passed() {
		t.Errorf("expected request from known bad IP 9.0.0.1 to be blocked, but it passed")
	}
}

func TestIntegrationRateLimit(t *testing.T) {
	s := newTestServer(t)

	sendFromIP := func(label string) *detection.Result {
		t.Helper()
		req := makeRequest(t, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
		req.RemoteAddr = "9.0.0.2:12345"
		dc, err := detection.MakeContextWithRequest(req)
		if err != nil {
			t.Fatalf("%s: MakeContextWithRequest error: %v", label, err)
		}
		result, err := s.DetectRequestInCtx(dc)
		if err != nil {
			t.Fatalf("%s: DetectRequestInCtx error: %v", label, err)
		}
		t.Logf("%s: passed=%v head=%c status=%d event_id=%s ExtraBody=%q",
			label, result.Passed(), result.Head, result.StatusCode(), result.EventID(), string(result.ExtraBody))
		return result
	}

	r1 := sendFromIP("req1")
	if !r1.Passed() {
		t.Logf("req1 was blocked (may be residual from prior rate window), waiting 6s to reset")
		time.Sleep(6 * time.Second)
		r1 = sendFromIP("req1-retry")
	}
	if !r1.Passed() {
		t.Fatalf("first request should pass after rate window reset")
	}

	r2 := sendFromIP("req2-immediate")
	if !r2.Blocked() {
		t.Errorf("second immediate request should be rate-limited, but it passed")
	}

	t.Log("waiting 6s for rate limit window to expire...")
	time.Sleep(6 * time.Second)

	r3 := sendFromIP("req3-after-wait")
	if !r3.Passed() {
		t.Errorf("request after rate window should pass, but got blocked (head=%c)", r3.Head)
	}
}

func TestIntegrationChannelPoolCommandInjection(t *testing.T) {
	pool := newTestChannelPool(t)

	raw := "POST /api/exec HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 27\r\n\r\n" +
		`{"cmd":"; cat /etc/passwd"}`
	req := makeRequest(t, raw)
	result, err := pool.DetectHttpRequest(req)
	if err != nil {
		t.Fatalf("DetectHttpRequest error: %v", err)
	}
	t.Logf("cmd injection result: passed=%v head=%c status=%d event_id=%s",
		result.Passed(), result.Head, result.StatusCode(), result.EventID())
}
