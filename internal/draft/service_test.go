package draft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestGenerateDigestFromContent_StripsMarkupAndIsRuneSafe(t *testing.T) {
	content := `<article><h1>标题 &amp; 简介</h1><p>第一段🙂第二段</p><script>alert("x")</script></article>`

	got := GenerateDigestFromContent(content, 7)

	want := "标题 & 简介..."
	if got != want {
		t.Fatalf("GenerateDigestFromContent() = %q, want %q", got, want)
	}
}

func TestGenerateDigestFromContent_DefaultLength(t *testing.T) {
	got := GenerateDigestFromContent("<p>hello</p>", 0)
	if got != "hello" {
		t.Fatalf("GenerateDigestFromContent() = %q, want %q", got, "hello")
	}
}

func TestStripHTML_NormalizesBlocksAndEntities(t *testing.T) {
	content := `<div>第一段&nbsp;内容</div><p>第二段</p><style>.x{color:red}</style><blockquote>引用</blockquote><br><span>尾部</span>`

	got := stripHTML(content)

	want := "第一段 内容\n第二段\n引用\n尾部"
	if got != want {
		t.Fatalf("stripHTML() = %q, want %q", got, want)
	}
}

func TestCreateDraftFromFileValidationErrors(t *testing.T) {
	svc := &Service{log: zap.NewNop()}
	dir := t.TempDir()

	invalidJSON := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidJSON, []byte("{"), 0600); err != nil {
		t.Fatalf("write invalid json: %v", err)
	}
	if _, err := svc.CreateDraftFromFile(invalidJSON); err == nil {
		t.Fatal("expected parse json error")
	}

	emptyArticles := filepath.Join(dir, "empty.json")
	data, _ := json.Marshal(DraftRequest{})
	if err := os.WriteFile(emptyArticles, data, 0600); err != nil {
		t.Fatalf("write empty draft json: %v", err)
	}
	if _, err := svc.CreateDraftFromFile(emptyArticles); err == nil {
		t.Fatal("expected no articles error")
	}

	missingTitle := filepath.Join(dir, "missing-title.json")
	data, _ = json.Marshal(DraftRequest{
		Articles: []Article{{Content: "<p>body</p>"}},
	})
	if err := os.WriteFile(missingTitle, data, 0600); err != nil {
		t.Fatalf("write missing title json: %v", err)
	}
	if _, err := svc.CreateDraftFromFile(missingTitle); err == nil {
		t.Fatal("expected title is required error")
	}
}

func TestArticleForUpdateSelectsRequestedIndex(t *testing.T) {
	req := DraftRequest{
		Articles: []Article{
			{Title: "First", Content: "one"},
			{Title: "Second", Content: "two"},
		},
	}

	got, err := articleForUpdate(req, 1)
	if err != nil {
		t.Fatalf("articleForUpdate() error = %v", err)
	}
	if got.Title != "Second" || got.Content != "two" {
		t.Fatalf("articleForUpdate() = %#v", got)
	}
}

func TestArticleForUpdateValidationErrors(t *testing.T) {
	if _, err := articleForUpdate(DraftRequest{}, 0); err == nil {
		t.Fatal("expected no articles error")
	}
	if _, err := articleForUpdate(DraftRequest{Articles: []Article{{Title: "Only", Content: "one"}}}, 1); err == nil {
		t.Fatal("expected index out of range error")
	}
	if _, err := articleForUpdate(DraftRequest{Articles: []Article{{Title: "Only", Content: "one"}}}, -1); err == nil {
		t.Fatal("expected negative index error")
	}
}

func TestBuildSDKArticleConvertsOptionalFields(t *testing.T) {
	got, err := buildSDKArticle(Article{
		Title:            "Title",
		Content:          "<p>body</p>",
		Author:           "Author",
		Digest:           "Digest",
		ThumbMediaID:     "thumb-id",
		ShowCoverPic:     1,
		ContentSourceURL: "https://example.com/source",
	})
	if err != nil {
		t.Fatalf("buildSDKArticle() error = %v", err)
	}

	if got.Title != "Title" || got.Content != "<p>body</p>" || got.Author != "Author" || got.Digest != "Digest" {
		t.Fatalf("buildSDKArticle() = %#v", got)
	}
	if got.ThumbMediaID != "thumb-id" || got.ShowCoverPic != 1 || got.ContentSourceURL != "https://example.com/source" {
		t.Fatalf("buildSDKArticle() optional fields = %#v", got)
	}
}

func TestBuildSDKArticlesValidatesAndPreservesOrder(t *testing.T) {
	got, err := buildSDKArticles([]Article{
		{Title: "First", Content: "one"},
		{Title: "Second", Content: "two"},
	})
	if err != nil {
		t.Fatalf("buildSDKArticles() error = %v", err)
	}
	if len(got) != 2 || got[0].Title != "First" || got[1].Title != "Second" {
		t.Fatalf("buildSDKArticles() = %#v", got)
	}

	if _, err := buildSDKArticles([]Article{{Content: "missing title"}}); err == nil {
		t.Fatal("expected validation error for missing title")
	}
}
