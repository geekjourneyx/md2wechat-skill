package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

// ArkProvider Volcano Engine Ark 图片生成服务提供者
type ArkProvider struct {
	apiKey  string
	baseURL string
	model   string
	size    string
	client  *http.Client
}

// NewArkProvider 创建 Ark Provider
func NewArkProvider(cfg *config.Config) (*ArkProvider, error) {
	model := cfg.ImageModel
	if model == "" {
		model = "doubao-seedream-5.0-lite"
	}

	size := cfg.ImageSize
	if size == "" || size == "1024x1024" {
		size = "2K"
	}

	baseURL := cfg.ImageAPIBase
	if baseURL == "" || baseURL == "https://api.openai.com/v1" || baseURL == "https://api-inference.modelscope.cn" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	return &ArkProvider{
		apiKey:  cfg.ImageAPIKey,
		baseURL: baseURL,
		model:   model,
		size:    size,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Name 返回提供者名称
func (p *ArkProvider) Name() string {
	return "Ark"
}

// Generate 生成图片
func (p *ArkProvider) Generate(ctx context.Context, prompt string) (*GenerateResult, error) {
	reqBody := map[string]any{
		"model":           p.model,
		"prompt":          prompt,
		"n":               1,
		"size":            p.size,
		"response_format": "url",
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

	url := p.baseURL + "/images/generations"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
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
			Message:  "网络请求失败，请检查网络连接",
			Hint:     "确认网络连接正常，API 地址正确",
			Original: err,
		}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp)
	}

	var result struct {
		Data []struct {
			URL           string `json:"url"`
			RevisedPrompt string `json:"revised_prompt,omitempty"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &GenerateError{
			Provider: p.Name(),
			Code:     "decode_error",
			Message:  "响应解析失败",
			Original: err,
		}
	}

	if len(result.Data) == 0 {
		return nil, &GenerateError{
			Provider: p.Name(),
			Code:     "no_image",
			Message:  "未生成图片",
			Hint:     "提示词可能不符合内容政策，请尝试修改提示词",
		}
	}

	return &GenerateResult{
		URL:           result.Data[0].URL,
		RevisedPrompt: result.Data[0].RevisedPrompt,
		Model:         p.model,
		Size:          p.size,
	}, nil
}

// handleErrorResponse 处理错误响应
func (p *ArkProvider) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	_ = json.Unmarshal(body, &errResp)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unauthorized",
			Message:  "Ark API Key 无效或已过期",
			Hint:     "请检查配置文件中的 api.image_key 是否正确，或前往火山引擎 Ark 控制台获取新的 API Key",
			Original: fmt.Errorf("status 401: %s", string(body)),
		}
	case http.StatusTooManyRequests:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "rate_limit",
			Message:  "请求过于频繁，请稍后重试",
			Hint:     "Ark API 有速率限制，请等待一段时间后再试",
			Original: fmt.Errorf("status 429: %s", string(body)),
		}
	case http.StatusBadRequest:
		msg := errResp.Error.Message
		if msg == "" {
			msg = string(body)
		}
		return &GenerateError{
			Provider: p.Name(),
			Code:     "bad_request",
			Message:  fmt.Sprintf("请求参数错误: %s", msg),
			Hint:     "请检查模型名称和 size 参数是否正确。Seedream 5.0 lite 推荐使用 2K、3K 或符合像素范围的 WIDTHxHEIGHT。",
			Original: fmt.Errorf("status 400: %s", string(body)),
		}
	case http.StatusPaymentRequired, http.StatusForbidden:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "payment_required",
			Message:  "Ark 账户余额不足或访问受限",
			Hint:     "请前往火山引擎 Ark 控制台检查账户余额和 API 使用权限",
			Original: fmt.Errorf("status %d: %s", resp.StatusCode, string(body)),
		}
	default:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unknown",
			Message:  fmt.Sprintf("Ark API 返回错误 (HTTP %d)", resp.StatusCode),
			Hint:     "请稍后重试，或访问火山引擎 Ark 控制台查看服务状态",
			Original: fmt.Errorf("status %d: %s", resp.StatusCode, string(body)),
		}
	}
}

// GetArkSupportedModels 返回 Ark 支持的模型列表
func GetArkSupportedModels() []string {
	return []string{
		"doubao-seedream-5.0-lite",
		"doubao-seedream-4.5",
		"doubao-seedream-4.0",
		"doubao-seedream-3.0-t2i",
	}
}
