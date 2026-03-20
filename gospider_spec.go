package xyrequests

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/net/http2/hpack"
)

// ==================== 类型定义 ====================

// GoSpiderSpec 表示解析后的 goSpiderSpec 指纹
type GoSpiderSpec struct {
	TLS *TLSSpec `json:"tls,omitempty"`
	H1  *H1Spec  `json:"h1,omitempty"`
	H2  *H2Spec  `json:"h2,omitempty"`
}

// TLSSpec 表示解析后的 TLS ClientHello 指纹
type TLSSpec struct {
	ContentType        uint8          `json:"contentType"`
	MessageVersion     uint16         `json:"messageVersion"`
	HandshakeVersion   uint16         `json:"handshakeVersion"`
	HandShakeType      uint8          `json:"handShakeType"`
	RandomTime         uint32         `json:"randomTime"`
	RandomBytes        []byte         `json:"randomBytes"`
	SessionId          []byte         `json:"sessionId"`
	CipherSuites       []uint16       `json:"cipherSuites"`
	CompressionMethods []byte         `json:"compressionMethods"`
	Extensions         []TLSExtension `json:"extensions"`
}

// TLSExtension 表示一个 TLS 扩展
type TLSExtension struct {
	Type uint16 `json:"type"`
	Data []byte `json:"-"` // JSON 中用 hex 字符串表示
}

// H1Spec 表示解析后的 HTTP/1.1 指纹
type H1Spec struct {
	OrderHeaders [][2]string `json:"orderHeaders"`
	Raw          string      `json:"raw"`
}

// H2Setting 表示 HTTP/2 设置参数
type H2Setting struct {
	ID  uint16 `json:"ID"`
	Val uint32 `json:"Val"`
}

// H2Priority 表示 HTTP/2 优先级参数
type H2Priority struct {
	Exclusive bool   `json:"exclusive"`
	StreamDep uint32 `json:"streamDep"`
	Weight    uint8  `json:"weight"`
}

// H2Stream 表示解析后的 HTTP/2 帧信息
type H2Stream struct {
	Name     string           `json:"name"`
	Type     uint8            `json:"type"`
	StreamID uint32           `json:"streamID"`
	Settings []h2StreamDetail `json:"settings,omitempty"`
	ConnFlow uint32           `json:"connFlow,omitempty"`
	Priority *H2Priority      `json:"priority,omitempty"`
	Headers  []h2Header       `json:"headers,omitempty"`
}

type h2StreamDetail struct {
	ID  uint16 `json:"id"`
	Val uint32 `json:"val"`
}

type h2Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// H2Spec 表示解析后的 HTTP/2 指纹
type H2Spec struct {
	Pri          string      `json:"pri"`
	Sm           string      `json:"sm"`
	Settings     []H2Setting `json:"settings"`
	ConnFlow     uint32      `json:"connFlow"`
	OrderHeaders [][2]string `json:"orderHeaders"`
	Priority     H2Priority  `json:"priority"`
	StreamID     uint32      `json:"-"`
	Streams      []H2Stream  `json:"streams"`
}

// ==================== 主要解析函数 ====================

// ParseGoSpiderSpec 解析 goSpiderSpec 字符串
// 格式: TLS_HEX @ H1_HEX @ H2_HEX (三部分用 @ 分隔)
func ParseGoSpiderSpec(value string) (*GoSpiderSpec, error) {
	parts := strings.Split(value, "@")
	if len(parts) != 3 {
		return nil, fmt.Errorf("goSpiderSpec 格式错误: 期望3部分, 实际 %d 部分", len(parts))
	}

	spec := &GoSpiderSpec{}

	// 解析 TLS 部分
	if parts[0] != "" {
		b, err := hex.DecodeString(parts[0])
		if err != nil {
			return nil, fmt.Errorf("TLS hex 解码失败: %w", err)
		}
		spec.TLS, err = parseTLSSpec(b)
		if err != nil {
			return nil, fmt.Errorf("TLS 解析失败: %w", err)
		}
	}

	// 解析 H1 部分
	if parts[1] != "" {
		b, err := hex.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("H1 hex 解码失败: %w", err)
		}
		spec.H1, err = parseH1Spec(b)
		if err != nil {
			return nil, fmt.Errorf("H1 解析失败: %w", err)
		}
	}

	// 解析 H2 部分
	if parts[2] != "" {
		b, err := hex.DecodeString(parts[2])
		if err != nil {
			return nil, fmt.Errorf("H2 hex 解码失败: %w", err)
		}
		spec.H2, err = parseH2Spec(b)
		if err != nil {
			return nil, fmt.Errorf("H2 解析失败: %w", err)
		}
	}

	return spec, nil
}

// ==================== TLS ClientHello 解析 ====================

func parseTLSSpec(raw []byte) (*TLSSpec, error) {
	spec := &TLSSpec{}
	s := cryptobyte.String(raw)

	// ContentType (1 byte) - 通常为 22 (handshake)
	if !s.ReadUint8(&spec.ContentType) {
		return nil, errors.New("读取 contentType 失败")
	}
	// MessageVersion (2 bytes) - 例如 0x0301 (TLS 1.0)
	if !s.ReadUint16(&spec.MessageVersion) {
		return nil, errors.New("读取 messageVersion 失败")
	}
	// Handshake 协议层
	var handShakeProtocol cryptobyte.String
	if !s.ReadUint16LengthPrefixed(&handShakeProtocol) {
		return nil, errors.New("读取 handshakeProtocol 失败")
	}
	// HandShakeType (1 byte) - 通常为 1 (ClientHello)
	if !handShakeProtocol.ReadUint8(&spec.HandShakeType) {
		return nil, errors.New("读取 handShakeType 失败")
	}
	// ClientHello 数据 (24-bit length prefixed)
	var handShakeData cryptobyte.String
	if !handShakeProtocol.ReadUint24LengthPrefixed(&handShakeData) {
		return nil, errors.New("读取 handShakeData 失败")
	}
	// HandshakeVersion (2 bytes)
	if !handShakeData.ReadUint16(&spec.HandshakeVersion) {
		return nil, errors.New("读取 handshakeVersion 失败")
	}
	// RandomTime (4 bytes)
	if !handShakeData.ReadUint32(&spec.RandomTime) {
		return nil, errors.New("读取 randomTime 失败")
	}
	// RandomBytes (28 bytes)
	if !handShakeData.ReadBytes(&spec.RandomBytes, 28) {
		return nil, errors.New("读取 randomBytes 失败")
	}
	// SessionId (uint8 length-prefixed)
	var sessionId cryptobyte.String
	if !handShakeData.ReadUint8LengthPrefixed(&sessionId) {
		return nil, errors.New("读取 sessionId 失败")
	}
	spec.SessionId = []byte(sessionId)

	// CipherSuites (uint16 length-prefixed, 每个 2 bytes)
	var cipherSuitesData cryptobyte.String
	if !handShakeData.ReadUint16LengthPrefixed(&cipherSuitesData) {
		return nil, errors.New("读取 cipherSuites 失败")
	}
	for !cipherSuitesData.Empty() {
		var cs uint16
		if cipherSuitesData.ReadUint16(&cs) {
			spec.CipherSuites = append(spec.CipherSuites, cs)
		}
	}

	// CompressionMethods (uint8 length-prefixed)
	var compressionMethods cryptobyte.String
	if !handShakeData.ReadUint8LengthPrefixed(&compressionMethods) {
		return nil, errors.New("读取 compressionMethods 失败")
	}
	spec.CompressionMethods = []byte(compressionMethods)

	// Extensions (uint16 length-prefixed)
	var extensionsData cryptobyte.String
	if !handShakeData.ReadUint16LengthPrefixed(&extensionsData) {
		return nil, errors.New("读取 extensions 失败")
	}
	for !extensionsData.Empty() {
		var extType uint16
		var extData cryptobyte.String
		if extensionsData.ReadUint16(&extType) && extensionsData.ReadUint16LengthPrefixed(&extData) {
			spec.Extensions = append(spec.Extensions, TLSExtension{
				Type: extType,
				Data: []byte(extData),
			})
		}
	}

	return spec, nil
}

// ==================== TLS 扩展解析辅助方法 ====================

// ServerName 提取 SNI 服务器名称 (扩展类型 0)
func (spec *TLSSpec) ServerName() string {
	for _, ext := range spec.Extensions {
		if ext.Type == 0 && len(ext.Data) >= 5 {
			// SNI 格式: 2字节列表长度, 1字节类型, 2字节名称长度, 名称
			nameLen := int(ext.Data[3])<<8 | int(ext.Data[4])
			if 5+nameLen <= len(ext.Data) {
				return string(ext.Data[5 : 5+nameLen])
			}
		}
	}
	return ""
}

// Protocols 提取 ALPN 协议列表 (扩展类型 16)
func (spec *TLSSpec) Protocols() []string {
	for _, ext := range spec.Extensions {
		if ext.Type == 16 && len(ext.Data) >= 2 {
			var protocols []string
			data := ext.Data[2:] // 跳过 2 字节列表长度
			for len(data) > 0 {
				pLen := int(data[0])
				data = data[1:]
				if pLen > len(data) {
					break
				}
				protocols = append(protocols, string(data[:pLen]))
				data = data[pLen:]
			}
			return protocols
		}
	}
	return nil
}

// Versions 提取支持的 TLS 版本 (扩展类型 43)
func (spec *TLSSpec) Versions() []uint16 {
	for _, ext := range spec.Extensions {
		if ext.Type == 43 && len(ext.Data) >= 1 {
			listLen := int(ext.Data[0])
			data := ext.Data[1:]
			if listLen > len(data) {
				listLen = len(data)
			}
			var versions []uint16
			for i := 0; i+1 < listLen; i += 2 {
				versions = append(versions, uint16(data[i])<<8|uint16(data[i+1]))
			}
			return versions
		}
	}
	return nil
}

// Algorithms 提取签名算法 (扩展类型 13)
func (spec *TLSSpec) Algorithms() []uint16 {
	for _, ext := range spec.Extensions {
		if ext.Type == 13 && len(ext.Data) >= 2 {
			listLen := int(ext.Data[0])<<8 | int(ext.Data[1])
			data := ext.Data[2:]
			if listLen > len(data) {
				listLen = len(data)
			}
			var algs []uint16
			for i := 0; i+1 < listLen; i += 2 {
				algs = append(algs, uint16(data[i])<<8|uint16(data[i+1]))
			}
			return algs
		}
	}
	return nil
}

// Curves 提取支持的曲线/组 (扩展类型 10)
func (spec *TLSSpec) Curves() []uint16 {
	for _, ext := range spec.Extensions {
		if ext.Type == 10 && len(ext.Data) >= 2 {
			listLen := int(ext.Data[0])<<8 | int(ext.Data[1])
			data := ext.Data[2:]
			if listLen > len(data) {
				listLen = len(data)
			}
			var curves []uint16
			for i := 0; i+1 < listLen; i += 2 {
				curves = append(curves, uint16(data[i])<<8|uint16(data[i+1]))
			}
			return curves
		}
	}
	return nil
}

// Points 提取支持的点格式 (扩展类型 11)
func (spec *TLSSpec) Points() []uint8 {
	for _, ext := range spec.Extensions {
		if ext.Type == 11 && len(ext.Data) >= 1 {
			listLen := int(ext.Data[0])
			data := ext.Data[1:]
			if listLen > len(data) {
				listLen = len(data)
			}
			return data[:listLen]
		}
	}
	return nil
}

// ==================== TLS Map 输出 ====================

// Map 将 TLS 指纹转换为 map (与 gospider007/fp 输出格式一致)
func (spec *TLSSpec) Map() map[string]any {
	extensions := make([]map[string]any, len(spec.Extensions))
	for i, ext := range spec.Extensions {
		extensions[i] = map[string]any{
			"type": ext.Type,
			"data": hex.EncodeToString(ext.Data),
		}
	}
	return map[string]any{
		"contentType":        spec.ContentType,
		"messageVersion":     spec.MessageVersion,
		"handshakeVersion":   spec.HandshakeVersion,
		"handShakeType":      spec.HandShakeType,
		"randomTime":         spec.RandomTime,
		"randomBytes":        spec.RandomBytes,
		"sessionId":          spec.SessionId,
		"cipherSuites":       spec.CipherSuites,
		"compressionMethods": spec.CompressionMethods,
		"extensions":         extensions,
		"serverName":         spec.ServerName(),
		"protocols":          spec.Protocols(),
		"versions":           spec.Versions(),
		"algorithms":         spec.Algorithms(),
		"curves":             spec.Curves(),
		"points":             spec.Points(),
	}
}

// ==================== H1 解析 ====================

func parseH1Spec(raw []byte) (*H1Spec, error) {
	i := bytes.Index(raw, []byte("\r\n\r\n"))
	if i == -1 {
		return nil, errors.New("H1: 未找到 \\r\\n\\r\\n")
	}
	content := raw[:i]
	var orderHeaders [][2]string
	for idx, line := range bytes.Split(content, []byte("\r\n")) {
		if idx == 0 {
			continue // 跳过请求行
		}
		parts := bytes.SplitN(line, []byte(": "), 2)
		if len(parts) < 2 {
			continue
		}
		orderHeaders = append(orderHeaders, [2]string{
			string(parts[0]),
			string(parts[1]),
		})
	}
	return &H1Spec{
		OrderHeaders: orderHeaders,
		Raw:          string(raw),
	}, nil
}

// Map 将 H1 指纹转换为 map
func (spec *H1Spec) Map() map[string]any {
	return map[string]any{
		"orderHeaders": spec.OrderHeaders,
		"raw":          spec.Raw,
	}
}

// ==================== H2 解析 ====================

// h2RawFrame 表示一个原始的 HTTP/2 帧
type h2RawFrame struct {
	Length   uint32
	Type     uint8
	Flags    uint8
	StreamID uint32
	Payload  []byte
}

// readH2Frame 从 reader 中读取一个 HTTP/2 帧
func readH2Frame(r io.Reader) (*h2RawFrame, error) {
	var header [9]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	length := uint32(header[0])<<16 | uint32(header[1])<<8 | uint32(header[2])

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}

	return &h2RawFrame{
		Length:   length,
		Type:     header[3],
		Flags:    header[4],
		StreamID: binary.BigEndian.Uint32(header[5:]) & 0x7fffffff,
		Payload:  payload,
	}, nil
}

func parseH2Spec(raw []byte) (*H2Spec, error) {
	// 查找 "PRI * HTTP/2.0\r\n\r\n"
	priEnd := bytes.Index(raw, []byte("\r\n\r\n"))
	if priEnd == -1 {
		return nil, errors.New("H2: 未找到 PRI 行")
	}
	pri := string(raw[:priEnd])
	remaining := raw[priEnd+4:]

	// 查找 "SM\r\n\r\n"
	smEnd := bytes.Index(remaining, []byte("\r\n\r\n"))
	if smEnd == -1 {
		return nil, errors.New("H2: 未找到 SM 行")
	}
	sm := string(remaining[:smEnd])
	frameData := remaining[smEnd+4:]

	// 解析 HTTP/2 帧
	reader := bytes.NewReader(frameData)
	decoder := hpack.NewDecoder(65536, nil)

	spec := &H2Spec{
		Pri:      pri,
		Sm:       sm,
		Settings: []H2Setting{},
		Streams:  []H2Stream{},
	}

	var headersFound bool

	for {
		frame, err := readH2Frame(reader)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("读取 H2 帧失败: %w", err)
		}

		switch frame.Type {
		case 0x04: // SETTINGS
			if frame.Flags&0x01 != 0 { // ACK, 跳过
				continue
			}
			stream := H2Stream{
				Name:     "Http2SettingsFrame",
				Type:     frame.Type,
				StreamID: frame.StreamID,
			}
			for i := 0; i+6 <= len(frame.Payload); i += 6 {
				id := binary.BigEndian.Uint16(frame.Payload[i : i+2])
				val := binary.BigEndian.Uint32(frame.Payload[i+2 : i+6])
				spec.Settings = append(spec.Settings, H2Setting{ID: id, Val: val})
				stream.Settings = append(stream.Settings, h2StreamDetail{ID: id, Val: val})
				// 更新 HPACK 动态表大小
				if id == 0x01 {
					decoder.SetMaxDynamicTableSize(val)
				}
			}
			spec.Streams = append(spec.Streams, stream)

		case 0x08: // WINDOW_UPDATE
			if len(frame.Payload) >= 4 {
				spec.ConnFlow = binary.BigEndian.Uint32(frame.Payload[:4]) & 0x7fffffff
			}
			spec.Streams = append(spec.Streams, H2Stream{
				Name:     "Http2WindowUpdateFrame",
				Type:     frame.Type,
				StreamID: frame.StreamID,
				ConnFlow: spec.ConnFlow,
			})

		case 0x01: // HEADERS
			p := frame.Payload
			var priority H2Priority

			// 处理 PADDED 标志
			var padLen byte
			if frame.Flags&0x08 != 0 {
				if len(p) < 1 {
					return nil, errors.New("HEADERS 帧 pad length 读取失败")
				}
				padLen = p[0]
				p = p[1:]
			}

			// 处理 PRIORITY 标志
			if frame.Flags&0x20 != 0 {
				if len(p) < 5 {
					return nil, errors.New("HEADERS 帧 priority 数据不足")
				}
				v := binary.BigEndian.Uint32(p[:4])
				priority = H2Priority{
					StreamDep: v & 0x7fffffff,
					Exclusive: v != v&0x7fffffff,
					Weight:    p[4],
				}
				p = p[5:]
			}

			spec.Priority = priority
			spec.StreamID = frame.StreamID

			// 去除 padding
			if int(padLen) > len(p) {
				return nil, errors.New("HEADERS 帧 padding 超出数据范围")
			}
			headerBlock := p[:len(p)-int(padLen)]

			// 如果 END_HEADERS 未设置, 继续读取 CONTINUATION 帧
			endHeaders := frame.Flags&0x04 != 0
			for !endHeaders {
				contFrame, err := readH2Frame(reader)
				if err != nil {
					return nil, fmt.Errorf("读取 CONTINUATION 帧失败: %w", err)
				}
				if contFrame.Type != 0x09 {
					return nil, errors.New("期望 CONTINUATION 帧")
				}
				headerBlock = append(headerBlock, contFrame.Payload...)
				endHeaders = contFrame.Flags&0x04 != 0
			}

			// HPACK 解码
			fields, err := decoder.DecodeFull(headerBlock)
			if err != nil {
				return nil, fmt.Errorf("HPACK 解码失败: %w", err)
			}

			stream := H2Stream{
				Name:     "Http2MetaHeadersFrame",
				Type:     frame.Type,
				StreamID: frame.StreamID,
				Priority: &priority,
			}

			for _, f := range fields {
				if !f.IsPseudo() {
					spec.OrderHeaders = append(spec.OrderHeaders, [2]string{f.Name, f.Value})
					stream.Headers = append(stream.Headers, h2Header{
						Name:  f.Name,
						Value: f.Value,
					})
				}
			}
			spec.Streams = append(spec.Streams, stream)
			headersFound = true

			// HEADERS 帧已完成, 停止解析
			break

		case 0x06: // PING
			spec.Streams = append(spec.Streams, H2Stream{
				Name:     "Http2PingFrame",
				Type:     frame.Type,
				StreamID: frame.StreamID,
			})

		default:
			// 忽略其他帧类型
		}

		if headersFound {
			break
		}
	}

	return spec, nil
}

// Map 将 H2 指纹转换为 map (与 gospider007/fp 输出格式一致)
func (spec *H2Spec) Map() map[string]any {
	streams := make([]map[string]any, len(spec.Streams))
	for i, s := range spec.Streams {
		m := map[string]any{
			"name":     s.Name,
			"type":     s.Type,
			"streamID": s.StreamID,
		}
		if s.Settings != nil {
			m["settings"] = s.Settings
		}
		if s.ConnFlow > 0 {
			m["connFlow"] = s.ConnFlow
		}
		if s.Priority != nil {
			m["priority"] = map[string]any{
				"exclusive": s.Priority.Exclusive,
				"streamDep": s.Priority.StreamDep,
				"weight":    s.Priority.Weight,
			}
		}
		if s.Headers != nil {
			m["headers"] = s.Headers
		}
		streams[i] = m
	}

	return map[string]any{
		"pri":          spec.Pri,
		"sm":           spec.Sm,
		"settings":     spec.Settings,
		"connFlow":     spec.ConnFlow,
		"orderHeaders": spec.OrderHeaders,
		"priority": map[string]any{
			"exclusive": spec.Priority.Exclusive,
			"streamDep": spec.Priority.StreamDep,
			"weight":    spec.Priority.Weight,
		},
		"streams": streams,
	}
}

// ==================== GoSpiderSpec Map 输出 ====================

// Map 将完整的指纹转换为 map
func (spec *GoSpiderSpec) Map() map[string]any {
	result := map[string]any{}
	if spec.TLS != nil {
		result["tls"] = spec.TLS.Map()
	}
	if spec.H1 != nil {
		result["h1"] = spec.H1.Map()
	}
	if spec.H2 != nil {
		result["h2"] = spec.H2.Map()
	}
	return result
}

// ==================== GREASE 判断工具 ====================

// IsGREASE 判断一个 uint16 值是否为 GREASE 值
func IsGREASE(v uint16) bool {
	return ((v >> 8) == v&0xff) && v&0xf == 0xa
}

// ExtensionTypes 返回所有扩展类型的列表 (用于 JA3 计算等)
func (spec *TLSSpec) ExtensionTypes() []uint16 {
	types := make([]uint16, len(spec.Extensions))
	for i, ext := range spec.Extensions {
		types[i] = ext.Type
	}
	return types
}
