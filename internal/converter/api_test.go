package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"go.uber.org/zap"
)

func TestResolveAPIConvertURLUsesOneNormalizedEndpoint(t *testing.T) {
	for _, tt := range []struct {
		base string
		want string
	}{
		{"", "https://www.md2wechat.cn/api/convert"},
		{"https://www.md2wechat.cn", "https://www.md2wechat.cn/api/convert"},
		{"https://www.md2wechat.cn/", "https://www.md2wechat.cn/api/convert"},
		{"https://www.md2wechat.cn/api/convert", "https://www.md2wechat.cn/api/convert"},
		{"http://127.0.0.1:3000/", "http://127.0.0.1:3000/api/convert"},
	} {
		if got := ResolveAPIConvertURL(tt.base); got != tt.want {
			t.Fatalf("ResolveAPIConvertURL(%q) = %q, want %q", tt.base, got, tt.want)
		}
	}
}

func TestAPIRequestRejectsMissingAndWhitespaceMarkdownBeforeTransport(t *testing.T) {
	for _, markdown := range []string{"", " ", "\n\t"} {
		conv := &converter{cfg: &config.Config{}}
		if err := conv.validateRequest(&ConvertRequest{Markdown: markdown, Mode: ModeAPI}); err == nil || !strings.Contains(err.Error(), "markdown") {
			t.Fatalf("validateRequest(%q) error = %v, want markdown rejection", markdown, err)
		}
	}
}

func TestPostAPIConvertSerializesSharedRequestAndRetriesOnlyTransientFailures(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Method != http.MethodPost || r.URL.Path != "/api/convert" {
			t.Fatalf("request = %s %s", r.Method, r.URL)
		}
		if got := r.Header.Get("X-API-Key"); got != "secret" {
			t.Fatalf("X-API-Key = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(payload) != `{"markdown":"# title","theme":"default","fontSize":"medium","backgroundType":"none"}` {
			t.Fatalf("payload = %s", payload)
		}
		if attempts < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(successEnvelope("<p>ok</p>")))
	}))
	defer server.Close()

	resp, err := PostAPIConvert(&http.Client{Timeout: time.Second}, server.URL, "secret", APIRequest{
		Markdown: "# title", Theme: "default", FontSize: "medium", BackgroundType: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if _, err := DecodeAPIResponse(resp); err != nil {
		t.Fatal(err)
	}
}

func TestPostAPIConvertReturnsContractFailuresWithoutRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	resp, err := PostAPIConvert(server.Client(), server.URL, "secret", APIRequest{Markdown: "# title", Theme: "default"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestPostAPIConvertRetriesTransportErrorsExactlyTwiceThenFails(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, io.ErrUnexpectedEOF
	})}
	_, err := PostAPIConvert(client, "https://transport.invalid", "secret", APIRequest{Markdown: "# title", Theme: "default"})
	if err == nil || !strings.Contains(err.Error(), "send request") {
		t.Fatalf("error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("transport attempts = %d, want initial request plus two retries", attempts)
	}
}

func TestAPILocalParameterMatrixUsesDiscoveryThemeAndExactSharedFields(t *testing.T) {
	themes := NewThemeManager()
	if err := themes.LoadThemes(); err != nil {
		t.Fatal(err)
	}
	apiThemes := themes.ListAPIThemes()
	if len(apiThemes) == 0 {
		t.Fatal("theme discovery returned no API-selectable theme")
	}
	selected, err := themes.ResolveThemeForMode(ModeAPI, apiThemes[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := themes.ResolveThemeForMode(ModeAPI, "not-a-theme"); err == nil {
		t.Fatal("unknown API theme must be rejected before transport")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request APIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(request.Markdown) == "" || request.Theme != selected.APITheme {
			_, _ = w.Write([]byte(`{"code":422,"msg":"invalid request"}`))
			return
		}
		if request.FontSize == "huge" {
			_, _ = w.Write([]byte(`{"code":422,"msg":"invalid font size"}`))
			return
		}
		font := request.FontSize
		if font == "" {
			font = "medium"
		}
		background := request.BackgroundType
		switch background {
		case "", "default":
			background = "default"
		case "grid", "none":
		default:
			background = "default"
		}
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"html":"<p>ok</p>","theme":%q,"fontSize":%q,"backgroundType":%q,"wordCount":2,"estimatedReadTime":1}}`, request.Theme, font, background)
	}))
	defer server.Close()

	for _, tt := range []struct {
		name, font, background, wantFont, wantBackground string
		wantErr                                          bool
	}{
		{"server defaults", "", "", "medium", "default", false},
		{"small grid", "small", "grid", "small", "grid", false},
		{"medium none", "medium", "none", "medium", "none", false},
		{"large invalid background defaults", "large", "unexpected", "large", "default", false},
		{"huge rejected", "huge", "none", "", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := PostAPIConvert(server.Client(), server.URL, "secret", APIRequest{Markdown: "# title", Theme: selected.APITheme, FontSize: tt.font, BackgroundType: tt.background})
			if err != nil {
				t.Fatal(err)
			}
			data, err := DecodeAPIResponse(resp)
			if tt.wantErr {
				if err == nil {
					t.Fatal("invalid parameter response must be rejected")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if data.Theme != selected.APITheme || data.FontSize != tt.wantFont || data.BackgroundType != tt.wantBackground {
				t.Fatalf("response = %+v", data)
			}
		})
	}
}

func TestAPIConverterDoesNotRetryNonzeroContractResponse(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		_, _ = w.Write([]byte(`{"code":422,"msg":"invalid request"}`))
	}))
	defer server.Close()
	_, err := NewAPIConverterWithURL(zap.NewNop(), server.URL).Convert(&APIRequest{Markdown: "# title", Theme: "default"}, "secret")
	if err == nil || !strings.Contains(err.Error(), "API_ERROR") {
		t.Fatalf("error = %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestDecodeAPIResponseRejectsIncompleteOrMalformedSuccessData(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"missing required field", `{"code":0,"data":{"html":"<p>ok</p>","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":1}}`, "estimatedReadTime"},
		{"wrong required type", `{"code":0,"data":{"html":"<p>ok</p>","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":"1","estimatedReadTime":1}}`, "wordCount"},
		{"empty html", `{"code":0,"data":{"html":" ","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":1,"estimatedReadTime":1}}`, "empty HTML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(tt.body))}
			_, err := DecodeAPIResponse(resp)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodeAPIResponse() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestDecodeAPIResponseRejectsNullAndWrongTypeForEveryRequiredField(t *testing.T) {
	valid := `{"code":0,"data":{"html":"<p>ok</p>","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":1,"estimatedReadTime":1}}`
	tests := []struct {
		name, before, after, want string
	}{
		{"html null", `"html":"<p>ok</p>"`, `"html":null`, "html"},
		{"html number", `"html":"<p>ok</p>"`, `"html":1`, "html"},
		{"theme null", `"theme":"default"`, `"theme":null`, "theme"},
		{"theme number", `"theme":"default"`, `"theme":1`, "theme"},
		{"fontSize null", `"fontSize":"medium"`, `"fontSize":null`, "fontSize"},
		{"fontSize number", `"fontSize":"medium"`, `"fontSize":1`, "fontSize"},
		{"backgroundType null", `"backgroundType":"none"`, `"backgroundType":null`, "backgroundType"},
		{"backgroundType number", `"backgroundType":"none"`, `"backgroundType":1`, "backgroundType"},
		{"wordCount null", `"wordCount":1`, `"wordCount":null`, "wordCount"},
		{"wordCount string", `"wordCount":1`, `"wordCount":"1"`, "wordCount"},
		{"estimatedReadTime null", `"estimatedReadTime":1`, `"estimatedReadTime":null`, "estimatedReadTime"},
		{"estimatedReadTime string", `"estimatedReadTime":1`, `"estimatedReadTime":"1"`, "estimatedReadTime"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.Replace(valid, tt.before, tt.after, 1)
			resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
			if _, err := DecodeAPIResponse(resp); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodeAPIResponse() error = %v, want field %q", err, tt.want)
			}
		})
	}
}

func TestDecodeAPIResponseAcceptsAdditiveFields(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(successEnvelope("<p>ok</p>")))}
	data, err := DecodeAPIResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	if data.HTML != "<p>ok</p>" || data.Theme != "default" || data.FontSize != "medium" || data.BackgroundType != "none" || data.WordCount != 2 || data.EstimatedReadTime != 1 {
		t.Fatalf("data = %+v", data)
	}
}

func TestAPIConverterUsesSharedResponseDecoder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(successEnvelope("<p>ok</p>"))) }))
	defer server.Close()
	html, err := NewAPIConverterWithURL(zap.NewNop(), server.URL).Convert(&APIRequest{Markdown: "# title", Theme: "default"}, "secret")
	if err != nil || html != "<p>ok</p>" {
		t.Fatalf("Convert() = %q, %v", html, err)
	}
}

func successEnvelope(html string) string {
	return `{"code":0,"msg":"ok","data":{"html":"` + html + `","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":2,"estimatedReadTime":1,"future":"allowed"}}`
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
