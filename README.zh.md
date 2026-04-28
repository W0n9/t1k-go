# t1k-go

[![Go Reference](https://pkg.go.dev/badge/github.com/chaitin/t1k-go.svg)](https://pkg.go.dev/github.com/chaitin/t1k-go)
[![CI](https://github.com/chaitin/t1k-go/actions/workflows/go.yml/badge.svg)](https://github.com/chaitin/t1k-go/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

T1K 协议的 Go SDK，用于与 [长亭 SafeLine](https://github.com/chaitin/safeline) Web 应用防火墙检测引擎通信。

[English](README.md)

## 功能特性

- 完整的 T1K 二进制协议实现（TLV 封装、多 Section 消息）
- HTTP 请求和响应检测
- 两种连接池实现：
  - `Server` — 固定大小连接池，带后台心跳
  - `ChannelPool` — 可配置连接池，支持空闲超时和最大容量控制
- Bot 检测支持（`BotQuery` / `BotBody` 解析）
- 健康检查（支持 T1K 心跳和 HTTP 两种协议）
- 可配置的心跳保活间隔
- 零外部依赖

## 安装

```bash
go get github.com/chaitin/t1k-go
```

## 快速开始

使用内置 `Server` 连接池检测 HTTP 请求的最简示例：

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

## 使用 ChannelPool

`ChannelPool` 提供更精细的连接管理控制：

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

### PoolConfig 参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| `InitialCap` | `int` | 连接池启动时创建的初始连接数 |
| `MaxIdle` | `int` | 连接池中保持的最大空闲连接数 |
| `MaxCap` | `int` | 允许的最大连接总数（活跃 + 空闲） |
| `Factory` | `ConnectionFactory` | 用于创建、关闭和探活连接的工厂接口 |
| `IdleTimeout` | `time.Duration` | 空闲超过此时间的连接将被丢弃 |

约束：`InitialCap <= MaxIdle <= MaxCap`。

## 健康检查

健康检查以后台 goroutine 运行，支持两种协议：

```go
server, _ := t1k.NewWithPoolSize("127.0.0.1:8000", 10)

server.UpdateHealthCheckConfig(&t1k.HealthCheckConfig{
	Interval:            2,                              // 每 2 秒检查一次
	HealthThreshold:     3,                              // 连续 3 次成功恢复健康
	UnhealthThreshold:   5,                              // 连续 5 次失败标记为不健康
	Addresses:           []string{"127.0.0.1:8001"},     // 健康检查端点
	HealthCheckProtocol: t1k.HEALTH_CHECK_T1K_PROTOCOL,  // 或 HEALTH_CHECK_HTTP_PROTOCOL
})

// 查询健康状态
fmt.Println(server.IsHealth())
fmt.Printf("%+v\n", server.HealthCheckStats())
```

## 检测上下文

用于请求 + 响应两阶段检测的高级用法：

```go
import "github.com/chaitin/t1k-go/detection"

// 第一阶段：请求检测
dc, _ := detection.MakeContextWithRequest(httpReq)
reqResult, _ := server.DetectRequestInCtx(dc)

// dc.T1KContext 会自动从引擎响应中填充
// 并传递到响应检测阶段

// 第二阶段：响应检测
detection.MakeHttpResponseInCtx(httpRsp, dc)
rspResult, _ := server.DetectResponseInCtx(dc)
```

`DetectionContext` 自动处理：
- 请求关联的 UUID 生成
- 从 `http.Request` 提取 Scheme、IP、端口
- ServerName 推导（TLS SNI → Host 头回退）
- 请求/响应阶段的时间戳记录
- 请求到响应阶段的 `T1KContext` 传递

## Result API

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `Passed()` | `bool` | 引擎放行请求时返回 `true`（`Head == '.'`） |
| `Blocked()` | `bool` | 引擎拦截请求时返回 `true` |
| `StatusCode()` | `int` | 引擎返回的 HTTP 状态码（默认 403） |
| `EventID()` | `string` | 从 `ExtraBody` 中提取的唯一事件标识符 |
| `BotDetected()` | `bool` | `BotQuery` 非空时返回 `true` |
| `BlockMessage()` | `map[string]interface{}` | 包含 status、message 和 event_id 的结构化拦截响应 |

### Result 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `Head` | `byte` | 检测判定：`'.'` = 放行，`'?'` = 拦截，`'!'` = Bot 检测 |
| `Body` | `[]byte` | 引擎返回的状态码字符串 |
| `ExtraHeader` | `[]byte` | 引擎提供的额外响应头 |
| `ExtraBody` | `[]byte` | 额外响应体（HTML 注释中包含 event_id） |
| `T1KContext` | `[]byte` | 请求→响应阶段传递的不透明上下文 |
| `Cookie` | `[]byte` | 引擎返回的防篡改 Cookie 数据 |
| `BotQuery` | `[]byte` | Bot 挑战查询字符串 |
| `BotBody` | `[]byte` | Bot 挑战响应体 |
| `WebLog` | `[]byte` | 引擎返回的 Web 日志标志 |

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `T1K_HEARTBEAT_INTERVAL` | 心跳间隔秒数（仅 Server 连接池） | `20` |
| `DETECTOR_ADDR` | 示例中使用的引擎地址 | — |
| `T1K_ADDR` | 集成测试使用的引擎地址（仅测试用，非生产配置） | — |

## 测试

```bash
# 单元测试（不需要引擎）
go test ./...

# 集成测试（需要可达的 SafeLine 引擎）
go test -tags integration -v ./...

# 自定义引擎地址
T1K_ADDR=your-engine:8000 go test -tags integration -v ./...
```

集成测试覆盖：正常请求放行、SQL 注入 / XSS / 路径穿越 / 命令注入拦截、POST Body 检测、已知恶意 IP 拦截、速率限制、连接池并发、DetectionContext 用法。

## 架构

```
t1k-go/
├── server.go          # Server：固定大小连接池 + 心跳 goroutine
├── channel.go         # ChannelPool：可配置连接池，支持空闲超时
├── conn.go            # 连接封装，失败时自动重连
├── detect.go          # 检测编排：序列化 → 发送 → 解析
├── factory.go         # ConnectionFactory 接口 + TcpFactory
├── pool.go            # Pool 接口 + PoolConfig
├── heartbeat.go       # 心跳：零长度 FIRST|LAST section
├── health_check.go    # 健康检查服务，基于阈值的状态机
├── health_check_protocol.go  # T1K 和 HTTP 健康检查协议
├── detection/
│   ├── context.go     # DetectionContext：单次请求元数据 + T1KContext
│   ├── request.go     # Request 接口 + HttpRequest 实现
│   ├── response.go    # Response 接口 + HttpResponse 实现
│   ├── result.go      # Result：Passed/Blocked/EventID/BotDetected
│   └── extra.go       # Extra 元数据构建器（key:value 格式）
├── t1k/
│   ├── tag.go         # Tag 常量 + MASK_FIRST/MASK_LAST 标志位
│   ├── header.go      # 5 字节二进制头（tag + 小端序 uint32 长度）
│   └── section.go     # Section 读写（SimpleSection + ReaderSection）
└── misc/
    ├── gen_uuid.go    # UUID 生成（基于 MT19937）
    ├── errors.go      # 错误封装，附带调用者信息
    └── now.go         # 微秒级时间戳
```

### 检测流程

```
HTTP 请求
    │
    ▼
序列化为 T1K section：
  TAG_HEADER|MASK_FIRST → TAG_BODY → TAG_EXTRA → TAG_VERSION|MASK_LAST
    │
    ▼
通过 TCP 发送到 SafeLine 引擎
    │
    ▼
读取多 section TLV 响应
    │
    ▼
解析为 Result（Head, Body, ExtraBody, Context, Cookie, BotQuery, ...）
    │
    ▼
Passed() / Blocked() / EventID() / BotDetected()
```

## 许可证

[Apache License 2.0](License)
