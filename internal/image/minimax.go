package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

const (
	miniMaxDefaultBaseURL = "https://api.minimax.io"
	miniMaxDefaultModel   = "image-01"
	miniMaxDefaultSize    = "1:1"
	miniMaxImagePath      = "/v1/image_generation"
	miniMaxSubjectType    = "character"
	miniMaxErrorBodyLimit = 8 << 10
)

// MiniMaxProvider generates images with optional portrait references.
type MiniMaxProvider struct {
	apiKey  string
	baseURL string
	model   string
	size    string
	client  *http.Client
}

func NewMiniMaxProvider(cfg *config.Config) (*MiniMaxProvider, error) {
	if err := validateMiniMaxConfig(cfg); err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.ImageAPIBase), "/")
	if baseURL == "" {
		baseURL = miniMaxDefaultBaseURL
	}
	model := strings.TrimSpace(cfg.ImageModel)
	if model == "" {
		model = miniMaxDefaultModel
	}
	size := strings.TrimSpace(cfg.ImageSize)
	if size == "" {
		size = miniMaxDefaultSize
	}
	return &MiniMaxProvider{
		apiKey: cfg.ImageAPIKey, baseURL: baseURL, model: model, size: size,
		client: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (p *MiniMaxProvider) Name() string { return "MiniMax" }

func (p *MiniMaxProvider) Generate(ctx context.Context, prompt string) (*GenerateResult, error) {
	return p.generate(ctx, prompt, "")
}

// GenerateWithSubject generates an image guided by a portrait reference. The
// reference must be a public HTTP(S) image URL and the configured model must
// accept subject references.
func (p *MiniMaxProvider) GenerateWithSubject(ctx context.Context, prompt, subjectReference string) (*GenerateResult, error) {
	reference, err := p.validateSubjectReference(subjectReference)
	if err != nil {
		return nil, err
	}
	if !ModelSupportsSubjectReference("minimax", p.model) {
		return nil, &GenerateError{
			Provider: p.Name(),
			Code:     "subject_reference_unsupported",
			Message:  fmt.Sprintf("model %s does not support subject references", p.model),
			Hint:     SubjectReferenceModelsHint("minimax"),
		}
	}
	return p.generate(ctx, prompt, reference)
}

// validateSubjectReference keeps the supported contract limited to public
// HTTP(S) image URLs so unusable values fail before the request is sent.
func (p *MiniMaxProvider) validateSubjectReference(subjectReference string) (string, error) {
	reference := strings.TrimSpace(subjectReference)
	if err := ValidateSubjectReferenceURL(reference); err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "invalid_subject_reference",
			Message:  err.Error(),
			Hint:     "Pass a public HTTP(S) URL of a single front-facing portrait in JPG, JPEG, or PNG format",
			Original: err,
		}
	}
	return reference, nil
}

func (p *MiniMaxProvider) generate(ctx context.Context, prompt, subjectReference string) (*GenerateResult, error) {
	payload := map[string]any{
		"model": p.model, "prompt": prompt,
		"response_format": "url", "n": 1,
	}
	if dimensions := strings.Split(p.size, "x"); len(dimensions) == 2 {
		width, widthErr := strconv.Atoi(dimensions[0])
		height, heightErr := strconv.Atoi(dimensions[1])
		if widthErr != nil || heightErr != nil {
			return nil, &GenerateError{Provider: p.Name(), Code: "invalid_size", Message: "image size must be an aspect ratio or WIDTHxHEIGHT"}
		}
		payload["width"] = width
		payload["height"] = height
	} else {
		payload["aspect_ratio"] = p.size
	}
	if subjectReference != "" {
		payload["subject_reference"] = []map[string]string{{"type": miniMaxSubjectType, "image_file": subjectReference}}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, &GenerateError{Provider: p.Name(), Code: "marshal_error", Message: "request encoding failed", Original: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+miniMaxImagePath, bytes.NewReader(body))
	if err != nil {
		return nil, &GenerateError{Provider: p.Name(), Code: "request_error", Message: "request creation failed", Original: err}
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, &GenerateError{Provider: p.Name(), Code: "network_error", Message: "image request failed", Original: err}
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp)
	}
	var response struct {
		Data struct {
			ImageURLs []string `json:"image_urls"`
		} `json:"data"`
		Metadata struct {
			SuccessCount *flexibleInt `json:"success_count"`
			FailedCount  *flexibleInt `json:"failed_count"`
		} `json:"metadata"`
		BaseResp struct {
			StatusCode flexibleInt `json:"status_code"`
			StatusMsg  string      `json:"status_msg"`
		} `json:"base_resp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, &GenerateError{Provider: p.Name(), Code: "decode_error", Message: "response decoding failed", Original: err}
	}
	if code := response.BaseResp.StatusCode.Int(); code != 0 {
		return nil, p.apiError(code, response.BaseResp.StatusMsg, nil)
	}
	if count := response.Metadata.SuccessCount; count != nil && count.Int() < 1 {
		return nil, &GenerateError{
			Provider: p.Name(),
			Code:     "no_image",
			Message:  fmt.Sprintf("provider reported %d successful and %d failed images", count.Int(), response.Metadata.FailedCount.Int()),
			Hint:     "The prompt may violate the content policy; try rewording it",
		}
	}
	if len(response.Data.ImageURLs) == 0 || strings.TrimSpace(response.Data.ImageURLs[0]) == "" {
		return nil, &GenerateError{Provider: p.Name(), Code: "no_image", Message: "response did not include a generated image"}
	}
	return &GenerateResult{URL: response.Data.ImageURLs[0], Model: p.model, Size: p.size}, nil
}

// handleErrorResponse preserves the upstream HTTP status and MiniMax base_resp
// details instead of discarding the response body.
func (p *MiniMaxProvider) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, miniMaxErrorBodyLimit))
	snippet := strings.TrimSpace(string(body))

	var payload struct {
		BaseResp struct {
			StatusCode flexibleInt `json:"status_code"`
			StatusMsg  string      `json:"status_msg"`
		} `json:"base_resp"`
	}
	_ = json.Unmarshal(body, &payload)

	original := fmt.Errorf("status %d: %s", resp.StatusCode, snippet)
	if code := payload.BaseResp.StatusCode.Int(); code != 0 {
		return p.apiError(code, payload.BaseResp.StatusMsg, original)
	}

	message := strings.TrimSpace(payload.BaseResp.StatusMsg)
	if message == "" {
		message = snippet
	}
	if message == "" {
		message = fmt.Sprintf("image request returned HTTP %d", resp.StatusCode)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unauthorized",
			Message:  fmt.Sprintf("MiniMax rejected the API key: %s", message),
			Hint:     "Check api.image_key in the config file or the IMAGE_API_KEY environment variable",
			Original: original,
		}
	case resp.StatusCode == http.StatusPaymentRequired:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "payment_required",
			Message:  fmt.Sprintf("MiniMax account balance is insufficient: %s", message),
			Hint:     "Top up the MiniMax account and retry",
			Original: original,
		}
	case resp.StatusCode == http.StatusTooManyRequests:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "rate_limit",
			Message:  fmt.Sprintf("MiniMax rate limit triggered: %s", message),
			Hint:     "Wait a moment and retry",
			Original: original,
		}
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "bad_request",
			Message:  fmt.Sprintf("MiniMax rejected the request parameters: %s", message),
			Hint:     "Check the model name, aspect ratio or WIDTHxHEIGHT size, and subject reference URL",
			Original: original,
		}
	case resp.StatusCode >= http.StatusInternalServerError:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "server_error",
			Message:  fmt.Sprintf("MiniMax service error: %s", message),
			Hint:     "The upstream service is unavailable; retry later",
			Original: original,
		}
	default:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "http_error",
			Message:  fmt.Sprintf("image request returned HTTP %d: %s", resp.StatusCode, message),
			Original: original,
		}
	}
}

// apiError maps documented MiniMax base_resp status codes to actionable errors
// while keeping the upstream code and message visible.
func (p *MiniMaxProvider) apiError(code int, statusMsg string, original error) error {
	message := strings.TrimSpace(statusMsg)
	withUpstream := func(text string) string {
		if message == "" {
			return fmt.Sprintf("%s (base_resp.status_code=%d)", text, code)
		}
		return fmt.Sprintf("%s (base_resp.status_code=%d: %s)", text, code, message)
	}

	switch code {
	case 1004:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unauthorized",
			Message:  withUpstream("MiniMax account authentication failed"),
			Hint:     "Check api.image_key in the config file or the IMAGE_API_KEY environment variable",
			Original: original,
		}
	case 2049:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unauthorized",
			Message:  withUpstream("MiniMax API key is invalid"),
			Hint:     "Create a new API key in the MiniMax console and update api.image_key",
			Original: original,
		}
	case 1002:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "rate_limit",
			Message:  withUpstream("MiniMax rate limit triggered"),
			Hint:     "Wait a moment and retry",
			Original: original,
		}
	case 1008:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "payment_required",
			Message:  withUpstream("MiniMax account balance is insufficient"),
			Hint:     "Top up the MiniMax account and retry",
			Original: original,
		}
	case 1026:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "safety_blocked",
			Message:  withUpstream("MiniMax flagged sensitive content in the prompt"),
			Hint:     "Reword the prompt to comply with the content policy",
			Original: original,
		}
	case 2013:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "bad_request",
			Message:  withUpstream("MiniMax rejected the request parameters"),
			Hint:     "Check the model name, aspect ratio or WIDTHxHEIGHT size, and subject reference URL",
			Original: original,
		}
	default:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "api_error",
			Message:  withUpstream("MiniMax image generation failed"),
			Original: original,
		}
	}
}

// flexibleInt decodes integers that the MiniMax API may encode either as JSON
// numbers or as quoted strings, which the official response examples do.
type flexibleInt int

func (v *flexibleInt) Int() int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func (v *flexibleInt) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		*v = 0
		return nil
	}
	if strings.HasPrefix(text, `"`) {
		var quoted string
		if err := json.Unmarshal(data, &quoted); err != nil {
			return err
		}
		quoted = strings.TrimSpace(quoted)
		if quoted == "" {
			*v = 0
			return nil
		}
		parsed, err := strconv.Atoi(quoted)
		if err != nil {
			return fmt.Errorf("decode %q as integer: %w", quoted, err)
		}
		*v = flexibleInt(parsed)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}
	parsed, err := number.Int64()
	if err != nil {
		asFloat, floatErr := number.Float64()
		if floatErr != nil {
			return err
		}
		parsed = int64(asFloat)
	}
	*v = flexibleInt(parsed)
	return nil
}
