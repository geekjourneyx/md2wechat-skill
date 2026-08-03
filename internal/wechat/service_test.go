package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
	"github.com/silenceper/wechat/v2/officialaccount/draft"
	"github.com/silenceper/wechat/v2/util"
	"go.uber.org/zap"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewServiceConfiguresWechatHTTPClientProxyAndTimeout(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})

	svc := NewService(&config.Config{
		WechatProxyURL: "http://user:pass@proxy.example.com:8080",
		HTTPTimeout:    45,
	}, zap.NewNop())

	if svc.httpClient == nil {
		t.Fatal("service http client is nil")
	}
	if svc.httpClient.Timeout != 45*time.Second {
		t.Fatalf("timeout = %v, want 45s", svc.httpClient.Timeout)
	}
	if util.DefaultHTTPClient != oldClient {
		t.Fatal("NewService should not permanently replace SDK default HTTP client")
	}
	transport, ok := svc.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", svc.httpClient.Transport)
	}
	req := &http.Request{URL: mustParseURL(t, "https://api.weixin.qq.com/cgi-bin/token")}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport proxy error: %v", err)
	}
	if proxyURL.String() != "http://user:pass@proxy.example.com:8080" {
		t.Fatalf("proxy url = %q", proxyURL.String())
	}
}

func TestNewServiceUsesDefaultTimeoutAndNoCustomTransportWithoutProxy(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})

	svc := NewService(&config.Config{}, zap.NewNop())

	if svc.httpClient == nil {
		t.Fatal("service http client is nil")
	}
	if svc.httpClient.Timeout != 60*time.Second {
		t.Fatalf("timeout = %v, want 60s", svc.httpClient.Timeout)
	}
	if svc.httpClient.Transport != nil {
		t.Fatalf("transport = %T, want nil", svc.httpClient.Transport)
	}
	if util.DefaultHTTPClient != oldClient {
		t.Fatal("NewService should not permanently replace SDK default HTTP client")
	}
}

func TestSDKOperationUsesServiceClientAndRestoresGlobalClient(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	sentinelClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("sentinel client should not be used")
		}),
	}
	util.DefaultHTTPClient = sentinelClient
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})

	calls := 0
	var serviceClient *http.Client
	serviceClient = &http.Client{
		Timeout: 37 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if util.DefaultHTTPClient != serviceClient {
				t.Fatal("SDK default HTTP client was not set to service client during operation")
			}
			if !strings.Contains(req.URL.String(), "/cgi-bin/token") {
				t.Fatalf("unexpected request url: %s", req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-123","expires_in":7200}`)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	svc := &Service{
		cfg: &config.Config{
			WechatAppID:  "appid",
			WechatSecret: "secret",
		},
		log:        zap.NewNop(),
		httpClient: serviceClient,
	}

	result, err := svc.GetAccessToken()
	if err != nil {
		t.Fatalf("GetAccessToken() error = %v", err)
	}
	if result.AccessToken != "token-123" {
		t.Fatalf("access token = %q, want token-123", result.AccessToken)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if util.DefaultHTTPClient != sentinelClient {
		t.Fatal("SDK default HTTP client was not restored after operation")
	}
}

func TestOlderServiceOperationIsNotContaminatedByLaterNewService(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})

	firstClientCalls := 0
	secondClientCalls := 0
	firstClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			firstClientCalls++
			switch {
			case strings.Contains(req.URL.String(), "/cgi-bin/token"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"first-token","expires_in":7200}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			case strings.Contains(req.URL.String(), "/cgi-bin/draft/add"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":0,"errmsg":"ok","media_id":"first-media"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected first client request url: %s", req.URL.String())
				return nil, nil
			}
		}),
	}
	firstSvc := NewService(&config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
	}, zap.NewNop())
	firstSvc.httpClient = firstClient

	secondSvc := NewService(&config.Config{
		WechatProxyURL: "http://second-proxy.example.com:8080",
		HTTPTimeout:    30,
		WechatAppID:    "appid",
		WechatSecret:   "secret",
	}, zap.NewNop())
	secondSvc.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			secondClientCalls++
			return nil, fmt.Errorf("second service client should not be used by first service")
		}),
	}
	util.DefaultHTTPClient = secondSvc.httpClient

	result, err := firstSvc.CreateNewspicDraft([]NewspicArticle{{
		Title:       "Title",
		Content:     "Body",
		ArticleType: "newspic",
		ImageInfo: NewspicImageInfo{
			ImageList: []NewspicImageItem{{ImageMediaID: "media-1"}},
		},
	}})
	if err != nil {
		t.Fatalf("CreateNewspicDraft() error = %v", err)
	}
	if result.MediaID != "first-media" {
		t.Fatalf("media id = %q, want first-media", result.MediaID)
	}
	if firstClientCalls != 2 {
		t.Fatalf("first client calls = %d, want 2", firstClientCalls)
	}
	if secondClientCalls != 0 {
		t.Fatalf("second client calls = %d, want 0", secondClientCalls)
	}
	if util.DefaultHTTPClient != secondSvc.httpClient {
		t.Fatal("SDK default HTTP client was not restored to later service client")
	}
}

func TestCreateNewspicDraftRejectsInvalidProxyBeforeNetwork(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	networkCalls := 0
	util.DefaultHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			networkCalls++
			return nil, fmt.Errorf("unexpected network call to %s", req.URL.String())
		}),
	}
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})

	svc := NewService(&config.Config{
		WechatAppID:    "appid",
		WechatSecret:   "secret",
		WechatProxyURL: "://bad-proxy",
		HTTPTimeout:    30,
	}, zap.NewNop())

	_, err := svc.CreateNewspicDraft([]NewspicArticle{{
		Title:       "Title",
		Content:     "Body",
		ArticleType: "newspic",
		ImageInfo: NewspicImageInfo{
			ImageList: []NewspicImageItem{{ImageMediaID: "media-1"}},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "wechat proxy url") {
		t.Fatalf("CreateNewspicDraft() error = %v, want proxy validation error", err)
	}
	if networkCalls != 0 {
		t.Fatalf("network calls = %d, want 0", networkCalls)
	}
}

func mustParseURL(t *testing.T, rawURL string) *neturl.URL {
	t.Helper()
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		t.Fatalf("Parse(%q): %v", rawURL, err)
	}
	return parsed
}

func TestDownloadFileReturnsLocalPathForExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "cover.png")
	if err := os.WriteFile(localPath, []byte("local"), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	got, err := DownloadFile(localPath)
	if err != nil {
		t.Fatalf("DownloadFile() error = %v", err)
	}
	if got != localPath {
		t.Fatalf("DownloadFile() = %q, want %q", got, localPath)
	}
}

func TestDownloadFileKeepsHTTPStatusPresentation(t *testing.T) {
	err := downloadFileError(&remotefile.Error{
		Kind:       remotefile.ErrorHTTPStatus,
		StatusCode: http.StatusTooManyRequests,
		Err:        fmt.Errorf("upstream status"),
	})
	if err.Error() != "download failed with status: 429" {
		t.Fatalf("download error = %q", err)
	}
}

func TestCreateMultipartFormDataBuildsImagePayload(t *testing.T) {
	contentType, body, boundary := CreateMultipartFormData("media", "cover.png", []byte("pngdata"))
	if contentType == "" || body == nil || boundary == "" {
		t.Fatalf("CreateMultipartFormData() returned empty metadata")
	}
	if !strings.Contains(contentType, "multipart/form-data") {
		t.Fatalf("content type = %q", contentType)
	}
	if !strings.Contains(contentType, boundary) {
		t.Fatalf("boundary %q not embedded in content type %q", boundary, contentType)
	}
	if !strings.Contains(body.String(), "cover.png") || !strings.Contains(body.String(), "pngdata") {
		t.Fatalf("unexpected multipart body: %s", body.String())
	}
}

func TestUploadMaterialFromBytesUsesTemporaryFileExtensionAndDeletesTempFile(t *testing.T) {
	calls := 0
	var uploadedPath string
	svc := &Service{
		log: zap.NewNop(),
		uploadMaterialFunc: func(filePath string) (*UploadMaterialResult, error) {
			calls++
			uploadedPath = filePath
			if filepath.Ext(filePath) != ".png" {
				t.Fatalf("temp file ext = %q, want .png", filepath.Ext(filePath))
			}
			if _, err := os.Stat(filePath); err != nil {
				t.Fatalf("temp file should exist during upload: %v", err)
			}
			return &UploadMaterialResult{MediaID: "media-123", WechatURL: "https://wechat.local/image"}, nil
		},
	}

	result, err := svc.UploadMaterialFromBytes([]byte("pngdata"), "cover.png")
	if err != nil {
		t.Fatalf("UploadMaterialFromBytes() error = %v", err)
	}
	if result == nil || result.MediaID != "media-123" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if calls != 1 {
		t.Fatalf("upload calls = %d, want 1", calls)
	}
	if uploadedPath == "" {
		t.Fatal("upload path not captured")
	}
	if _, err := os.Stat(uploadedPath); !os.IsNotExist(err) {
		t.Fatalf("temp file should be removed after upload, stat err = %v", err)
	}
}

func TestUploadMaterialWithRetryRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	sleeps := 0
	svc := &Service{
		log: zap.NewNop(),
		uploadMaterialFunc: func(filePath string) (*UploadMaterialResult, error) {
			attempts++
			if attempts < 3 {
				return nil, fmt.Errorf("temporary failure %d", attempts)
			}
			return &UploadMaterialResult{MediaID: "media-123", WechatURL: "https://wechat.local/image"}, nil
		},
		sleep: func(d time.Duration) {
			if d != time.Second {
				t.Fatalf("unexpected sleep duration: %v", d)
			}
			sleeps++
		},
	}

	result, err := svc.UploadMaterialWithRetry("/tmp/image.png", 3)
	if err != nil {
		t.Fatalf("UploadMaterialWithRetry() error = %v", err)
	}
	if result == nil || result.MediaID != "media-123" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if sleeps != 2 {
		t.Fatalf("sleeps = %d, want 2", sleeps)
	}
}

func TestUploadMaterialWithRetryReturnsLastError(t *testing.T) {
	attempts := 0
	svc := &Service{
		log: zap.NewNop(),
		uploadMaterialFunc: func(filePath string) (*UploadMaterialResult, error) {
			attempts++
			return nil, fmt.Errorf("permanent failure")
		},
		sleep: func(time.Duration) {},
	}

	result, err := svc.UploadMaterialWithRetry("/tmp/image.png", 2)
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestCreateDraftUsesAccessTokenAndReturnsMediaIDOnly(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	util.DefaultHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.Contains(req.URL.String(), "/cgi-bin/token"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-123","expires_in":7200}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			case strings.Contains(req.URL.String(), "/cgi-bin/draft/add"):
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				if !strings.Contains(string(body), `"title":"Title"`) {
					t.Fatalf("draft payload missing title: %s", string(body))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":0,"errmsg":"ok","media_id":"draft-media-123"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		}),
	}
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})
	httpClient := util.DefaultHTTPClient

	svc := &Service{
		cfg: &config.Config{
			WechatAppID:  "appid",
			WechatSecret: "secret",
		},
		log:        zap.NewNop(),
		httpClient: httpClient,
	}

	result, err := svc.CreateDraft([]*draft.Article{
		{
			Title:   "Title",
			Content: "<p>content</p>",
			Digest:  "Digest",
		},
	})
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if result.MediaID != "draft-media-123" {
		t.Fatalf("media id = %q", result.MediaID)
	}
	if result.DraftURL != "" {
		t.Fatalf("draft url = %q, want empty", result.DraftURL)
	}
}

func TestCreateNewspicDraftPostsJSONAndReturnsMediaID(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	var draftRequestBody []byte
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.Contains(req.URL.String(), "/cgi-bin/token"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-123","expires_in":7200}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			case strings.Contains(req.URL.String(), "/cgi-bin/draft/add"):
				if req.Method != http.MethodPost {
					t.Fatalf("request method = %s, want POST", req.Method)
				}
				if got := req.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("content type = %q, want application/json", got)
				}
				if !strings.Contains(req.URL.String(), "access_token=token-123") {
					t.Fatalf("request url missing token: %s", req.URL.String())
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				draftRequestBody = append([]byte(nil), body...)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":0,"errmsg":"ok","media_id":"news-media-123"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		}),
	}
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})
	util.DefaultHTTPClient = httpClient

	svc := &Service{
		cfg: &config.Config{
			WechatAppID:  "appid",
			WechatSecret: "secret",
		},
		log:        zap.NewNop(),
		httpClient: httpClient,
	}

	result, err := svc.CreateNewspicDraft([]NewspicArticle{
		{
			Title:       "Title",
			Content:     "Body",
			ArticleType: "newspic",
			ImageInfo: NewspicImageInfo{
				ImageList: []NewspicImageItem{
					{ImageMediaID: "media-1"},
					{ImageMediaID: "media-2"},
				},
			},
			NeedOpenComment:    1,
			OnlyFansCanComment: 1,
		},
	})
	if err != nil {
		t.Fatalf("CreateNewspicDraft() error = %v", err)
	}
	if result.MediaID != "news-media-123" {
		t.Fatalf("media id = %q, want %q", result.MediaID, "news-media-123")
	}
	if result.DraftURL != "" {
		t.Fatalf("draft url = %q, want empty", result.DraftURL)
	}

	var req NewspicDraftRequest
	if err := json.Unmarshal(draftRequestBody, &req); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if len(req.Articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(req.Articles))
	}
	article := req.Articles[0]
	if article.Title != "Title" || article.Content != "Body" || article.ArticleType != "newspic" {
		t.Fatalf("article = %#v", article)
	}
	if article.NeedOpenComment != 1 || article.OnlyFansCanComment != 1 {
		t.Fatalf("comment flags = %#v", article)
	}
	if len(article.ImageInfo.ImageList) != 2 || article.ImageInfo.ImageList[0].ImageMediaID != "media-1" {
		t.Fatalf("image info = %#v", article.ImageInfo)
	}
}

func TestCreateNewspicDraftSurfacesWechatApiErrors(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.Contains(req.URL.String(), "/cgi-bin/token"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-123","expires_in":7200}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			case strings.Contains(req.URL.String(), "/cgi-bin/draft/add"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":40013,"errmsg":"invalid credential"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		}),
	}
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})
	util.DefaultHTTPClient = httpClient

	svc := &Service{
		cfg: &config.Config{
			WechatAppID:  "appid",
			WechatSecret: "secret",
		},
		log:        zap.NewNop(),
		httpClient: httpClient,
	}

	_, err := svc.CreateNewspicDraft([]NewspicArticle{
		{
			Title:       "Title",
			Content:     "Body",
			ArticleType: "newspic",
			ImageInfo: NewspicImageInfo{
				ImageList: []NewspicImageItem{
					{ImageMediaID: "media-1"},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "wechat api error") {
		t.Fatalf("CreateNewspicDraft() error = %v", err)
	}
}

func TestCreateNewspicDraftExplainsKnownWeChatLimitErrors(t *testing.T) {
	oldClient := util.DefaultHTTPClient
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.Contains(req.URL.String(), "/cgi-bin/token"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-123","expires_in":7200}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			case strings.Contains(req.URL.String(), "/cgi-bin/draft/add"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":45004,"errmsg":"description size out of limit"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected request url: %s", req.URL.String())
				return nil, nil
			}
		}),
	}
	t.Cleanup(func() {
		util.DefaultHTTPClient = oldClient
	})
	util.DefaultHTTPClient = httpClient

	svc := &Service{
		cfg: &config.Config{
			WechatAppID:  "appid",
			WechatSecret: "secret",
		},
		log:        zap.NewNop(),
		httpClient: httpClient,
	}

	_, err := svc.CreateNewspicDraft([]NewspicArticle{{
		Title:       "Title",
		Content:     "Body",
		ArticleType: "newspic",
		ImageInfo: NewspicImageInfo{
			ImageList: []NewspicImageItem{{ImageMediaID: "media-1"}},
		},
	}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "description size out of limit") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "shorten --digest") {
		t.Fatalf("error missing digest hint: %v", err)
	}
}

func TestExplainDraftErrorAddsHintsForKnownCodes(t *testing.T) {
	err := ExplainDraftError(fmt.Errorf("wechat api error: 45003 - title size out of limit"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "32 characters or fewer") {
		t.Fatalf("error = %v", err)
	}

	unknown := ExplainDraftError(fmt.Errorf("wechat api error: 40013 - invalid credential"))
	if unknown.Error() != "wechat api error: 40013 - invalid credential" {
		t.Fatalf("unexpected error rewrite: %v", unknown)
	}
}
