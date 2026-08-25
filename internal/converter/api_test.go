package converter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	defer resp.Body.Close()
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
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
