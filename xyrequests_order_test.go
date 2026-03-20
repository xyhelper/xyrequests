package xyrequests

import (
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

func TestApplyHeaderOrderingFromSpec(t *testing.T) {
	client := &Client{
		defaultHeaderOrder:       []string{"sec-fetch-site", "priority", "user-agent"},
		defaultPseudoHeaderOrder: []string{":method", ":authority", ":scheme", ":path"},
	}

	header := http.Header{
		"User-Agent":      []string{"ua"},
		"Priority":        []string{"u=0, i"},
		"Sec-Fetch-Site":  []string{"none"},
		"Accept-Language": []string{"en-US"},
	}

	client.applyHeaderOrdering(header)

	order, ok := header[http.HeaderOrderKey]
	if !ok {
		t.Fatal("HeaderOrderKey 未写入")
	}
	if len(order) < 4 {
		t.Fatalf("HeaderOrderKey 数量异常: %v", order)
	}

	expectedPrefix := []string{"sec-fetch-site", "priority", "user-agent"}
	for i, expected := range expectedPrefix {
		if order[i] != expected {
			t.Fatalf("HeaderOrderKey[%d] 期望 %s, 实际 %s", i, expected, order[i])
		}
	}

	pseudo, ok := header[http.PHeaderOrderKey]
	if !ok {
		t.Fatal("PHeaderOrderKey 未写入")
	}
	expectedPseudo := []string{":method", ":authority", ":scheme", ":path"}
	if len(pseudo) != len(expectedPseudo) {
		t.Fatalf("PHeaderOrderKey 长度期望 %d, 实际 %d", len(expectedPseudo), len(pseudo))
	}
	for i, expected := range expectedPseudo {
		if pseudo[i] != expected {
			t.Fatalf("PHeaderOrderKey[%d] 期望 %s, 实际 %s", i, expected, pseudo[i])
		}
	}
}
