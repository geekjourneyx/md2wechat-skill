package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

// officialMiniMaxResponse mirrors the official MiniMax image generation example,
// which encodes metadata.success_count and failed_count as JSON strings.
const officialMiniMaxResponse = `{"data":{"image_urls":["https://cdn.example/result.png"]},"metadata":{"success_count":"1","failed_count":"0"},"base_resp":{"status_code":0,"status_msg":"success"}}`

func newTestMiniMaxProvider(t *testing.T, cfg *config.Config, fn roundTripFunc) *MiniMaxProvider {
	t.Helper()
	provider, err := NewMiniMaxProvider(cfg)
	if err != nil {
		t.Fatalf("NewMiniMaxProvider() error = %v", err)
	}
	provider.client = newMockHTTPClient(fn)
	return provider
}

func miniMaxGenerateError(t *testing.T, err error) *GenerateError {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	var genErr *GenerateError
	if !errors.As(err, &genErr) {
		t.Fatalf("error %v is not a *GenerateError", err)
	}
	return genErr
}

func TestMiniMaxGenerateWithSubject(t *testing.T) {
	provider := newTestMiniMaxProvider(t, &config.Config{
		ImageAPIKey: "image-key", ImageAPIBase: "https://api.minimax.test",
		ImageModel: "image-01", ImageSize: "16:9",
	}, func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/image_generation" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer image-key" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
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
		if payload["response_format"] != "url" {
			t.Fatalf("response_format = %#v", payload["response_format"])
		}
		return jsonResponse(http.StatusOK, officialMiniMaxResponse), nil
	})

	result, err := provider.GenerateWithSubject(context.Background(), "Keep the character", "https://cdn.example/portrait.png")
	if err != nil {
		t.Fatalf("GenerateWithSubject() error = %v", err)
	}
	if result.URL != "https://cdn.example/result.png" || result.Model != "image-01" {
		t.Fatalf("result = %#v", result)
	}
	if result.Size != "16:9" {
		t.Fatalf("result size = %q", result.Size)
	}
}

// TestMiniMaxAcceptsNumericAndStringMetadataCounts guards the official response
// shape, which quotes the counts, as well as the schema shape, which does not.
func TestMiniMaxAcceptsNumericAndStringMetadataCounts(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "official_string_counts", body: officialMiniMaxResponse},
		{name: "numeric_counts", body: `{"data":{"image_urls":["https://cdn.example/result.png"]},"metadata":{"success_count":1,"failed_count":0},"base_resp":{"status_code":0}}`},
		{name: "string_status_code", body: `{"data":{"image_urls":["https://cdn.example/result.png"]},"metadata":{"success_count":"2","failed_count":"0"},"base_resp":{"status_code":"0"}}`},
		{name: "missing_metadata", body: `{"data":{"image_urls":["https://cdn.example/result.png"]},"base_resp":{"status_code":0}}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, testCase.body), nil
			})
			result, err := provider.Generate(context.Background(), "prompt")
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if result.URL != "https://cdn.example/result.png" {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestMiniMaxAppliesDocumentedDefaults(t *testing.T) {
	provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "api.minimax.io" {
			t.Fatalf("host = %q, want the global endpoint", request.URL.Host)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != "image-01" || payload["aspect_ratio"] != "1:1" {
			t.Fatalf("payload = %#v", payload)
		}
		return jsonResponse(http.StatusOK, officialMiniMaxResponse), nil
	})
	if _, err := provider.Generate(context.Background(), "prompt"); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestMiniMaxUsesConfiguredChinaEndpoint(t *testing.T) {
	provider := newTestMiniMaxProvider(t, &config.Config{
		ImageAPIKey: "image-key", ImageAPIBase: "https://api.minimaxi.com/",
	}, func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://api.minimaxi.com/v1/image_generation" {
			t.Fatalf("request URL = %q", request.URL.String())
		}
		return jsonResponse(http.StatusOK, officialMiniMaxResponse), nil
	})
	if _, err := provider.Generate(context.Background(), "prompt"); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestMiniMaxSizeOverrides(t *testing.T) {
	cases := []struct {
		name        string
		size        string
		wantWidth   float64
		wantHeight  float64
		wantAspect  string
		wantErrCode string
	}{
		{name: "aspect_ratio", size: "3:4", wantAspect: "3:4"},
		{name: "explicit_dimensions", size: "1024x768", wantWidth: 1024, wantHeight: 768},
		{name: "invalid_dimensions", size: "widthxheight", wantErrCode: "invalid_size"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := newTestMiniMaxProvider(t, &config.Config{
				ImageAPIKey: "image-key", ImageSize: testCase.size,
			}, func(request *http.Request) (*http.Response, error) {
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if testCase.wantAspect != "" && payload["aspect_ratio"] != testCase.wantAspect {
					t.Fatalf("aspect_ratio = %#v", payload["aspect_ratio"])
				}
				if testCase.wantWidth != 0 {
					if payload["width"] != testCase.wantWidth || payload["height"] != testCase.wantHeight {
						t.Fatalf("width/height = %#v/%#v", payload["width"], payload["height"])
					}
					if _, ok := payload["aspect_ratio"]; ok {
						t.Fatal("aspect_ratio must be omitted when explicit dimensions are used")
					}
				}
				return jsonResponse(http.StatusOK, officialMiniMaxResponse), nil
			})

			_, err := provider.Generate(context.Background(), "prompt")
			if testCase.wantErrCode == "" {
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}
				return
			}
			if got := miniMaxGenerateError(t, err).Code; got != testCase.wantErrCode {
				t.Fatalf("error code = %q, want %q", got, testCase.wantErrCode)
			}
		})
	}
}

func TestMiniMaxModelOverrideIsForwarded(t *testing.T) {
	provider := newTestMiniMaxProvider(t, &config.Config{
		ImageAPIKey: "image-key", ImageModel: "image-01-live",
	}, func(request *http.Request) (*http.Response, error) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != "image-01-live" {
			t.Fatalf("model = %#v", payload["model"])
		}
		return jsonResponse(http.StatusOK, officialMiniMaxResponse), nil
	})
	if _, err := provider.Generate(context.Background(), "prompt"); err != nil {
		t.Fatalf("Generate() error = %v", err)
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

// TestMiniMaxRejectsUnsupportedSubjectReferenceValues keeps the supported
// contract limited to validated public HTTP(S) URLs.
func TestMiniMaxRejectsUnsupportedSubjectReferenceValues(t *testing.T) {
	cases := []struct {
		name      string
		reference string
	}{
		{name: "empty", reference: "   "},
		{name: "data_url", reference: "data:image/jpeg;base64,QUJD"},
		{name: "local_path", reference: "/tmp/portrait.png"},
		{name: "relative_path", reference: "portrait.png"},
		{name: "unsupported_scheme", reference: "ftp://cdn.example/portrait.png"},
		{name: "missing_host", reference: "https://"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(*http.Request) (*http.Response, error) {
				t.Fatal("provider must not send a request for an invalid subject reference")
				return nil, nil
			})
			_, err := provider.GenerateWithSubject(context.Background(), "prompt", testCase.reference)
			if got := miniMaxGenerateError(t, err).Code; got != "invalid_subject_reference" {
				t.Fatalf("error code = %q, want invalid_subject_reference", got)
			}
		})
	}
}

func TestMiniMaxRejectsSubjectReferenceForUnsupportedModel(t *testing.T) {
	provider := newTestMiniMaxProvider(t, &config.Config{
		ImageAPIKey: "image-key", ImageModel: "image-01-live",
	}, func(*http.Request) (*http.Response, error) {
		t.Fatal("provider must not send a request for an unsupported model")
		return nil, nil
	})

	_, err := provider.GenerateWithSubject(context.Background(), "prompt", "https://cdn.example/portrait.png")
	genErr := miniMaxGenerateError(t, err)
	if genErr.Code != "subject_reference_unsupported" {
		t.Fatalf("error code = %q", genErr.Code)
	}
	if !strings.Contains(genErr.Message, "image-01-live") {
		t.Fatalf("error message = %q", genErr.Message)
	}
	if !strings.Contains(genErr.Hint, "image-01") {
		t.Fatalf("error hint = %q", genErr.Hint)
	}
}

// TestMiniMaxMapsAPIStatusCodes covers the documented base_resp status codes
// returned alongside HTTP 200.
func TestMiniMaxMapsAPIStatusCodes(t *testing.T) {
	cases := []struct {
		statusCode int
		wantCode   string
	}{
		{statusCode: 1004, wantCode: "unauthorized"},
		{statusCode: 2049, wantCode: "unauthorized"},
		{statusCode: 1002, wantCode: "rate_limit"},
		{statusCode: 1008, wantCode: "payment_required"},
		{statusCode: 1026, wantCode: "safety_blocked"},
		{statusCode: 1027, wantCode: "safety_blocked"},
		{statusCode: 2013, wantCode: "bad_request"},
		{statusCode: 1039, wantCode: "api_error"},
	}
	for _, testCase := range cases {
		t.Run(testCase.wantCode+"_"+strconv.Itoa(testCase.statusCode), func(t *testing.T) {
			body := `{"base_resp":{"status_code":` + strconv.Itoa(testCase.statusCode) + `,"status_msg":"upstream detail"}}`
			provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, body), nil
			})
			_, err := provider.Generate(context.Background(), "prompt")
			genErr := miniMaxGenerateError(t, err)
			if genErr.Code != testCase.wantCode {
				t.Fatalf("error code = %q, want %q", genErr.Code, testCase.wantCode)
			}
			if !strings.Contains(genErr.Message, "upstream detail") {
				t.Fatalf("error message %q must preserve the upstream status message", genErr.Message)
			}
			if !strings.Contains(genErr.Message, strconv.Itoa(testCase.statusCode)) {
				t.Fatalf("error message %q must preserve the upstream status code", genErr.Message)
			}
		})
	}
}

func TestMiniMaxMapsSensitiveOutputStatus(t *testing.T) {
	provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"base_resp":{"status_code":1027,"status_msg":"sensitive output"}}`), nil
	})
	_, err := provider.Generate(context.Background(), "prompt")
	genErr := miniMaxGenerateError(t, err)
	if genErr.Code != "safety_blocked" {
		t.Fatalf("error code = %q, want safety_blocked", genErr.Code)
	}
	if !strings.Contains(genErr.Message, "generated output") {
		t.Fatalf("error message = %q, want output-specific detail", genErr.Message)
	}
	if !strings.Contains(genErr.Hint, "subject") {
		t.Fatalf("error hint = %q, want subject guidance", genErr.Hint)
	}
}

// TestMiniMaxMapsHTTPErrorStatuses covers non-200 responses, whose body was
// previously discarded.
func TestMiniMaxMapsHTTPErrorStatuses(t *testing.T) {
	cases := []struct {
		httpStatus int
		wantCode   string
	}{
		{httpStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{httpStatus: http.StatusForbidden, wantCode: "unauthorized"},
		{httpStatus: http.StatusPaymentRequired, wantCode: "payment_required"},
		{httpStatus: http.StatusTooManyRequests, wantCode: "rate_limit"},
		{httpStatus: http.StatusBadRequest, wantCode: "bad_request"},
		{httpStatus: http.StatusUnprocessableEntity, wantCode: "bad_request"},
		{httpStatus: http.StatusInternalServerError, wantCode: "server_error"},
		{httpStatus: http.StatusTeapot, wantCode: "http_error"},
	}
	for _, testCase := range cases {
		t.Run(testCase.wantCode+"_"+strconv.Itoa(testCase.httpStatus), func(t *testing.T) {
			provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(*http.Request) (*http.Response, error) {
				return jsonResponse(testCase.httpStatus, `{"detail":"upstream body"}`), nil
			})
			_, err := provider.Generate(context.Background(), "prompt")
			genErr := miniMaxGenerateError(t, err)
			if genErr.Code != testCase.wantCode {
				t.Fatalf("error code = %q, want %q", genErr.Code, testCase.wantCode)
			}
			if genErr.Original == nil || !strings.Contains(genErr.Original.Error(), "upstream body") {
				t.Fatalf("original error must preserve the response body, got %v", genErr.Original)
			}
		})
	}
}

// TestMiniMaxHTTPErrorPrefersBaseResp keeps MiniMax error details even when the
// transport status is not 200.
func TestMiniMaxHTTPErrorPrefersBaseResp(t *testing.T) {
	provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"base_resp":{"status_code":1008,"status_msg":"insufficient balance"}}`), nil
	})
	_, err := provider.Generate(context.Background(), "prompt")
	genErr := miniMaxGenerateError(t, err)
	if genErr.Code != "payment_required" {
		t.Fatalf("error code = %q, want payment_required", genErr.Code)
	}
	if !strings.Contains(genErr.Message, "insufficient balance") {
		t.Fatalf("error message = %q", genErr.Message)
	}
}

func TestMiniMaxRejectsUnusableResponses(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode string
	}{
		{name: "malformed_json", body: `{"data":`, wantCode: "decode_error"},
		{name: "empty_body", body: ``, wantCode: "decode_error"},
		{name: "empty_object", body: `{}`, wantCode: "no_image"},
		{name: "no_image_urls", body: `{"data":{"image_urls":[]},"metadata":{"success_count":"1"},"base_resp":{"status_code":0}}`, wantCode: "no_image"},
		{name: "blank_image_url", body: `{"data":{"image_urls":["  "]},"metadata":{"success_count":"1"},"base_resp":{"status_code":0}}`, wantCode: "no_image"},
		{name: "zero_success_count", body: `{"data":{"image_urls":["https://cdn.example/result.png"]},"metadata":{"success_count":"0","failed_count":"1"},"base_resp":{"status_code":0}}`, wantCode: "no_image"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := newTestMiniMaxProvider(t, &config.Config{ImageAPIKey: "image-key"}, func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, testCase.body), nil
			})
			_, err := provider.Generate(context.Background(), "prompt")
			if got := miniMaxGenerateError(t, err).Code; got != testCase.wantCode {
				t.Fatalf("error code = %q, want %q", got, testCase.wantCode)
			}
		})
	}
}

func TestNewMiniMaxProviderRequiresAPIKey(t *testing.T) {
	if _, err := NewMiniMaxProvider(&config.Config{}); err == nil {
		t.Fatal("expected a missing API key error")
	}
}

func TestNewProviderResolvesMiniMax(t *testing.T) {
	provider, err := NewProvider(&config.Config{ImageProvider: "minimax", ImageAPIKey: "image-key"})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if provider.Name() != "MiniMax" {
		t.Fatalf("provider name = %q", provider.Name())
	}
	if _, ok := provider.(SubjectReferenceProvider); !ok {
		t.Fatal("MiniMax provider must implement SubjectReferenceProvider")
	}
}

func TestSubjectReferenceCapabilityMetadata(t *testing.T) {
	if !ProviderSupportsSubjectReference("minimax") {
		t.Fatal("minimax must advertise subject reference support")
	}
	if ProviderSupportsSubjectReference("openai") {
		t.Fatal("openai must not advertise subject reference support")
	}
	if ProviderSupportsSubjectReference("does-not-exist") {
		t.Fatal("unknown providers must not advertise subject reference support")
	}
	if !ModelSupportsSubjectReference("minimax", "image-01") {
		t.Fatal("image-01 must support subject references")
	}
	if !ModelSupportsSubjectReference("minimax", "") {
		t.Fatal("an empty model must resolve to the default model")
	}
	if ModelSupportsSubjectReference("minimax", "image-01-live") {
		t.Fatal("image-01-live must not support subject references")
	}
	if ModelSupportsSubjectReference("minimax", "unknown-model") {
		t.Fatal("unknown models must not support subject references")
	}
	if ModelSupportsSubjectReference("openai", "gpt-image-2") {
		t.Fatal("openai models must not support subject references")
	}
	if got := SubjectReferenceModelNames("minimax"); len(got) != 1 || got[0] != "image-01" {
		t.Fatalf("SubjectReferenceModelNames() = %#v", got)
	}
	if got := SubjectReferenceProviderNames(); len(got) != 1 || got[0] != "minimax" {
		t.Fatalf("SubjectReferenceProviderNames() = %#v", got)
	}
}

func TestUnknownProviderHintListsMiniMax(t *testing.T) {
	_, err := NewProvider(&config.Config{ImageProvider: "nope", ImageAPIKey: "image-key"})
	if err == nil {
		t.Fatal("expected an unknown provider error")
	}
	var cfgErr *config.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("error %v is not a *config.ConfigError", err)
	}
	if !strings.Contains(cfgErr.Hint, "minimax") {
		t.Fatalf("unknown provider hint = %q, must list minimax", cfgErr.Hint)
	}
}
