# xyrequests

精简版请见 [README.quick.md](README.quick.md)。

`xyrequests` 是一个基于 `bogdanfinn/tls-client` 的 Go 请求库，目标是更方便地使用 TLS 指纹、HTTP/2 参数与 WebSocket 指纹连接能力。

项目支持通过 `goSpiderSpec` 字符串构建自定义客户端指纹，也支持常规 HTTP 请求、反向代理传输层、WebSocket 连接与 WebSocket 反向代理。

## 功能特性

- 支持 TLS 指纹请求（基于 `tls-client`）
- 支持通过 `goSpiderSpec` 构建自定义 `ClientProfile`
- 支持解析 `goSpiderSpec`（TLS / H1 / H2）并输出结构化结果
- 支持默认请求头、代理、超时、重定向控制、CookieJar
- 支持单次请求临时代理
- 支持 WebSocket 指纹连接
- 支持标准库 `ReverseProxy` 的 `Transport` 对接
- 支持 WebSocket 反向代理（客户端 <-> 上游双向转发）

## 环境要求

- Go `1.24.1+`

当前项目 `go.mod` 已对依赖版本做过兼容下调，适配 Go 1.24.x。

## 项目结构

- `xyrequests.go`：客户端主体、HTTP 请求方法、WebSocket 建连
- `gospider_client.go`：`goSpiderSpec` -> `ClientProfile` 构建逻辑
- `gospider_spec.go`：`goSpiderSpec` 解析与结构化输出
- `transport.go`：`net/http.RoundTripper` 适配层（用于反向代理）
- `ws_proxy.go`：WebSocket 反向代理
- `xyjar.go`：CookieJar 扩展封装

## 快速开始

### 1. 初始化客户端

```go
package main

import (
    "context"
    "fmt"
    "io"

    "xyrequests"
)

func main() {
    ctx := context.Background()

    client, err := xyrequests.NewClient(ctx, xyrequests.ClientOption{
        TimeoutSeconds: 15,
        Debug:          false,
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    resp, err := client.Get(ctx, "https://httpbin.org/get")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    b, _ := io.ReadAll(resp.Body)
    fmt.Println(resp.StatusCode, string(b))
}
```

### 2. POST JSON

```go
resp, err := client.Post(ctx, "https://httpbin.org/post", xyrequests.RequestOption{
    Json: map[string]any{
        "name": "xyrequests",
        "lang": "go",
    },
    Headers: map[string]string{
        "X-Test": "1",
    },
})
```

### 3. 单次请求走临时代理

```go
resp, err := client.Get(ctx, "https://httpbin.org/ip", xyrequests.RequestOption{
    Proxy: "http://127.0.0.1:7890",
})
```

## 使用 goSpiderSpec

`goSpiderSpec` 格式为：

```text
TLS_HEX@H1_HEX@H2_HEX
```

在 `ClientOption.Spec` 不为空时，库会优先用该指纹构建 `ClientProfile`，并尝试从 H2 头顺序中提取默认请求头。

```go
client, err := xyrequests.NewClient(ctx, xyrequests.ClientOption{
    Spec: yourGoSpiderSpec,
})
```

你也可以单独解析指纹：

```go
spec, err := xyrequests.ParseGoSpiderSpec(yourGoSpiderSpec)
if err != nil {
    panic(err)
}

// 结构化输出
m := spec.Map()
_ = m

// 一些便捷方法
_ = spec.TLS.ServerName()
_ = spec.TLS.Protocols()
_ = spec.TLS.ExtensionTypes()
```

## CookieJar 用法

```go
jar := xyrequests.NewJar()
_ = jar.SetCookiesByMap("https://example.com", map[string]string{
    "token": "abc",
    "uid":   "1001",
})

client, err := xyrequests.NewClient(ctx, xyrequests.ClientOption{
    CookieJar: jar,
})
```

清空 Cookie：

```go
jar.Clear()
```

## 反向代理 Transport

可直接把指纹传输层挂到标准库反代：

```go
proxy := &httputil.ReverseProxy{
    Director: func(req *http.Request) {
        req.URL.Scheme = "https"
        req.URL.Host = "target-api.example.com"
        req.Host = "target-api.example.com"
    },
    Transport: client.Transport(),
}

_ = http.ListenAndServe(":8080", proxy)
```

如果希望由 `xyrequests` 的 CookieJar 统一管理，可启用：

```go
Transport: client.Transport(true)
```

启用后会：
- 忽略客户端传入的 `Cookie`
- 不向客户端透传上游的 `Set-Cookie`

### Header 处理模式

反向代理时，有两种 header 处理模式：

**1. 混合模式（默认，`StrictFingerprint=false`）**
- 客户端 header 优先，gospec 的 header 只在缺失时补充
- 适合通用反代场景

**2. 严格模式（`StrictFingerprint=true`）**
- 完全使用 gospec 的 header，**覆盖**客户端的同名项
- 适合需要保持完整指纹一致性的场景（避免客户端修改 header 破坏指纹特征）

该模式下支持**白名单转义**，某些 header（如 `Authorization`）可以设置为始终透传：

```go
// 启用严格指纹模式，并指定透传白名单
client, _ := xyrequests.NewClient(ctx, xyrequests.ClientOption{
    Spec:              "...",
    StrictFingerprint: true,  // 启用严格指纹
    PassthroughHeaders: []string{"Authorization", "X-Custom-Token"},  // 这些 header 可以透传
})

proxy := &httputil.ReverseProxy{
    Director: func(req *http.Request) {
        req.URL.Scheme = "https"
        req.URL.Host = "target-api.example.com"
        req.Host = "target-api.example.com"
    },
    Transport: client.Transport(),
}

http.ListenAndServe(":8080", proxy)
```

**严格模式的行为：**
- 清除客户端的所有 header 和特殊字段
- 使用 gospec 解析的 header 替换
- header 顺序按 gospec 指定的顺序
- 作用范围：Direct HTTP 请求（`Do` 方法） + Transport + WebSocket 三个路径

**白名单转义的行为：**
- `PassthroughHeaders` 中列举的 header 在严格模式下保留，其值来自客户端（会覆盖 spec 中的同名项）
- `Cookie` 也需要显式在 `PassthroughHeaders` 中声明才会透传；否则被删除
- 使用场景：保护 Authorization、业务认证类、跟踪 ID、Cookie 等动态信息

**与 ManageCookies 的配合：**
- 如果启用 `Transport(true)` 的 `ManageCookies`，客户端的 Cookie 会被删除，Cookie 由 jar 统一管理
- 如果想同时使用 jar 的 Cookie 和客户端的 Cookie，在 `PassthroughHeaders` 中声明 Cookie，但须确保上游正确处理多个 Cookie

示例场景：
```go
// 场景1：使用 jar 管理所有 Cookie（通过 ManageCookies）
proxy := &httputil.ReverseProxy{
    Transport: client.Transport(true),  // ManageCookies = true，完全由 jar 管理
}

// 场景2：保留客户端的 Cookie，但使用 gospec header
proxy := &httputil.ReverseProxy{
    Transport: client.Transport(),
    // 同时在 ClientOption 中
    PassthroughHeaders: []string{"Cookie"},
}

// 场景3：混合使用（既用 jar 又透传客户端 Cookie）
// 注意：这样上游会收到两组 Cookie，需要特殊处理
proxy := &httputil.ReverseProxy{
    Transport: client.Transport(true),
    // ... 但这时客户端的 Cookie 已被删除，所以只有 jar 的 Cookie
}
```

示例 - 客户端请求到达反代时（不启用 ManageCookies）：
```
客户端请求 Header:
  User-Agent: Mozilla/...（自定义）      ❌ 被删除，使用 spec 的值
  Authorization: Bearer xxx              ✅ 保留（在 PassthroughHeaders 中）
  X-Request-ID: 12345                    ❌ 被删除，不在白名单中
  Cookie: ...                            ❌ 被删除（未在 PassthroughHeaders 中声明）

最终转发到上游的 Header:
  User-Agent: Mozilla/...（spec的值）
  Accept-Language: en-US,en;q=0.9（spec的值）
  ...（其他 spec header）
  Authorization: Bearer xxx              ✅ 来自客户端
  Cookie: ...                            ✅ 来自客户端，或由 ManageCookies 管理
```

## WebSocket

### 1. 直接建立 WebSocket 连接

```go
ws, err := client.NewWebsocket(ctx, "wss://echo.websocket.events", xyrequests.WebsocketOption{
    Headers: map[string]string{
        "Origin": "https://example.com",
    },
    HandshakeTimeoutMs: 8000,
    ReadBufferSize:     4096,
    WriteBufferSize:    4096,
})
if err != nil {
    panic(err)
}
defer ws.Close()
```

### 2. WebSocket 反向代理

```go
http.Handle("/ws", client.WebsocketProxy(ctx, "wss://upstream.example.com/ws"))
_ = http.ListenAndServe(":8080", nil)
```

## ClientOption 说明

- `TimeoutSeconds`：请求超时秒数
- `Proxy`：客户端级别代理地址
- `NotFollowRedirects`：是否禁止跟随重定向
- `CookieJar`：自定义 CookieJar
- `Debug`：是否启用 `tls-client` 调试
- `DefaultHeaders`：自定义默认请求头（优先级最高）
- `ClientProfile`：预置 profile 名称（当前内置 `Okhttp4Android13`、`ConfirmedAndroid2`、`Chrome_131`）
- `Spec`：`goSpiderSpec` 指纹字符串（设置后优先于 `ClientProfile`）
- `ForceHttp1`：强制使用 HTTP/1.1（WebSocket 场景建议开启）

## RequestOption 说明

- `Headers`：单次请求头
- `Body`：请求体（字节）
- `Json`：自动 JSON 序列化请求体
- `Proxy`：单次请求代理（优先于客户端默认代理）

## 开发与测试

```bash
go mod tidy
go test ./...
```

## 版本发布

项目内置发布脚本 [release.sh](release.sh)，用于统一执行版本校验、测试、打标签与推送。

```bash
# 首次使用需要赋予执行权限
chmod +x release.sh

# 正式发布（会执行测试并推送分支与标签）
./release.sh v0.1.0

# 仅本地打标签，不推送
./release.sh v0.1.0 --no-push

# 指定发布分支
./release.sh v0.1.0 --branch master

# 跳过测试（不建议常态使用）
./release.sh v0.1.0 --skip-tests
```

## 注意事项

- `Spec` 必须是合法的 `TLS_HEX@H1_HEX@H2_HEX` 三段格式。
- 若同时设置 `DefaultHeaders` 与 `Spec`，优先使用 `DefaultHeaders`。
- `WebsocketProxy` 内置 `CheckOrigin: true`，生产环境请结合业务需求自行加固。

## License

当前仓库未包含许可证文件，建议补充 `LICENSE` 后再对外发布。
