package xyrequests

import (
"context"
"encoding/hex"
"strings"
"testing"
)

func TestNewClientWithSpec(t *testing.T) {
ctx := context.Background()

client, err := NewClient(ctx, ClientOption{
Spec: testGoSpiderSpec,
})
if err != nil {
t.Fatalf("NewClient with Spec 失败: %v", err)
}
if client == nil {
t.Fatal("创建的客户端为 nil")
}
t.Log("成功通过 Spec 创建 tls-client")
}

func TestNewClientWithSpecAndOptions(t *testing.T) {
ctx := context.Background()

client, err := NewClient(ctx, ClientOption{
Spec:           testGoSpiderSpec,
TimeoutSeconds: 30,
Debug:          false,
})
if err != nil {
t.Fatalf("NewClient with Spec 失败: %v", err)
}
if client == nil {
t.Fatal("创建的客户端为 nil")
}
t.Log("成功通过 Spec + 选项创建 tls-client")
}

func TestNewClientWithoutSpec(t *testing.T) {
ctx := context.Background()

client, err := NewClient(ctx, ClientOption{
TimeoutSeconds: 10,
})
if err != nil {
t.Fatalf("NewClient 失败: %v", err)
}
if client == nil {
t.Fatal("创建的客户端为 nil")
}
t.Log("成功通过默认 profile 创建 tls-client")
}

func TestBuildClientProfile(t *testing.T) {
spec, err := ParseGoSpiderSpec(testGoSpiderSpec)
if err != nil {
t.Fatalf("ParseGoSpiderSpec 失败: %v", err)
}

parts := strings.Split(testGoSpiderSpec, "@")
if len(parts) != 3 {
t.Fatal("分割 goSpiderSpec 失败")
}

tlsRaw, err := hex.DecodeString(parts[0])
if err != nil {
t.Fatalf("TLS hex 解码失败: %v", err)
}

profile, err := buildClientProfile(tlsRaw, spec.H2)
if err != nil {
t.Fatalf("buildClientProfile 失败: %v", err)
}

chSpec, err := profile.GetClientHelloSpec()
if err != nil {
t.Fatalf("GetClientHelloSpec 失败: %v", err)
}

if len(chSpec.CipherSuites) == 0 {
t.Fatal("CipherSuites 不应为空")
}
t.Logf("CipherSuites 数量: %d", len(chSpec.CipherSuites))

if len(chSpec.Extensions) == 0 {
t.Fatal("Extensions 不应为空")
}
t.Logf("Extensions 数量: %d", len(chSpec.Extensions))

settings := profile.GetSettings()
if settings == nil {
t.Fatal("Settings 不应为 nil")
}
t.Logf("Settings: %v", settings)

settingsOrder := profile.GetSettingsOrder()
if len(settingsOrder) == 0 {
t.Fatal("SettingsOrder 不应为空")
}
t.Logf("SettingsOrder: %v", settingsOrder)

connFlow := profile.GetConnectionFlow()
if connFlow == 0 {
t.Fatal("ConnectionFlow 不应为 0")
}
t.Logf("ConnectionFlow: %d", connFlow)

pseudoHeaders := profile.GetPseudoHeaderOrder()
if len(pseudoHeaders) == 0 {
t.Fatal("PseudoHeaderOrder 不应为空")
}
t.Logf("PseudoHeaderOrder: %v", pseudoHeaders)

headerPrio := profile.GetHeaderPriority()
t.Logf("HeaderPriority: %v", headerPrio)
}

func TestNewClientWithInvalidSpec(t *testing.T) {
ctx := context.Background()

_, err := NewClient(ctx, ClientOption{
Spec: "invalid",
})
if err == nil {
t.Fatal("期望错误但未返回")
}
t.Logf("正确返回错误: %v", err)
}
