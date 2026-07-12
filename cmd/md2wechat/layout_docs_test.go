package main

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
)

var layoutCountAnchorRE = regexp.MustCompile(`<!-- layout-count-contract: recommended_scenarios=(\d+) recommended_syntaxes=(\d+) compatibility_modules=(\d+) base_enhancements=(\d+) render_syntaxes=(\d+) -->`)
var markdownCodeFenceRE = regexp.MustCompile("(?ms)^```[^\\n]*\\n(.*?)^```[ \\t]*$")
var concreteLayoutOpenerRE = regexp.MustCompile(`(?m)^:::[a-z]`)

func TestLayoutDocumentationCountContract(t *testing.T) {
	files := []string{
		"../../README.md",
		"../../docs/README.md",
		"../../docs/LAYOUT.md",
		"../../docs/DISCOVERY.md",
		"../../docs/FAQ.md",
		"../../docs/AGENT-GUIDE.md",
		"../../AGENTS.md",
		"../../.github/copilot-instructions.md",
	}

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, stale := range []string{"43" + " 个", "43" + " embedded", "43" + " built-in", "6" + " categories"} {
			if strings.Contains(text, stale) {
				t.Errorf("%s has stale layout wording %q", path, stale)
			}
		}
	}

	layoutText := readDocumentationFile(t, "../../docs/LAYOUT.md")
	match := layoutCountAnchorRE.FindStringSubmatch(layoutText)
	if match == nil {
		t.Fatal("docs/LAYOUT.md must define the semantic layout-count-contract anchor")
	}
	counts := make([]int, 0, len(match)-1)
	for _, raw := range match[1:] {
		value, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatal(err)
		}
		counts = append(counts, value)
	}
	if want := []int{68, 53, 3, 4, 60}; !equalInts(counts, want) {
		t.Fatalf("layout count contract = %v, want %v", counts, want)
	}
	if counts[1]+counts[2]+counts[3] != counts[4] {
		t.Fatalf("render syntax relationship invalid: %d + %d + %d != %d", counts[1], counts[2], counts[3], counts[4])
	}
	if counts[0] <= counts[1] {
		t.Fatalf("scenario count %d must remain a broader dimension than syntax count %d", counts[0], counts[1])
	}

	discoveryText := readDocumentationFile(t, "../../docs/DISCOVERY.md")
	semanticCounts := map[string]int{
		"recommended_scenario_count": 68,
		"recommended_syntax_count":   53,
		"compatibility_module_count": 3,
		"base_enhancement_count":     4,
		"render_syntax_count":        60,
	}
	for key, value := range semanticCounts {
		pattern := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s*:\s*` + strconv.Itoa(value) + `\b`)
		if !pattern.MatchString(discoveryText) {
			t.Errorf("docs/DISCOVERY.md must define %s=%d", key, value)
		}
	}

	readmeText := readDocumentationFile(t, "../../README.md")
	for _, value := range counts {
		if !regexp.MustCompile(`\b` + strconv.Itoa(value) + `\b`).MatchString(readmeText) {
			t.Errorf("README.md must expose layout count %d and link to the semantic contract", value)
		}
	}
	if !strings.Contains(readmeText, "docs/LAYOUT.md") {
		t.Error("README.md must link to docs/LAYOUT.md as the semantic count contract")
	}

	releaseCheck := readDocumentationFile(t, "../../scripts/release-check.sh")
	if !strings.Contains(releaseCheck, "-run '^TestLayoutDocumentation'") {
		t.Error("release-check must execute the layout documentation contracts")
	}
	if strings.Contains(releaseCheck, "grep -q '53 个主推'") || strings.Contains(releaseCheck, "grep -q '68 个主推'") {
		t.Error("release-check must not depend on locale-specific layout count prose")
	}
}

func TestLayoutDocumentationBodyFormatTroubleshootingContract(t *testing.T) {
	text := readDocumentationFile(t, "../../docs/LAYOUT.md")
	start := strings.Index(text, "### 错误 5：")
	if start < 0 {
		t.Fatal("docs/LAYOUT.md must retain the body-format troubleshooting section")
	}
	section := text[start:]
	if end := strings.Index(section, "\n---"); end >= 0 {
		section = section[:end]
	}
	for _, format := range []string{"fields", "rows", "json_object", "json_array", "markdown_images", "markdown_fields", "split", "lines", "dialogue"} {
		if !strings.Contains(section, "`"+format+"`") {
			t.Errorf("body-format troubleshooting must cover %q", format)
		}
	}
	for _, phrase := range []string{"question", "primary `dialogue`", "legacy-compatible `json_array`", "Opener", "Body", "Rows", "Fields"} {
		if !strings.Contains(section, phrase) {
			t.Errorf("body-format troubleshooting must define %q", phrase)
		}
	}
}

func TestLayoutDocumentationDeclaresBuiltinOnlyCatalog(t *testing.T) {
	files := []string{
		"../../README.md",
		"../../docs/README.md",
		"../../docs/LAYOUT.md",
		"../../docs/DISCOVERY.md",
		"../../docs/FAQ.md",
		"../../docs/AGENT-GUIDE.md",
		"../../AGENTS.md",
		"../../.github/copilot-instructions.md",
		"../../skills/md2wechat/SKILL.md",
		"../../platforms/openclaw/md2wechat/SKILL.md",
	}
	for _, path := range files {
		text := readDocumentationFile(t, path)
		for _, obsolete := range []string{
			"MD2WECHAT_LAYOUT_DIR",
			"~/.config/md2wechat/layout",
			"effective_recommended_syntax_count",
			"effective_compatibility_module_count",
			"local_override_module_count",
			"remote_renderer_available",
			"Module Override",
			"自定义模块",
			"custom layout module",
		} {
			if strings.Contains(text, obsolete) {
				t.Errorf("%s still documents removed layout override contract %q", path, obsolete)
			}
		}
	}
	discovery := readDocumentationFile(t, "../../docs/DISCOVERY.md")
	if !strings.Contains(discovery, "内置 catalog 是唯一事实源") {
		t.Error("docs/DISCOVERY.md must declare the embedded catalog as the single source of truth")
	}
	releaseCheck := readDocumentationFile(t, "../../scripts/release-check.sh")
	if !strings.Contains(releaseCheck, "-run '^TestLayoutDocumentation'") {
		t.Error("release-check must execute the complete layout documentation contract set")
	}
}

func TestLayoutDocumentationSeparatesLocalValidationFromRemoteConformance(t *testing.T) {
	agentGuide := readDocumentationFile(t, "../../docs/AGENT-GUIDE.md")
	if strings.Contains(agentGuide, `LAYOUT_VALIDATED" | 排版语法正确 | 可以转换`) ||
		strings.Contains(agentGuide, `LAYOUT_VALIDATED`+"` → 语法正确，可以转换") {
		t.Error("docs/AGENT-GUIDE.md must not treat local validation as remote rendering proof")
	}
	if !strings.Contains(agentGuide, "本地 catalog/schema") {
		t.Error("docs/AGENT-GUIDE.md must state the local validation boundary")
	}
	dateStampedResult := regexp.MustCompile(`20\d\d-\d\d-\d\d[^\n]*80 pass`)
	for _, path := range []string{"../../docs/LAYOUT.md", "../../docs/SMOKE.md"} {
		if dateStampedResult.MatchString(readDocumentationFile(t, path)) {
			t.Errorf("%s must not embed a dated one-off conformance result", path)
		}
	}
}

func TestLayoutDocumentationE2EFixtureMatchesCurrentCatalog(t *testing.T) {
	markdown := readDocumentationFile(t, "../../examples/layout-e2e-test.md")
	catalog := layoutcatalog.NewCatalog()
	err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	report := catalog.Validate(markdown)
	if len(report.Errors) != 0 || len(report.Warnings) != 0 {
		t.Fatalf("layout E2E fixture drifted: errors=%+v warnings=%+v", report.Errors, report.Warnings)
	}
}

func TestLayoutDocumentationConcreteExamplesMatchCatalog(t *testing.T) {
	text := readDocumentationFile(t, "../../docs/LAYOUT.md")
	catalog := layoutcatalog.NewCatalog()
	if err := catalog.Load(); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, match := range markdownCodeFenceRE.FindAllStringSubmatch(text, -1) {
		snippet := match[1]
		if !concreteLayoutOpenerRE.MatchString(snippet) {
			continue
		}
		checked++
		report := catalog.Validate(snippet)
		if len(report.Errors) != 0 || len(report.Warnings) != 0 {
			t.Errorf("concrete layout snippet %d drifted: errors=%+v warnings=%+v\n%s", checked, report.Errors, report.Warnings, snippet)
		}
	}
	if checked < 20 {
		t.Fatalf("checked %d concrete layout snippets, want at least 20", checked)
	}
}

func readDocumentationFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func equalInts(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
