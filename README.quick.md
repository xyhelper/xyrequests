# xyrequests 精简版

一个基于 tls-client 的 Go 请求库，支持 TLS 指纹、goSpiderSpec、自定义代理、WebSocket 和反向代理 Transport。

## 1 分钟上手

### 环境

- Go 1.24.1+

### 安装

```bash
go mod tidy
```

### 最小可用示例

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

    c, err := xyrequests.NewClient(ctx, xyrequests.ClientOption{
        TimeoutSeconds: 15,
    })
    if err != nil {
        panic(err)
    }
    defer c.Close()

    resp, err := c.Get(ctx, "https://httpbin.org/get")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    b, _ := io.ReadAll(resp.Body)
    fmt.Println(resp.StatusCode, string(b))
}
```

## 常用场景

### POST JSON

```go
resp, err := c.Post(ctx, "https://httpbin.org/post", xyrequests.RequestOption{
    Json: map[string]any{"k": "v"},
})
_ = resp
_ = err
```

### 单次请求临时代理

```go
resp, err := c.Get(ctx, "https://httpbin.org/ip", xyrequests.RequestOption{
    Proxy: "http://127.0.0.1:7890",
})
_ = resp
_ = err
```

### 使用 goSpiderSpec 指纹

```go
c, err := xyrequests.NewClient(ctx, xyrequests.ClientOption{
    Spec: yourGoSpiderSpec, // TLS_HEX@H1_HEX@H2_HEX
})
_ = c
_ = err
```

### 解析 goSpiderSpec

```go
spec, err := xyrequests.ParseGoSpiderSpec(yourGoSpiderSpec)
if err != nil {
    panic(err)
}
_ = spec.Map()
_ = spec.TLS.ServerName()
```

### WebSocket 连接

```go
ws, err := c.NewWebsocket(ctx, "wss://echo.websocket.events")
_ = ws
_ = err
```

### 反向代理 Transport

```go
proxy := &httputil.ReverseProxy{
    Director: func(req *http.Request) {
        req.URL.Scheme = "https"
        req.URL.Host = "target-api.example.com"
        req.Host = "target-api.example.com"
    },
    Transport: c.Transport(),
}
_ = proxy
```

## 最常用参数

- ClientOption.TimeoutSeconds：超时秒数
- ClientOption.Proxy：全局代理
- ClientOption.Spec：goSpiderSpec 指纹
- ClientOption.ForceHttp1：强制 HTTP/1.1（WebSocket 场景常用）
- RequestOption.Headers：单次请求头
- RequestOption.Json：自动 JSON 请求体
- RequestOption.Proxy：单次请求代理

## 开发测试

```bash
go test ./...
```

## 版本发布

```bash
# 正式发布
./release.sh v0.1.0

# 仅本地打标签，不推送
./release.sh v0.1.0 --no-push

# 指定分支发布
./release.sh v0.1.0 --branch master
```
