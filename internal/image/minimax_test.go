package image

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestMiniMaxGenerateWithSubject(t *testing.T) {
	provider, err := NewMiniMaxProvider(&config.Config{
		ImageAPIKey: "image-key", ImageAPIBase: "https://api.minimax.test",
		ImageModel: "image-01", ImageSize: "16:9",
	})
	if err != nil {
		t.Fatalf("NewMiniMaxProvider() error = %v", err)
	}
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/image_generation" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer image-key" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		references, ok := payload["subject_reference"].([]any)
		if !ok || len(references) != 1 {
			t.Fatalf("subject_reference = %#v", payload["subject_reference"])
		}
		reference := references[0].(map[string]any)
		if reference["type"] != "character" || reference["image_file"] != "https://cdn.example/portrait.png" {
			t.Fatalf("subject reference = %#v", reference)
		}
		if payload["aspect_ratio"] != "16:9" || payload["model"] != "image-01" {
			t.Fatalf("payload = %#v", payload)
		}
		return jsonResponse(http.StatusOK, `{"data":{"image_urls":["https://cdn.example/result.png"]},"metadata":{"success_count":1,"failed_count":0},"base_resp":{"status_code":0,"status_msg":"success"}}`), nil
	})}

	result, err := provider.GenerateWithSubject(context.Background(), "Keep the character", "https://cdn.example/portrait.png")
	if err != nil {
		t.Fatalf("GenerateWithSubject() error = %v", err)
	}
	if result.URL != "https://cdn.example/result.png" || result.Model != "image-01" {
		t.Fatalf("result = %#v", result)
	}
}

func TestMiniMaxGenerateRejectsMissingSubject(t *testing.T) {
	provider, err := NewMiniMaxProvider(&config.Config{ImageAPIKey: "image-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.GenerateWithSubject(context.Background(), "prompt", " "); err == nil {
		t.Fatal("expected missing subject reference error")
	}
}
