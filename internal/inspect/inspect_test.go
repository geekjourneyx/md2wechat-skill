package inspect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestRunResolvesMetadataAndDetectsDuplicateH1(t *testing.T) {
	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "article.md")
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter 标题",
		"author: 张三",
		"summary: Frontmatter 摘要",
		"---",
		"",
		"# Frontmatter 标题",
		"",
		"正文",
	}, "\n")

	result, err := Run(&Input{
		MarkdownFile: markdownPath,
		Markdown:     markdown,
		Mode:         "api",
		Theme:        "default",
		FontSize:     "medium",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Metadata.Title.Value != "Frontmatter 标题" || result.Metadata.Title.Source != "frontmatter.title" {
		t.Fatalf("title = %#v", result.Metadata.Title)
	}
	if result.Metadata.Author.Value != "张三" || result.Metadata.Author.Source != "frontmatter.author" {
		t.Fatalf("author = %#v", result.Metadata.Author)
	}
	if result.Metadata.Digest.Value != "Frontmatter 摘要" || result.Metadata.Digest.Source != "frontmatter.summary" {
		t.Fatalf("digest = %#v", result.Metadata.Digest)
	}
	if !result.Structure.BodyH1.Present || result.Structure.BodyH1.Text != "Frontmatter 标题" {
		t.Fatalf("body_h1 = %#v", result.Structure.BodyH1)
	}
	if !result.Structure.DuplicateTitleRisk {
		t.Fatal("expected duplicate title risk")
	}
}

func TestRunDoesNotUseFirstBodyLineAsTitleAndFlagsMissingRequirements(t *testing.T) {
	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "article.md")
	localImage := filepath.Join("images", "missing.png")
	markdown := strings.Join([]string{
		"正文第一行不是标题",
		"",
		"![missing](images/missing.png)",
	}, "\n")

	result, err := Run(&Input{
		MarkdownFile:    markdownPath,
		Markdown:        markdown,
		Mode:            "api",
		Theme:           "default",
		FontSize:        "medium",
		DraftRequested:  true,
		UploadRequested: true,
		Config:          &config.Config{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Metadata.Title.Value != "未命名文章" || result.Metadata.Title.Source != "fallback.untitled" {
		t.Fatalf("title = %#v", result.Metadata.Title)
	}
	if result.Readiness.ConvertReady {
		t.Fatal("expected convert_ready false without API key")
	}
	if result.Readiness.UploadReady {
		t.Fatal("expected upload_ready false without WeChat config")
	}
	if result.Readiness.DraftReady {
		t.Fatal("expected draft_ready false without cover and config")
	}

	foundLocalImageMissing := false
	foundMissingAPIKey := false
	foundMissingCover := false
	foundMissingWeChatConfig := false
	for _, check := range result.Checks {
		switch check.Code {
		case "LOCAL_IMAGE_MISSING":
			foundLocalImageMissing = strings.Contains(check.Message, localImage)
		case "MISSING_API_KEY":
			foundMissingAPIKey = true
		case "MISSING_COVER":
			foundMissingCover = true
		case "MISSING_WECHAT_CONFIG":
			foundMissingWeChatConfig = true
		}
	}
	if !foundLocalImageMissing || !foundMissingAPIKey || !foundMissingCover || !foundMissingWeChatConfig {
		t.Fatalf("checks = %#v", result.Checks)
	}
}

func TestRunMarksLocalImageExists(t *testing.T) {
	dir := t.TempDir()
	imageDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	imagePath := filepath.Join(imageDir, "ok.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := Run(&Input{
		MarkdownFile: filepath.Join(dir, "article.md"),
		Markdown:     "![ok](images/ok.png)\n",
		Mode:         "ai",
		Theme:        "autumn-warm",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(result.Assets) != 1 || !result.Assets[0].Exists {
		t.Fatalf("assets = %#v", result.Assets)
	}
	if result.Readiness.PreviewFidelity != PreviewFidelityDegraded {
		t.Fatalf("preview_fidelity = %q", result.Readiness.PreviewFidelity)
	}
}

func TestRunIncludesReadinessTargets(t *testing.T) {
	result, err := Run(&Input{
		MarkdownFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:     "# 标题\n",
		Mode:         "api",
		Config:       &config.Config{MD2WechatAPIKey: "api-key"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	readiness := result.Readiness
	if readiness.SchemaVersion != ReadinessSchemaVersion {
		t.Fatalf("schema_version = %q", readiness.SchemaVersion)
	}
	if readiness.Targets.Preview != ReadinessTargetReady {
		t.Fatalf("preview target = %q", readiness.Targets.Preview)
	}
	if readiness.Targets.Convert != ReadinessTargetReady {
		t.Fatalf("convert target = %q", readiness.Targets.Convert)
	}
	if readiness.Targets.Upload != ReadinessTargetNotRequested {
		t.Fatalf("upload target = %q", readiness.Targets.Upload)
	}
	if readiness.Targets.Draft != ReadinessTargetNotRequested {
		t.Fatalf("draft target = %q", readiness.Targets.Draft)
	}
	if len(readiness.Blockers) != 0 {
		t.Fatalf("blockers = %#v", readiness.Blockers)
	}
	if readiness.Blockers == nil {
		t.Fatal("blockers should be an empty array, not nil")
	}

	payload, err := json.Marshal(readiness)
	if err != nil {
		t.Fatalf("marshal readiness: %v", err)
	}
	if !strings.Contains(string(payload), `"blockers":[]`) {
		t.Fatalf("readiness JSON should include stable empty blockers array: %s", payload)
	}
}

func TestRunAICustomPromptMirrorsConvertThemeException(t *testing.T) {
	result, err := Run(&Input{
		MarkdownFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:     "# 标题\n\n正文",
		Mode:         "ai",
		Theme:        "default",
		CustomPrompt: "Use a compact editorial layout.",
		FontSize:     "medium",
		Config:       &config.Config{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Context.CustomPrompt != true {
		t.Fatalf("context.custom_prompt = %t", result.Context.CustomPrompt)
	}
	if result.Readiness.Targets.Convert != ReadinessTargetReady {
		t.Fatalf("convert target = %q, checks = %#v", result.Readiness.Targets.Convert, result.Checks)
	}
	for _, check := range result.Checks {
		if check.Code == "THEME_MODE_MISMATCH" {
			t.Fatalf("unexpected theme mismatch with custom prompt: %#v", result.Checks)
		}
	}
}

func TestBlockedReadinessTargetsContract(t *testing.T) {
	cases := []struct {
		code string
		want []string
	}{
		{code: "TITLE_TOO_LONG", want: []string{"convert", "upload", "draft"}},
		{code: "AUTHOR_TOO_LONG", want: []string{"convert", "upload", "draft"}},
		{code: "DIGEST_TOO_LONG", want: []string{"convert", "upload", "draft"}},
		{code: "MISSING_API_KEY", want: []string{"convert", "upload", "draft"}},
		{code: "THEME_NOT_FOUND", want: []string{"convert", "upload", "draft"}},
		{code: "THEME_NOT_SELECTABLE", want: []string{"convert", "upload", "draft"}},
		{code: "THEME_MODE_MISMATCH", want: []string{"convert", "upload", "draft"}},
		{code: "THEME_INVALID", want: []string{"convert", "upload", "draft"}},
		{code: "LOCAL_IMAGE_MISSING", want: []string{"upload", "draft"}},
		{code: "MISSING_WECHAT_CONFIG", want: []string{"upload", "draft"}},
		{code: "MISSING_COVER", want: []string{"draft"}},
		{code: "COVER_IMAGE_MISSING", want: []string{"draft"}},
		{code: "CONFLICTING_COVER_INPUTS", want: []string{"draft"}},
		{code: "COVER_MEDIA_ID_INVALID", want: []string{"draft"}},
		{code: "DUPLICATE_H1"},
		{code: "DIGEST_METADATA_ONLY"},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			if got := blockedReadinessTargets(tc.code); !slices.Equal(got, tc.want) {
				t.Fatalf("blockedReadinessTargets(%q) = %#v, want %#v", tc.code, got, tc.want)
			}
		})
	}
}

func TestRunReadinessTargetsAndBlockersStayAligned(t *testing.T) {
	fullCfg := &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAppID:     "appid",
		WechatSecret:    "secret",
	}

	cases := []struct {
		name              string
		input             Input
		wantTargets       ReadinessTargets
		wantBlockerCode   string
		wantBlockerBlocks []string
	}{
		{
			name: "metadata limit blocks all publish targets via TITLE_TOO_LONG",
			input: Input{
				MarkdownFile:    filepath.Join(t.TempDir(), "article.md"),
				Markdown:        "# 标题\n",
				Mode:            "api",
				TitleOverride:   strings.Repeat("标", 33),
				UploadRequested: true,
				DraftRequested:  true,
				CoverMediaID:    "existing-cover-id",
				Config:          fullCfg,
			},
			wantTargets: ReadinessTargets{
				Preview: ReadinessTargetReady,
				Convert: ReadinessTargetBlocked,
				Upload:  ReadinessTargetBlocked,
				Draft:   ReadinessTargetBlocked,
			},
			wantBlockerCode:   "TITLE_TOO_LONG",
			wantBlockerBlocks: []string{"convert", "upload", "draft"},
		},
		{
			name: "theme mismatch blocks all publish targets via THEME_MODE_MISMATCH",
			input: Input{
				MarkdownFile:    filepath.Join(t.TempDir(), "article.md"),
				Markdown:        "# 标题\n",
				Mode:            "api",
				Theme:           "autumn-warm",
				UploadRequested: true,
				DraftRequested:  true,
				CoverMediaID:    "existing-cover-id",
				Config:          fullCfg,
			},
			wantTargets: ReadinessTargets{
				Preview: ReadinessTargetReady,
				Convert: ReadinessTargetBlocked,
				Upload:  ReadinessTargetBlocked,
				Draft:   ReadinessTargetBlocked,
			},
			wantBlockerCode:   "THEME_MODE_MISMATCH",
			wantBlockerBlocks: []string{"convert", "upload", "draft"},
		},
		{
			name: "missing local image blocks upload and draft only via LOCAL_IMAGE_MISSING",
			input: Input{
				MarkdownFile:    filepath.Join(t.TempDir(), "article.md"),
				Markdown:        "# 标题\n\n![missing](images/missing.png)\n",
				Mode:            "api",
				UploadRequested: true,
				DraftRequested:  true,
				CoverMediaID:    "existing-cover-id",
				Config:          fullCfg,
			},
			wantTargets: ReadinessTargets{
				Preview: ReadinessTargetReady,
				Convert: ReadinessTargetReady,
				Upload:  ReadinessTargetBlocked,
				Draft:   ReadinessTargetBlocked,
			},
			wantBlockerCode:   "LOCAL_IMAGE_MISSING",
			wantBlockerBlocks: []string{"upload", "draft"},
		},
		{
			name: "missing cover blocks draft only via MISSING_COVER",
			input: Input{
				MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
				Markdown:       "# 标题\n",
				Mode:           "api",
				DraftRequested: true,
				Config:         fullCfg,
			},
			wantTargets: ReadinessTargets{
				Preview: ReadinessTargetReady,
				Convert: ReadinessTargetReady,
				Upload:  ReadinessTargetReady,
				Draft:   ReadinessTargetBlocked,
			},
			wantBlockerCode:   "MISSING_COVER",
			wantBlockerBlocks: []string{"draft"},
		},
		{
			name: "not requested upload and draft remain not_requested",
			input: Input{
				MarkdownFile: filepath.Join(t.TempDir(), "article.md"),
				Markdown:     "# 标题\n",
				Mode:         "api",
				Config:       &config.Config{MD2WechatAPIKey: "api-key"},
			},
			wantTargets: ReadinessTargets{
				Preview: ReadinessTargetReady,
				Convert: ReadinessTargetReady,
				Upload:  ReadinessTargetNotRequested,
				Draft:   ReadinessTargetNotRequested,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Run(&tc.input)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if result.Readiness.Targets != tc.wantTargets {
				t.Fatalf("targets = %#v, want %#v", result.Readiness.Targets, tc.wantTargets)
			}
			assertReadinessLegacyFieldsMatchTargets(t, result.Readiness)

			if tc.wantBlockerCode == "" {
				if len(result.Readiness.Blockers) != 0 {
					t.Fatalf("blockers = %#v", result.Readiness.Blockers)
				}
				return
			}

			blocker, ok := findReadinessBlocker(result.Readiness.Blockers, tc.wantBlockerCode)
			if !ok {
				t.Fatalf("missing blocker %s in %#v", tc.wantBlockerCode, result.Readiness.Blockers)
			}
			if !slices.Equal(blocker.Blocks, tc.wantBlockerBlocks) {
				t.Fatalf("%s blocks = %#v, want %#v", tc.wantBlockerCode, blocker.Blocks, tc.wantBlockerBlocks)
			}
		})
	}
}

func TestRunMapsReadinessBlockersToTargets(t *testing.T) {
	result, err := Run(&Input{
		MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
		Markdown:       "# 标题\n",
		Mode:           "api",
		DraftRequested: true,
		Config: &config.Config{
			MD2WechatAPIKey: "api-key",
			WechatAppID:     "appid",
			WechatSecret:    "secret",
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Readiness.Targets.Draft != ReadinessTargetBlocked {
		t.Fatalf("draft target = %q", result.Readiness.Targets.Draft)
	}
	if result.Readiness.Targets.Upload != ReadinessTargetReady {
		t.Fatalf("upload target = %q", result.Readiness.Targets.Upload)
	}
	blocker, ok := findReadinessBlocker(result.Readiness.Blockers, "MISSING_COVER")
	if !ok {
		t.Fatalf("missing MISSING_COVER blocker in %#v", result.Readiness.Blockers)
	}
	if !slices.Equal(blocker.Blocks, []string{"draft"}) {
		t.Fatalf("MISSING_COVER blocks = %#v", blocker.Blocks)
	}
	if blocker.SuggestedFix == "" {
		t.Fatal("MISSING_COVER suggested_fix is empty")
	}
}

func TestBuildReadinessDoesNotMirrorNonErrorChecks(t *testing.T) {
	blockers := buildReadinessBlockers([]Check{
		{
			Level:        LevelWarn,
			Code:         "DUPLICATE_H1",
			Message:      "Body H1 matches the final article title",
			Field:        "title",
			SuggestedFix: "remove the body H1 or change the final title",
		},
		{
			Level:        LevelInfo,
			Code:         "DIGEST_METADATA_ONLY",
			Message:      "Digest affects draft metadata, not body HTML rendering",
			Field:        "digest",
			SuggestedFix: "use body content or theme behavior to control visible article summary inside the HTML",
		},
	})

	if len(blockers) != 0 {
		t.Fatalf("blockers = %#v", blockers)
	}
}

func TestRunMetadataSourcePrecedence(t *testing.T) {
	t.Run("cli overrides frontmatter", func(t *testing.T) {
		dir := t.TempDir()
		result, err := Run(&Input{
			MarkdownFile: filepath.Join(dir, "article.md"),
			Markdown: strings.Join([]string{
				"---",
				"title: Frontmatter 标题",
				"author: Frontmatter 作者",
				"digest: Frontmatter 摘要",
				"---",
				"",
				"# 正文标题",
			}, "\n"),
			TitleOverride:  "CLI 标题",
			AuthorOverride: "CLI 作者",
			DigestOverride: "CLI 摘要",
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if result.Metadata.Title.Source != "cli.title" || result.Metadata.Title.Value != "CLI 标题" {
			t.Fatalf("title = %#v", result.Metadata.Title)
		}
		if result.Metadata.Author.Source != "cli.author" || result.Metadata.Author.Value != "CLI 作者" {
			t.Fatalf("author = %#v", result.Metadata.Author)
		}
		if result.Metadata.Digest.Source != "cli.digest" || result.Metadata.Digest.Value != "CLI 摘要" {
			t.Fatalf("digest = %#v", result.Metadata.Digest)
		}
	})

	t.Run("empty frontmatter title falls back to markdown heading", func(t *testing.T) {
		dir := t.TempDir()
		result, err := Run(&Input{
			MarkdownFile: filepath.Join(dir, "article.md"),
			Markdown: strings.Join([]string{
				"---",
				"title: \"\"",
				"author: 张三",
				"---",
				"",
				"# Heading 标题",
				"",
				"正文",
			}, "\n"),
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if result.Metadata.Title.Source != "markdown.heading" || result.Metadata.Title.Value != "Heading 标题" {
			t.Fatalf("title = %#v", result.Metadata.Title)
		}
		if result.Metadata.Author.Source != "frontmatter.author" {
			t.Fatalf("author = %#v", result.Metadata.Author)
		}
	})
}

func TestRunDigestSourcePrecedence(t *testing.T) {
	cases := []struct {
		name        string
		frontmatter []string
		wantValue   string
		wantSource  string
	}{
		{
			name: "digest wins",
			frontmatter: []string{
				"digest: Digest 文本",
				"summary: Summary 文本",
				"description: Description 文本",
			},
			wantValue:  "Digest 文本",
			wantSource: "frontmatter.digest",
		},
		{
			name: "summary wins when digest missing",
			frontmatter: []string{
				"summary: Summary 文本",
				"description: Description 文本",
			},
			wantValue:  "Summary 文本",
			wantSource: "frontmatter.summary",
		},
		{
			name: "description used last",
			frontmatter: []string{
				"description: Description 文本",
			},
			wantValue:  "Description 文本",
			wantSource: "frontmatter.description",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			markdown := strings.Join(append([]string{"---"}, append(tc.frontmatter, "---", "", "# 标题")...), "\n")
			result, err := Run(&Input{
				MarkdownFile: filepath.Join(dir, "article.md"),
				Markdown:     markdown,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if result.Metadata.Digest.Value != tc.wantValue || result.Metadata.Digest.Source != tc.wantSource {
				t.Fatalf("digest = %#v", result.Metadata.Digest)
			}
		})
	}
}

func TestRunReadinessMatrix(t *testing.T) {
	fullCfg := &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAppID:     "appid",
		WechatSecret:    "secret",
	}
	apiOnlyCfg := &config.Config{MD2WechatAPIKey: "api-key"}
	validCoverPath := filepath.Join(t.TempDir(), "cover.jpg")
	if err := os.WriteFile(validCoverPath, []byte("cover"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cases := []struct {
		name         string
		input        Input
		wantConvert  bool
		wantUpload   bool
		wantDraft    bool
		wantPreview  string
		wantChecks   []string
		absentChecks []string
	}{
		{
			name: "api without config or cover",
			input: Input{
				MarkdownFile:    filepath.Join(t.TempDir(), "article.md"),
				Markdown:        "# 标题\n",
				Mode:            "api",
				DraftRequested:  true,
				UploadRequested: true,
				Config:          &config.Config{},
			},
			wantConvert: false,
			wantUpload:  false,
			wantDraft:   false,
			wantPreview: PreviewFidelityDegraded,
			wantChecks:  []string{"MISSING_API_KEY", "MISSING_WECHAT_CONFIG", "MISSING_COVER"},
		},
		{
			name: "ai without config or cover",
			input: Input{
				MarkdownFile:    filepath.Join(t.TempDir(), "article.md"),
				Markdown:        "# 标题\n",
				Mode:            "ai",
				Theme:           "autumn-warm",
				DraftRequested:  true,
				UploadRequested: true,
				Config:          &config.Config{},
			},
			wantConvert:  true,
			wantUpload:   false,
			wantDraft:    false,
			wantPreview:  PreviewFidelityDegraded,
			wantChecks:   []string{"AI_MODE_ACTION_REQUIRED", "MISSING_WECHAT_CONFIG", "MISSING_COVER"},
			absentChecks: []string{"MISSING_API_KEY"},
		},
		{
			name: "api with api key only",
			input: Input{
				MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
				Markdown:       "# 标题\n",
				Mode:           "api",
				DraftRequested: true,
				Config:         apiOnlyCfg,
			},
			wantConvert:  true,
			wantUpload:   false,
			wantDraft:    false,
			wantPreview:  PreviewFidelityExact,
			wantChecks:   []string{"MISSING_WECHAT_CONFIG", "MISSING_COVER"},
			absentChecks: []string{"MISSING_API_KEY"},
		},
		{
			name: "api fully ready",
			input: Input{
				MarkdownFile:    filepath.Join(t.TempDir(), "article.md"),
				Markdown:        "# 标题\n",
				Mode:            "api",
				DraftRequested:  true,
				UploadRequested: true,
				CoverImagePath:  validCoverPath,
				Config:          fullCfg,
			},
			wantConvert:  true,
			wantUpload:   true,
			wantDraft:    true,
			wantPreview:  PreviewFidelityExact,
			absentChecks: []string{"MISSING_API_KEY", "MISSING_WECHAT_CONFIG", "MISSING_COVER"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Run(&tc.input)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if result.Readiness.ConvertReady != tc.wantConvert {
				t.Fatalf("convert_ready = %t", result.Readiness.ConvertReady)
			}
			if result.Readiness.UploadReady != tc.wantUpload {
				t.Fatalf("upload_ready = %t", result.Readiness.UploadReady)
			}
			if result.Readiness.DraftReady != tc.wantDraft {
				t.Fatalf("draft_ready = %t", result.Readiness.DraftReady)
			}
			if result.Readiness.PreviewFidelity != tc.wantPreview {
				t.Fatalf("preview_fidelity = %q", result.Readiness.PreviewFidelity)
			}
			for _, code := range tc.wantChecks {
				if !hasCheckCode(result.Checks, code) {
					t.Fatalf("missing check %s in %#v", code, result.Checks)
				}
			}
			for _, code := range tc.absentChecks {
				if hasCheckCode(result.Checks, code) {
					t.Fatalf("unexpected check %s in %#v", code, result.Checks)
				}
			}
		})
	}
}

func TestRunMakesReadinessFalseForBlockingChecks(t *testing.T) {
	fullCfg := &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAppID:     "appid",
		WechatSecret:    "secret",
	}

	t.Run("metadata limit errors block convert and downstream readiness", func(t *testing.T) {
		result, err := Run(&Input{
			MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
			Markdown:       "# 标题\n",
			Mode:           "api",
			TitleOverride:  strings.Repeat("标", 33),
			DraftRequested: true,
			CoverImagePath: filepath.Join(t.TempDir(), "cover.png"),
			Config:         fullCfg,
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !hasCheckCode(result.Checks, "TITLE_TOO_LONG") {
			t.Fatalf("checks = %#v", result.Checks)
		}
		if result.Readiness.ConvertReady {
			t.Fatal("expected convert_ready false when title is too long")
		}
		if result.Readiness.UploadReady {
			t.Fatal("expected upload_ready false when convert is blocked")
		}
		if result.Readiness.DraftReady {
			t.Fatal("expected draft_ready false when convert is blocked")
		}
	})

	t.Run("cover path errors block draft target state", func(t *testing.T) {
		result, err := Run(&Input{
			MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
			Markdown:       "# 标题\n",
			Mode:           "api",
			DraftRequested: true,
			CoverImagePath: filepath.Join(t.TempDir(), "missing-cover.png"),
			Config:         fullCfg,
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !hasCheckCode(result.Checks, "COVER_IMAGE_MISSING") {
			t.Fatalf("checks = %#v", result.Checks)
		}
		if !result.Readiness.ConvertReady {
			t.Fatal("expected convert_ready true when only the cover path is invalid")
		}
		if !result.Readiness.UploadReady {
			t.Fatal("expected upload_ready true when config is valid and no upload blocker exists")
		}
		if result.Readiness.DraftReady {
			t.Fatal("expected draft_ready false when the cover image path is invalid")
		}
	})

	t.Run("existing cover media id makes draft ready without local cover path", func(t *testing.T) {
		result, err := Run(&Input{
			MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
			Markdown:       "# 标题\n",
			Mode:           "api",
			DraftRequested: true,
			Config:         fullCfg,
			CoverMediaID:   "existing-cover-id",
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !result.Readiness.ConvertReady || !result.Readiness.UploadReady || !result.Readiness.DraftReady {
			t.Fatalf("readiness = %#v", result.Readiness)
		}
		if hasCheckCode(result.Checks, "MISSING_COVER") || hasCheckCode(result.Checks, "COVER_MEDIA_ID_INVALID") {
			t.Fatalf("checks = %#v", result.Checks)
		}
	})

	t.Run("conflicting cover inputs block draft target state", func(t *testing.T) {
		coverPath := filepath.Join(t.TempDir(), "cover.png")
		if err := os.WriteFile(coverPath, []byte("cover"), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		result, err := Run(&Input{
			MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
			Markdown:       "# 标题\n",
			Mode:           "api",
			DraftRequested: true,
			Config:         fullCfg,
			CoverImagePath: coverPath,
			CoverMediaID:   "existing-cover-id",
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !hasCheckCode(result.Checks, "CONFLICTING_COVER_INPUTS") {
			t.Fatalf("checks = %#v", result.Checks)
		}
		if result.Readiness.DraftReady {
			t.Fatal("expected draft_ready false when cover inputs conflict")
		}
	})

	t.Run("url-like cover media id blocks draft target state", func(t *testing.T) {
		result, err := Run(&Input{
			MarkdownFile:   filepath.Join(t.TempDir(), "article.md"),
			Markdown:       "# 标题\n",
			Mode:           "api",
			DraftRequested: true,
			Config:         fullCfg,
			CoverMediaID:   "https://mmbiz.qpic.cn/example",
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !hasCheckCode(result.Checks, "COVER_MEDIA_ID_INVALID") {
			t.Fatalf("checks = %#v", result.Checks)
		}
		if result.Readiness.DraftReady {
			t.Fatal("expected draft_ready false when cover_media_id is a URL")
		}
	})
}

func TestRunBlocksConvertForThemeCompatibilityErrors(t *testing.T) {
	fullCfg := &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAppID:     "appid",
		WechatSecret:    "secret",
	}

	cases := []struct {
		name  string
		theme string
		code  string
	}{
		{
			name:  "api mode with ai theme",
			theme: "autumn-warm",
			code:  "THEME_MODE_MISMATCH",
		},
		{
			name:  "api mode with missing theme",
			theme: "missing-theme",
			code:  "THEME_NOT_FOUND",
		},
		{
			name:  "api mode with non-selectable api collection",
			theme: "api-collection",
			code:  "THEME_NOT_SELECTABLE",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Run(&Input{
				MarkdownFile:    filepath.Join(t.TempDir(), "article.md"),
				Markdown:        "# 标题\n",
				Mode:            "api",
				Theme:           tc.theme,
				UploadRequested: true,
				DraftRequested:  true,
				CoverMediaID:    "existing-cover-id",
				Config:          fullCfg,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			check, ok := findCheck(result.Checks, tc.code)
			if !ok {
				t.Fatalf("missing check %s in %#v", tc.code, result.Checks)
			}
			if check.Level != LevelError || check.Field != "theme" {
				t.Fatalf("theme check = %#v", check)
			}
			if result.Readiness.ConvertReady {
				t.Fatal("expected convert_ready false when theme compatibility fails")
			}
			if result.Readiness.Targets.Convert != ReadinessTargetBlocked {
				t.Fatalf("convert target = %q", result.Readiness.Targets.Convert)
			}

			blocker, ok := findReadinessBlocker(result.Readiness.Blockers, tc.code)
			if !ok {
				t.Fatalf("missing blocker %s in %#v", tc.code, result.Readiness.Blockers)
			}
			if !slices.Equal(blocker.Blocks, []string{"convert", "upload", "draft"}) {
				t.Fatalf("%s blocks = %#v", tc.code, blocker.Blocks)
			}
		})
	}
}

func TestRunRejectsInvalidMode(t *testing.T) {
	_, err := Run(&Input{
		MarkdownFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:     "# 标题\n",
		Mode:         "foo",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid convert mode") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunInspectsAbsoluteRemoteAndAIAssets(t *testing.T) {
	dir := t.TempDir()
	absPath := filepath.Join(dir, "cover.png")
	if err := os.WriteFile(absPath, []byte("png"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	markdown := strings.Join([]string{
		"# 标题",
		"",
		"![local](" + absPath + ")",
		"![remote](https://example.com/a.png)",
		"![ai](__generate:draw a cover__)",
	}, "\n")

	result, err := Run(&Input{
		MarkdownFile: filepath.Join(dir, "article.md"),
		Markdown:     markdown,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Structure.Images.Total != 3 || result.Structure.Images.Local != 1 || result.Structure.Images.Remote != 1 || result.Structure.Images.AI != 1 {
		t.Fatalf("image counters = %#v", result.Structure.Images)
	}
	if len(result.Assets) != 3 {
		t.Fatalf("assets = %#v", result.Assets)
	}
	if result.Assets[0].ResolvedSource != absPath || !result.Assets[0].Exists {
		t.Fatalf("local asset = %#v", result.Assets[0])
	}
	if result.Assets[1].Kind != "online" || !result.Assets[1].Exists {
		t.Fatalf("remote asset = %#v", result.Assets[1])
	}
	if result.Assets[2].Kind != "ai" || !result.Assets[2].Exists {
		t.Fatalf("ai asset = %#v", result.Assets[2])
	}
}

func TestRunAddsMetadataAndImageIntentChecks(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "cover.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	markdown := strings.Join([]string{
		"---",
		"title: 草稿标题",
		"summary: 这是摘要",
		"---",
		"",
		"# 正文一级标题",
		"",
		"![local](cover.png)",
	}, "\n")

	result, err := Run(&Input{
		MarkdownFile: filepath.Join(dir, "article.md"),
		Markdown:     markdown,
		Mode:         "ai",
		Theme:        "autumn-warm",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, code := range []string{
		"TITLE_BODY_MISMATCH",
		"DIGEST_METADATA_ONLY",
		"IMAGE_REPLACEMENT_REQUIRES_UPLOAD_OR_DRAFT",
		"AI_MODE_ACTION_REQUIRED",
	} {
		if !hasCheckCode(result.Checks, code) {
			t.Fatalf("missing check %s in %#v", code, result.Checks)
		}
	}
	if hasCheckCode(result.Checks, "DUPLICATE_H1") {
		t.Fatalf("unexpected duplicate h1 check in %#v", result.Checks)
	}
}

func hasCheckCode(checks []Check, code string) bool {
	for _, check := range checks {
		if check.Code == code {
			return true
		}
	}
	return false
}

func findCheck(checks []Check, code string) (Check, bool) {
	for _, check := range checks {
		if check.Code == code {
			return check, true
		}
	}
	return Check{}, false
}

func findReadinessBlocker(blockers []ReadinessBlocker, code string) (ReadinessBlocker, bool) {
	for _, blocker := range blockers {
		if blocker.Code == code {
			return blocker, true
		}
	}
	return ReadinessBlocker{}, false
}

func assertReadinessLegacyFieldsMatchTargets(t *testing.T, readiness Readiness) {
	t.Helper()

	if got, want := readiness.Targets.Preview, previewReadinessTarget(readiness.PreviewFidelity); got != want {
		t.Fatalf("preview target = %q, want %q for preview_fidelity %q", got, want, readiness.PreviewFidelity)
	}
	if got, want := readiness.Targets.Convert, readyReadinessTarget(readiness.ConvertReady); got != want {
		t.Fatalf("convert target = %q, want %q for convert_ready %t", got, want, readiness.ConvertReady)
	}
	assertRequestedReadinessTargetMatchesLegacy(t, "upload", readiness.Targets.Upload, readiness.UploadReady)
	assertRequestedReadinessTargetMatchesLegacy(t, "draft", readiness.Targets.Draft, readiness.DraftReady)
}

func assertRequestedReadinessTargetMatchesLegacy(t *testing.T, name, target string, ready bool) {
	t.Helper()

	if target == ReadinessTargetNotRequested {
		return
	}
	if got, want := target, readyReadinessTarget(ready); got != want {
		t.Fatalf("%s target = %q, want %q for legacy ready %t", name, got, want, ready)
	}
}
