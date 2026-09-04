package converter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"go.uber.org/zap"
)

func TestExtractImagesPreservesDocumentOrder(t *testing.T) {
	conv := NewConverter(&config.Config{}, zap.NewNop())
	markdown := strings.Join([]string{
		"![local](images/a.png)",
		"![online](https://example.com/b.png)",
		"![ai](__generate:draw a cat__)",
	}, "\n")

	images := conv.ExtractImages(markdown)
	if len(images) != 3 {
		t.Fatalf("expected 3 images, got %d", len(images))
	}

	if images[0].Type != ImageTypeLocal || images[0].Original != "images/a.png" {
		t.Fatalf("unexpected first image: %+v", images[0])
	}
	if images[1].Type != ImageTypeOnline || images[1].Original != "https://example.com/b.png" {
		t.Fatalf("unexpected second image: %+v", images[1])
	}
	if images[2].Type != ImageTypeAI || images[2].AIPrompt != "draw a cat" {
		t.Fatalf("unexpected third image: %+v", images[2])
	}
}

func TestParseMarkdownImagesSupportsLocalVariantsAndTitles(t *testing.T) {
	markdown := strings.Join([]string{
		"![relative](./a.png \"cover\")",
		"![nested](images/b.png)",
		"![parent](../c.png)",
		"![absolute](/tmp/d.png)",
		"![angle](<images/my cat.png>)",
	}, "\n")

	images := ParseMarkdownImages(markdown)
	if len(images) != 5 {
		t.Fatalf("expected 5 images, got %d", len(images))
	}

	want := []string{"./a.png", "images/b.png", "../c.png", "/tmp/d.png", "images/my cat.png"}
	for i, expected := range want {
		if images[i].Type != ImageTypeLocal {
			t.Fatalf("image %d expected local type, got %+v", i, images[i])
		}
		if images[i].Original != expected {
			t.Fatalf("image %d original = %q, want %q", i, images[i].Original, expected)
		}
		if images[i].Placeholder == "" {
			t.Fatalf("image %d missing placeholder", i)
		}
	}
}

func TestParseArticleMetadataPrefersFrontMatter(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter Title",
		"author: Jane Doe",
		"summary: Frontmatter summary",
		"---",
		"",
		"# Heading Title",
		"",
		"body",
	}, "\n")

	meta := ParseArticleMetadata(markdown)
	if meta.Title != "Frontmatter Title" {
		t.Fatalf("title = %q", meta.Title)
	}
	if meta.Author != "Jane Doe" {
		t.Fatalf("author = %q", meta.Author)
	}
	if meta.Digest != "Frontmatter summary" {
		t.Fatalf("digest = %q", meta.Digest)
	}
}

func TestParseArticleDocumentStripsFrontMatterFromBody(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter Title",
		"author: Jane Doe",
		"summary: Frontmatter summary",
		"---",
		"",
		"# Heading Title",
		"",
		"body",
	}, "\n")

	doc := ParseArticleDocument(markdown)
	if doc.Metadata.Title != "Frontmatter Title" {
		t.Fatalf("title = %q", doc.Metadata.Title)
	}
	if doc.Metadata.Author != "Jane Doe" {
		t.Fatalf("author = %q", doc.Metadata.Author)
	}
	if doc.Metadata.Digest != "Frontmatter summary" {
		t.Fatalf("digest = %q", doc.Metadata.Digest)
	}
	if strings.Contains(doc.Body, "title: Frontmatter Title") {
		t.Fatalf("body still contains frontmatter: %q", doc.Body)
	}
	if !strings.HasPrefix(doc.Body, "\n# Heading Title") && !strings.HasPrefix(doc.Body, "# Heading Title") {
		t.Fatalf("unexpected body = %q", doc.Body)
	}
}

func TestParseArticleMetadataFallsBackToMarkdownTitle(t *testing.T) {
	markdown := "# Heading Title\n\nbody"

	meta := ParseArticleMetadata(markdown)
	if meta.Title != "Heading Title" {
		t.Fatalf("title = %q", meta.Title)
	}
	if meta.Author != "" || meta.Digest != "" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestParseArticleMetadataDoesNotUseFirstBodyLineAsTitle(t *testing.T) {
	markdown := "This is body text without a heading.\n\nMore body."

	meta := ParseArticleMetadata(markdown)
	if meta.Title != "未命名文章" {
		t.Fatalf("title = %q", meta.Title)
	}
}

func TestParseArticleMetadataFallsBackToBodyTitleWhenFrontMatterHasNoTitle(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"author: Jane Doe",
		"summary: Frontmatter summary",
		"---",
		"",
		"# Heading Title",
		"",
		"body",
	}, "\n")

	meta := ParseArticleMetadata(markdown)
	if meta.Title != "Heading Title" {
		t.Fatalf("title = %q", meta.Title)
	}
	if meta.Author != "Jane Doe" {
		t.Fatalf("author = %q", meta.Author)
	}
	if meta.Digest != "Frontmatter summary" {
		t.Fatalf("digest = %q", meta.Digest)
	}
}

func TestParseArticleMetadataSupportsCRLFFrontMatter(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"summary: Windows newline summary",
		"---",
		"",
		"# Heading Title",
		"",
		"body",
	}, "\r\n")

	meta := ParseArticleMetadata(markdown)
	if meta.Title != "Heading Title" {
		t.Fatalf("title = %q", meta.Title)
	}
	if meta.Digest != "Windows newline summary" {
		t.Fatalf("digest = %q", meta.Digest)
	}
}

func TestConvertReturnsValidationErrors(t *testing.T) {
	conv := &converter{
		cfg:           &config.Config{},
		log:           zap.NewNop(),
		theme:         NewThemeManager(),
		promptBuilder: NewPromptBuilder(),
	}

	result := conv.Convert(&ConvertRequest{Markdown: "", Mode: ModeAPI})
	if result.Success || !strings.Contains(result.Error, ErrEmptyMarkdown.Error()) {
		t.Fatalf("unexpected empty markdown result: %+v", result)
	}

	result = conv.Convert(&ConvertRequest{Markdown: "# title", Mode: ModeAPI})
	if result.Success || !strings.Contains(result.Error, ErrMissingAPIKey.Error()) {
		t.Fatalf("unexpected missing key result: %+v", result)
	}
}

func TestAIRequestHelpersExposePreparedPrompt(t *testing.T) {
	result := &ConvertResult{
		Status: action.StatusActionRequired,
		Action: action.ActionConvert,
		Prompt: "prompt body",
		Images: []ImageRef{{Index: 0, Original: "./a.png"}},
	}

	if !IsAIRequest(result) {
		t.Fatal("expected AI request result")
	}
	if ExtractAIRequest(result) != "prompt body" {
		t.Fatalf("ExtractAIRequest() = %q", ExtractAIRequest(result))
	}
	prompt, images, ok := GetAIRequestInfo(result)
	if !ok || prompt != "prompt body" || len(images) != 1 {
		t.Fatalf("GetAIRequestInfo() = (%q, %#v, %v)", prompt, images, ok)
	}
	if result.Status != action.StatusActionRequired || result.Action != action.ActionConvert {
		t.Fatalf("unexpected typed state: %+v", result)
	}
}

func TestBuildAIPromptPrefersResolvedMetadataTitle(t *testing.T) {
	tm := NewThemeManager()
	tm.themes["ai-test"] = Theme{
		Name:   "ai-test",
		Type:   "ai",
		Prompt: "Title: {{TITLE}}\n\n{{MARKDOWN}}",
	}
	conv := &converter{
		cfg:           &config.Config{},
		log:           zap.NewNop(),
		theme:         tm,
		promptBuilder: NewPromptBuilder(),
	}

	prompt, err := conv.buildAIPrompt(&ConvertRequest{
		Markdown: strings.Join([]string{
			"---",
			"title: Frontmatter 标题",
			"---",
			"",
			"# 正文标题",
			"",
			"正文",
		}, "\n"),
		Metadata: ArticleMetadata{
			Title: "命令行标题",
		},
		Mode:  ModeAI,
		Theme: "ai-test",
	})
	if err != nil {
		t.Fatalf("buildAIPrompt() error = %v", err)
	}
	if !strings.Contains(prompt, "命令行标题") {
		t.Fatalf("prompt missing override title: %q", prompt)
	}
	if !strings.Contains(prompt, "Title: 命令行标题") {
		t.Fatalf("prompt did not apply override title variable: %q", prompt)
	}
	if strings.Contains(prompt, "title: Frontmatter 标题") {
		t.Fatalf("prompt should not include raw frontmatter: %q", prompt)
	}
}

func TestPrepareAIRequestStripsFrontMatterFromMarkdown(t *testing.T) {
	tm := NewThemeManager()
	tm.themes["ai-test"] = Theme{
		Name:   "ai-test",
		Type:   "ai",
		Prompt: "Body:\n\n{{MARKDOWN}}",
	}
	conv := &converter{
		cfg:           &config.Config{},
		log:           zap.NewNop(),
		theme:         tm,
		promptBuilder: NewPromptBuilder(),
	}

	req, err := conv.PrepareAIRequest(&ConvertRequest{
		Markdown: strings.Join([]string{
			"---",
			"title: Frontmatter 标题",
			"author: Jane Doe",
			"---",
			"",
			"# 正文标题",
			"",
			"正文",
		}, "\n"),
		Mode:  ModeAI,
		Theme: "ai-test",
	})
	if err != nil {
		t.Fatalf("PrepareAIRequest() error = %v", err)
	}
	if strings.Contains(req.Markdown, "title: Frontmatter 标题") {
		t.Fatalf("prepared markdown should not contain frontmatter: %q", req.Markdown)
	}
	if !strings.Contains(req.Markdown, "# 正文标题") {
		t.Fatalf("prepared markdown missing body content: %q", req.Markdown)
	}
}

func TestConvertViaAPIStripsFrontMatterBeforeSendingMarkdown(t *testing.T) {
	var received APIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"html":"<p>ok</p>","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":2,"estimatedReadTime":1}}`))
	}))
	defer server.Close()

	conv := &converter{
		cfg: &config.Config{
			MD2WechatAPIKey:  "api-key",
			MD2WechatBaseURL: server.URL,
		},
		log:           zap.NewNop(),
		theme:         NewThemeManager(),
		promptBuilder: NewPromptBuilder(),
	}

	result := conv.Convert(&ConvertRequest{
		Markdown: strings.Join([]string{
			"---",
			"title: Frontmatter Title",
			"author: Jane Doe",
			"---",
			"",
			"# Heading Title",
			"",
			"body",
		}, "\n"),
		Mode:  ModeAPI,
		Theme: "default",
	})
	if !result.Success {
		t.Fatalf("Convert() failed: %+v", result)
	}
	if strings.Contains(received.Markdown, "title: Frontmatter Title") {
		t.Fatalf("api request still contains frontmatter: %q", received.Markdown)
	}
	if !strings.Contains(received.Markdown, "# Heading Title") || !strings.Contains(received.Markdown, "body") {
		t.Fatalf("api request missing body content: %q", received.Markdown)
	}
}

func TestAPIConverterRejectsSuccessfulResponseWithoutHTML(t *testing.T) {
	for _, html := range []string{"", " \n\t "} {
		t.Run(fmt.Sprintf("html_%q", html), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code": 0,
					"msg":  "ok",
					"data": map[string]any{"html": html, "theme": "default", "fontSize": "medium", "backgroundType": "none", "wordCount": 2, "estimatedReadTime": 1},
				})
			}))
			defer server.Close()

			api := NewAPIConverterWithURL(zap.NewNop(), server.URL)
			got, err := api.Convert(&APIRequest{Markdown: "# Title", Theme: "default"}, "api-key")
			if err == nil {
				t.Fatalf("Convert() error = nil, got HTML %q", got)
			}
			if got != "" {
				t.Fatalf("Convert() HTML = %q, want empty on error", got)
			}
		})
	}
}

func TestConvertAPIModeRejectsAIThemeBeforeRemoteCall(t *testing.T) {
	conv := &converter{
		cfg: &config.Config{
			MD2WechatAPIKey:  "api-key",
			MD2WechatBaseURL: "http://127.0.0.1:1",
		},
		log:           zap.NewNop(),
		theme:         NewThemeManager(),
		promptBuilder: NewPromptBuilder(),
	}

	result := conv.Convert(&ConvertRequest{
		Markdown: "# Title\n\nBody",
		Mode:     ModeAPI,
		Theme:    "autumn-warm",
	})
	if result.Success {
		t.Fatalf("expected failed result, got %+v", result)
	}
	if !strings.Contains(result.Error, "not valid for api mode") {
		t.Fatalf("error = %q", result.Error)
	}
	if strings.Contains(result.Error, "API call failed") {
		t.Fatalf("remote API should not be called for theme mismatch: %q", result.Error)
	}
}

func TestConvertAIModeRejectsAPIThemeInsteadOfGenericPrompt(t *testing.T) {
	conv := &converter{
		cfg:           &config.Config{},
		log:           zap.NewNop(),
		theme:         NewThemeManager(),
		promptBuilder: NewPromptBuilder(),
	}

	result := conv.Convert(&ConvertRequest{
		Markdown: "# Title\n\nBody",
		Mode:     ModeAI,
		Theme:    "default",
	})
	if result.Success {
		t.Fatalf("expected failed result, got %+v", result)
	}
	if result.Prompt != "" || strings.Contains(result.Error, aiModePrefix) {
		t.Fatalf("expected no generic AI prompt, got %+v", result)
	}
	if !strings.Contains(result.Error, "not valid for ai mode") {
		t.Fatalf("error = %q", result.Error)
	}
}

func TestCompleteAIConversionMarksCompletedState(t *testing.T) {
	result := CompleteAIConversion("<p>ok</p>", nil, "default")
	if result.Status != action.StatusCompleted {
		t.Fatalf("Status = %q", result.Status)
	}
	if result.Action != action.ActionConvert {
		t.Fatalf("Action = %q", result.Action)
	}
	if !result.Success {
		t.Fatalf("expected success result: %+v", result)
	}
	if IsAIRequest(result) {
		t.Fatalf("completed result should not require AI: %+v", result)
	}
}

func TestValidatePromptContentRejectsDangerousContent(t *testing.T) {
	result := ValidatePromptContent(`<script>alert(1)</script>`)
	if result.Valid {
		t.Fatalf("expected invalid result: %#v", result)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected validation errors: %#v", result)
	}
}

func TestBuildPromptWithTemplateAppliesVariables(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPromptWithTemplate("Title: {{.title}}\nBody: {{.markdown}}", map[string]string{
		"TITLE":    "My Title",
		"MARKDOWN": "Body",
	})
	if err != nil {
		t.Fatalf("BuildPromptWithTemplate() error = %v", err)
	}
	if !strings.Contains(prompt, "My Title") || !strings.Contains(prompt, "Body") {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestInsertAndReplaceImagePlaceholders(t *testing.T) {
	html := `<p>before</p><img src="./a.png" alt="a"><p>middle</p><img src="https://example.com/b.png" alt="b">`
	images := []ImageRef{
		{Index: 0, Original: "./a.png", Placeholder: "<!-- IMG:0 -->", WechatURL: "https://wechat.local/a"},
		{Index: 1, Original: "https://example.com/b.png", Placeholder: "<!-- IMG:1 -->", WechatURL: "https://wechat.local/b"},
	}

	withPlaceholders := InsertImagePlaceholders(html, images)
	if !strings.Contains(withPlaceholders, "<!-- IMG:0 -->") || !strings.Contains(withPlaceholders, "<!-- IMG:1 -->") {
		t.Fatalf("expected placeholders to be inserted, got %s", withPlaceholders)
	}

	replaced := ReplaceImagePlaceholders(withPlaceholders, images)
	if strings.Contains(replaced, "./a.png") || strings.Contains(replaced, "https://example.com/b.png") {
		t.Fatalf("expected original sources to be replaced, got %s", replaced)
	}
	if !strings.Contains(replaced, "https://wechat.local/a") || !strings.Contains(replaced, "https://wechat.local/b") {
		t.Fatalf("expected WeChat URLs in final HTML, got %s", replaced)
	}
}

func TestInsertImagePlaceholdersFallsBackToDocumentOrder(t *testing.T) {
	html := `<p>before</p><img src="https://cdn.example.com/a.png" alt="a"><p>middle</p><img src="https://cdn.example.com/b.png" alt="b">`
	images := []ImageRef{
		{Index: 0, Original: "./a.png", Placeholder: "<!-- IMG:0 -->", WechatURL: "https://wechat.local/a"},
		{Index: 1, Original: "./b.png", Placeholder: "<!-- IMG:1 -->", WechatURL: "https://wechat.local/b"},
	}

	withPlaceholders := InsertImagePlaceholders(html, images)
	if strings.Count(withPlaceholders, "<!-- IMG:") != 2 {
		t.Fatalf("expected ordered fallback placeholders, got %s", withPlaceholders)
	}

	replaced := ReplaceImagePlaceholders(withPlaceholders, images)
	if !strings.Contains(replaced, "https://wechat.local/a") || !strings.Contains(replaced, "https://wechat.local/b") {
		t.Fatalf("expected fallback replacement to use WeChat URLs, got %s", replaced)
	}
	if strings.Contains(replaced, "cdn.example.com") {
		t.Fatalf("expected original rewritten src values to be removed, got %s", replaced)
	}
}
