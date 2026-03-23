package converter

import (
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

	result := conv.Convert(&ConvertRequest{Markdown: "", Mode: ModeAI})
	if result.Success || !strings.Contains(result.Error, ErrEmptyMarkdown.Error()) {
		t.Fatalf("unexpected empty markdown result: %+v", result)
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
