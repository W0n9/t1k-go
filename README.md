# t1k-go

[![Go Reference](https://pkg.go.dev/badge/github.com/chaitin/t1k-go.svg)](https://pkg.go.dev/github.com/chaitin/t1k-go)
[![CI](https://github.com/chaitin/t1k-go/actions/workflows/go.yml/badge.svg)](https://github.com/chaitin/t1k-go/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Go SDK for the T1K protocol, used to communicate with [Chaitin SafeLine](https://github.com/chaitin/safeline) Web Application Firewall detection engine.

[中文文档](README.zh.md)

## Features

- Full T1K binary protocol implementation (TLV framing, multi-section messages)
- HTTP request and response detection
- Two connection pool implementations:
  - `Server` — fixed-size pool with background heartbeat
  - `ChannelPool` — configurable pool with idle timeout and max capacity
- Bot detection support (`BotQuery` / `BotBody` parsing)
- Health check with T1K heartbeat and HTTP protocol support
- Configurable heartbeat keepalive interval
- Zero external dependencies

## Installation

```bash
go get github.com/chaitin/t1k-go
```

## Quick Start

The simplest way to detect HTTP requests using the built-in `Server` pool:

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/chaitin/t1k-go"
)

func main() {
	server, err := t1k.NewWithPoolSizeWithTimeout("127.0.0.1:8000", 10, 10*time.Second)
	if err != nil {
		panic(err)
	}
	defer server.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		result, err := server.DetectHttpRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if result.Blocked() {
			http.Error(w, fmt.Sprintf("blocked, event_id: %s", result.EventID()), result.StatusCode())
			return
		}
		w.Write([]byte("allowed"))
	})
	http.ListenAndServe(":80", nil)
}
```

## ChannelPool Usage

`ChannelPool` provides more control over connection management:

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/chaitin/t1k-go"
)

func main() {
	pool, err := t1k.NewChannelPool(&t1k.PoolConfig{
		InitialCap:  1,
		MaxIdle:     16,
		MaxCap:      32,
		Factory:     &t1k.TcpFactory{Addr: "127.0.0.1:8000"},
		IdleTimeout: 30 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	defer pool.Release()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		result, err := pool.DetectHttpRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if result.Blocked() {
			http.Error(w, fmt.Sprintf("blocked, event_id: %s", result.EventID()), result.StatusCode())
			return
		}
		w.Write([]byte("allowed"))
	})
	http.ListenAndServe(":80", nil)
}
```

### PoolConfig Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `InitialCap` | `int` | Initial number of connections created at pool startup |
| `MaxIdle` | `int` | Maximum number of idle connections kept in the pool |
| `MaxCap` | `int` | Maximum total connections allowed (active + idle) |
| `Factory` | `ConnectionFactory` | Factory for creating, closing, and pinging connections |
| `IdleTimeout` | `time.Duration` | Connections idle longer than this are discarded |

Constraint: `InitialCap <= MaxIdle <= MaxCap`.

## Health Check

Health check runs as a background goroutine and supports two protocols:

```go
server, _ := t1k.NewWithPoolSize("127.0.0.1:8000", 10)

server.UpdateHealthCheckConfig(&t1k.HealthCheckConfig{
	Interval:            2,                              // check every 2 seconds
	HealthThreshold:     3,                              // 3 consecutive successes to recover
	UnhealthThreshold:   5,                              // 5 consecutive failures to mark unhealthy
	Addresses:           []string{"127.0.0.1:8001"},     // health check endpoint(s)
	HealthCheckProtocol: t1k.HEALTH_CHECK_T1K_PROTOCOL,  // or HEALTH_CHECK_HTTP_PROTOCOL
})

// Query health status
fmt.Println(server.IsHealth())
fmt.Printf("%+v\n", server.HealthCheckStats())
```

## Detection Context

For advanced use cases requiring request + response two-phase detection:

```go
import "github.com/chaitin/t1k-go/detection"

// Phase 1: Request detection
dc, _ := detection.MakeContextWithRequest(httpReq)
reqResult, _ := server.DetectRequestInCtx(dc)

// dc.T1KContext is automatically populated from the engine response
// and carried over to the response detection phase

// Phase 2: Response detection
detection.MakeHttpResponseInCtx(httpRsp, dc)
rspResult, _ := server.DetectResponseInCtx(dc)
```

`DetectionContext` automatically handles:
- UUID generation for request correlation
- Scheme, IP, port extraction from `http.Request`
- ServerName derivation (TLS SNI → Host header fallback)
- Timestamp recording for request/response phases
- `T1KContext` carry-over from request to response phase

## Result API

| Method | Return Type | Description |
|--------|-------------|-------------|
| `Passed()` | `bool` | `true` if the engine allows the request (`Head == '.'`) |
| `Blocked()` | `bool` | `true` if the engine blocks the request |
| `StatusCode()` | `int` | HTTP status code from engine response (defaults to 403) |
| `EventID()` | `string` | Unique event identifier extracted from `ExtraBody` |
| `BotDetected()` | `bool` | `true` if `BotQuery` is non-empty |
| `BlockMessage()` | `map[string]interface{}` | Structured block response with status, message, and event_id |

### Result Fields

| Field | Type | Description |
|-------|------|-------------|
| `Head` | `byte` | Detection verdict: `'.'` = pass, `'?'` = block, `'!'` = bot detected |
| `Body` | `[]byte` | Status code string from engine |
| `ExtraHeader` | `[]byte` | Extra headers to inject (engine-supplied) |
| `ExtraBody` | `[]byte` | Extra body content (contains event_id in HTML comment) |
| `T1KContext` | `[]byte` | Opaque context for request→response phase carry-over |
| `Cookie` | `[]byte` | Anti-tamper cookie data from engine |
| `BotQuery` | `[]byte` | Bot challenge query string |
| `BotBody` | `[]byte` | Bot challenge body |
| `WebLog` | `[]byte` | Web log flag from engine |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `T1K_HEARTBEAT_INTERVAL` | Heartbeat interval in seconds (Server pool only) | `20` |
| `DETECTOR_ADDR` | Engine address used in examples | — |
| `T1K_ADDR` | Engine address for integration tests (fallback default, not for production) | — |

## Testing

```bash
# Unit tests (no engine required)
go test ./...

# Integration tests (requires a reachable SafeLine engine)
go test -tags integration -v ./...

# Override engine address
T1K_ADDR=your-engine:8000 go test -tags integration -v ./...
```

Integration tests cover: normal request pass-through, SQL injection / XSS / path traversal / command injection blocking, POST body detection, known bad IP blocking, rate limiting, connection pool concurrency, and DetectionContext usage.

## Architecture

```
t1k-go/
├── server.go          # Server: fixed-size connection pool + heartbeat goroutine
├── channel.go         # ChannelPool: configurable pool with idle timeout
├── conn.go            # Connection wrapper with auto-reconnect on failure
├── detect.go          # Detection orchestration: serialize → send → parse
├── factory.go         # ConnectionFactory interface + TcpFactory
├── pool.go            # Pool interface + PoolConfig
├── heartbeat.go       # Heartbeat: zero-length FIRST|LAST section
├── health_check.go    # Health check service with threshold state machine
├── health_check_protocol.go  # T1K and HTTP health check protocols
├── detection/
│   ├── context.go     # DetectionContext: per-request metadata + T1KContext
│   ├── request.go     # Request interface + HttpRequest implementation
│   ├── response.go    # Response interface + HttpResponse implementation
│   ├── result.go      # Result: Passed/Blocked/EventID/BotDetected
│   └── extra.go       # Extra metadata builder (key:value format)
├── t1k/
│   ├── tag.go         # Tag constants + MASK_FIRST/MASK_LAST flags
│   ├── header.go      # 5-byte binary header (tag + LE uint32 size)
│   └── section.go     # Section read/write (SimpleSection + ReaderSection)
└── misc/
    ├── gen_uuid.go    # UUID generation (MT19937-based)
    ├── errors.go      # Error wrapping with caller info
    └── now.go         # Microsecond timestamp
```

### Detection Flow

```
HTTP Request
    │
    ▼
serialize as T1K sections:
  TAG_HEADER|MASK_FIRST → TAG_BODY → TAG_EXTRA → TAG_VERSION|MASK_LAST
    │
    ▼
send over TCP to SafeLine engine
    │
    ▼
read multi-section TLV response
    │
    ▼
parse into Result (Head, Body, ExtraBody, Context, Cookie, BotQuery, ...)
    │
    ▼
Passed() / Blocked() / EventID() / BotDetected()
```

## License

[Apache License 2.0](License)
