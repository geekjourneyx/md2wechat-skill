package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

type AtlasCloudProvider struct {
	apiKey       string
	baseURL      string
	model        string
	size         string
	client       *http.Client
	pollInterval time.Duration
	maxPollTime  time.Duration
}

type atlasCloudPrediction struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Status  string   `json:"status"`
	Outputs []string `json:"outputs"`
	Error   string   `json:"error"`
	URLs    struct {
		Get string `json:"get"`
	} `json:"urls"`
}

type atlasCloudResponse struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    atlasCloudPrediction `json:"data"`
}

func NewAtlasCloudProvider(cfg *config.Config) (*AtlasCloudProvider, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.ImageAPIBase), "/")
	if baseURL == "" {
		baseURL = DefaultProviderBaseURL("atlascloud")
	}
	model := strings.TrimSpace(cfg.ImageModel)
	if model == "" {
		model = DefaultProviderModel("atlascloud")
	}
	size := strings.TrimSpace(cfg.ImageSize)
	if size == "" {
		size = "1024x1024"
	}

	return &AtlasCloudProvider{
		apiKey:       cfg.ImageAPIKey,
		baseURL:      baseURL,
		model:        model,
		size:         size,
		client:       &http.Client{Timeout: 30 * time.Second},
		pollInterval: 3 * time.Second,
		maxPollTime:  180 * time.Second,
	}, nil
}

func (p *AtlasCloudProvider) Name() string {
	return "Atlas Cloud"
}

func (p *AtlasCloudProvider) Generate(ctx context.Context, prompt string) (*GenerateResult, error) {
	prediction, err := p.createTask(ctx, prompt)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(prediction.Status, "completed") {
		return p.resultFromPrediction(prediction)
	}

	pollURL, err := p.resolvePollURL(prediction)
	if err != nil {
		return nil, err
	}
	prediction, err = p.pollTask(ctx, pollURL)
	if err != nil {
		return nil, err
	}
	return p.resultFromPrediction(prediction)
}

func (p *AtlasCloudProvider) createTask(ctx context.Context, prompt string) (atlasCloudPrediction, error) {
	body := map[string]any{
		"model":  p.model,
		"prompt": prompt,
		"size":   p.size,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return atlasCloudPrediction{}, p.generateError("marshal_error", "请求构造失败", "", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/generateImage", bytes.NewReader(data))
	if err != nil {
		return atlasCloudPrediction{}, p.generateError("request_error", "创建请求失败", "", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return atlasCloudPrediction{}, p.generateError("network_error", "网络请求失败，请检查网络连接", "确认网络连接正常，API 地址正确", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return atlasCloudPrediction{}, p.handleErrorResponse(resp)
	}

	prediction, err := decodeAtlasCloudResponse(resp.Body)
	if err != nil {
		return atlasCloudPrediction{}, p.generateError("decode_error", "响应解析失败", "", err)
	}
	if prediction.ID == "" && !strings.EqualFold(prediction.Status, "completed") {
		return atlasCloudPrediction{}, p.generateError("no_task_id", "未获取到任务 ID", "请检查 Atlas Cloud API 响应", nil)
	}
	return prediction, nil
}

func (p *AtlasCloudProvider) pollTask(ctx context.Context, pollURL string) (atlasCloudPrediction, error) {
	pollCtx, cancel := context.WithTimeout(ctx, p.maxPollTime)
	defer cancel()
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			code := "canceled"
			message := "操作已取消"
			if ctx.Err() == nil {
				code = "timeout"
				message = fmt.Sprintf("图片生成超时（超过 %v）", p.maxPollTime)
			}
			return atlasCloudPrediction{}, p.generateError(code, message, "请稍后重试", pollCtx.Err())
		case <-ticker.C:
			prediction, err := p.getPrediction(pollCtx, pollURL)
			if err != nil {
				return atlasCloudPrediction{}, err
			}
			switch strings.ToLower(strings.TrimSpace(prediction.Status)) {
			case "completed", "succeeded":
				return prediction, nil
			case "failed", "canceled", "cancelled":
				message := "图片生成任务失败"
				if strings.TrimSpace(prediction.Error) != "" {
					message += ": " + strings.TrimSpace(prediction.Error)
				}
				return atlasCloudPrediction{}, p.generateError("task_failed", message, "请检查提示词和模型配置", nil)
			}
		}
	}
}

func (p *AtlasCloudProvider) getPrediction(ctx context.Context, pollURL string) (atlasCloudPrediction, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return atlasCloudPrediction{}, p.generateError("request_error", "创建轮询请求失败", "", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return atlasCloudPrediction{}, p.generateError("network_error", "查询任务状态失败", "确认网络连接正常", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return atlasCloudPrediction{}, p.handleErrorResponse(resp)
	}
	prediction, err := decodeAtlasCloudResponse(resp.Body)
	if err != nil {
		return atlasCloudPrediction{}, p.generateError("decode_error", "任务状态响应解析失败", "", err)
	}
	return prediction, nil
}

func (p *AtlasCloudProvider) resolvePollURL(prediction atlasCloudPrediction) (string, error) {
	pollURL := strings.TrimSpace(prediction.URLs.Get)
	if pollURL == "" {
		pollURL = p.baseURL + "/prediction/" + url.PathEscape(prediction.ID)
	}

	parsedPollURL, err := url.Parse(pollURL)
	if err != nil {
		return "", p.generateError("invalid_poll_url", "任务轮询地址无效", "", err)
	}
	base, err := url.Parse(p.baseURL)
	if err != nil {
		return "", p.generateError("invalid_base_url", "Atlas Cloud API 地址无效", "", err)
	}
	if !parsedPollURL.IsAbs() {
		parsedPollURL = base.ResolveReference(parsedPollURL)
	}
	if !strings.EqualFold(parsedPollURL.Scheme, base.Scheme) || !strings.EqualFold(parsedPollURL.Host, base.Host) {
		return "", p.generateError("invalid_poll_url", "任务轮询地址不属于已配置的 Atlas Cloud API", "", nil)
	}
	return parsedPollURL.String(), nil
}

func (p *AtlasCloudProvider) resultFromPrediction(prediction atlasCloudPrediction) (*GenerateResult, error) {
	if len(prediction.Outputs) == 0 || strings.TrimSpace(prediction.Outputs[0]) == "" {
		return nil, p.generateError("no_image", "任务已完成但没有返回图片 URL", "请稍后重试或检查模型配置", nil)
	}
	model := strings.TrimSpace(prediction.Model)
	if model == "" {
		model = p.model
	}
	return &GenerateResult{URL: prediction.Outputs[0], Model: model, Size: p.size}, nil
}

func (p *AtlasCloudProvider) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func decodeAtlasCloudResponse(body io.Reader) (atlasCloudPrediction, error) {
	var response atlasCloudResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return atlasCloudPrediction{}, err
	}
	if response.Code != 0 && response.Code != http.StatusOK {
		return atlasCloudPrediction{}, fmt.Errorf("Atlas Cloud API code %d: %s", response.Code, response.Message)
	}
	return response.Data, nil
}

func (p *AtlasCloudProvider) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var response struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &response)
	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = strings.TrimSpace(string(body))
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		return p.generateError("bad_request", "请求参数错误: "+message, ProviderSupportedModelsHint("atlascloud"), fmt.Errorf("status 400"))
	case http.StatusUnauthorized:
		return p.generateError("unauthorized", "Atlas Cloud API Key 无效或已过期", "请检查 api.image_key 或 IMAGE_API_KEY", fmt.Errorf("status 401"))
	case http.StatusPaymentRequired, http.StatusForbidden:
		return p.generateError("payment_required", "Atlas Cloud 账户余额不足或访问受限", "请检查账户余额和模型权限", fmt.Errorf("status %d", resp.StatusCode))
	case http.StatusTooManyRequests:
		return p.generateError("rate_limit", "请求过于频繁，请稍后重试", "Atlas Cloud API 有速率限制", fmt.Errorf("status 429"))
	default:
		return p.generateError("unknown", fmt.Sprintf("Atlas Cloud API 返回错误 (HTTP %d): %s", resp.StatusCode, message), "请稍后重试", fmt.Errorf("status %d", resp.StatusCode))
	}
}

func (p *AtlasCloudProvider) generateError(code, message, hint string, original error) error {
	return &GenerateError{Provider: p.Name(), Code: code, Message: message, Hint: hint, Original: original}
}
