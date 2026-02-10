package image

import (
	"testing"
)

func TestMapSizeToGeminiAspectRatio(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// 直接使用宽高比
		{"1:1", "1:1"},
		{"16:9", "16:9"},
		{"9:16", "9:16"},
		{"4:3", "4:3"},
		{"3:4", "3:4"},
		{"3:2", "3:2"},
		{"2:3", "2:3"},
		{"4:5", "4:5"},
		{"5:4", "5:4"},
		{"21:9", "21:9"},
		// Gemini 3 Pro 官方尺寸映射（1K）
		{"1024x1024", "1:1"}, // 1K
		{"848x1264", "2:3"},  // 1K
		{"1264x848", "3:2"},  // 1K
		{"896x1200", "3:4"},  // 1K
		{"1200x896", "4:3"},  // 1K
		{"928x1152", "4:5"},  // 1K
		{"1152x928", "5:4"},  // 1K
		{"768x1376", "9:16"}, // 1K
		{"1376x768", "16:9"}, // 1K
		{"1584x672", "21:9"}, // 1K
		// 2K 尺寸
		{"2048x2048", "1:1"},
		{"1696x2528", "2:3"},
		{"2528x1696", "3:2"},
		{"1792x2400", "3:4"},
		{"2400x1792", "4:3"},
		{"1856x2304", "4:5"},
		{"2304x1856", "5:4"},
		{"1536x2752", "9:16"},
		{"2752x1536", "16:9"},
		{"3168x1344", "21:9"},
		// 4K 尺寸
		{"4096x4096", "1:1"},
		{"3392x5056", "2:3"},
		{"5056x3392", "3:2"},
		{"3584x4800", "3:4"},
		{"4800x3584", "4:3"},
		{"3712x4608", "4:5"},
		{"4608x3712", "5:4"},
		{"3072x5504", "9:16"},
		{"5504x3072", "16:9"},
		{"6336x2688", "21:9"},
		// 默认值
		{"", "1:1"},
		{"unknown", "1:1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapSizeToGeminiAspectRatio(tt.input)
			if got != tt.want {
				t.Errorf("mapSizeToGeminiAspectRatio(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetGeminiSupportedModels(t *testing.T) {
	models := GetGeminiSupportedModels()
	if len(models) == 0 {
		t.Error("No supported models returned")
	}

	// 检查默认模型在列表中
	found := false
	for _, m := range models {
		if m == "gemini-3-pro-image-preview" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default model gemini-3-pro-image-preview not in supported list")
	}
}

func TestGetGeminiSupportedAspectRatios(t *testing.T) {
	ratios := GetGeminiSupportedAspectRatios()
	if len(ratios) == 0 {
		t.Error("No supported aspect ratios returned")
	}

	// 检查常用比例
	expected := []string{"1:1", "16:9", "9:16"}
	for _, e := range expected {
		found := false
		for _, r := range ratios {
			if r == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected aspect ratio %s not found", e)
		}
	}
}

// 注意：由于 Gemini SDK 需要真实的 API Key 才能测试，
// 完整的集成测试需要在有 API Key 的环境中运行
// 以下是模拟测试的占位符

func TestGeminiProviderName(t *testing.T) {
	// 创建一个最小的 provider 来测试 Name 方法
	// 注意：实际创建需要有效的 API Key
	p := &GeminiProvider{
		apiKey: "test-key",
		model:  "gemini-3-pro-image-preview",
	}

	if p.Name() != "Gemini" {
		t.Errorf("Name() = %v, want Gemini", p.Name())
	}
}
