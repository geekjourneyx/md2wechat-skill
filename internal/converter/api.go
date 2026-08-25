package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
	"go.uber.org/zap"
)

const (
	// DefaultAPIConvertURL is the production endpoint used by API-mode convert
	// and opt-in conformance checks.
	DefaultAPIConvertURL = "https://www.md2wechat.cn/api/convert"
	apiConvertMaxRetries = 2
)

// APIResponse md2wechat.cn API 响应
type APIResponse struct {
	Code int             `json:"code"` // 0 表示成功
	Msg  string          `json:"msg"`  // 错误信息
	Data APIResponseData `json:"data"`
}

// APIResponseData is the stable successful conversion result. Consumers that
// only need HTML may ignore the remaining server-provided metadata.
type APIResponseData struct {
	HTML              string `json:"html"`
	Theme             string `json:"theme"`
	FontSize          string `json:"fontSize"`
	BackgroundType    string `json:"backgroundType"`
	WordCount         int    `json:"wordCount"`
	EstimatedReadTime int    `json:"estimatedReadTime"`
}

// APIRequest md2wechat.cn API 请求
type APIRequest struct {
	Markdown       string `json:"markdown"`
	Theme          string `json:"theme"`
	FontSize       string `json:"fontSize,omitempty"`
	BackgroundType string `json:"backgroundType,omitempty"` // default/grid/none
}

// apiConverter API 模式转换器
type apiConverter struct {
	log     *zap.Logger
	baseURL string
	timeout time.Duration
}

// NewAPIConverter 创建 API 转换器
func NewAPIConverter(log *zap.Logger) *apiConverter {
	return &apiConverter{
		log:     log,
		baseURL: DefaultAPIConvertURL,
		timeout: 30 * time.Second,
	}
}

// NewAPIConverterWithURL 创建 API 转换器（指定 URL）
func NewAPIConverterWithURL(log *zap.Logger, baseURL string) *apiConverter {
	return &apiConverter{
		log:     log,
		baseURL: ResolveAPIConvertURL(baseURL),
		timeout: 30 * time.Second,
	}
}

// convertViaAPI 通过 API 执行转换
func (c *converter) convertViaAPI(req *ConvertRequest) *ConvertResult {
	result := &ConvertResult{
		Mode:      ModeAPI,
		Theme:     req.Theme,
		Status:    action.StatusFailed,
		Action:    action.ActionConvert,
		Retryable: true,
		Success:   false,
	}

	theme, err := c.theme.ResolveThemeForMode(ModeAPI, req.Theme)
	if err != nil {
		result.Error = err.Error()
		result.Retryable = false
		return result
	}

	// 创建 API 转换器，传入配置中的 base URL
	apiConv := NewAPIConverterWithURL(c.log, ResolveAPIConvertURL(c.cfg.MD2WechatBaseURL))

	// 调用 API
	html, err := apiConv.Convert(&APIRequest{
		Markdown:       req.Markdown,
		Theme:          theme.APITheme,
		FontSize:       req.FontSize,
		BackgroundType: req.BackgroundType,
	}, req.APIKey)

	if err != nil {
		result.Error = fmt.Sprintf("API call failed: %s", err.Error())
		c.log.Error("API conversion failed",
			zap.String("theme", req.Theme),
			zap.Error(err))
		return result
	}

	// 提取图片引用
	images := c.ExtractImages(req.Markdown)

	result.HTML = html
	result.Images = images
	result.Status = action.StatusCompleted
	result.Retryable = false
	result.Success = true

	c.log.Info("API conversion succeeded",
		zap.String("theme", req.Theme),
		zap.Int("image_count", len(images)))

	return result
}

// Convert 调用 md2wechat.cn API 进行转换
func (a *apiConverter) Convert(req *APIRequest, apiKey string) (string, error) {
	client := &http.Client{
		Timeout: a.timeout,
	}
	resp, err := PostAPIConvert(client, a.baseURL, apiKey, *req)
	if err != nil {
		return "", err
	}
	data, err := DecodeAPIResponse(resp)
	if err != nil {
		return "", err
	}
	return data.HTML, nil
}

// ResolveAPIConvertURL normalizes a configured base URL to the sole convert
// endpoint. It deliberately accepts a full endpoint to preserve existing
// config values.
func ResolveAPIConvertURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return DefaultAPIConvertURL
	}
	if strings.HasSuffix(baseURL, "/api/convert") {
		return baseURL
	}
	return baseURL + "/api/convert"
}

// PostAPIConvert sends the shared request envelope. Only transport failures
// and 5xx responses are retried; contract and authorization responses are
// returned directly for the caller to decode and categorize.
func PostAPIConvert(client *http.Client, baseURL, apiKey string, payload APIRequest) (*http.Response, error) {
	if client == nil {
		return nil, fmt.Errorf("send request: nil HTTP client")
	}
	for attempt := 0; ; attempt++ {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		req, err := http.NewRequest(http.MethodPost, ResolveAPIConvertURL(baseURL), bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("X-API-Key", apiKey)
		}
		resp, err := client.Do(req)
		if err == nil && (resp.StatusCode < http.StatusInternalServerError || attempt == apiConvertMaxRetries) {
			return resp, nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		if attempt == apiConvertMaxRetries {
			return nil, fmt.Errorf("send request: %w", err)
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
}

// DecodeAPIResponse validates the successful conversion envelope while
// accepting additive server fields. It always closes resp.Body.
func DecodeAPIResponse(resp *http.Response) (APIResponseData, error) {
	if resp == nil {
		return APIResponseData{}, fmt.Errorf("nil response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return APIResponseData{}, fmt.Errorf("http status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return APIResponseData{}, fmt.Errorf("read response: %w", err)
	}
	var raw struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return APIResponseData{}, fmt.Errorf("decode envelope: %w", err)
	}
	if raw.Code != 0 {
		return APIResponseData{}, &ConvertError{Code: "API_ERROR", Message: fmt.Sprintf("API returned error code %d: %s", raw.Code, raw.Msg)}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw.Data, &fields); err != nil {
		return APIResponseData{}, fmt.Errorf("parse response data: %w", err)
	}
	for _, name := range []string{"html", "theme", "fontSize", "backgroundType", "wordCount", "estimatedReadTime"} {
		if _, ok := fields[name]; !ok {
			return APIResponseData{}, fmt.Errorf("parse response data: missing required field %s", name)
		}
	}
	var data APIResponseData
	if err := json.Unmarshal(raw.Data, &data); err != nil {
		return APIResponseData{}, fmt.Errorf("parse response data: %w", err)
	}
	if strings.TrimSpace(data.HTML) == "" {
		return APIResponseData{}, fmt.Errorf("API returned empty HTML")
	}
	return data, nil
}

// SetBaseURL 设置 API 基础 URL（用于测试）
func (a *apiConverter) SetBaseURL(url string) {
	a.baseURL = ResolveAPIConvertURL(url)
}

// SetTimeout 设置请求超时
func (a *apiConverter) SetTimeout(timeout time.Duration) {
	a.timeout = timeout
}
