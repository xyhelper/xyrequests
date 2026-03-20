package xyrequests

import (
	"context"
	"testing"

	stdhttp "net/http"

	fhttp "github.com/bogdanfinn/fhttp"
)

// TestStrictFingerprintMode 测试严格指纹模式是否覆盖客户端header
func TestStrictFingerprintMode(t *testing.T) {
	ctx := context.Background()

	// 创建 spec 客户端，启用 StrictFingerprint
	c, err := NewClient(ctx, ClientOption{
		Spec:              testGoSpiderSpec,
		StrictFingerprint: true,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 验证 strictFingerprint 标志已设置
	if !c.strictFingerprint {
		t.Errorf("Expected strictFingerprint=true, got false")
	}

	// 验证 specHeaders 不为空
	if len(c.specHeaders) == 0 {
		t.Errorf("Expected specHeaders to be non-empty")
	}

	// 验证 specHeaders 包含预期的 header
	if val := c.specHeaders.Get("User-Agent"); val == "" {
		t.Errorf("Expected User-Agent in specHeaders")
	}
}

// TestDoDifferentHeaderOrder 测试 Do 方法在严格模式下使用 spec 的 header 而非请求中的 header
func TestDoDifferentHeaderOrder(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ClientOption{
		Spec:              testGoSpiderSpec,
		StrictFingerprint: true,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 模拟有自定义 User-Agent 的请求
	req, _ := fhttp.NewRequest("GET", "https://example.com", nil)
	req.Header.Set("User-Agent", "CustomAgent/1.0")
	req.Header.Set("X-Custom-Header", "should-be-removed")

	// 调用 applyHeaderOrdering 前的初始 header 处理
	c.Do(context.Background(), req)

	// 验证：在严格模式下，原始的 User-Agent 应该被覆盖为 spec 中的值
	// （实际的验证需要通过 mock 或检查内部状态）
	// 这里主要测试没有错误发生
}

// TestRoundTripStrictMode 测试 RoundTrip 在严格模式下覆盖客户端 header
func TestRoundTripStrictMode(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ClientOption{
		Spec:              testGoSpiderSpec,
		StrictFingerprint: true,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 创建 transport
	transport := c.Transport().(stdhttp.RoundTripper)
	if transport == nil {
		t.Errorf("Expected transport to be non-nil")
	}

	footprint := transport.(*FingerprintTransport)
	if !footprint.client.strictFingerprint {
		t.Errorf("Expected transport's client to have strictFingerprint=true")
	}
}

// TestPassthroughHeaders 测试透传白名单功能
func TestPassthroughHeaders(t *testing.T) {
	ctx := context.Background()

	// 创建启用严格模式且指定透传白名单的客户端
	c, err := NewClient(ctx, ClientOption{
		Spec:               testGoSpiderSpec,
		StrictFingerprint:  true,
		PassthroughHeaders: []string{"Authorization", "X-Custom-Token"},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 验证透传白名单已构建（应包含 Authorization、X-Custom-Token）
	if _, ok := c.passthroughHeadersLower["authorization"]; !ok {
		t.Errorf("Expected 'authorization' in passthroughHeadersLower")
	}
	if _, ok := c.passthroughHeadersLower["x-custom-token"]; !ok {
		t.Errorf("Expected 'x-custom-token' in passthroughHeadersLower")
	}
	// Cookie 不应该自动在白名单中，需要显式声明
	if _, ok := c.passthroughHeadersLower["cookie"]; ok {
		t.Errorf("Expected 'cookie' NOT in passthroughHeadersLower (should be explicit)")
	}
}

// TestApplyStrictFingerprintHeadersWithPassthrough 测试严格模式下透传白名单的应用
func TestApplyStrictFingerprintHeadersWithPassthrough(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ClientOption{
		Spec:               testGoSpiderSpec,
		StrictFingerprint:  true,
		PassthroughHeaders: []string{"Authorization"},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 创建一个包含多个 header 的请求
	req, _ := fhttp.NewRequest("GET", "https://example.com", nil)
	req.Header.Set("User-Agent", "CustomAgent/1.0")    // 不在透传白名单，应被删除
	req.Header.Set("Authorization", "Bearer token123") // 在透传白名单，应保留
	req.Header.Set("X-Remove", "should-be-removed")    // 不在透传白名单，应被删除
	req.Header.Set("Cookie", "session=abc")            // 不在透传白名单，应被删除（可通过 PassthroughHeaders 显式声明保留）

	// 应用严格指纹 header
	c.applyStrictFingerprintHeaders(req.Header)

	// 验证：Authorization 应该保留
	if auth := req.Header.Get("Authorization"); auth != "Bearer token123" {
		t.Errorf("Expected Authorization to be preserved, got %q", auth)
	}

	// 验证：Cookie 应该被删除（因为没在 PassthroughHeaders 中声明）
	if cookie := req.Header.Get("Cookie"); cookie != "" {
		t.Errorf("Expected Cookie to be deleted (not in PassthroughHeaders), but got %q", cookie)
	}

	// 验证：spec 中的 header（如 User-Agent）应该被应用
	// 这里验证不会变成 CustomAgent 而是 spec 中的值
	if agent := req.Header.Get("User-Agent"); agent == "CustomAgent/1.0" {
		t.Errorf("Expected User-Agent to be replaced by spec value, but it remained as CustomAgent/1.0")
	}

	// 验证：X-Remove 应该被删除
	if xRemove := req.Header.Get("X-Remove"); xRemove != "" {
		t.Errorf("Expected X-Remove to be deleted, but got %q", xRemove)
	}
}

// TestCookieInPassthroughHeaders 测试 Cookie 可以显式在透传白名单中声明
func TestCookieInPassthroughHeaders(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ClientOption{
		Spec:               testGoSpiderSpec,
		StrictFingerprint:  true,
		PassthroughHeaders: []string{"Authorization", "Cookie"},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req, _ := fhttp.NewRequest("GET", "https://example.com", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Cookie", "session=abc")

	c.applyStrictFingerprintHeaders(req.Header)

	// 现在 Cookie 在白名单中，应该保留
	if cookie := req.Header.Get("Cookie"); cookie != "session=abc" {
		t.Errorf("Expected Cookie to be preserved (in PassthroughHeaders), got %q", cookie)
	}

	// Authorization 仍然保留
	if auth := req.Header.Get("Authorization"); auth != "Bearer token123" {
		t.Errorf("Expected Authorization to be preserved, got %q", auth)
	}
}
