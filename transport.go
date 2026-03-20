package xyrequests

import (
	"io"
	stdhttp "net/http"

	fhttp "github.com/bogdanfinn/fhttp"
)

// FingerprintTransport 实现 net/http.RoundTripper 接口，
// 可直接用于 httputil.ReverseProxy 的 Transport 字段，
// 所有请求将通过 tls-client 发送，保留 TLS 指纹伪装。
//
// 用法示例:
//
//	client, _ := xyrequests.NewClient(ctx, xyrequests.ClientOption{
//	    Spec: "...",
//	})
//	proxy := &httputil.ReverseProxy{
//	    Director:  director,
//	    Transport: client.Transport(),
//	}
// client, _ := xyrequests.NewClient(ctx, xyrequests.ClientOption{
//     Spec: "...",
// })

// proxy := &httputil.ReverseProxy{
//     Director: func(req *http.Request) {
//         req.URL.Scheme = "https"
//         req.URL.Host = "target-api.example.com"
//         req.Host = "target-api.example.com"
//     },
//     Transport: client.Transport(), // 带 TLS 指纹的反向代理
// }

// http.ListenAndServe(":8080", proxy)
type FingerprintTransport struct {
	client        *Client
	ManageCookies bool // 为true时：忽略客户端发送的Cookie，不向客户端转发Set-Cookie，所有Cookie由client的CookieJar管理
}

// Transport 返回一个实现了 net/http.RoundTripper 的指纹传输层，
// 可用于 httputil.ReverseProxy 等标准库组件。
func (c *Client) Transport(manageCookies ...bool) stdhttp.RoundTripper {
	mc := false
	if len(manageCookies) > 0 {
		mc = manageCookies[0]
	}
	return &FingerprintTransport{client: c, ManageCookies: mc}
}

// RoundTrip 实现 net/http.RoundTripper 接口。
// 将标准库的 net/http.Request 转换为 fhttp.Request，
// 通过 tls-client 发送后，再将 fhttp.Response 转换回 net/http.Response。
func (t *FingerprintTransport) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	// 构建 fhttp.Request
	fReq := &fhttp.Request{
		Method:           req.Method,
		URL:              req.URL,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Header:           convertToFHTTPHeader(req.Header),
		Body:             req.Body,
		GetBody:          req.GetBody,
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Close:            req.Close,
		Host:             req.Host,
		Trailer:          convertToFHTTPHeader(req.Trailer),
	}

	// ManageCookies 模式下移除客户端发来的 Cookie，由 CookieJar 统一管理
	if t.ManageCookies {
		fReq.Header.Del("Cookie")
	}

	// 合并默认请求头（不覆盖已有的）
	defaultHeaders := t.client.defaultHeaders()
	for key, values := range defaultHeaders {
		if fReq.Header.Get(key) == "" {
			fReq.Header[key] = values
		}
	}
	t.client.applyHeaderOrdering(fReq.Header)

	// 通过 tls-client 发送请求
	fResp, err := t.client.HttpClient.Do(fReq)
	if err != nil {
		return nil, err
	}

	// 转换 fhttp.Response → net/http.Response
	respHeader := convertToStdHeader(fResp.Header)

	// ManageCookies 模式下移除上游返回的 Set-Cookie，不暴露给客户端
	if t.ManageCookies {
		respHeader.Del("Set-Cookie")
	}

	resp := &stdhttp.Response{
		Status:           fResp.Status,
		StatusCode:       fResp.StatusCode,
		Proto:            fResp.Proto,
		ProtoMajor:       fResp.ProtoMajor,
		ProtoMinor:       fResp.ProtoMinor,
		Header:           respHeader,
		Body:             fResp.Body,
		ContentLength:    fResp.ContentLength,
		TransferEncoding: fResp.TransferEncoding,
		Close:            fResp.Close,
		Uncompressed:     fResp.Uncompressed,
		Trailer:          convertToStdHeader(fResp.Trailer),
		Request:          req,
	}

	return resp, nil
}

// convertToFHTTPHeader 将 net/http.Header 转换为 fhttp.Header
func convertToFHTTPHeader(src stdhttp.Header) fhttp.Header {
	if src == nil {
		return nil
	}
	dst := make(fhttp.Header, len(src))
	for k, v := range src {
		vv := make([]string, len(v))
		copy(vv, v)
		dst[k] = vv
	}
	return dst
}

// convertToStdHeader 将 fhttp.Header 转换为 net/http.Header
func convertToStdHeader(src fhttp.Header) stdhttp.Header {
	if src == nil {
		return nil
	}
	dst := make(stdhttp.Header, len(src))
	for k, v := range src {
		// 跳过 fhttp 内部的特殊 key（如 HeaderOrderKey、PHeaderOrderKey）
		if k == fhttp.HeaderOrderKey || k == fhttp.PHeaderOrderKey {
			continue
		}
		vv := make([]string, len(v))
		copy(vv, v)
		dst[k] = vv
	}
	return dst
}

// RoundTripReadBody 与 RoundTrip 相同，但额外读取完整响应体后关闭连接。
// 适用于需要立即获取完整响应的场景。
func (t *FingerprintTransport) RoundTripReadBody(req *stdhttp.Request) (*stdhttp.Response, []byte, error) {
	resp, err := t.RoundTrip(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}

	return resp, body, nil
}
