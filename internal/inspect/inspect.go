package inspect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"gopkg.in/yaml.v3"
)

const (
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"

	PreviewFidelityExact    = "exact"
	PreviewFidelityDegraded = "degraded"

	ReadinessSchemaVersion      = "v1"
	ReadinessTargetReady        = "ready"
	ReadinessTargetBlocked      = "blocked"
	ReadinessTargetDegraded     = "degraded"
	ReadinessTargetNotRequested = "not_requested"
)

type Input struct {
	MarkdownFile    string
	Markdown        string
	Mode            string
	Theme           string
	FontSize        string
	BackgroundType  string
	CustomPrompt    string
	TitleOverride   string
	AuthorOverride  string
	DigestOverride  string
	CoverImagePath  string
	CoverMediaID    string
	UploadRequested bool
	DraftRequested  bool
	Config          *config.Config
}

type Result struct {
	SourceFile string        `json:"source_file"`
	SourceDir  string        `json:"source_dir"`
	Context    Context       `json:"context"`
	Metadata   MetadataState `json:"metadata"`
	Structure  Structure     `json:"structure"`
	Assets     []AssetState  `json:"assets,omitempty"`
	Readiness  Readiness     `json:"readiness"`
	Checks     []Check       `json:"checks,omitempty"`
	Body       string        `json:"-"`
}

type Context struct {
	Mode           string `json:"mode"`
	Theme          string `json:"theme"`
	FontSize       string `json:"font_size"`
	BackgroundType string `json:"background_type"`
	CustomPrompt   bool   `json:"custom_prompt"`
	Upload         bool   `json:"upload"`
	Draft          bool   `json:"draft"`
}

type MetadataState struct {
	Title  MetadataField `json:"title"`
	Author MetadataField `json:"author"`
	Digest MetadataField `json:"digest"`
}

type MetadataField struct {
	Value  string `json:"value"`
	Source string `json:"source"`
	Limit  int    `json:"limit"`
	Length int    `json:"length"`
	Valid  bool   `json:"valid"`
}

type Structure struct {
	BodyH1             HeadingInfo   `json:"body_h1"`
	DuplicateTitleRisk bool          `json:"duplicate_title_risk"`
	Headings           []Heading     `json:"headings,omitempty"`
	Images             ImageCounters `json:"images"`
}

type HeadingInfo struct {
	Present bool   `json:"present"`
	Text    string `json:"text,omitempty"`
}

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

type ImageCounters struct {
	Total  int `json:"total"`
	Local  int `json:"local"`
	Remote int `json:"remote"`
	AI     int `json:"ai"`
}

type AssetState struct {
	Index          int    `json:"index"`
	Kind           string `json:"kind"`
	Source         string `json:"source"`
	ResolvedSource string `json:"resolved_source,omitempty"`
	Exists         bool   `json:"exists"`
}

type Readiness struct {
	ConvertReady    bool               `json:"convert_ready"`
	UploadReady     bool               `json:"upload_ready"`
	DraftReady      bool               `json:"draft_ready"`
	PreviewFidelity string             `json:"preview_fidelity"`
	SchemaVersion   string             `json:"schema_version"`
	Targets         ReadinessTargets   `json:"targets"`
	Blockers        []ReadinessBlocker `json:"blockers"`
}

type Check struct {
	Level        string `json:"level"`
	Code         string `json:"code"`
	Message      string `json:"message"`
	Field        string `json:"field,omitempty"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
}

type ReadinessTargets struct {
	Preview string `json:"preview"`
	Convert string `json:"convert"`
	Upload  string `json:"upload"`
	Draft   string `json:"draft"`
}

type ReadinessBlocker struct {
	Code         string   `json:"code"`
	Field        string   `json:"field,omitempty"`
	Level        string   `json:"level"`
	Message      string   `json:"message"`
	Blocks       []string `json:"blocks"`
	SuggestedFix string   `json:"suggested_fix,omitempty"`
}

func Run(input *Input) (*Result, error) {
	if input == nil {
		return nil, fmt.Errorf("inspect input is required")
	}
	if input.MarkdownFile == "" {
		return nil, fmt.Errorf("markdown file is required")
	}

	doc := converter.ParseArticleDocument(input.Markdown)
	fm, _, hasFrontMatter := parseFrontMatterSource(input.Markdown)
	mode := firstNonEmpty(input.Mode, "api")
	if mode != "api" && mode != "ai" {
		return nil, fmt.Errorf("invalid convert mode: %s", mode)
	}
	theme := firstNonEmpty(input.Theme, "default")
	fontSize := firstNonEmpty(input.FontSize, "medium")
	backgroundType := firstNonEmpty(input.BackgroundType, "none")

	title, titleSource := resolveTitle(input.TitleOverride, doc, fm, hasFrontMatter)
	author, authorSource := resolveAuthor(input.AuthorOverride, fm, hasFrontMatter)
	digest, digestSource := resolveDigest(input.DigestOverride, fm, hasFrontMatter)

	result := &Result{
		SourceFile: input.MarkdownFile,
		SourceDir:  filepath.Dir(input.MarkdownFile),
		Context: Context{
			Mode:           mode,
			Theme:          theme,
			FontSize:       fontSize,
			BackgroundType: backgroundType,
			CustomPrompt:   strings.TrimSpace(input.CustomPrompt) != "",
			Upload:         input.UploadRequested,
			Draft:          input.DraftRequested,
		},
		Metadata: MetadataState{
			Title:  buildMetadataField(title, titleSource, 32),
			Author: buildMetadataField(author, authorSource, 16),
			Digest: buildMetadataField(digest, digestSource, 128),
		},
		Body: doc.Body,
	}

	result.Structure.Headings, result.Structure.BodyH1 = parseHeadings(doc.Body)
	result.Structure.DuplicateTitleRisk = result.Structure.BodyH1.Present && result.Structure.BodyH1.Text == result.Metadata.Title.Value && result.Metadata.Title.Value != ""

	images := converter.ParseMarkdownImages(doc.Body)
	result.Structure.Images, result.Assets = inspectAssets(images, result.SourceDir)

	result.Readiness.PreviewFidelity = previewFidelityFor(mode, apiKeyAvailable(input.Config))
	result.Checks = buildChecks(input, result)
	result.Readiness.ConvertReady = convertReadyFor(mode, input.Config) && !hasBlockingCheck(result.Checks, blocksConvert)
	result.Readiness.UploadReady = uploadReadyFor(input.Config) && result.Readiness.ConvertReady && !hasBlockingCheck(result.Checks, blocksUpload)
	result.Readiness.DraftReady = result.Readiness.UploadReady && coverConfigured(input) && !hasBlockingCheck(result.Checks, blocksDraft)
	result.Readiness.SchemaVersion = ReadinessSchemaVersion
	result.Readiness.Targets = buildReadinessTargets(result)
	result.Readiness.Blockers = buildReadinessBlockers(result.Checks)
	return result, nil
}

func resolveTitle(override string, doc converter.ArticleDocument, fm frontMatterSource, hasFrontMatter bool) (string, string) {
	if value := strings.TrimSpace(override); value != "" {
		return value, "cli.title"
	}
	if hasFrontMatter {
		if value := strings.TrimSpace(fm.Title); value != "" {
			return value, "frontmatter.title"
		}
	}
	if value := strings.TrimSpace(converter.ParseMarkdownTitle(doc.Body)); value != "" && value != "未命名文章" {
		return value, "markdown.heading"
	}
	return "未命名文章", "fallback.untitled"
}

func resolveAuthor(override string, fm frontMatterSource, hasFrontMatter bool) (string, string) {
	if value := strings.TrimSpace(override); value != "" {
		return value, "cli.author"
	}
	if hasFrontMatter {
		if value := strings.TrimSpace(fm.Author); value != "" {
			return value, "frontmatter.author"
		}
	}
	return "", "fallback.empty"
}

func resolveDigest(override string, fm frontMatterSource, hasFrontMatter bool) (string, string) {
	if value := strings.TrimSpace(override); value != "" {
		return value, "cli.digest"
	}
	if !hasFrontMatter {
		return "", "fallback.empty"
	}
	if value := strings.TrimSpace(fm.Digest); value != "" {
		return value, "frontmatter.digest"
	}
	if value := strings.TrimSpace(fm.Summary); value != "" {
		return value, "frontmatter.summary"
	}
	if value := strings.TrimSpace(fm.Description); value != "" {
		return value, "frontmatter.description"
	}
	return "", "fallback.empty"
}

type frontMatterSource struct {
	Title       string `yaml:"title"`
	Author      string `yaml:"author"`
	Digest      string `yaml:"digest"`
	Summary     string `yaml:"summary"`
	Description string `yaml:"description"`
}

func parseFrontMatterSource(markdown string) (frontMatterSource, string, bool) {
	normalized := strings.ReplaceAll(strings.TrimPrefix(markdown, "\uFEFF"), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	var fm frontMatterSource
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return fm, markdown, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "---" {
			continue
		}
		frontMatterBody := strings.Join(lines[1:i], "\n")
		if err := yaml.Unmarshal([]byte(frontMatterBody), &fm); err != nil {
			return frontMatterSource{}, markdown, false
		}
		body := strings.Join(lines[i+1:], "\n")
		return fm, body, true
	}
	return frontMatterSource{}, markdown, false
}

func buildMetadataField(value, source string, limit int) MetadataField {
	length := utf8.RuneCountInString(value)
	return MetadataField{
		Value:  value,
		Source: source,
		Limit:  limit,
		Length: length,
		Valid:  length <= limit,
	}
}

func parseHeadings(markdown string) ([]Heading, HeadingInfo) {
	lines := strings.Split(markdown, "\n")
	headings := make([]Heading, 0)
	bodyH1 := HeadingInfo{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		level := 0
		for level < len(line) && line[level] == '#' {
			level++
		}
		text := strings.TrimSpace(line[level:])
		if level == 0 || text == "" {
			continue
		}
		headings = append(headings, Heading{Level: level, Text: text})
		if level == 1 && !bodyH1.Present {
			bodyH1 = HeadingInfo{Present: true, Text: text}
		}
	}
	return headings, bodyH1
}

func inspectAssets(images []converter.ImageRef, sourceDir string) (ImageCounters, []AssetState) {
	counts := ImageCounters{Total: len(images)}
	assets := make([]AssetState, 0, len(images))
	for _, img := range images {
		asset := AssetState{
			Index:  img.Index,
			Kind:   string(img.Type),
			Source: img.Original,
		}
		switch img.Type {
		case converter.ImageTypeLocal:
			counts.Local++
			resolved := img.Original
			if resolved != "" && !filepath.IsAbs(resolved) {
				resolved = filepath.Join(sourceDir, resolved)
			}
			asset.ResolvedSource = resolved
			if resolved != "" {
				if _, err := os.Stat(resolved); err == nil {
					asset.Exists = true
				}
			}
		case converter.ImageTypeOnline:
			counts.Remote++
			asset.Exists = true
		case converter.ImageTypeAI:
			counts.AI++
			asset.Exists = true
		}
		assets = append(assets, asset)
	}
	return counts, assets
}

func buildChecks(input *Input, result *Result) []Check {
	checks := make([]Check, 0)
	appendFieldLimitCheck := func(field string, state MetadataField, code string) {
		if state.Valid {
			return
		}
		checks = append(checks, Check{
			Level:        LevelError,
			Code:         code,
			Message:      fmt.Sprintf("%s exceeds %d characters", field, state.Limit),
			Field:        field,
			SuggestedFix: fmt.Sprintf("shorten %s to %d characters or fewer", field, state.Limit),
		})
	}
	appendFieldLimitCheck("title", result.Metadata.Title, "TITLE_TOO_LONG")
	appendFieldLimitCheck("author", result.Metadata.Author, "AUTHOR_TOO_LONG")
	appendFieldLimitCheck("digest", result.Metadata.Digest, "DIGEST_TOO_LONG")

	if result.Structure.DuplicateTitleRisk {
		checks = append(checks, Check{
			Level:        LevelWarn,
			Code:         "DUPLICATE_H1",
			Message:      "Body H1 matches the final article title",
			Field:        "title",
			SuggestedFix: "remove the body H1 or change the final title",
		})
	} else if result.Structure.BodyH1.Present && strings.TrimSpace(result.Metadata.Title.Value) != "" && result.Structure.BodyH1.Text != result.Metadata.Title.Value {
		checks = append(checks, Check{
			Level:        LevelInfo,
			Code:         "TITLE_BODY_MISMATCH",
			Message:      "Final article title differs from the body H1",
			Field:        "title",
			SuggestedFix: "confirm whether draft metadata title and body H1 are intended to differ",
		})
	}

	if strings.TrimSpace(result.Metadata.Digest.Value) != "" {
		checks = append(checks, Check{
			Level:        LevelInfo,
			Code:         "DIGEST_METADATA_ONLY",
			Message:      "Digest affects draft metadata, not body HTML rendering",
			Field:        "digest",
			SuggestedFix: "use body content or theme behavior to control visible article summary inside the HTML",
		})
	}

	for _, asset := range result.Assets {
		if asset.Kind != string(converter.ImageTypeLocal) {
			continue
		}
		if !asset.Exists {
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "LOCAL_IMAGE_MISSING",
				Message:      fmt.Sprintf("Local image not found: %s", asset.Source),
				Field:        "images",
				SuggestedFix: "fix the image path or remove the missing image reference",
			})
			continue
		}
		info, err := os.Stat(asset.ResolvedSource)
		if err != nil {
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "LOCAL_IMAGE_MISSING",
				Message:      fmt.Sprintf("Local image not found: %s", asset.Source),
				Field:        "images",
				SuggestedFix: "fix the image path or remove the missing image reference",
			})
			continue
		}
		if !info.Mode().IsRegular() {
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "LOCAL_IMAGE_NOT_FILE",
				Message:      fmt.Sprintf("Local image is not a regular file: %s", asset.Source),
				Field:        "images",
				SuggestedFix: "replace the image path with an existing regular image file",
			})
			continue
		}
		if !image.IsValidImageFormat(asset.ResolvedSource) {
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "LOCAL_IMAGE_UNSUPPORTED_FORMAT",
				Message:      fmt.Sprintf("Local image has an unsupported format: %s", asset.Source),
				Field:        "images",
				SuggestedFix: "use a supported image file: jpg, jpeg, png, gif, bmp, or webp",
			})
		}
	}

	if len(result.Assets) > 0 && !input.UploadRequested && !input.DraftRequested {
		checks = append(checks, Check{
			Level:        LevelInfo,
			Code:         "IMAGE_REPLACEMENT_REQUIRES_UPLOAD_OR_DRAFT",
			Message:      "Images are only uploaded and replaced during upload or draft flows",
			Field:        "images",
			SuggestedFix: "use --upload or --draft when you need local/remote/AI image references rewritten to published URLs",
		})
	}

	if result.Context.Mode == "api" && !apiKeyAvailable(input.Config) {
		checks = append(checks, Check{
			Level:        LevelError,
			Code:         "MISSING_API_KEY",
			Message:      "API mode requires MD2WECHAT_API_KEY",
			Field:        "mode",
			SuggestedFix: "set MD2WECHAT_API_KEY or switch to --mode ai",
		})
	}

	appendThemeCompatibilityCheck(result, &checks)

	if result.Context.Mode == "ai" {
		checks = append(checks, Check{
			Level:        LevelInfo,
			Code:         "AI_MODE_ACTION_REQUIRED",
			Message:      "AI mode produces a prompt/request instead of final HTML until an external model completes the flow",
			Field:        "mode",
			SuggestedFix: "use --mode api for exact preview or complete the AI prompt externally",
		})
	}

	if input.UploadRequested || input.DraftRequested {
		if !uploadReadyFor(input.Config) {
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "MISSING_WECHAT_CONFIG",
				Message:      "Upload and draft flows require WECHAT_APPID and WECHAT_SECRET",
				Field:        "config",
				SuggestedFix: "set WECHAT_APPID and WECHAT_SECRET or configure them in ~/.config/md2wechat/config.yaml",
			})
		}
		if publishAPIKeyRequired(input.Config) && !apiKeyAvailable(input.Config) {
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "MISSING_PUBLISH_API_KEY",
				Message:      "Named-account and proxy publish paths require MD2WECHAT_API_KEY",
				Field:        "config",
				SuggestedFix: "set MD2WECHAT_API_KEY before upload or draft, or use direct WeChat credentials without a named account or proxy",
			})
		}
	}

	if input.DraftRequested && strings.TrimSpace(input.CoverImagePath) != "" && strings.TrimSpace(input.CoverMediaID) != "" {
		checks = append(checks, Check{
			Level:        LevelError,
			Code:         "CONFLICTING_COVER_INPUTS",
			Message:      "Draft mode cannot use --cover and --cover-media-id together",
			Field:        "cover",
			SuggestedFix: "choose either a local cover path or an existing WeChat media_id",
		})
	}
	if input.DraftRequested && strings.TrimSpace(input.CoverMediaID) != "" && looksLikeURL(input.CoverMediaID) {
		checks = append(checks, Check{
			Level:        LevelError,
			Code:         "COVER_MEDIA_ID_INVALID",
			Message:      fmt.Sprintf("Cover media_id looks like a URL: %s", input.CoverMediaID),
			Field:        "cover_media_id",
			SuggestedFix: "pass a WeChat media_id to --cover-media-id, or use --cover /path/to/cover.jpg for a local file",
		})
	}
	if input.DraftRequested && !coverConfigured(input) {
		checks = append(checks, Check{
			Level:        LevelError,
			Code:         "MISSING_COVER",
			Message:      "Draft mode requires --cover or --cover-media-id",
			Field:        "cover",
			SuggestedFix: "pass --cover /path/to/cover.jpg or --cover-media-id <media_id>",
		})
	}
	if input.DraftRequested && strings.TrimSpace(input.CoverImagePath) != "" {
		info, err := os.Stat(input.CoverImagePath)
		switch {
		case err != nil:
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "COVER_IMAGE_MISSING",
				Message:      fmt.Sprintf("Cover image not found: %s", input.CoverImagePath),
				Field:        "cover",
				SuggestedFix: "fix the --cover path or provide an existing local image file",
			})
		case !info.Mode().IsRegular():
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "COVER_IMAGE_NOT_FILE",
				Message:      fmt.Sprintf("Cover image is not a regular file: %s", input.CoverImagePath),
				Field:        "cover",
				SuggestedFix: "pass --cover with an existing regular image file",
			})
		case !image.IsValidImageFormat(input.CoverImagePath):
			checks = append(checks, Check{
				Level:        LevelError,
				Code:         "COVER_IMAGE_UNSUPPORTED_FORMAT",
				Message:      fmt.Sprintf("Cover image has an unsupported format: %s", input.CoverImagePath),
				Field:        "cover",
				SuggestedFix: "use a supported cover image: jpg, jpeg, png, gif, bmp, or webp",
			})
		}
	}

	return checks
}

func appendThemeCompatibilityCheck(result *Result, checks *[]Check) {
	mode := converter.ConvertMode(strings.TrimSpace(result.Context.Mode))
	if mode == "" {
		mode = converter.ModeAPI
	}
	theme := strings.TrimSpace(result.Context.Theme)
	if theme == "" {
		theme = "default"
	}
	if mode == converter.ModeAI && result.Context.CustomPrompt {
		return
	}

	themeManager := converter.NewThemeManager()
	if _, err := themeManager.ResolveThemeForMode(mode, theme); err != nil {
		code := "THEME_INVALID"
		switch {
		case converter.IsThemeNotFound(err):
			code = "THEME_NOT_FOUND"
		case converter.IsThemeNotSelectable(err):
			code = "THEME_NOT_SELECTABLE"
		case converter.IsThemeModeMismatch(err):
			code = "THEME_MODE_MISMATCH"
		}

		*checks = append(*checks, Check{
			Level:        LevelError,
			Code:         code,
			Message:      err.Error(),
			Field:        "theme",
			SuggestedFix: "run md2wechat themes list --json and choose a selectable theme for the selected mode",
		})
	}
}

func hasBlockingCheck(checks []Check, blocks func(string) bool) bool {
	for _, check := range checks {
		if check.Level == LevelError && blocks(check.Code) {
			return true
		}
	}
	return false
}

func blocksConvert(code string) bool {
	switch code {
	case "TITLE_TOO_LONG", "AUTHOR_TOO_LONG", "DIGEST_TOO_LONG", "MISSING_API_KEY",
		"THEME_NOT_FOUND", "THEME_NOT_SELECTABLE", "THEME_MODE_MISMATCH", "THEME_INVALID":
		return true
	default:
		return false
	}
}

func blocksUpload(code string) bool {
	if blocksConvert(code) {
		return true
	}
	switch code {
	case "LOCAL_IMAGE_MISSING", "LOCAL_IMAGE_NOT_FILE", "LOCAL_IMAGE_UNSUPPORTED_FORMAT",
		"MISSING_WECHAT_CONFIG", "MISSING_PUBLISH_API_KEY":
		return true
	default:
		return false
	}
}

func blocksDraft(code string) bool {
	if blocksUpload(code) {
		return true
	}
	switch code {
	case "MISSING_COVER", "COVER_IMAGE_MISSING", "COVER_IMAGE_NOT_FILE", "COVER_IMAGE_UNSUPPORTED_FORMAT",
		"CONFLICTING_COVER_INPUTS", "COVER_MEDIA_ID_INVALID":
		return true
	default:
		return false
	}
}

func buildReadinessTargets(result *Result) ReadinessTargets {
	return ReadinessTargets{
		Preview: previewReadinessTarget(result.Readiness.PreviewFidelity),
		Convert: readyReadinessTarget(result.Readiness.ConvertReady),
		Upload:  requestedReadinessTarget(result.Context.Upload || result.Context.Draft, result.Readiness.UploadReady),
		Draft:   requestedReadinessTarget(result.Context.Draft, result.Readiness.DraftReady),
	}
}

func readyReadinessTarget(ready bool) string {
	if ready {
		return ReadinessTargetReady
	}
	return ReadinessTargetBlocked
}

func requestedReadinessTarget(requested, ready bool) string {
	if !requested {
		return ReadinessTargetNotRequested
	}
	return readyReadinessTarget(ready)
}

func previewReadinessTarget(fidelity string) string {
	if fidelity == PreviewFidelityExact {
		return ReadinessTargetReady
	}
	return ReadinessTargetDegraded
}

func buildReadinessBlockers(checks []Check) []ReadinessBlocker {
	blockers := make([]ReadinessBlocker, 0)
	for _, check := range checks {
		if check.Level != LevelError {
			continue
		}
		blocks := blockedReadinessTargets(check.Code)
		if len(blocks) == 0 {
			continue
		}
		blockers = append(blockers, ReadinessBlocker{
			Code:         check.Code,
			Field:        check.Field,
			Level:        check.Level,
			Message:      check.Message,
			Blocks:       blocks,
			SuggestedFix: check.SuggestedFix,
		})
	}
	return blockers
}

func blockedReadinessTargets(code string) []string {
	switch {
	case blocksConvert(code):
		return []string{"convert", "upload", "draft"}
	case blocksUpload(code):
		return []string{"upload", "draft"}
	case blocksDraft(code):
		return []string{"draft"}
	default:
		return nil
	}
}

func coverConfigured(input *Input) bool {
	return strings.TrimSpace(input.CoverImagePath) != "" || strings.TrimSpace(input.CoverMediaID) != ""
}

func looksLikeURL(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func apiKeyAvailable(cfg *config.Config) bool {
	return cfg != nil && strings.TrimSpace(cfg.MD2WechatAPIKey) != ""
}

func publishAPIKeyRequired(cfg *config.Config) bool {
	return cfg != nil && (cfg.WechatAccountNamed || strings.TrimSpace(cfg.WechatProxyURL) != "")
}

func convertReadyFor(mode string, cfg *config.Config) bool {
	if mode == "ai" {
		return true
	}
	return apiKeyAvailable(cfg)
}

func uploadReadyFor(cfg *config.Config) bool {
	return cfg != nil && cfg.ValidateForWeChat() == nil
}

func previewFidelityFor(mode string, apiKeyReady bool) string {
	if mode == "api" && apiKeyReady {
		return PreviewFidelityExact
	}
	return PreviewFidelityDegraded
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
