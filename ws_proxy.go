package xyrequests

import (
	"io"
	stdhttp "net/http"
	"sync"

	bfwebsocket "github.com/bogdanfinn/websocket"
	"github.com/gogf/gf/v2/frame/g"
	gorillawebsocket "github.com/gorilla/websocket"
)

// WebsocketProxyOption WebSocket 反代选项
type WebsocketProxyOption struct {
	Headers            map[string]string // 连接上游时的自定义请求头
	HandshakeTimeoutMs int               // 上游握手超时(毫秒)
	ReadBufferSize     int               // 读缓冲区大小
	WriteBufferSize    int               // 写缓冲区大小
}

// wsUpgrader 服务端 WebSocket 升级器（接受客户端连接）
var wsUpgrader = gorillawebsocket.Upgrader{
	CheckOrigin: func(r *stdhttp.Request) bool {
		return true // 反代场景不限制 Origin
	},
}

// WebsocketProxy 返回一个 net/http.Handler，用于 WebSocket 反向代理。
// 客户端的 WebSocket 连接将通过 tls-client 的 TLS 指纹连接到上游 targetURL，
// 双向转发所有消息。
//
// 用法示例:
//
//	client, _ := xyrequests.NewClient(ctx, xyrequests.ClientOption{
//	    Spec: "...",
//	})
//	http.Handle("/ws", client.WebsocketProxy(ctx, "wss://upstream.example.com/ws"))
func (c *Client) WebsocketProxy(ctx g.Ctx, targetURL string, option ...WebsocketProxyOption) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		// 1. 接受客户端的 WebSocket 连接
		clientConn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			g.Log().Error(ctx, "WebSocket upgrade failed:", err)
			return
		}
		defer clientConn.Close()

		// 2. 构建上游 WebSocket 连接选项
		wsOpt := WebsocketOption{}
		if len(option) > 0 {
			opt := option[0]
			wsOpt.Headers = opt.Headers
			wsOpt.HandshakeTimeoutMs = opt.HandshakeTimeoutMs
			wsOpt.ReadBufferSize = opt.ReadBufferSize
			wsOpt.WriteBufferSize = opt.WriteBufferSize
		}

		// 3. 用 tls-client 指纹连接上游
		upstreamConn, err := c.NewWebsocket(ctx, targetURL, wsOpt)
		if err != nil {
			g.Log().Error(ctx, "WebSocket upstream connect failed:", err)
			closeMsg := gorillawebsocket.FormatCloseMessage(
				gorillawebsocket.CloseInternalServerErr, "upstream connect failed")
			clientConn.WriteMessage(gorillawebsocket.CloseMessage, closeMsg)
			return
		}
		defer upstreamConn.Close()

		// 4. 双向转发
		var wg sync.WaitGroup
		wg.Add(2)

		// 客户端 → 上游
		go func() {
			defer wg.Done()
			proxyClientToUpstream(clientConn, upstreamConn)
		}()

		// 上游 → 客户端
		go func() {
			defer wg.Done()
			proxyUpstreamToClient(upstreamConn, clientConn)
		}()

		wg.Wait()
	})
}

// proxyClientToUpstream 从客户端读取消息转发到上游
func proxyClientToUpstream(client *gorillawebsocket.Conn, upstream *bfwebsocket.Conn) {
	for {
		msgType, msg, err := client.ReadMessage()
		if err != nil {
			// 客户端断开，向上游发送关闭帧
			if gorillawebsocket.IsCloseError(err,
				gorillawebsocket.CloseNormalClosure,
				gorillawebsocket.CloseGoingAway,
				gorillawebsocket.CloseNoStatusReceived) {
				upstream.WriteMessage(bfwebsocket.CloseMessage,
					bfwebsocket.FormatCloseMessage(bfwebsocket.CloseNormalClosure, ""))
			}
			return
		}
		if err := upstream.WriteMessage(msgType, msg); err != nil {
			return
		}
	}
}

// proxyUpstreamToClient 从上游读取消息转发到客户端
func proxyUpstreamToClient(upstream *bfwebsocket.Conn, client *gorillawebsocket.Conn) {
	for {
		msgType, reader, err := upstream.NextReader()
		if err != nil {
			// 上游断开，向客户端发送关闭帧
			if bfwebsocket.IsCloseError(err,
				bfwebsocket.CloseNormalClosure,
				bfwebsocket.CloseGoingAway,
				bfwebsocket.CloseNoStatusReceived) {
				client.WriteMessage(gorillawebsocket.CloseMessage,
					gorillawebsocket.FormatCloseMessage(gorillawebsocket.CloseNormalClosure, ""))
			}
			return
		}
		writer, err := client.NextWriter(msgType)
		if err != nil {
			return
		}
		if _, err := io.Copy(writer, reader); err != nil {
			writer.Close()
			return
		}
		writer.Close()
	}
}
