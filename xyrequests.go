package xyrequests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/bogdanfinn/websocket"
	"github.com/gogf/gf/v2/frame/g"
)

type ClientOption struct {
	TimeoutSeconds               int         // 超时秒数（总超时，含 body 读取）。设置 ResponseHeaderTimeoutSeconds 时本字段被忽略。
	ResponseHeaderTimeoutSeconds int         // 响应头超时秒数：仅限制「发出请求→收到响应头」的时间，body 读取不受限（适合流式/长响应）。设置后 TimeoutSeconds 将被忽略。
	Proxy                        string      // 代理地址
	NotFollowRedirects           bool        // 不跟随重定向
	CookieJar                    *XyJar      // 自定义CookieJar
	ProxyUrl                     string      // 代理URL
	Closed                       bool        // 是否已关闭
	Debug                        bool        // 是否启用调试模式
	DefaultHeaders               http.Header // 默认请求头
	ClientProfile                string      // 客户端配置文件名称
	Spec                         string      // goSpiderSpec 指纹字符串，不为空时优先使用
	ForceHttp1                   bool        // 强制使用HTTP/1.1（WebSocket场景需要设为true）
	StrictFingerprint            bool        // 反向代理严格模式：为true时完全使用gospec的header，覆盖客户端的；为false时客户端header优先（默认）
	PassthroughHeaders           []string    // StrictFingerprint模式下允许透传的header列表。比如Authorization、X-Custom-Token、Cookie等。Cookie需要显式声明，若不声明则严格模式下会被删除（如启用ManageCookies则由jar管理）
}

type Client struct {
	HttpClient               tls_client.HttpClient
	ClientOption             ClientOption
	defaultHeaderOrder       []string
	defaultPseudoHeaderOrder []string
	strictFingerprint        bool // 严格指纹模式标志
	specHeaders              http.Header
	passthroughHeadersLower  map[string]struct{} // 透传header白名单（小写），用于case-insensitive判定
}

type RequestOption struct {
	Headers  map[string]string // 自定义请求头
	Body     []byte            // 请求体
	Json     any               // JSON数据
	Proxy    string            // 单次请求代理地址，为空时使用客户端默认代理
	MaxRetry int               // 最大重试次数，默认为0不重试
}

var (
	DefaultHeaders = http.Header{
		"User-Agent":      []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36"},
		"Accept-Language": []string{"en-US,en;q=0.9"},
	}
	ProfileMap = map[string]profiles.ClientProfile{
		"Okhttp4Android13":  profiles.Okhttp4Android13,
		"ConfirmedAndroid2": profiles.ConfirmedAndroid2,
		"Chrome_131":        profiles.Chrome_131,
	}
)

// NewClient 创建一个新的HTTP客户端
func NewClient(ctx g.Ctx, option ...ClientOption) (*Client, error) {
	// 创建TLS客户端配置
	var profile profiles.ClientProfile

	var specHeaders http.Header
	var specHeaderOrder []string
	var specPseudoHeaderOrder []string
	if len(option) > 0 && option[0].Spec != "" {
		// 通过 goSpiderSpec 指纹创建自定义 profile
		p, sh, sho, psho, err := buildProfileFromSpec(option[0].Spec)
		if err != nil {
			g.Log().Error(ctx, "Failed to build profile from spec:", err)
			return nil, err
		}
		profile = p
		specHeaders = sh
		specHeaderOrder = sho
		specPseudoHeaderOrder = psho
	} else {
		profile = profiles.Okhttp4Android12
		if len(option) > 0 && option[0].ClientProfile != "" {
			if p, ok := ProfileMap[option[0].ClientProfile]; ok {
				profile = p
			}
		}
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profile),
		tls_client.WithRandomTLSExtensionOrder(),
	}
	if len(option) > 0 && option[0].ForceHttp1 {
		options = append(options, tls_client.WithForceHttp1())
	}
	clientOpt := ClientOption{}
	if len(option) > 0 {
		clientOpt = option[0]
		if clientOpt.ResponseHeaderTimeoutSeconds > 0 {
			// 禁用总超时，由 doOnce 在响应头阶段单独控制超时
			options = append(options, tls_client.WithTimeoutSeconds(0))
		} else if clientOpt.TimeoutSeconds > 0 {
			options = append(options, tls_client.WithTimeoutSeconds(clientOpt.TimeoutSeconds))
		}
		if clientOpt.NotFollowRedirects {
			options = append(options, tls_client.WithNotFollowRedirects())
		}
		if clientOpt.CookieJar != nil {
			options = append(options, tls_client.WithCookieJar(clientOpt.CookieJar))
		}
		if clientOpt.Proxy != "" {
			options = append(options, tls_client.WithProxyUrl(clientOpt.Proxy))
		}
		if clientOpt.Debug {
			options = append(options, tls_client.WithDebug())
		}
	}

	// 设置默认请求头：优先用户自定义 > spec 中提取 > 全局默认
	if len(clientOpt.DefaultHeaders) > 0 {
		options = append(options, tls_client.WithDefaultHeaders(clientOpt.DefaultHeaders))
	} else if len(specHeaders) > 0 {
		options = append(options, tls_client.WithDefaultHeaders(specHeaders))
		clientOpt.DefaultHeaders = specHeaders
	} else {
		options = append(options, tls_client.WithDefaultHeaders(DefaultHeaders))
	}

	httpClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		g.Log().Error(ctx, "Failed to create HTTP client:", err)
		return nil, err
	}

	// 构建透传header白名单（小写）
	passthroughHeadersLower := make(map[string]struct{}, len(clientOpt.PassthroughHeaders))
	for _, h := range clientOpt.PassthroughHeaders {
		passthroughHeadersLower[strings.ToLower(h)] = struct{}{}
	}

	return &Client{
		HttpClient:               httpClient,
		ClientOption:             clientOpt,
		defaultHeaderOrder:       append([]string(nil), specHeaderOrder...),
		defaultPseudoHeaderOrder: append([]string(nil), specPseudoHeaderOrder...),
		strictFingerprint:        clientOpt.StrictFingerprint,
		specHeaders:              specHeaders,
		passthroughHeadersLower:  passthroughHeadersLower,
	}, nil
}

// Close 关闭HTTP客户端
func (c *Client) Close() {
	c.HttpClient.CloseIdleConnections()
	c.ClientOption.Closed = true

}

// defaultHeaders returns a non-nil copy of the default headers.
func (c *Client) defaultHeaders() http.Header {
	src := DefaultHeaders
	if len(c.ClientOption.DefaultHeaders) > 0 {
		src = c.ClientOption.DefaultHeaders
	}
	dst := make(http.Header, len(src))
	for k, v := range src {
		vv := make([]string, len(v))
		copy(vv, v)
		dst[k] = vv
	}
	return dst
}

func headerExistsCaseInsensitive(header http.Header, name string) bool {
	for k := range header {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}

// applyStrictFingerprintHeaders 在严格指纹模式下应用 spec 的 header
// 会删除客户端不在透传白名单中的 header，然后使用 spec 的 header 替换
func (c *Client) applyStrictFingerprintHeaders(header http.Header) {
	if header == nil || !c.strictFingerprint || len(c.specHeaders) == 0 {
		return
	}

	// 保存需要透传的客户端 header（白名单中的）
	passthroughHeaders := make(http.Header)
	for k, v := range header {
		lowerK := strings.ToLower(k)
		if _, ok := c.passthroughHeadersLower[lowerK]; ok {
			passthroughHeaders[k] = append([]string(nil), v...)
		}
	}

	// 清空所有 header
	for k := range header {
		header.Del(k)
	}

	// 使用 spec 的 header
	for key, values := range c.specHeaders {
		header[key] = append([]string(nil), values...)
	}

	// 将白名单中的客户端 header 合并回来（这些 header 会覆盖 spec 的同名项）
	for key, values := range passthroughHeaders {
		header[key] = values
	}
}

func (c *Client) applyHeaderOrdering(header http.Header) {
	if header == nil {
		return
	}

	if len(c.defaultHeaderOrder) > 0 {
		seen := make(map[string]struct{}, len(header))
		order := make([]string, 0, len(header))
		for _, name := range c.defaultHeaderOrder {
			lower := strings.ToLower(name)
			if _, exists := seen[lower]; exists {
				continue
			}
			if !headerExistsCaseInsensitive(header, lower) {
				continue
			}
			seen[lower] = struct{}{}
			order = append(order, lower)
		}

		extra := make([]string, 0, len(header))
		for k := range header {
			if k == http.HeaderOrderKey || k == http.PHeaderOrderKey {
				continue
			}
			if strings.HasPrefix(k, ":") {
				continue
			}
			lower := strings.ToLower(k)
			if _, exists := seen[lower]; exists {
				continue
			}
			seen[lower] = struct{}{}
			extra = append(extra, lower)
		}
		sort.Strings(extra)
		order = append(order, extra...)
		if len(order) > 0 {
			header[http.HeaderOrderKey] = order
		}
	}

	if len(c.defaultPseudoHeaderOrder) > 0 {
		pseudo := make([]string, 0, len(c.defaultPseudoHeaderOrder))
		seenPseudo := make(map[string]struct{}, len(c.defaultPseudoHeaderOrder))
		for _, name := range c.defaultPseudoHeaderOrder {
			if !strings.HasPrefix(name, ":") {
				continue
			}
			if _, exists := seenPseudo[name]; exists {
				continue
			}
			seenPseudo[name] = struct{}{}
			pseudo = append(pseudo, name)
		}
		if len(pseudo) > 0 {
			header[http.PHeaderOrderKey] = pseudo
		}
	}
}

// doWithRetry 根据 maxRetry 执行请求重试。
// maxRetry=0 表示仅尝试一次；maxRetry=n 表示最多尝试 n+1 次。
func (c *Client) doWithRetry(ctx g.Ctx, maxRetry int, doOnce func() (*http.Response, error)) (*http.Response, error) {
	attempts := maxRetry + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		resp, err := doOnce()
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		g.Log().Error(ctx, "HTTP request failed after retries:", lastErr)
	}
	return nil, lastErr
}

// bodyWithCancel 包装 ReadCloser，在 Close 时调用 cancel 释放请求 context。
// 用于 ResponseHeaderTimeoutSeconds 模式：收到响应头后 context 保持存活直到 body 关闭。
type bodyWithCancel struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *bodyWithCancel) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

// doOnce 执行一次实际 HTTP 请求。
// 若设置了 ResponseHeaderTimeoutSeconds，仅对「收到响应头」限时，body 读取不受约束。
func (c *Client) doOnce(req *http.Request) (*http.Response, error) {
	headerTimeout := c.ClientOption.ResponseHeaderTimeoutSeconds
	if headerTimeout <= 0 {
		return c.HttpClient.Do(req)
	}

	ctx, cancel := context.WithCancel(req.Context())

	type result struct {
		resp *http.Response
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		resp, err := c.HttpClient.Do(req.WithContext(ctx))
		ch <- result{resp, err}
	}()

	timer := time.NewTimer(time.Duration(headerTimeout) * time.Second)
	defer timer.Stop()

	select {
	case r := <-ch:
		if r.err != nil {
			cancel()
			return nil, r.err
		}
		// 收到响应头，将 cancel 推迟到 body 关闭时再调用
		r.resp.Body = &bodyWithCancel{ReadCloser: r.resp.Body, cancel: cancel}
		return r.resp, nil
	case <-timer.C:
		cancel()
		return nil, fmt.Errorf("xyrequests: response header timeout after %ds", headerTimeout)
	case <-req.Context().Done():
		cancel()
		return nil, req.Context().Err()
	}
}

// Do 发送HTTP请求
func (c *Client) Do(ctx g.Ctx, req *http.Request) (*http.Response, error) {
	c.applyStrictFingerprintHeaders(req.Header)
	c.applyHeaderOrdering(req.Header)
	resp, err := c.doOnce(req)
	if err != nil {
		g.Log().Error(ctx, "HTTP request failed:", err)
		return nil, err
	}
	return resp, nil
}

// DoWithProxy 发送HTTP请求，临时使用指定代理，请求完成后恢复原代理
func (c *Client) DoWithProxy(ctx g.Ctx, req *http.Request, proxy string) (*http.Response, error) {
	originalProxy := c.HttpClient.GetProxy()
	if err := c.HttpClient.SetProxy(proxy); err != nil {
		g.Log().Error(ctx, "Failed to set request proxy:", err)
		return nil, err
	}
	defer func() {
		if err := c.HttpClient.SetProxy(originalProxy); err != nil {
			g.Log().Error(ctx, "Failed to restore original proxy:", err)
		}
	}()
	return c.Do(ctx, req)
}

// Get 发送GET请求
func (c *Client) Get(ctx g.Ctx, url string, option ...RequestOption) (*http.Response, error) {
	opt := RequestOption{}
	if len(option) > 0 {
		opt = option[0]
	}

	return c.doWithRetry(ctx, opt.MaxRetry, func() (*http.Response, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			g.Log().Error(ctx, "Failed to create GET request:", err)
			return nil, err
		}
		req.Header = c.defaultHeaders()
		for key, value := range opt.Headers {
			req.Header.Set(key, value)
		}
		if opt.Proxy != "" {
			return c.DoWithProxy(ctx, req, opt.Proxy)
		}
		return c.Do(ctx, req)
	})
}

// Post 发送POST请求
func (c *Client) Post(ctx g.Ctx, url string, option ...RequestOption) (*http.Response, error) {
	bodyBytes := []byte(nil)
	contentType := ""
	opt := RequestOption{}

	if len(option) > 0 {
		opt = option[0]
		if opt.Json != nil {
			jsonBytes, err := json.Marshal(opt.Json)
			if err != nil {
				g.Log().Error(ctx, "Failed to marshal JSON:", err)
				return nil, err
			}
			bodyBytes = jsonBytes
			contentType = "application/json"
		} else if opt.Body != nil {
			bodyBytes = opt.Body
		}
	}

	return c.doWithRetry(ctx, opt.MaxRetry, func() (*http.Response, error) {
		req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			g.Log().Error(ctx, "Failed to create POST request:", err)
			return nil, err
		}
		req.Header = c.defaultHeaders()
		if contentType != "" && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", contentType)
		}
		for key, value := range opt.Headers {
			req.Header.Set(key, value)
		}
		if opt.Proxy != "" {
			return c.DoWithProxy(ctx, req, opt.Proxy)
		}
		return c.Do(ctx, req)
	})
}

// DO 转发请求
func (c *Client) DO(ctx g.Ctx, method string, url string, option ...RequestOption) (*http.Response, error) {
	bodyBytes := []byte(nil)
	contentType := ""
	opt := RequestOption{}

	if len(option) > 0 {
		opt = option[0]
		if opt.Json != nil {
			jsonBytes, err := json.Marshal(opt.Json)
			if err != nil {
				g.Log().Error(ctx, "Failed to marshal JSON:", err)
				return nil, err
			}
			bodyBytes = jsonBytes
			contentType = "application/json"
		} else if opt.Body != nil {
			bodyBytes = opt.Body
		}
	}

	return c.doWithRetry(ctx, opt.MaxRetry, func() (*http.Response, error) {
		req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			g.Log().Error(ctx, "Failed to create request:", err)
			return nil, err
		}
		req.Header = c.defaultHeaders()
		if contentType != "" && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", contentType)
		}
		for key, value := range opt.Headers {
			req.Header.Set(key, value)
		}
		if opt.Proxy != "" {
			return c.DoWithProxy(ctx, req, opt.Proxy)
		}
		return c.Do(ctx, req)
	})
}

// WebsocketOption WebSocket连接选项
type WebsocketOption struct {
	Headers            map[string]string // 自定义请求头
	HandshakeTimeoutMs int               // 握手超时(毫秒)
	ReadBufferSize     int               // 读缓冲区大小
	WriteBufferSize    int               // 写缓冲区大小
}

// NewWebsocket 使用当前客户端的TLS指纹配置创建WebSocket连接
func (c *Client) NewWebsocket(ctx g.Ctx, url string, option ...WebsocketOption) (*websocket.Conn, error) {
	wsOptions := []tls_client.WebsocketOption{
		tls_client.WithTlsClient(c.HttpClient),
		tls_client.WithUrl(url),
	}

	// 构建请求头：默认头 + 自定义头
	headers := c.defaultHeaders()
	if len(option) > 0 {
		opt := option[0]
		for key, value := range opt.Headers {
			headers.Set(key, value)
		}
		if opt.HandshakeTimeoutMs > 0 {
			wsOptions = append(wsOptions, tls_client.WithHandshakeTimeoutMilliseconds(opt.HandshakeTimeoutMs))
		}
		if opt.ReadBufferSize > 0 {
			wsOptions = append(wsOptions, tls_client.WithReadBufferSize(opt.ReadBufferSize))
		}
		if opt.WriteBufferSize > 0 {
			wsOptions = append(wsOptions, tls_client.WithWriteBufferSize(opt.WriteBufferSize))
		}
	}
	c.applyHeaderOrdering(headers)
	wsOptions = append(wsOptions, tls_client.WithHeaders(headers))

	// 如果客户端配置了CookieJar，传递给WebSocket
	if c.ClientOption.CookieJar != nil {
		wsOptions = append(wsOptions, tls_client.WithCookiejar(c.ClientOption.CookieJar))
	}

	ws, err := tls_client.NewWebsocket(tls_client.NewNoopLogger(), wsOptions...)
	if err != nil {
		g.Log().Error(ctx, "Failed to create websocket:", err)
		return nil, err
	}

	conn, err := ws.Connect(context.Background())
	if err != nil {
		g.Log().Error(ctx, "Failed to connect websocket:", err)
		return nil, err
	}

	return conn, nil
}
