package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/silenceper/wechat/v2/officialaccount/draft"
	"github.com/silenceper/wechat/v2/util"
	"go.uber.org/zap"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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

func TestValidateRemoteDownloadURLRejectsLocalTargets(t *testing.T) {
	oldLookup := downloadLookupIP
	downloadLookupIP = func(host string) ([]net.IP, error) {
		if host != "internal.example" {
			t.Fatalf("unexpected host lookup: %s", host)
		}
		return []net.IP{net.ParseIP("10.0.0.5")}, nil
	}
	t.Cleanup(func() {
		downloadLookupIP = oldLookup
	})

	cases := []string{
		"http://localhost/image.png",
		"http://127.0.0.1/image.png",
		"http://169.254.169.254/image.png",
		"http://internal.example/image.png",
		"http://example.com:8080/image.png",
	}

	for _, rawURL := range cases {
		parsed, err := neturl.Parse(rawURL)
		if err != nil {
			t.Fatalf("Parse(%q): %v", rawURL, err)
		}
		if err := validateRemoteDownloadURL(parsed); err == nil {
			t.Fatalf("validateRemoteDownloadURL(%q) expected error", rawURL)
		}
	}
}

func TestDownloadFileDownloadsPublicRemoteImages(t *testing.T) {
	oldLookup := downloadLookupIP
	oldFactory := newDownloadHTTPClient
	downloadLookupIP = func(host string) ([]net.IP, error) {
		if host != "example.com" {
			t.Fatalf("unexpected host lookup: %s", host)
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	newDownloadHTTPClient = func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://example.com/path/image.png?size=large" {
					t.Fatalf("unexpected request url: %s", req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("image-bytes")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}
	}
	t.Cleanup(func() {
		downloadLookupIP = oldLookup
		newDownloadHTTPClient = oldFactory
	})

	path, err := DownloadFile("https://example.com/path/image.png?size=large")
	if err != nil {
		t.Fatalf("DownloadFile() error = %v", err)
	}
	defer func() {
		_ = os.Remove(path)
	}()

	if filepath.Ext(path) != ".png" {
		t.Fatalf("download path ext = %q, want .png", filepath.Ext(path))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(data) != "image-bytes" {
		t.Fatalf("downloaded body = %q", string(data))
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

func TestUpdateDraftPostsJSONAndReturnsExistingMediaID(t *testing.T) {
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
			case strings.Contains(req.URL.String(), "/cgi-bin/draft/update"):
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
					Body:       io.NopCloser(strings.NewReader(`{"errcode":0,"errmsg":"ok"}`)),
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

	result, err := svc.UpdateDraft("draft-media-123", 0, &draft.Article{
		Title:        "Updated title",
		Content:      "<p>updated</p>",
		Digest:       "Updated digest",
		ThumbMediaID: "thumb-123",
	})
	if err != nil {
		t.Fatalf("UpdateDraft() error = %v", err)
	}
	if result.MediaID != "draft-media-123" {
		t.Fatalf("media id = %q, want existing media id", result.MediaID)
	}

	var req UpdateDraftRequest
	if err := json.Unmarshal(draftRequestBody, &req); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if req.MediaID != "draft-media-123" || req.Index != 0 {
		t.Fatalf("request identity = %#v", req)
	}
	if req.Articles.Title != "Updated title" || req.Articles.Content != "<p>updated</p>" || req.Articles.ThumbMediaID != "thumb-123" {
		t.Fatalf("request article = %#v", req.Articles)
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
