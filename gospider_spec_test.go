package xyrequests

import (
	"encoding/json"
	"fmt"
	"testing"
)

// 来自 sample.json 的 goSpiderSpec 测试数据
const testGoSpiderSpec = "16030106d2010006ce03033c2b834c59248a7e109bd5cdf53664bf6fc822ca6046860aa14fca310551957620120fb492b3b5531b2eec9b0e8dd1ecfa0654799ca270ae6725d323e9299728ff00202a2a130313011302cca9cca8c02bc02fc02cc030c013c014009c009d002f0035010006651a1a0000002b0007069a9a03040303002d00020101000a000c000a4a4a11ec001d00170018ff01000100000500050100000000000b0002010000230000001b00030200020010000e000c02683208687474702f312e31003304ef04ed4a4a00010011ec04c0ac5a6c1009c61cc5a004b362955bb21e468c89dccdef799be233b8e9b3314280bad716119c5626ea31ca6b70492956357ed18cdb013ca06aa9570996ebc27c0e3602a3431182224ab3259bac71ade281b6e517adb9880f3628178fd69b64b62c75674b09398a6ad760434884fd9145f0cc36e521bf72998065d32864f20843b48809a606c64aaa29123ade5c484f5a0ca535a2df885a71037347861606ac9b9d14c6fd145908aa80625b33896b8b7a58c7cab02ccc39afb83b5731d46363a633c2c45b3607a8fc598920916b540b3b01ac8528338f0175264fc56d82511b7962b26e9768fd1c5d82a86b1103c24ad20cbc924e6066cecbb33e41648092095b9f5c03fbab79104b1b6ea48e66099109a9a623456cc906cb85465080c1b836d633c669a639219da5b07c7ce699fbc4951668ad0acba6e3e7028f540b67a9896f40173500ce5b2aaede15c0915885e7e0cbf68ab4cdf39798088540fbae3e36696bdca6f8ab9b0b54741c5c14d69a8592a39b743448997415f1aa1bafa42de2234044baa24be0c74c253ff786c921c8b3e6d96d6d95735862c6459b7b1160b73c6199bf1b0381259b75d5604025000379bdcb1a086f97c0ce4982f469cff50473e7d162bf610670b42b16988412b81ff9473af8ab1c2e8247a3b39f32a569b24c8cc0708b5c72b1a8f986ceda57b3e77227400f5a350649d84958da5fc7c6310da893cb8002d38572259c49ace84cde8c6380b27633574840b7a523f843a1f18493c799bf5285122b611eb317ec67439fa15f8fac95adf8a4b0c06ea1ec9fc03bc521c2aa8f96a68c72222377152c821ff4f0c6e2d64d4f46c9664c525f90b0c1cbccc0b580a1c0c8c6b51858e49793d8c635d367d2781a9a327ce5d77a45489c3157c803976c37d1395d5068751a3c90d27a86c69644a9160337a77c85354ca534f2b981ec161dbbf6347bec6f32b1834771207164447b01140390156bb1048f86acc8a83518d7029dc9713601bf319096e69b09ae1a15a3697898f781418a85b6056bb1d832c10951587093b90808da2115259b5592c46f43db86ee1b1738d20223e749a2b4244cac6353504f00688203419db2d4686c381f384941bdac47cdec9fc9d49409fb8b5bc0150bf42c173b626d9736a6ca59ecea95c0880f4b089b30455763a48db8817e30b00fd6781d7928a5086ca6b9183164209906100532c96c8a73cc808141dafb402c2bc8dd56ba57c80a419552488aa6534515036ac0f71a3936412121ebcc3173752f1268b02755edd661c6a69406c8aacf8084e15063a4c578c4b5a160bcba73a1485a18ba672c21fc4451d3b916219194e0171e2bb99804144fbf6243ae78183d538be3f79341b3ce59a2a3c9fc4797098f7ab8b04ea1c59481162d9515da31b08713900ed5c20933cde8da79040654f89118040811f68a0a33a85d1d12473ed918e574aeadc2904489af7d9968c9c864fec5556bfbb2caaa1650c3a78b648d0817be3bd36629e225d1c838e718c084ca764cd9936c29b3aae91744946cf3241471d42acfb88717135de0a077d2f6124904073c503d9249c5d6940d4497cf04b2634bea6c4efa5cdbc32677d83cde83628a94cc6cd16ad1ca60b738b036acaafeec72e5bb5bf270881e4cb62c7fad030cb28ffcdc37f0bc3a0550149554731eb5fe1c4ede48ccb85a5663c6a3cce095d41404f145c100001d0020d8fb76a0385f42fc29c0288e2ce9cbb8cf11814f0205fe6ac0aced17b3a25244001200000000000e000c0000096c6f63616c686f737444cd00050003026832000d001200100403080404010503080505010806060100170000fe0d00da0000010003d10020efb18015c7b7a668897237e23d003814aa9822b9ec3b444a205aa2c6e627b94e00b016897902ad135724600af28c8a8319a66d39d7f16b2f153815e8d5594dfb613f7a271dab3a17946377737b5ed8e9995f09d041420ce2406baab6e3a6e3304601a7f6879d04bb689b5e0bc5fd4d54ed3a1b73d9243ea5d6cf53cacde9dfbe3ebe276a2647f736ac8c9e384a9094b8fa26b875e91239ff8c783b463d27d72f0881002db57817ba67fe07308fa39203c78739b4caf3b1c0bbf648fc7cf72ec6f6570b7404728c58c79d0d327009293b9abffafa000100@@505249202a20485454502f322e300d0a0d0a534d0d0a0d0a00001804000000000000010001000000020000000000040060000000060004000000000408000000000000ef00010001c601250000000180000000ff82418aa0e41d139d09b8f3efbf878440874148b1275ad1ffb8fe749d37215aed83aa4fe7efbc1fcbefff3f4a7f388e79a82a97a7b0f497f9fbef07f2169bfe7e94fe6f4f61e935b4ff3f7de0fe42d37fcf408b4148b1275ad1ad49e33505023f30408d4148b1275ad1ad5d034ca7b29f88fe791aa90fe11fcf4092b6b9ac1c8558d520a4b6c2ad617b5a54251f01317ad5d07f66a281b0dae053fae46aa43f8429a77a8102e0fb5391aa71afb53cb8d7f6a435d74179163cc64b0db2eaecb8a7f59b1efd19fe94a0dd4aa62293a9ffb52f4f61e92b0169b5c0b817029b8728ec330db2eaecb953e5497ca589d34d1f43aeba0c41a4c7a98f33a69a3fdf9a68fa1d75d0620d263d4c79a68fbed00177fe8d48e62b03ee697e8d48e62b1e0b1d7f46a4731581d754df5f2c7cfdf6800bbdf43aeba0c41a4c7a9841a6a8b22c5f249c754c5fbef046cfdf6800bbbf408a4148b4a549275906497f83a8f517408a4148b4a549275a93c85f86a87dcd30d25f408a4148b4a549275ad416cf023f31408a4148b4a549275a42a13f8690e4b692d49f50929bd9abfa5242cb40d25fa523b3e94f684c9f518b2d4b70ddf45abefb4005df4086aec31ec327d785b6007d286f"

func TestParseGoSpiderSpec(t *testing.T) {
	spec, err := ParseGoSpiderSpec(testGoSpiderSpec)
	if err != nil {
		t.Fatalf("ParseGoSpiderSpec 失败: %v", err)
	}

	// 验证 TLS 部分
	if spec.TLS == nil {
		t.Fatal("TLS spec 不应为空")
	}
	t.Logf("ContentType: %d", spec.TLS.ContentType)
	if spec.TLS.ContentType != 22 {
		t.Errorf("ContentType 期望 22, 实际 %d", spec.TLS.ContentType)
	}
	t.Logf("MessageVersion: %d (0x%04x)", spec.TLS.MessageVersion, spec.TLS.MessageVersion)
	if spec.TLS.MessageVersion != 769 {
		t.Errorf("MessageVersion 期望 769, 实际 %d", spec.TLS.MessageVersion)
	}
	t.Logf("HandshakeVersion: %d (0x%04x)", spec.TLS.HandshakeVersion, spec.TLS.HandshakeVersion)
	if spec.TLS.HandshakeVersion != 771 {
		t.Errorf("HandshakeVersion 期望 771, 实际 %d", spec.TLS.HandshakeVersion)
	}
	t.Logf("HandShakeType: %d", spec.TLS.HandShakeType)
	if spec.TLS.HandShakeType != 1 {
		t.Errorf("HandShakeType 期望 1, 实际 %d", spec.TLS.HandShakeType)
	}
	t.Logf("RandomTime: %d", spec.TLS.RandomTime)
	if spec.TLS.RandomTime != 1009484620 {
		t.Errorf("RandomTime 期望 1009484620, 实际 %d", spec.TLS.RandomTime)
	}

	// 验证 CipherSuites
	expectedCiphers := []uint16{10794, 4867, 4865, 4866, 52393, 52392, 49195, 49199, 49196, 49200, 49171, 49172, 156, 157, 47, 53}
	if len(spec.TLS.CipherSuites) != len(expectedCiphers) {
		t.Errorf("CipherSuites 长度期望 %d, 实际 %d", len(expectedCiphers), len(spec.TLS.CipherSuites))
	} else {
		for i, cs := range spec.TLS.CipherSuites {
			if cs != expectedCiphers[i] {
				t.Errorf("CipherSuites[%d] 期望 %d, 实际 %d", i, expectedCiphers[i], cs)
			}
		}
	}
	t.Logf("CipherSuites: %v", spec.TLS.CipherSuites)

	// 验证 Extensions
	t.Logf("Extensions 数量: %d", len(spec.TLS.Extensions))
	for i, ext := range spec.TLS.Extensions {
		t.Logf("  Extension[%d]: type=%d (0x%04x), data_len=%d", i, ext.Type, ext.Type, len(ext.Data))
	}

	// 验证派生字段
	t.Logf("ServerName: %s", spec.TLS.ServerName())
	if spec.TLS.ServerName() != "localhost" {
		t.Errorf("ServerName 期望 'localhost', 实际 '%s'", spec.TLS.ServerName())
	}

	t.Logf("Protocols: %v", spec.TLS.Protocols())
	protocols := spec.TLS.Protocols()
	if len(protocols) != 2 || protocols[0] != "h2" || protocols[1] != "http/1.1" {
		t.Errorf("Protocols 期望 [h2, http/1.1], 实际 %v", protocols)
	}

	t.Logf("Versions: %v", spec.TLS.Versions())
	t.Logf("Algorithms: %v", spec.TLS.Algorithms())
	t.Logf("Curves: %v", spec.TLS.Curves())

	algorithms := spec.TLS.Algorithms()
	expectedAlgorithms := []uint16{1027, 2052, 1025, 1283, 2053, 1281, 2054, 1537}
	if len(algorithms) != len(expectedAlgorithms) {
		t.Errorf("Algorithms 长度期望 %d, 实际 %d", len(expectedAlgorithms), len(algorithms))
	} else {
		for i, a := range algorithms {
			if a != expectedAlgorithms[i] {
				t.Errorf("Algorithms[%d] 期望 %d, 实际 %d", i, expectedAlgorithms[i], a)
			}
		}
	}

	curves := spec.TLS.Curves()
	expectedCurves := []uint16{19018, 4588, 29, 23, 24}
	if len(curves) != len(expectedCurves) {
		t.Errorf("Curves 长度期望 %d, 实际 %d", len(expectedCurves), len(curves))
	} else {
		for i, c := range curves {
			if c != expectedCurves[i] {
				t.Errorf("Curves[%d] 期望 %d, 实际 %d", i, expectedCurves[i], c)
			}
		}
	}

	// 验证 H1 部分 (应为空)
	if spec.H1 != nil {
		t.Error("H1 spec 应为空")
	}

	// 验证 H2 部分
	if spec.H2 == nil {
		t.Fatal("H2 spec 不应为空")
	}
	t.Logf("H2 Pri: %s", spec.H2.Pri)
	if spec.H2.Pri != "PRI * HTTP/2.0" {
		t.Errorf("H2 Pri 期望 'PRI * HTTP/2.0', 实际 '%s'", spec.H2.Pri)
	}
	t.Logf("H2 Sm: %s", spec.H2.Sm)
	if spec.H2.Sm != "SM" {
		t.Errorf("H2 Sm 期望 'SM', 实际 '%s'", spec.H2.Sm)
	}

	// 验证 H2 Settings
	t.Logf("H2 Settings: %+v", spec.H2.Settings)
	expectedSettings := []H2Setting{
		{ID: 1, Val: 65536},
		{ID: 2, Val: 0},
		{ID: 4, Val: 6291456},
		{ID: 6, Val: 262144},
	}
	if len(spec.H2.Settings) != len(expectedSettings) {
		t.Errorf("H2 Settings 长度期望 %d, 实际 %d", len(expectedSettings), len(spec.H2.Settings))
	} else {
		for i, s := range spec.H2.Settings {
			if s.ID != expectedSettings[i].ID || s.Val != expectedSettings[i].Val {
				t.Errorf("H2 Settings[%d] 期望 %+v, 实际 %+v", i, expectedSettings[i], s)
			}
		}
	}

	// 验证 H2 ConnFlow
	t.Logf("H2 ConnFlow: %d", spec.H2.ConnFlow)
	if spec.H2.ConnFlow != 15663105 {
		t.Errorf("H2 ConnFlow 期望 15663105, 实际 %d", spec.H2.ConnFlow)
	}

	// 验证 H2 Priority
	t.Logf("H2 Priority: %+v", spec.H2.Priority)
	if !spec.H2.Priority.Exclusive || spec.H2.Priority.StreamDep != 0 || spec.H2.Priority.Weight != 255 {
		t.Errorf("H2 Priority 期望 {Exclusive:true StreamDep:0 Weight:255}, 实际 %+v", spec.H2.Priority)
	}

	// 验证 H2 OrderHeaders
	t.Logf("H2 OrderHeaders 数量: %d", len(spec.H2.OrderHeaders))
	for i, h := range spec.H2.OrderHeaders {
		t.Logf("  Header[%d]: %s = %s", i, h[0], h[1])
	}
	if len(spec.H2.OrderHeaders) == 0 {
		t.Error("H2 OrderHeaders 不应为空")
	}

	// 验证已知的 headers
	expectedHeaders := map[string]string{
		"sec-ch-ua-mobile": "?0",
		"sec-fetch-site":   "none",
		"sec-fetch-mode":   "navigate",
		"sec-fetch-user":   "?1",
		"sec-fetch-dest":   "document",
		"priority":         "u=0, i",
	}
	headerMap := make(map[string]string)
	for _, h := range spec.H2.OrderHeaders {
		headerMap[h[0]] = h[1]
	}
	for k, v := range expectedHeaders {
		if headerMap[k] != v {
			t.Errorf("Header '%s' 期望 '%s', 实际 '%s'", k, v, headerMap[k])
		}
	}

	// 验证 Streams
	t.Logf("H2 Streams 数量: %d", len(spec.H2.Streams))
	if len(spec.H2.Streams) < 3 {
		t.Errorf("H2 Streams 期望至少 3 个, 实际 %d", len(spec.H2.Streams))
	} else {
		if spec.H2.Streams[0].Name != "Http2SettingsFrame" {
			t.Errorf("第一个 stream 期望 Http2SettingsFrame, 实际 %s", spec.H2.Streams[0].Name)
		}
		if spec.H2.Streams[1].Name != "Http2WindowUpdateFrame" {
			t.Errorf("第二个 stream 期望 Http2WindowUpdateFrame, 实际 %s", spec.H2.Streams[1].Name)
		}
		if spec.H2.Streams[2].Name != "Http2MetaHeadersFrame" {
			t.Errorf("第三个 stream 期望 Http2MetaHeadersFrame, 实际 %s", spec.H2.Streams[2].Name)
		}
	}

	// 输出完整的 JSON
	result := spec.Map()
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}
	fmt.Println(string(jsonBytes))
}

func TestIsGREASE(t *testing.T) {
	greaseValues := []uint16{0x0a0a, 0x1a1a, 0x2a2a, 0x3a3a, 0x4a4a, 0x5a5a, 0x6a6a, 0x7a7a, 0x8a8a, 0x9a9a, 0xaaaa, 0xbaba, 0xcaca, 0xdada, 0xeaea, 0xfafa}
	for _, v := range greaseValues {
		if !IsGREASE(v) {
			t.Errorf("IsGREASE(%d/0x%04x) 应为 true", v, v)
		}
	}

	nonGreaseValues := []uint16{0x0001, 0x0303, 0x0304, 0x1301, 0xc02b, 0xff01}
	for _, v := range nonGreaseValues {
		if IsGREASE(v) {
			t.Errorf("IsGREASE(%d/0x%04x) 应为 false", v, v)
		}
	}
}
