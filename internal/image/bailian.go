package image

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"strings"
"time"

"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

// BailianProvider 阿里云百炼 (DashScope) qwen-image 提供者
type BailianProvider struct {
apiKey  string
baseURL string
model   string
size    string
client  *http.Client
}

// NewBailianProvider 创建 Bailian Provider
func NewBailianProvider(cfg *config.Config) (*BailianProvider, error) {
model := cfg.ImageModel
if model == "" {
model = "qwen-image-2.0" // 推荐的默认模型
}

size := cfg.ImageSize
if size == "" {
// qwen-image-2.0 默认推荐 2048*2048, 其他老模型如 max/plus 默认 1664*928
if strings.Contains(model, "2.0") {
size = "2048*2048"
} else {
size = "1664*928"
}
}
// 将常见的分辨率x替换为百炼期望的 * 
size = strings.ReplaceAll(size, "x", "*")

baseURL := cfg.ImageAPIBase
if baseURL == "" {
// 官方推荐使用的同步请求接口
baseURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
}

return &BailianProvider{
apiKey:  cfg.ImageAPIKey,
baseURL: baseURL,
model:   model,
size:    size,
client: &http.Client{
// 图像生成耗时较长，需要相对充裕的超时设置
Timeout: 120 * time.Second,
},
}, nil
}

// Name 返回提供者名称
func (p *BailianProvider) Name() string {
return "Aliyun Bailian"
}

// Generate 生成图片
func (p *BailianProvider) Generate(ctx context.Context, prompt string) (*GenerateResult, error) {
// 构造百炼 Qwen-Image 同步请求格式
reqBody := map[string]any{
"model": p.model,
"input": map[string]any{
"messages": []map[string]any{
{
"role": "user",
"content": []map[string]any{
{
"text": prompt,
},
},
},
},
},
"parameters": map[string]any{
"size":          p.size,
// 开启 prompt 智能改写功能，丰富生成图像的细节
"prompt_extend": true,
"watermark":     false,
},
}

jsonData, err := json.Marshal(reqBody)
if err != nil {
return nil, &GenerateError{
Provider: p.Name(),
Code:     "marshal_error",
Message:  "请求构造失败",
Original: err,
}
}

req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
if err != nil {
return nil, &GenerateError{
Provider: p.Name(),
Code:     "request_error",
Message:  "创建请求失败",
Original: err,
}
}

req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+p.apiKey)

resp, err := p.client.Do(req)
if err != nil {
return nil, &GenerateError{
Provider: p.Name(),
Code:     "network_error",
Message:  "网络请求失败",
Original: err,
}
}
defer resp.Body.Close()

body, _ := io.ReadAll(resp.Body)

// 处理同步响应
var result struct {
Output struct {
Choices []struct {
Message struct {
Content []struct {
Image string `json:"image"`
} `json:"content"`
} `json:"message"`
} `json:"choices"`
} `json:"output"`
Code    string `json:"code"`
Message string `json:"message"`
}

if err := json.Unmarshal(body, &result); err != nil {
return nil, &GenerateError{
Provider: p.Name(),
Code:     "decode_error",
Message:  "响应解析失败",
Original: err,
}
}

if result.Code != "" || result.Message != "" {
return nil, &GenerateError{
Provider: p.Name(),
Code:     result.Code,
Message:  result.Message,
Original: fmt.Errorf("api error: %s", string(body)),
}
}

if len(result.Output.Choices) > 0 && len(result.Output.Choices[0].Message.Content) > 0 {
return &GenerateResult{
URL:   result.Output.Choices[0].Message.Content[0].Image,
Model: p.model,
Size:  p.size,
}, nil
}

return nil, &GenerateError{
Provider: p.Name(),
Code:     "no_image",
Message:  "未生成图片",
}
}
