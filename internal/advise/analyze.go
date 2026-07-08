package advise

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"gopkg.in/yaml.v3"
)

const (
	defaultSourceFile = "article.md"
	maxActions        = 3
)

var (
	orderedListLinePattern = regexp.MustCompile(`^\s*\d+[\.)]\s+`)
	metricPattern          = regexp.MustCompile(`(?i)\d+(?:\.\d+)?\s*(?:%|％|倍|x|元|万元|亿元|人|次|小时|分钟|篇|个|k|m)`)
)

// Analyze returns deterministic, local-only enhancement advice for a Markdown article.
func Analyze(input Input) (*Result, error) {
	if strings.TrimSpace(input.Markdown) == "" {
		return nil, errors.New("markdown is required")
	}

	source := strings.TrimSpace(input.SourceFile)
	if source == "" {
		source = defaultSourceFile
	}

	doc := converter.ParseArticleDocument(input.Markdown)
	stats, facts := analyzeBody(doc.Body)
	hasTitle := facts.h1Count > 0 || hasFrontmatterTitle(input.Markdown)

	result := &Result{
		SchemaVersion: SchemaVersion,
		Command:       "advise",
		Source:        source,
		Decision:      DecisionNoEnhancementNeeded,
		Article: ArticleProfile{
			Type:          inferArticleType(facts),
			Confidence:    inferConfidence(facts),
			Stats:         stats,
			Signals:       articleSignals(hasTitle, facts),
			Uncertainties: articleUncertainties(hasTitle, facts),
		},
		Actions:    []Action{},
		Layout:     buildLayoutAdvice(facts),
		MicroEdits: buildMicroEdits(facts),
		DoNotDo:    []string{},
	}

	if !hasTitle {
		result.Actions = appendAction(result.Actions, Action{
			Tool:                 ToolTitle,
			State:                ActionStateRecommended,
			Reason:               "article has no observable title",
			Evidence:             []Evidence{messageEvidence("missing_title", "No H1 heading or frontmatter title was found.")},
			CommandHint:          fmt.Sprintf("md2wechat title suggest %s --json", shellQuote(source)),
			CommandArgs:          []string{"md2wechat", "title", "suggest", source, "--json"},
			RequiresConfirmation: true,
		})
	}

	if len(result.Layout.Recommended) > 0 {
		result.Actions = appendAction(result.Actions, Action{
			Tool:                 ToolLayout,
			State:                ActionStateRecommended,
			Reason:               "article structure can support a small number of layout modules",
			Evidence:             []Evidence{messageEvidence("layout_candidates", "Observable structure supports deterministic layout advice.")},
			CommandHint:          "md2wechat layout list --json",
			RequiresConfirmation: true,
		})
	}

	if facts.imageCount == 0 && enoughContentForCover(stats) {
		result.Actions = appendAction(result.Actions, Action{
			Tool:                 ToolCover,
			State:                ActionStateOptional,
			Reason:               "article has enough content for a planned cover but no image",
			Evidence:             []Evidence{messageEvidence("no_images", "No Markdown image references were found in the parsed article body.")},
			CommandHint:          fmt.Sprintf("md2wechat generate_cover --article %s --plan --json", shellQuote(source)),
			CommandArgs:          []string{"md2wechat", "generate_cover", "--article", source, "--plan", "--json"},
			RequiresConfirmation: true,
		})
	}

	if len(result.Actions) > 0 || len(result.Layout.Recommended) > 0 || len(result.MicroEdits) > 0 {
		result.Decision = DecisionEnhanceMinimally
	}

	return result, nil
}

type bodyFacts struct {
	h1Count          int
	h2Count          int
	paragraphCount   int
	imageCount       int
	blockquoteCount  int
	orderedListCount int
	wordCount        int
	hasMetrics       bool
	hasCTA           bool
}

func analyzeBody(body string) (Stats, bodyFacts) {
	lines := strings.Split(normalizeNewlines(body), "\n")
	facts := bodyFacts{}
	inParagraph := false
	inFence := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if isFenceDelimiter(line) {
			inFence = !inFence
			inParagraph = false
			continue
		}
		if inFence {
			continue
		}
		if line == "" {
			inParagraph = false
			continue
		}

		facts.imageCount += strings.Count(line, "![")

		switch {
		case isH1Line(line):
			facts.h1Count++
			inParagraph = false
		case isH2Line(line):
			facts.h2Count++
			inParagraph = false
		case strings.HasPrefix(line, ">"):
			facts.blockquoteCount++
			inParagraph = false
		case orderedListLinePattern.MatchString(line):
			facts.orderedListCount++
			inParagraph = false
		case strings.HasPrefix(line, "!["):
			inParagraph = false
		default:
			if !inParagraph {
				facts.paragraphCount++
				inParagraph = true
			}
		}

		if lineHasMetric(line) {
			facts.hasMetrics = true
		}
		if lineHasCTA(line) {
			facts.hasCTA = true
		}
		facts.wordCount += approximateWordCount(line)
	}

	stats := Stats{
		WordCount:        facts.wordCount,
		H1Count:          facts.h1Count,
		H2Count:          facts.h2Count,
		ParagraphCount:   facts.paragraphCount,
		ImageCount:       facts.imageCount,
		BlockquoteCount:  facts.blockquoteCount,
		OrderedListCount: facts.orderedListCount,
	}

	return stats, facts
}

func buildLayoutAdvice(facts bodyFacts) LayoutAdvice {
	advice := LayoutAdvice{
		MaxModules:  3,
		Recommended: []ModuleAdvice{},
		Rejected:    []ModuleAdvice{},
	}

	if facts.h2Count >= 3 {
		advice.Recommended = appendModule(advice.Recommended, ModuleAdvice{
			Name:     "toc",
			State:    ActionStateRecommended,
			Reason:   "article has enough second-level sections for a table of contents",
			Evidence: []Evidence{messageEvidence("h2_count", fmt.Sprintf("Found %d H2 headings.", facts.h2Count))},
		})
	}
	if facts.orderedListCount >= 2 {
		advice.Recommended = appendModule(advice.Recommended, ModuleAdvice{
			Name:     "steps",
			State:    ActionStateRecommended,
			Reason:   "article contains multiple ordered-list steps",
			Evidence: []Evidence{messageEvidence("ordered_list", fmt.Sprintf("Found %d ordered-list lines.", facts.orderedListCount))},
		})
	}
	if facts.blockquoteCount > 0 {
		advice.Recommended = appendModule(advice.Recommended, ModuleAdvice{
			Name:     "quote",
			State:    ActionStateRecommended,
			Reason:   "article contains quote-ready blockquote content",
			Evidence: []Evidence{messageEvidence("blockquote", fmt.Sprintf("Found %d blockquote line(s).", facts.blockquoteCount))},
		})
	}
	if facts.hasMetrics {
		advice.Recommended = appendModule(advice.Recommended, ModuleAdvice{
			Name:     "metrics",
			State:    ActionStateRecommended,
			Reason:   "article contains numeric evidence suitable for a metrics module",
			Evidence: []Evidence{messageEvidence("numeric_metric", "Found numeric metric evidence.")},
		})
	} else {
		advice.Rejected = append(advice.Rejected, ModuleAdvice{
			Name:     "metrics",
			State:    ActionStateSkip,
			Reason:   "no numeric metric evidence found",
			Evidence: []Evidence{messageEvidence("no_metrics", "Metrics module requires observable numeric evidence.")},
		})
	}
	if facts.hasCTA {
		advice.Recommended = appendModule(advice.Recommended, ModuleAdvice{
			Name:     "cta",
			State:    ActionStateRecommended,
			Reason:   "article contains observable action intent",
			Evidence: []Evidence{messageEvidence("cta_intent", "Found action-oriented wording.")},
		})
	} else {
		advice.Rejected = append(advice.Rejected, ModuleAdvice{
			Name:     "cta",
			State:    ActionStateSkip,
			Reason:   "no action intent found",
			Evidence: []Evidence{messageEvidence("no_cta", "CTA module requires observable action intent.")},
		})
	}

	return advice
}

func buildMicroEdits(facts bodyFacts) []MicroEdit {
	edits := []MicroEdit{}
	if facts.paragraphCount <= 0 {
		return edits
	}
	if facts.h2Count == 0 && facts.paragraphCount >= 4 {
		edits = append(edits, MicroEdit{
			Kind:                 "section_breaks",
			Reason:               "multiple paragraphs have no H2 section boundary",
			Evidence:             []Evidence{messageEvidence("dense_paragraphs", fmt.Sprintf("Found %d paragraphs and no H2 headings.", facts.paragraphCount))},
			RequiresConfirmation: true,
		})
	}
	if facts.wordCount >= 700 && len(edits) < 2 {
		edits = append(edits, MicroEdit{
			Kind:                 "tighten_long_article",
			Reason:               "article is long enough to benefit from manual tightening",
			Evidence:             []Evidence{messageEvidence("word_count", fmt.Sprintf("Approximate word count is %d.", facts.wordCount))},
			RequiresConfirmation: true,
		})
	}
	return edits
}

func appendAction(actions []Action, action Action) []Action {
	if len(actions) >= maxActions {
		return actions
	}
	return append(actions, action)
}

func appendModule(modules []ModuleAdvice, module ModuleAdvice) []ModuleAdvice {
	if len(modules) >= 3 {
		return modules
	}
	return append(modules, module)
}

func hasFrontmatterTitle(markdown string) bool {
	type frontMatter struct {
		Title string `yaml:"title"`
	}

	normalized := normalizeNewlines(strings.TrimPrefix(markdown, "\uFEFF"))
	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "---" {
			continue
		}
		var fm frontMatter
		if err := yaml.Unmarshal([]byte(strings.Join(lines[1:i], "\n")), &fm); err != nil {
			return false
		}
		return strings.TrimSpace(fm.Title) != ""
	}
	return false
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func isH1Line(line string) bool {
	return strings.HasPrefix(line, "# ") && strings.TrimSpace(strings.TrimPrefix(line, "#")) != ""
}

func isH2Line(line string) bool {
	return strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "###") && strings.TrimSpace(strings.TrimPrefix(line, "##")) != ""
}

func isFenceDelimiter(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}

func articleSignals(hasTitle bool, facts bodyFacts) []string {
	signals := []string{}
	if hasTitle {
		signals = append(signals, "has_title")
	}
	if facts.orderedListCount >= 2 {
		signals = append(signals, "has_steps")
	}
	if facts.blockquoteCount > 0 {
		signals = append(signals, "has_quotes")
	}
	if facts.hasCTA {
		signals = append(signals, "has_cta")
	}
	if facts.hasMetrics {
		signals = append(signals, "has_metrics")
	}
	return signals
}

func articleUncertainties(hasTitle bool, facts bodyFacts) []string {
	uncertainties := []string{}
	if !hasTitle {
		uncertainties = append(uncertainties, "title_missing")
	}
	if facts.wordCount < 80 {
		uncertainties = append(uncertainties, "short_article")
	}
	return uncertainties
}

func inferArticleType(facts bodyFacts) string {
	switch {
	case facts.orderedListCount >= 2:
		return ArticleTypeTutorial
	case facts.hasMetrics:
		return ArticleTypeReport
	case facts.blockquoteCount > 0:
		return ArticleTypeEssay
	default:
		return ArticleTypeUnknown
	}
}

func inferConfidence(facts bodyFacts) string {
	if facts.orderedListCount >= 2 || facts.hasMetrics || facts.h2Count >= 3 {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

func enoughContentForCover(stats Stats) bool {
	return stats.WordCount >= 120 || stats.H2Count >= 2 || stats.ParagraphCount >= 4
}

func lineHasCTA(line string) bool {
	lower := strings.ToLower(line)
	negativeMarkers := []string{"没有行动号召", "没有 cta", "没有cta", "无行动号召", "no cta", "no call to action", "without a cta", "without cta"}
	for _, marker := range negativeMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	phrases := []string{
		"点击", "扫码", "关注", "订阅", "报名", "下载", "留言", "转发", "立即",
		"subscribe", "sign up", "signup", "register", "download", "contact us", "learn more",
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func lineHasMetric(line string) bool {
	if orderedListLinePattern.MatchString(line) {
		return false
	}
	return metricPattern.MatchString(line)
}

func approximateWordCount(line string) int {
	words := 0
	inASCIIWord := false
	for _, r := range stripMarkdownMarkers(line) {
		switch {
		case unicode.Is(unicode.Han, r):
			words++
			inASCIIWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inASCIIWord {
				words++
				inASCIIWord = true
			}
		default:
			inASCIIWord = false
		}
	}
	return words
}

func stripMarkdownMarkers(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "#>")
	line = orderedListLinePattern.ReplaceAllString(line, "")
	return strings.TrimSpace(strings.ReplaceAll(line, "![", ""))
}

func normalizeNewlines(markdown string) string {
	markdown = strings.TrimPrefix(markdown, "\uFEFF")
	return strings.ReplaceAll(markdown, "\r\n", "\n")
}

func messageEvidence(code, message string) Evidence {
	return Evidence{
		Code:    code,
		Message: message,
	}
}
