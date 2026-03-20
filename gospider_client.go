package xyrequests

import (
	"encoding/hex"
	"fmt"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/http2"
	"github.com/bogdanfinn/tls-client/profiles"
	tls "github.com/bogdanfinn/utls"
)

// buildProfileFromSpec 从 goSpiderSpec 字符串构建 profiles.ClientProfile 和默认请求头
func buildProfileFromSpec(goSpiderSpec string) (profiles.ClientProfile, http.Header, error) {
	parts := strings.Split(goSpiderSpec, "@")
	if len(parts) != 3 {
		return profiles.ClientProfile{}, nil, fmt.Errorf("goSpiderSpec 格式错误: 期望3部分, 实际 %d 部分", len(parts))
	}

	// 解码 TLS 原始字节
	tlsRaw, err := hex.DecodeString(parts[0])
	if err != nil {
		return profiles.ClientProfile{}, nil, fmt.Errorf("TLS hex 解码失败: %w", err)
	}

	// 解码 H2 原始字节并解析
	var h2Spec *H2Spec
	if parts[2] != "" {
		h2Raw, err := hex.DecodeString(parts[2])
		if err != nil {
			return profiles.ClientProfile{}, nil, fmt.Errorf("H2 hex 解码失败: %w", err)
		}
		h2Spec, err = parseH2Spec(h2Raw)
		if err != nil {
			return profiles.ClientProfile{}, nil, fmt.Errorf("H2 解析失败: %w", err)
		}
	}

	profile, err := buildClientProfile(tlsRaw, h2Spec)
	if err != nil {
		return profiles.ClientProfile{}, nil, err
	}

	// 从 H2 OrderHeaders 提取默认请求头（跳过伪头部）
	specHeaders := extractDefaultHeaders(h2Spec)

	return profile, specHeaders, nil
}

// extractDefaultHeaders 从 H2Spec 的 OrderHeaders 中提取非伪头部作为默认请求头
func extractDefaultHeaders(h2 *H2Spec) http.Header {
	if h2 == nil || len(h2.OrderHeaders) == 0 {
		return nil
	}
	headers := make(http.Header)
	for _, h := range h2.OrderHeaders {
		name := h[0]
		value := h[1]
		// 跳过伪头部（以 : 开头）
		if len(name) > 0 && name[0] == ':' {
			continue
		}
		headers.Set(name, value)
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

// buildClientProfile 从 TLS 原始字节和 H2 规格构建 profiles.ClientProfile
func buildClientProfile(tlsRaw []byte, h2 *H2Spec) (profiles.ClientProfile, error) {
	rawCopy := make([]byte, len(tlsRaw))
	copy(rawCopy, tlsRaw)

	clientHelloID := tls.ClientHelloID{
		Client:  "GoSpider",
		Version: "Custom",
		SpecFactory: func() (tls.ClientHelloSpec, error) {
			var spec tls.ClientHelloSpec
			if err := spec.FromRaw(rawCopy, true); err != nil {
				return tls.ClientHelloSpec{}, fmt.Errorf("FromRaw 解析失败: %w", err)
			}
			return spec, nil
		},
	}

	settings, settingsOrder, connFlow, pseudoHeaderOrder, headerPriority, priorities, streamID := buildH2Params(h2)

	return profiles.NewClientProfile(
		clientHelloID,
		settings,
		settingsOrder,
		pseudoHeaderOrder,
		connFlow,
		priorities,
		headerPriority,
		streamID,
		false, // allowHTTP
		nil,   // http3Settings
		nil,   // http3SettingsOrder
		0,     // http3PriorityParam
		nil,   // http3PseudoHeaderOrder
		false, // http3SendGreaseFrames
	), nil
}

// buildH2Params 从 H2Spec 构建 HTTP/2 参数
func buildH2Params(h2 *H2Spec) (
	settings map[http2.SettingID]uint32,
	settingsOrder []http2.SettingID,
	connFlow uint32,
	pseudoHeaderOrder []string,
	headerPriority *http2.PriorityParam,
	priorities []http2.Priority,
	streamID uint32,
) {
	if h2 == nil {
		settings = map[http2.SettingID]uint32{
			http2.SettingHeaderTableSize:   65536,
			http2.SettingEnablePush:        0,
			http2.SettingInitialWindowSize: 6291456,
			http2.SettingMaxHeaderListSize: 262144,
		}
		settingsOrder = []http2.SettingID{
			http2.SettingHeaderTableSize,
			http2.SettingEnablePush,
			http2.SettingInitialWindowSize,
			http2.SettingMaxHeaderListSize,
		}
		connFlow = 15663105
		pseudoHeaderOrder = []string{":method", ":authority", ":scheme", ":path"}
		return
	}

	settings = make(map[http2.SettingID]uint32, len(h2.Settings))
	settingsOrder = make([]http2.SettingID, 0, len(h2.Settings))
	for _, s := range h2.Settings {
		sid := http2.SettingID(s.ID)
		settings[sid] = s.Val
		settingsOrder = append(settingsOrder, sid)
	}

	connFlow = h2.ConnFlow

	if len(h2.OrderHeaders) > 0 {
		pseudoHeaderOrder = make([]string, 0, 4)
		for _, h := range h2.OrderHeaders {
			if len(h[0]) > 0 && h[0][0] == ':' {
				pseudoHeaderOrder = append(pseudoHeaderOrder, h[0])
			}
		}
	}
	if len(pseudoHeaderOrder) == 0 {
		pseudoHeaderOrder = []string{":method", ":authority", ":scheme", ":path"}
	}

	if h2.Priority.StreamDep != 0 || h2.Priority.Weight != 0 || h2.Priority.Exclusive {
		headerPriority = &http2.PriorityParam{
			StreamDep: h2.Priority.StreamDep,
			Exclusive: h2.Priority.Exclusive,
			Weight:    h2.Priority.Weight,
		}
	}

	streamID = h2.StreamID

	return
}
