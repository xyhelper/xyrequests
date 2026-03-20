package xyrequests

import (
	"bytes"
	"context"
	"encoding/json"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/bogdanfinn/websocket"
	"github.com/gogf/gf/v2/frame/g"
)

type ClientOption struct {
	TimeoutSeconds     int         // 超时秒数
	Proxy              string      // 代理地址
	NotFollowRedirects bool        // 不跟随重定向
	CookieJar          *XyJar      // 自定义CookieJar
	ProxyUrl           string      // 代理URL
	Closed             bool        // 是否已关闭
	Debug              bool        // 是否启用调试模式
	DefaultHeaders     http.Header // 默认请求头
	ClientProfile      string      // 客户端配置文件名称
	Spec               string      // goSpiderSpec 指纹字符串，不为空时优先使用
	ForceHttp1         bool        // 强制使用HTTP/1.1（WebSocket场景需要设为true）
}

type Client struct {
	HttpClient   tls_client.HttpClient
	ClientOption ClientOption
}

type RequestOption struct {
	Headers map[string]string // 自定义请求头
	Body    []byte            // 请求体
	Json    any               // JSON数据
	Proxy   string            // 单次请求代理地址，为空时使用客户端默认代理
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
	if len(option) > 0 && option[0].Spec != "" {
		// 通过 goSpiderSpec 指纹创建自定义 profile
		p, sh, err := buildProfileFromSpec(option[0].Spec)
		if err != nil {
			g.Log().Error(ctx, "Failed to build profile from spec:", err)
			return nil, err
		}
		profile = p
		specHeaders = sh
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
		if clientOpt.TimeoutSeconds > 0 {
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

	return &Client{
		HttpClient:   httpClient,
		ClientOption: clientOpt,
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

// Do 发送HTTP请求
func (c *Client) Do(ctx g.Ctx, req *http.Request) (*http.Response, error) {
	resp, err := c.HttpClient.Do(req)
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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		g.Log().Error(ctx, "Failed to create GET request:", err)
		return nil, err
	}
	req.Header = c.defaultHeaders()
	if len(option) > 0 {
		if option[0].Headers != nil {
			for key, value := range option[0].Headers {
				req.Header.Set(key, value)
			}
		}
		if option[0].Proxy != "" {
			return c.DoWithProxy(ctx, req, option[0].Proxy)
		}
	}
	return c.Do(ctx, req)
}

// Post 发送POST请求
func (c *Client) Post(ctx g.Ctx, url string, option ...RequestOption) (*http.Response, error) {
	bodyReader := bytes.NewReader(nil)
	contentType := ""

	if len(option) > 0 {
		opt := option[0]
		if opt.Json != nil {
			jsonBytes, err := json.Marshal(opt.Json)
			if err != nil {
				g.Log().Error(ctx, "Failed to marshal JSON:", err)
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonBytes)
			contentType = "application/json"
		} else if opt.Body != nil {
			bodyReader = bytes.NewReader(opt.Body)
		}
	}

	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		g.Log().Error(ctx, "Failed to create POST request:", err)
		return nil, err
	}
	req.Header = c.defaultHeaders()
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	if len(option) > 0 {
		if option[0].Headers != nil {
			for key, value := range option[0].Headers {
				req.Header.Set(key, value)
			}
		}
		if option[0].Proxy != "" {
			return c.DoWithProxy(ctx, req, option[0].Proxy)
		}
	}
	return c.Do(ctx, req)
}

// DO 转发请求
func (c *Client) DO(ctx g.Ctx, method string, url string, option ...RequestOption) (*http.Response, error) {
	bodyReader := bytes.NewReader(nil)
	contentType := ""

	if len(option) > 0 {
		opt := option[0]
		if opt.Json != nil {
			jsonBytes, err := json.Marshal(opt.Json)
			if err != nil {
				g.Log().Error(ctx, "Failed to marshal JSON:", err)
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonBytes)
			contentType = "application/json"
		} else if opt.Body != nil {
			bodyReader = bytes.NewReader(opt.Body)
		}
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		g.Log().Error(ctx, "Failed to create request:", err)
		return nil, err
	}
	req.Header = c.defaultHeaders()
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	if len(option) > 0 {
		if option[0].Headers != nil {
			for key, value := range option[0].Headers {
				req.Header.Set(key, value)
			}
		}
		if option[0].Proxy != "" {
			return c.DoWithProxy(ctx, req, option[0].Proxy)
		}
	}
	return c.Do(ctx, req)
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
