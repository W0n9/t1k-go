# CLAUDE.md

本文件用于指导 Claude Code（claude.ai/code）在本仓库中工作时的行为。

## 这个项目是什么

这是一个用于 [Chaitin SafeLine](https://github.com/chaitin/safeline) WAF 的 T1K 协议 Go SDK。它会把 HTTP 请求/响应序列化为 T1K 二进制格式，通过 TCP 发送到 SafeLine 检测引擎，并解析引擎返回的判定结果（放行/拦截、事件 ID、Bot 检测等）。

模块路径是 `github.com/chaitin/t1k-go`。这个仓库是维护在 `github.com/w0n9/t1k-go` 的一个 fork。

## 构建与测试

```bash
go build -v ./examples/main.go   # 编译检查（不会生成独立可执行文件）
go test -v ./...                  # 单元测试（不需要真实在线引擎）
go test -v -run TestCaclErrorCount ./...  # 单个测试
```

CI 使用 Go 1.26 进行运行。

## 架构

这里并存着两种连接池实现，请根据你的集成场景选择合适的一种：

| 类型 | 文件 | 模型 |
|------|------|------|
| `Server` | `server.go` + `conn.go` | 固定大小的带缓冲 channel 连接池，带后台心跳 goroutine。`GetConn` 时懒加载补齐，失败后通过 `conn.onErr` 重连。 |
| `ChannelPool` | `channel.go` + `factory.go` + `pool.go` | 使用 `ConnectionFactory` 接口的可配置连接池（`InitialCap`/`MaxIdle`/`MaxCap`/`IdleTimeout`）。达到最大容量时，等待者会在 `connReqs` 中排队。 |

两者都以 `DetectHttpRequest(req)` → `*detection.Result` 作为主要 API。

### T1K 线协议（`t1k/`）

二进制封装格式：5 字节头部（1 字节 tag + 4 字节小端序 `uint32` 长度）后跟消息体。消息是由多个 section 组成的序列，tag 字节上通过 `MASK_FIRST` / `MASK_LAST` 标志位标记首尾。tag 的定义在 `t1k/tag.go` 中。

### 检测流程（`detect.go`）

请求检测会按顺序写入四个 section：`TAG_HEADER|MASK_FIRST` → `TAG_BODY` → `TAG_EXTRA` → `TAG_VERSION|MASK_LAST`，然后通过 `readDetectionResult` 读取引擎返回的多 section 响应。响应检测采用类似流程，使用 `TAG_RSP_*` tag 以及 `TAG_CONTEXT|MASK_LAST`。

### 检测上下文（`detection/`）

- `DetectionContext` 保存单次请求的元数据（UUID、scheme、地址、时间戳），并在请求→响应阶段持续累积 `T1KContext`。
- `Result` 字段含义：`Head` 字节（`.` 表示放行，其他值表示拦截）、`Body`（状态码）、`ExtraBody`（在 HTML 注释中包含事件 ID）、`BotQuery`/`BotBody`（Bot 检测）。`BotDetected()` 只取决于 `BotQuery` 是否非空。
- `Extra` 元数据（`detection/extra.go`）是按行分隔的 `key:value` 格式，不是 JSON。

### 健康检查（`health_check.go` + `health_check_protocol.go`）

它会作为 `Server` 上的后台 goroutine 运行。支持 T1K heartbeat 或 HTTP `/stat` 端点协议。内部使用基于阈值的状态机（`CaclErrorCount`）：当 `ErrorCount > UnhealthThreshold` 时标记为不健康，恢复时则需要连续 `HealthThreshold` 次成功。

### 心跳（`heartbeat.go`）

会发送一个长度为 0 的 section，并设置 `MASK_FIRST|MASK_LAST` 标志，然后读取检测结果。它既用于保活（`Server.runHeartbeatCo` 每 20 秒执行一次，可通过环境变量 `T1K_HEARTBEAT_INTERVAL` 配置），也用于 `TcpFactory.Ping` 的健康探测。
