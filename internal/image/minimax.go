package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
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
		baseURL = "https://api.minimax.io"
	}
	model := strings.TrimSpace(cfg.ImageModel)
	if model == "" {
		model = "image-01"
	}
	size := strings.TrimSpace(cfg.ImageSize)
	if size == "" {
		size = "1:1"
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

func (p *MiniMaxProvider) GenerateWithSubject(ctx context.Context, prompt, subjectReference string) (*GenerateResult, error) {
	if strings.TrimSpace(subjectReference) == "" {
		return nil, &GenerateError{Provider: p.Name(), Code: "invalid_subject_reference", Message: "subject reference is required"}
	}
	return p.generate(ctx, prompt, subjectReference)
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
		payload["subject_reference"] = []map[string]string{{"type": "character", "image_file": subjectReference}}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, &GenerateError{Provider: p.Name(), Code: "marshal_error", Message: "request encoding failed", Original: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/image_generation", bytes.NewReader(body))
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
		return nil, &GenerateError{Provider: p.Name(), Code: "http_error", Message: fmt.Sprintf("image request returned HTTP %d", resp.StatusCode)}
	}
	var response struct {
		Data struct {
			ImageURLs []string `json:"image_urls"`
		} `json:"data"`
		Metadata struct {
			SuccessCount int `json:"success_count"`
			FailedCount  int `json:"failed_count"`
		} `json:"metadata"`
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, &GenerateError{Provider: p.Name(), Code: "decode_error", Message: "response decoding failed", Original: err}
	}
	if response.BaseResp.StatusCode != 0 {
		return nil, &GenerateError{Provider: p.Name(), Code: "api_error", Message: response.BaseResp.StatusMsg}
	}
	if response.Metadata.SuccessCount < 1 || len(response.Data.ImageURLs) == 0 || strings.TrimSpace(response.Data.ImageURLs[0]) == "" {
		return nil, &GenerateError{Provider: p.Name(), Code: "no_image", Message: "response did not include a generated image"}
	}
	return &GenerateResult{URL: response.Data.ImageURLs[0], Model: p.model, Size: p.size}, nil
}
