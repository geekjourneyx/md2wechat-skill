package layoutcatalog

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/assets"
	"gopkg.in/yaml.v3"
)

type upstreamAgentContractOracle struct {
	SourceCommit string                  `yaml:"source_commit"`
	SourceFile   string                  `yaml:"source_file"`
	SourceSHA256 string                  `yaml:"source_sha256"`
	Contracts    []upstreamAgentContract `yaml:"contracts"`
}

type upstreamAgentContract struct {
	Syntax   string              `yaml:"syntax"`
	Input    []string            `yaml:"input"`
	Required []string            `yaml:"required"`
	Optional []string            `yaml:"optional"`
	Enums    map[string][]string `yaml:"enums"`
	Defaults map[string]string   `yaml:"defaults"`
	Invalid  []string            `yaml:"invalid"`
	Ignored  []string            `yaml:"ignored"`
	Legacy   []string            `yaml:"legacy"`
}

type upstreamAgentContractProjectionOracle struct {
	SourceCommit string                            `yaml:"source_commit"`
	SourceFiles  []string                          `yaml:"source_files"`
	Projections  []upstreamAgentContractProjection `yaml:"projections"`
}

type upstreamAgentContractProjection struct {
	Syntax        string              `yaml:"syntax"`
	BodyFormat    string              `yaml:"body_format"`
	Applicability map[string][]string `yaml:"applicability"`
}

const upstreamAgentContractContentSHA256 = "c6ca6d8a26b1bc694a8cef72ff6c7d517366f4331bb7ee978dbdae5556636fbd"
const upstreamAgentContractProjectionSHA256 = "65db776313af832ede1b2fee087ea38ed55a264cec4ee5a1051d310b9fbd29f4"

func TestUpstreamAgentContractProjectionOracle(t *testing.T) {
	data, err := os.ReadFile("testdata/upstream_agent_contract_projections.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var oracle upstreamAgentContractProjectionOracle
	if err := yaml.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != upstreamAgentContractProjectionSHA256 {
		t.Fatalf("projection fixture digest = %q, want %q", got, upstreamAgentContractProjectionSHA256)
	}
	if oracle.SourceCommit != "edcde64ae1be56f1a08a0617bb1862471e7e00b1" {
		t.Fatalf("upstream source commit = %q", oracle.SourceCommit)
	}
	if want := []string{"__tests__/fixtures/advanced-layout-agent-contract.ts", "advanced-layout-modules-guide.md"}; !slices.Equal(oracle.SourceFiles, want) {
		t.Fatalf("projection source files = %v, want %v", oracle.SourceFiles, want)
	}
	if len(oracle.Projections) != 56 {
		t.Fatalf("projection count = %d, want 56", len(oracle.Projections))
	}
	seen := make(map[string]bool, len(oracle.Projections))
	for _, projection := range oracle.Projections {
		if projection.Syntax == "" || seen[projection.Syntax] {
			t.Fatalf("invalid duplicate projection syntax %q", projection.Syntax)
		}
		seen[projection.Syntax] = true
		if !ValidBodyFormats[projection.BodyFormat] || projection.Applicability == nil {
			t.Fatalf("projection %q has invalid body format/applicability", projection.Syntax)
		}
	}
	want := slices.Clone(recommendedModuleNames)
	slices.Sort(want)
	got := make([]string, 0, len(seen))
	for syntax := range seen {
		got = append(got, syntax)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("projection syntax names = %v, want %v", got, want)
	}
}

func TestUpstreamAgentContractOracle(t *testing.T) {
	oracle := readUpstreamAgentContracts(t)
	if oracle.SourceCommit != "edcde64ae1be56f1a08a0617bb1862471e7e00b1" {
		t.Fatalf("upstream source commit = %q", oracle.SourceCommit)
	}
	if oracle.SourceFile != "__tests__/fixtures/advanced-layout-agent-contract.ts" {
		t.Fatalf("upstream source file = %q", oracle.SourceFile)
	}
	if oracle.SourceSHA256 != "265b50ae88d3688614273423df1d5de7fddfb899fc7d496a6c88d37ec66ff1d3" {
		t.Fatalf("upstream source digest = %q", oracle.SourceSHA256)
	}
	if len(oracle.Contracts) != 56 {
		t.Fatalf("contract count = %d, want 56", len(oracle.Contracts))
	}

	seen := make(map[string]bool, len(oracle.Contracts))
	got := make([]string, 0, len(oracle.Contracts))
	for _, contract := range oracle.Contracts {
		if contract.Syntax == "" || seen[contract.Syntax] {
			t.Fatalf("invalid duplicate syntax %q", contract.Syntax)
		}
		seen[contract.Syntax] = true
		got = append(got, contract.Syntax)
	}
	want := slices.Clone(recommendedModuleNames)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("contract syntax names = %v, want %v", got, want)
	}
}

func TestUpstreamAgentContractOracleContentIntegrity(t *testing.T) {
	oracle := readUpstreamAgentContracts(t)
	for i, contract := range oracle.Contracts {
		if contract.Syntax == "" {
			t.Fatalf("contract %d has no syntax", i)
		}
		if contract.Input == nil || contract.Required == nil || contract.Optional == nil {
			t.Fatalf("contract %q omits an input/required/optional list", contract.Syntax)
		}
		if contract.Enums == nil || contract.Defaults == nil {
			t.Fatalf("contract %q omits an enums/defaults map", contract.Syntax)
		}
		if contract.Invalid == nil || contract.Ignored == nil || contract.Legacy == nil {
			t.Fatalf("contract %q omits an invalid/ignored/legacy list", contract.Syntax)
		}
		for field, values := range contract.Enums {
			if values == nil {
				t.Fatalf("contract %q enum %q omits its value list", contract.Syntax, field)
			}
		}
	}

	encoded, err := json.Marshal(oracle.Contracts)
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprintf("%x", sha256.Sum256(encoded))
	if got != upstreamAgentContractContentSHA256 {
		t.Fatalf("normalized contract digest = %q, want %q", got, upstreamAgentContractContentSHA256)
	}
}

func TestUpstreamAgentContractsUseSupportedInputPositions(t *testing.T) {
	oracle := readUpstreamAgentContracts(t)
	for _, contract := range oracle.Contracts {
		for _, input := range contract.Input {
			if !ValidAgentInputPositions[AgentInputPosition(input)] {
				t.Fatalf("contract %q input position %q is not supported by schema-v1", contract.Syntax, input)
			}
		}
	}
}

// TestBuiltinCatalogMatchesUpstreamAgentContracts keeps the author-facing
// contract in the builtin catalog aligned with the pinned upstream oracle.
// The upstream fields and ordered values are compared exactly. Canonical body
// format and branch applicability are checked against their generic runtime
// schema declarations; compatible parsing stays outside this authoring view.
func TestBuiltinCatalogMatchesUpstreamAgentContracts(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	projections := pinnedAgentContractProjectionsBySyntax(t)
	for _, want := range readUpstreamAgentContracts(t).Contracts {
		want := want
		t.Run(want.Syntax, func(t *testing.T) {
			spec, ok := c.Get(want.Syntax)
			if !ok {
				t.Fatal("missing recommended catalog entry")
			}
			got := projectAgentContract(spec)
			projection, ok := projections[want.Syntax]
			if !ok {
				t.Fatal("missing pinned body-format/applicability projection")
			}
			expected := expectedAgentContract(want, projection.BodyFormat, projection.Applicability)
			if diff := compareAgentContracts(got, expected); len(diff) != 0 {
				t.Errorf("agent contract differs in %v\n got: %#v\nwant: %#v", diff, got, expected)
			}
		})
	}
}

func TestPinnedProjectionRejectsCoordinatedRuntimeAndAgentContractDrift(t *testing.T) {
	base := func() *LayoutSpec {
		return &LayoutSpec{
			Name:       "cta",
			BodyFormat: BodyFormatFields,
			Fields:     &FieldsSpec{Optional: []FieldSpec{{Name: "points", AppliesTo: []string{"trial"}}}},
			AgentContract: &AgentContractSpec{
				BodyFormat:    BodyFormatFields,
				Applicability: map[string][]string{"points": {"trial"}},
			},
		}
	}
	projection := pinnedAgentContractProjectionsBySyntax(t)["cta"]
	for _, tt := range []struct {
		name, dimension string
		mutate          func(*LayoutSpec)
	}{
		{
			name: "body format", dimension: "body_format",
			mutate: func(spec *LayoutSpec) {
				spec.BodyFormat = BodyFormatRows
				spec.AgentContract.BodyFormat = BodyFormatRows
			},
		},
		{
			name: "applicability", dimension: "applicability",
			mutate: func(spec *LayoutSpec) {
				spec.Fields.Optional[0].AppliesTo = []string{"consult"}
				spec.AgentContract.Applicability["points"] = []string{"consult"}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spec := base()
			tt.mutate(spec)
			expected := projectAgentContract(base())
			expected.BodyFormat = projection.BodyFormat
			expected.Applicability = projection.Applicability
			if diff := compareAgentContracts(projectAgentContract(spec), expected); !slices.Equal(diff, []string{tt.dimension}) {
				t.Fatalf("coordinated drift dimensions = %v, want [%s]", diff, tt.dimension)
			}
		})
	}
}

type agentContractProjection struct {
	Syntax         string
	InputPositions []string
	BodyFormat     string
	Required       []string
	Optional       []string
	Enums          map[string][]string
	Defaults       map[string]string
	Applicability  map[string][]string
	Invalid        []string
	Ignored        []string
	Legacy         []string
}

func projectAgentContract(spec *LayoutSpec) agentContractProjection {
	contract := spec.AgentContract
	if contract == nil {
		return agentContractProjection{Syntax: spec.Name}
	}
	got := agentContractProjection{
		Syntax:        spec.Name,
		BodyFormat:    contract.BodyFormat,
		Required:      contract.Required,
		Optional:      contract.Optional,
		Enums:         contract.Enums,
		Defaults:      contract.Defaults,
		Applicability: contract.Applicability,
		Invalid:       contract.Invalid,
		Ignored:       contract.Ignored,
		Legacy:        contract.Legacy,
	}
	for _, input := range spec.InputPositions {
		got.InputPositions = append(got.InputPositions, string(input))
	}
	return got
}

func expectedAgentContract(want upstreamAgentContract, bodyFormat string, applicability map[string][]string) agentContractProjection {
	return agentContractProjection{
		Syntax:         want.Syntax,
		InputPositions: want.Input,
		BodyFormat:     bodyFormat,
		Required:       want.Required,
		Optional:       want.Optional,
		Enums:          want.Enums,
		Defaults:       want.Defaults,
		Applicability:  applicability,
		Invalid:        want.Invalid,
		Ignored:        want.Ignored,
		Legacy:         want.Legacy,
	}
}

func compareAgentContracts(got, want agentContractProjection) []string {
	var diff []string
	checks := []struct {
		name  string
		equal bool
	}{
		{"syntax", got.Syntax == want.Syntax},
		{"input_positions", slices.Equal(got.InputPositions, want.InputPositions)},
		{"body_format", got.BodyFormat == want.BodyFormat},
		{"required", slices.Equal(got.Required, want.Required)},
		{"optional", slices.Equal(got.Optional, want.Optional)},
		{"enums", reflect.DeepEqual(got.Enums, want.Enums)},
		{"defaults", reflect.DeepEqual(got.Defaults, want.Defaults)},
		{"applicability", reflect.DeepEqual(got.Applicability, want.Applicability)},
		{"invalid", slices.Equal(got.Invalid, want.Invalid)},
		{"ignored", slices.Equal(got.Ignored, want.Ignored)},
		{"legacy", slices.Equal(got.Legacy, want.Legacy)},
	}
	for _, check := range checks {
		if !check.equal {
			diff = append(diff, check.name)
		}
	}
	return diff
}

func TestAgentContractProjectionDetectsEveryMetadataDimensionDrift(t *testing.T) {
	projection := func() agentContractProjection {
		return agentContractProjection{
			Syntax:         "fixture",
			InputPositions: []string{"body-kv"},
			BodyFormat:     BodyFormatFields,
			Required:       []string{"title"},
			Optional:       []string{"variant"},
			Enums:          map[string][]string{"variant": {"plain", "card"}},
			Defaults:       map[string]string{"variant": "plain"},
			Applicability:  map[string][]string{"detail": {"card"}},
			Invalid:        []string{"blank-title"},
			Ignored:        []string{"unknown-fields"},
			Legacy:         []string{"case-folded-variant"},
		}
	}
	tests := []struct {
		name, dimension string
		mutate          func(*agentContractProjection)
	}{
		{name: "syntax", dimension: "syntax", mutate: func(got *agentContractProjection) { got.Syntax = "other" }},
		{name: "input positions", dimension: "input_positions", mutate: func(got *agentContractProjection) { got.InputPositions = append(got.InputPositions, "json") }},
		{name: "body format", dimension: "body_format", mutate: func(got *agentContractProjection) { got.BodyFormat = BodyFormatRows }},
		{name: "required", dimension: "required", mutate: func(got *agentContractProjection) { got.Required = nil }},
		{name: "optional", dimension: "optional", mutate: func(got *agentContractProjection) { got.Optional = nil }},
		{name: "required added to optional", dimension: "optional", mutate: func(got *agentContractProjection) { got.Optional = append(got.Optional, "title") }},
		{name: "catalog-only enum", dimension: "enums", mutate: func(got *agentContractProjection) { got.Enums["catalog-only"] = []string{"drift"} }},
		{name: "enum order", dimension: "enums", mutate: func(got *agentContractProjection) { got.Enums["variant"] = []string{"card", "plain"} }},
		{name: "missing default", dimension: "defaults", mutate: func(got *agentContractProjection) { delete(got.Defaults, "variant") }},
		{name: "default value", dimension: "defaults", mutate: func(got *agentContractProjection) { got.Defaults["variant"] = "card" }},
		{name: "applicability", dimension: "applicability", mutate: func(got *agentContractProjection) { got.Applicability["detail"] = []string{"plain"} }},
		{name: "invalid", dimension: "invalid", mutate: func(got *agentContractProjection) { got.Invalid = append(got.Invalid, "extra") }},
		{name: "ignored", dimension: "ignored", mutate: func(got *agentContractProjection) { got.Ignored = append(got.Ignored, "extra") }},
		{name: "legacy", dimension: "legacy", mutate: func(got *agentContractProjection) { got.Legacy = append(got.Legacy, "extra") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := projection()
			got := projection()
			tt.mutate(&got)
			if diff := compareAgentContracts(got, want); !slices.Equal(diff, []string{tt.dimension}) {
				t.Fatalf("drift dimensions = %v, want [%s]", diff, tt.dimension)
			}
		})
	}
}

func readUpstreamAgentContracts(t *testing.T) upstreamAgentContractOracle {
	t.Helper()
	data, err := os.ReadFile("testdata/upstream_agent_contracts.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var oracle upstreamAgentContractOracle
	if err := yaml.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	return oracle
}

func readUpstreamAgentContractProjections(t *testing.T) upstreamAgentContractProjectionOracle {
	t.Helper()
	data, err := os.ReadFile("testdata/upstream_agent_contract_projections.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var oracle upstreamAgentContractProjectionOracle
	if err := yaml.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	return oracle
}

func pinnedAgentContractProjectionsBySyntax(t *testing.T) map[string]upstreamAgentContractProjection {
	t.Helper()
	oracle := readUpstreamAgentContractProjections(t)
	bySyntax := make(map[string]upstreamAgentContractProjection, len(oracle.Projections))
	for _, projection := range oracle.Projections {
		bySyntax[projection.Syntax] = projection
	}
	return bySyntax
}

func TestBuiltinYAMLExplicitlyDeclaresLifecycle(t *testing.T) {
	categories, err := assets.ListBuiltinLayoutCategories()
	if err != nil {
		t.Fatal(err)
	}
	for _, category := range categories {
		names, err := assets.ListBuiltinLayouts(category)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			raw, err := assets.ReadBuiltinLayout(category, name)
			if err != nil {
				t.Fatal(err)
			}
			text := string(raw)
			if !strings.Contains(text, "\nlifecycle: recommended\n") && !strings.Contains(text, "\nlifecycle: compatibility\n") {
				t.Errorf("%s/%s does not explicitly declare a valid lifecycle", category, name)
			}
		}
	}
}

type imageModuleParamContract struct {
	name, defaultValue string
	enum               []string
}

type imageModuleFieldContract struct {
	name, example string
}

type imageModuleContract struct {
	format, category, lifecycle string
	minImages, maxImages        int
	paramStyle                  string
	params                      []imageModuleParamContract
	requiredFields              []imageModuleFieldContract
	optionalFields              []imageModuleFieldContract
	metadata                    LayoutMetadata
	example                     string
}

type freeLayoutContract struct {
	format, category, lifecycle string
	opener                      *OpenerSpec
	body                        *BodySpec
	rows                        *RowsSpec
	metadata                    LayoutMetadata
	example                     string
}

type compatibilityLayoutContract struct {
	format, replacement string
	opener              *OpenerSpec
	body                *BodySpec
	example             string
}

func TestFreeLayoutModuleContracts(t *testing.T) {
	contracts := map[string]freeLayoutContract{
		"split": {
			format: BodyFormatSplit, category: "free-layout", lifecycle: LifecycleRecommended,
			opener: &OpenerSpec{ParamStyle: ParamStyleBracket, Params: []ParamSpec{
				{Name: "ratio", Description: "左右两栏比例；省略时由渲染器使用默认 1:1", Example: "3:2"},
			}},
			body:     &BodySpec{Separator: "---", MinItems: 2},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#split"},
			example:  splitGuideSnippet,
		},
		"flow": {
			format: BodyFormatLines, category: "free-layout", lifecycle: LifecycleRecommended,
			opener:   &OpenerSpec{Caption: true},
			body:     &BodySpec{MinItems: 1},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#flow"},
			example:  flowGuideSnippet,
		},
		"matrix": {
			format: BodyFormatRows, category: "free-layout", lifecycle: LifecycleRecommended,
			opener: &OpenerSpec{ParamStyle: ParamStyleTokens, Params: []ParamSpec{
				{Name: "headers", Description: "逗号分隔的表头", Example: "能力,基础版,专业版,企业版"},
			}},
			rows: &RowsSpec{Delimiter: "|", MinColumns: 2, Schema: []FieldSpec{
				{Name: "dimension", Description: "对比维度"},
				{Name: "value", Description: "至少一个方案值"},
			}},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#matrix"},
			example:  matrixGuideSnippet,
		},
		"dialogue-pair": {
			format: BodyFormatDialogue, category: "free-layout", lifecycle: LifecycleRecommended,
			opener: &OpenerSpec{ParamStyle: ParamStyleTokens, Params: []ParamSpec{
				{Name: "left", Description: "左侧人物名称", Default: "用户"},
				{Name: "right", Description: "右侧人物名称", Default: "助手"},
			}},
			body:     &BodySpec{MinItems: 1, AllowedPrefixes: []string{"U:", "E:"}},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#dialogue-pair"},
			example:  dialoguePairGuideSnippet,
		},
	}

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for name, contract := range contracts {
		t.Run(name, func(t *testing.T) {
			spec, ok := c.Get(name)
			if !ok {
				t.Fatalf("missing %s", name)
			}
			if spec.BodyFormat != contract.format {
				t.Errorf("body_format = %q, want %q", spec.BodyFormat, contract.format)
			}
			if spec.Category != contract.category {
				t.Errorf("category = %q, want %q", spec.Category, contract.category)
			}
			if spec.Lifecycle != contract.lifecycle {
				t.Errorf("lifecycle = %q, want %q", spec.Lifecycle, contract.lifecycle)
			}
			if !reflect.DeepEqual(spec.Opener, contract.opener) {
				t.Errorf("opener = %#v, want %#v", spec.Opener, contract.opener)
			}
			if !reflect.DeepEqual(spec.Body, contract.body) {
				t.Errorf("body = %#v, want %#v", spec.Body, contract.body)
			}
			if !reflect.DeepEqual(spec.Rows, contract.rows) {
				t.Errorf("rows = %#v, want %#v", spec.Rows, contract.rows)
			}
			if !reflect.DeepEqual(spec.Metadata, contract.metadata) {
				t.Errorf("metadata = %#v, want %#v", spec.Metadata, contract.metadata)
			}
			if spec.Example != contract.example {
				t.Errorf("canonical example differs from pinned upstream guide snippet\ngot:  %q\nwant: %q", spec.Example, contract.example)
			}
			if err := c.ValidateWitness(WitnessContract{Module: name, Example: spec.Example}); err != nil {
				t.Fatalf("witness invalid: %v", err)
			}
		})
	}
}

func TestFreeLayoutModuleRejectedCases(t *testing.T) {
	tests := []struct {
		name, markdown, category string
	}{
		{name: "split missing separator", markdown: ":::split\n左侧\n右侧\n:::\n", category: "separator"},
		{name: "split unknown opener param", markdown: ":::split ratio=3:2\n左侧\n---\n右侧\n:::\n", category: "param_style"},
		{name: "flow empty", markdown: ":::flow[空流程]\n:::\n", category: "at least 1 items"},
		{name: "flow unknown opener param", markdown: ":::flow extra=value\n节点\n:::\n", category: "opener"},
		{name: "matrix markdown separator row", markdown: ":::matrix headers=能力,基础版,专业版\n| 能力 | 基础版 | 专业版 |\n|---|---|---|\n| 高级模块 | 有 | 有 |\n:::\n", category: "column 1"},
		{name: "matrix short row", markdown: ":::matrix headers=能力,基础版\n高级模块\n:::\n", category: "at least 2 columns"},
		{name: "matrix unknown opener param", markdown: ":::matrix columns=2\n高级模块|有\n:::\n", category: "undeclared opener parameter"},
		{name: "dialogue empty", markdown: ":::dialogue-pair\n:::\n", category: "at least 1 items"},
		{name: "dialogue unknown opener param", markdown: ":::dialogue-pair speaker=读者\nU: 问题\n:::\n", category: "undeclared opener parameter"},
		{name: "dialogue bracket personas", markdown: ":::dialogue-pair[读者 | 作者]\nU: 问题\n:::\n", category: "opener caption"},
		{name: "dialogue rejects unconfigured named speakers", markdown: ":::dialogue-pair\n读者：问题\n作者：回答\n:::\n", category: "not allowed"},
	}

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := c.Validate(tt.markdown)
			if len(report.Errors) == 0 {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(report.Errors[0].Message, tt.category) {
				t.Fatalf("error = %q, want category %q", report.Errors[0].Message, tt.category)
			}
		})
	}
}

func TestNewImageModuleContracts(t *testing.T) {
	contracts := map[string]imageModuleContract{
		"gallery-grid": {
			format: BodyFormatMarkdownImages, category: "evidence", lifecycle: LifecycleRecommended, minImages: 1,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "variant", enum: []string{"clean", "card"}, defaultValue: "clean"},
				{name: "density", enum: []string{"compact", "normal", "airy"}, defaultValue: "normal"},
				{name: "columns", enum: []string{"1", "2", "3", "4"}, defaultValue: "3"},
				{name: "image_shape", enum: []string{"square", "rounded", "phone", "poster", "original"}, defaultValue: "square"},
				{name: "caption_style", enum: []string{"none", "minimal", "numbered", "label"}, defaultValue: "minimal"},
				{name: "accent", enum: []string{"brand", "muted", "contrast"}, defaultValue: "brand"},
				{name: "wechat_safe_level", enum: []string{"strict", "normal"}, defaultValue: "normal"},
				{name: "title"}, {name: "description"},
			},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#gallery-grid"},
			example:  galleryGridGuideSnippet,
		},
		"gallery-story": {
			format: BodyFormatMarkdownImages, category: "evidence", lifecycle: LifecycleRecommended, minImages: 1,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "variant", enum: []string{"clean", "card"}, defaultValue: "card"},
				{name: "density", enum: []string{"compact", "normal", "airy"}, defaultValue: "normal"},
				{name: "caption_style", enum: []string{"none", "minimal", "numbered", "label"}, defaultValue: "minimal"},
				{name: "accent", enum: []string{"brand", "muted", "contrast"}, defaultValue: "brand"},
				{name: "wechat_safe_level", enum: []string{"strict", "normal"}, defaultValue: "normal"},
			},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#gallery-story"},
			example:  galleryStoryGuideSnippet,
		},
		"image-phone-shot": {
			format: BodyFormatMarkdownImages, category: "evidence", lifecycle: LifecycleRecommended, minImages: 1,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "columns", enum: []string{"1", "2", "3", "4"}, defaultValue: "2"},
				{name: "image_shape", enum: []string{"square", "rounded", "phone", "poster", "original"}, defaultValue: "phone"},
				{name: "variant", enum: []string{"clean", "card"}, defaultValue: "clean"},
				{name: "wechat_safe_level", enum: []string{"strict", "normal"}, defaultValue: "normal"}, {name: "title"},
			},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#image-phone-shot"},
			example:  imagePhoneShotGuideSnippet,
		},
		"figure-caption": {
			format: BodyFormatMarkdownFields, category: "evidence", lifecycle: LifecycleRecommended, minImages: 1, maxImages: 1,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "caption_style", enum: []string{"none", "minimal", "numbered", "label"}, defaultValue: "numbered"},
			},
			requiredFields: []imageModuleFieldContract{{name: "caption", example: "2026 年用户增长曲线"}},
			optionalFields: []imageModuleFieldContract{{name: "source", example: "内部实验数据"}},
			metadata:       LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#figure-caption"},
			example:        figureCaptionGuideSnippet,
		},
		"svg-reveal": {
			format: BodyFormatFields, category: "interactive", lifecycle: LifecycleRecommended,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "variant", enum: []string{"clean", "card"}, defaultValue: "card"},
				{name: "accent", enum: []string{"brand", "muted", "contrast"}, defaultValue: "brand"},
				{name: "density", enum: []string{"compact", "normal", "airy"}, defaultValue: "normal"},
				{name: "wechat_safe_level", enum: []string{"strict", "normal"}, defaultValue: "normal"},
				{name: "svg_fallback", enum: []string{"static", "first-layer"}, defaultValue: "first-layer"},
			},
			requiredFields: []imageModuleFieldContract{
				{name: "question", example: "点击查看答案"},
				{name: "answer", example: "42"},
			},
			optionalFields: []imageModuleFieldContract{},
			metadata:       LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#svg-reveal"},
			example:        svgRevealGuideSnippet,
		},
		"svg-swipe-gallery": {
			format: BodyFormatMarkdownImages, category: "interactive", lifecycle: LifecycleRecommended, minImages: 2,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "variant", enum: []string{"clean", "card"}, defaultValue: "card"},
				{name: "accent", enum: []string{"brand", "muted", "contrast"}, defaultValue: "brand"},
				{name: "density", enum: []string{"compact", "normal", "airy"}, defaultValue: "normal"},
				{name: "wechat_safe_level", enum: []string{"strict", "normal"}, defaultValue: "normal"},
				{name: "svg_fallback", enum: []string{"static", "first-layer"}, defaultValue: "first-layer"}, {name: "title"},
			},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#svg-swipe-gallery"},
			example:  svgSwipeGalleryGuideSnippet,
		},
	}

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for name, contract := range contracts {
		t.Run(name, func(t *testing.T) {
			spec, ok := c.Get(name)
			if !ok {
				t.Fatalf("missing %s", name)
			}
			if spec.BodyFormat != contract.format {
				t.Errorf("body_format = %q, want %q", spec.BodyFormat, contract.format)
			}
			if spec.Category != contract.category {
				t.Errorf("category = %q, want %q", spec.Category, contract.category)
			}
			if spec.Lifecycle != contract.lifecycle {
				t.Errorf("lifecycle = %q, want normalized %q", spec.Lifecycle, contract.lifecycle)
			}
			minImages, maxImages := bodyImageLimits(spec.Body)
			if minImages != contract.minImages || maxImages != contract.maxImages {
				t.Errorf("image limits = (%d, %d), want (%d, %d)", minImages, maxImages, contract.minImages, contract.maxImages)
			}
			if spec.Opener == nil {
				t.Fatal("missing opener contract")
			}
			if spec.Opener.ParamStyle != contract.paramStyle {
				t.Errorf("opener param_style = %q, want %q", spec.Opener.ParamStyle, contract.paramStyle)
			}
			if got := imageModuleParams(spec.Opener.Params); !reflect.DeepEqual(got, contract.params) {
				t.Errorf("opener params = %#v, want %#v", got, contract.params)
			}
			if got := imageModuleFields(spec.Fields, true); !reflect.DeepEqual(got, contract.requiredFields) {
				t.Errorf("required fields = %#v, want %#v", got, contract.requiredFields)
			}
			if got := imageModuleFields(spec.Fields, false); !reflect.DeepEqual(got, contract.optionalFields) {
				t.Errorf("optional fields = %#v, want %#v", got, contract.optionalFields)
			}
			if !reflect.DeepEqual(spec.Metadata, contract.metadata) {
				t.Errorf("metadata = %#v, want %#v", spec.Metadata, contract.metadata)
			}
			if spec.Example != contract.example {
				t.Errorf("canonical example differs from pinned upstream guide snippet\ngot:  %q\nwant: %q", spec.Example, contract.example)
			}
			if err := c.ValidateWitness(WitnessContract{Module: name, Example: spec.Example}); err != nil {
				t.Fatalf("witness invalid: %v", err)
			}
			if result := c.Validate(spec.Example); len(result.Errors) != 0 {
				t.Fatalf("example invalid: %+v", result.Errors)
			}
		})
	}
}

func bodyImageLimits(body *BodySpec) (int, int) {
	if body == nil {
		return 0, 0
	}
	return body.MinImages, body.MaxImages
}

func imageModuleParams(params []ParamSpec) []imageModuleParamContract {
	contracts := make([]imageModuleParamContract, 0, len(params))
	for _, param := range params {
		contracts = append(contracts, imageModuleParamContract{
			name: param.Name, enum: param.Enum, defaultValue: param.Default,
		})
	}
	return contracts
}

func imageModuleFields(fields *FieldsSpec, required bool) []imageModuleFieldContract {
	if fields == nil {
		return nil
	}
	items := fields.Optional
	if required {
		items = fields.Required
	}
	contracts := make([]imageModuleFieldContract, 0, len(items))
	for _, field := range items {
		contracts = append(contracts, imageModuleFieldContract{name: field.Name, example: field.Example})
	}
	return contracts
}

const galleryGridGuideSnippet = `:::gallery-grid{columns=3 variant=card}
![A](https://example.com/1.jpg) | 首页状态
![B](https://example.com/2.jpg) | 详情状态
![C](https://example.com/3.jpg) | 结果状态
:::
`

const galleryStoryGuideSnippet = `:::gallery-story{variant=card}
![第一站](https://example.com/story-1.jpg) | 第一站 | 用一张图建立现场感。
![第二站](https://example.com/story-2.jpg) | 第二站 | 用段落解释关键细节。
:::
`

const imagePhoneShotGuideSnippet = `:::image-phone-shot{columns=2 image_shape=phone}
![首页](https://example.com/home.jpg) | 首页截图
![详情](https://example.com/detail.jpg) | 详情截图
:::
`

const figureCaptionGuideSnippet = `:::figure-caption{caption_style=numbered}
![增长曲线](https://example.com/chart.jpg)
caption: 2026 年用户增长曲线
source: 内部实验数据
:::
`

const svgRevealGuideSnippet = `:::svg-reveal{accent=brand}
question: 点击查看答案
answer: 42
:::
`

const svgSwipeGalleryGuideSnippet = `:::svg-swipe-gallery
![A](https://mmbiz.qpic.cn/...) | 第一张
![B](https://mmbiz.qpic.cn/...) | 第二张
:::
`

const splitGuideSnippet = `:::split
## 主判断

高级排版不是把页面做花，而是让读者先看懂重点。

---

## 落地方式

左侧讲结论，右侧放证据、说明或下一步，让两块信息都能独立阅读。
:::
`

const flowGuideSnippet = `:::flow[Agent 发布流程]
草稿输入 → 结构判断 → 模块选择 → 视觉校准 → 发布检查
:::
`

const matrixGuideSnippet = `:::matrix headers=能力,基础版,专业版,企业版
高级模块|有|有|有
主题定制|无|有|有
API 调用|无|有|有
私有部署|无|无|有
:::
`

const dialoguePairGuideSnippet = `:::dialogue-pair left=读者 right=作者
U: 高级模块和普通 Markdown 有什么区别？
E: 高级模块把结构、层级和视觉节奏一起带进公众号正文，不只是把文字换个样式。

U: 我需要学多少个模块？
E: 先学开场、信息卡、证据、总结和 CTA，绝大多数文章就能明显升级。
:::
`

var recommendedModuleNames = []string{
	"audience-fit", "author-card", "bridge", "callout", "cards", "cases",
	"changelog", "checklist", "compare", "comparison-table", "cta", "definition",
	"dialogue-pair", "faq", "figure-caption", "flow", "gallery-grid", "gallery-story",
	"hero", "image-annotate", "image-compare", "image-phone-shot", "image-steps", "closing",
	"image-text", "infographic", "label-title", "logos", "manifesto", "matrix",
	"metrics", "myth-fact", "notice", "part", "people", "pricing", "question",
	"quote", "quote-card", "resource-list", "series", "specs", "split", "stat-row",
	"steps", "subscribe", "summary", "svg-reveal", "svg-swipe-gallery", "timeline", "epilogue", "section-title",
	"toc", "toolbox", "tweet", "verdict",
}

var compatibilityModuleNames = []string{"dialogue", "gallery", "longimage"}

func TestRecommendedSyntaxInventoryHasExactCount(t *testing.T) {
	if got := len(recommendedModuleNames); got != 56 {
		t.Fatalf("recommended syntax inventory = %d, want 56", got)
	}
}

func moduleNames(mods []*LayoutSpec) []string {
	names := make([]string, 0, len(mods))
	for _, mod := range mods {
		names = append(names, mod.Name)
	}
	slices.Sort(names)
	return names
}

func TestListFilteredDefaultsToRecommended(t *testing.T) {
	c := NewCatalog()
	c.modules["current"] = &LayoutSpec{Name: "current", Lifecycle: LifecycleRecommended}
	c.modules["legacy"] = &LayoutSpec{Name: "legacy", Lifecycle: LifecycleCompatibility}
	if got := moduleNames(c.ListFiltered(ListFilter{})); !slices.Equal(got, []string{"current"}) {
		t.Fatalf("default list = %v", got)
	}
	if got := moduleNames(c.ListFiltered(ListFilter{Lifecycle: LifecycleCompatibility})); !slices.Equal(got, []string{"legacy"}) {
		t.Fatalf("compatibility list = %v", got)
	}
}

func TestBuiltinRecommendedModuleSetMatchesUpstream(t *testing.T) {
	t.Skip("catalog projection is implemented in Tasks 3-4")

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	got := moduleNames(c.ListFiltered(ListFilter{}))
	want := slices.Clone(recommendedModuleNames)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestBuiltinCompatibilityModuleSetMatchesUpstream(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	got := moduleNames(c.ListFiltered(ListFilter{Lifecycle: LifecycleCompatibility}))
	want := slices.Clone(compatibilityModuleNames)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestCompatibilityModuleContracts(t *testing.T) {
	contracts := map[string]compatibilityLayoutContract{
		"gallery": {
			format: BodyFormatMarkdownImages, replacement: "gallery-grid",
			opener: &OpenerSpec{Caption: true}, body: &BodySpec{MinImages: 1},
			example: galleryCompatibilityGuideSnippet,
		},
		"dialogue": {
			format: BodyFormatDialogue, replacement: "dialogue-pair",
			opener: &OpenerSpec{Caption: true}, body: &BodySpec{MinItems: 1, AllowNamedSpeakers: true},
			example: dialogueCompatibilityGuideSnippet,
		},
		"longimage": {
			format: BodyFormatMarkdownImages, replacement: "image-text",
			opener: nil, body: &BodySpec{MinImages: 1, MaxImages: 1},
			example: longimageCompatibilityGuideSnippet,
		},
	}
	wantAssertions := map[string]string{
		"gallery":   "",
		"dialogue":  "你好",
		"longimage": "",
	}

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for name, contract := range contracts {
		t.Run(name, func(t *testing.T) {
			spec, ok := c.Get(name)
			if !ok {
				t.Fatalf("missing %s", name)
			}
			if spec.Lifecycle != LifecycleCompatibility {
				t.Errorf("lifecycle = %q, want %q", spec.Lifecycle, LifecycleCompatibility)
			}
			if spec.BodyFormat != contract.format {
				t.Errorf("body_format = %q, want %q", spec.BodyFormat, contract.format)
			}
			if !reflect.DeepEqual(spec.Opener, contract.opener) {
				t.Errorf("opener = %#v, want %#v", spec.Opener, contract.opener)
			}
			if !reflect.DeepEqual(spec.Body, contract.body) {
				t.Errorf("body = %#v, want %#v", spec.Body, contract.body)
			}
			if !strings.Contains(spec.WhenNotToUse, contract.replacement) {
				t.Errorf("when_not_to_use = %q, want replacement %q", spec.WhenNotToUse, contract.replacement)
			}
			if len(spec.PairsWellWith) != 0 || len(spec.Variants) != 0 {
				t.Errorf("compatibility module leaked recommendation guidance: pairs=%v variants=%v", spec.PairsWellWith, spec.Variants)
			}
			if spec.Example != contract.example {
				t.Errorf("canonical migration witness differs\ngot:  %q\nwant: %q", spec.Example, contract.example)
			}
			if spec.ExampleAssertContains != wantAssertions[name] {
				t.Errorf("example_assert_contains = %q, want %q", spec.ExampleAssertContains, wantAssertions[name])
			}
			if err := checkExecutableWitness(c, spec.Name, "", spec.Example, spec.ExampleAssertContains); err != nil {
				t.Fatalf("witness invalid: %v", err)
			}
		})
	}
}

func TestBuiltinCanonicalStableAssertions(t *testing.T) {
	want := map[string]string{
		"checklist":         "结构先搭好",
		"dialogue-pair":     "高级模块和普通 Markdown 有什么区别？",
		"figure-caption":    "2026 年用户增长曲线",
		"flow":              "草稿输入",
		"image-text":        "图和说明绑在一起，读者更容易跟上重点",
		"question":          "md2wechat 是排版工具吗？",
		"split":             "主判断",
		"svg-swipe-gallery": "第一张",
	}
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for name, assertion := range want {
		spec, ok := c.Get(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if spec.ExampleAssertContains != assertion {
			t.Errorf("%s example_assert_contains = %q, want %q", name, spec.ExampleAssertContains, assertion)
		}
		if !strings.Contains(spec.Example, assertion) {
			t.Errorf("%s example does not contain assertion %q", name, assertion)
		}
	}
}

func TestCompatibilityModuleRejectedCases(t *testing.T) {
	tests := []struct {
		name, markdown, category string
	}{
		{name: "gallery requires an image", markdown: ":::gallery[空画廊]\n只有文字\n:::\n", category: "at least 1 image"},
		{name: "dialogue requires full-width named separator", markdown: ":::dialogue[旧稿]\n用户: 你好\n:::\n", category: "full-width colon"},
		{name: "longimage allows one image", markdown: ":::longimage[旧稿]\n![一](https://example.com/1.jpg)\n![二](https://example.com/2.jpg)\n:::\n", category: "at most 1 image"},
	}
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := c.Validate(tt.markdown)
			if len(report.Errors) == 0 {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(report.Errors[0].Message, tt.category) {
				t.Fatalf("error = %q, want category %q", report.Errors[0].Message, tt.category)
			}
		})
	}
}

const galleryCompatibilityGuideSnippet = `:::gallery[主题画廊]
![图一](https://example.com/1.png)
![图二](https://example.com/2.png)
:::
`

const dialogueCompatibilityGuideSnippet = `:::dialogue[主题对话]
甲：你好
乙：你好
:::
`

const longimageCompatibilityGuideSnippet = `:::longimage
![长图](https://example.com/longimage.jpg)
:::
`

func TestKnownDriftContractsAreCalibrated(t *testing.T) {
	calibratedFields := map[string]struct {
		required []string
		any      [][]string
	}{
		"hero":           {required: []string{"title"}},
		"audience-fit":   {any: [][]string{{"fit", "avoid"}}},
		"verdict":        {any: [][]string{{"title", "body"}}},
		"bridge":         {any: [][]string{{"title", "body", "next"}}},
		"manifesto":      {any: [][]string{{"title", "believe"}}},
		"quote":          {any: [][]string{{"quote", "text"}}},
		"image-text":     {required: []string{"image"}, any: [][]string{{"title", "body"}}},
		"image-annotate": {required: []string{"image", "point"}},
		"author-card":    {required: []string{"name"}},
		"series":         {required: []string{"name", "title"}},
		"subscribe":      {required: []string{"title"}},
		"cta":            {required: []string{"title"}},
		"tweet":          {required: []string{"name", "text"}},
		"resource-list":  {required: []string{"name"}},
	}
	calibratedFormats := map[string]string{
		"callout": BodyFormatMarkdownFields, "image-steps": BodyFormatMarkdownFields, "question": BodyFormatDialogue,
	}
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for name, want := range calibratedFields {
		t.Run(name, func(t *testing.T) {
			spec, ok := c.Get(name)
			if !ok {
				t.Fatalf("missing %s", name)
			}
			if got := fieldNames(spec.Fields.Required); !slices.Equal(got, want.required) {
				t.Errorf("required = %v, want %v", got, want.required)
			}
			if got := spec.Fields.RequiredAny; !reflect.DeepEqual(got, want.any) {
				t.Errorf("required_any = %v, want %v", got, want.any)
			}
		})
	}
	for name, want := range calibratedFormats {
		spec, _ := c.Get(name)
		if spec.BodyFormat != want {
			t.Errorf("%s body_format = %q, want %q", name, spec.BodyFormat, want)
		}
	}
	question, _ := c.Get("question")
	if !slices.Equal(question.CompatibleBodyFormats, []string{BodyFormatJSONArray}) {
		t.Errorf("question compatible formats = %v", question.CompatibleBodyFormats)
	}
	callout, _ := c.Get("callout")
	wantOpener := &OpenerSpec{ParamStyle: ParamStyleToken, Params: []ParamSpec{{Name: "variant", Enum: []string{"info", "warning", "success", "danger"}}}}
	if !reflect.DeepEqual(callout.Opener, wantOpener) {
		t.Errorf("callout opener = %#v, want %#v", callout.Opener, wantOpener)
	}
}

func TestTitleAndClosureCatalogContracts(t *testing.T) {
	const sharedSymbols = "spark-solid,spark-outline,diamond-solid,diamond-outline,reference-mark,asterism,double-circle,circle,square-solid,square-outline,star,infinity"
	type contract struct {
		category, defaultVariant, defaultSymbol string
		variants                                []string
		inputPositions                          []AgentInputPosition
	}
	contracts := map[string]contract{
		"hero":          {category: "opening", defaultVariant: "editorial", variants: []string{"editorial", "briefing", "story", "masthead"}, inputPositions: []AgentInputPosition{InputBodyKV}},
		"section-title": {category: "opening", defaultVariant: "marker", variants: []string{"marker", "divider", "numbered", "frame", "focus", "vertical"}, inputPositions: []AgentInputPosition{InputBodyKV}},
		"epilogue":      {category: "opening", defaultSymbol: "infinity", inputPositions: []AgentInputPosition{InputBodyKV}},
		"closing":       {category: "conversion", defaultSymbol: "asterism", inputPositions: []AgentInputPosition{InputBodyKV}},
		"cta":           {category: "conversion", defaultVariant: "save-follow", variants: []string{"save-follow", "consult", "trial"}, inputPositions: []AgentInputPosition{InputBodyKV}},
	}
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for name, want := range contracts {
		t.Run(name, func(t *testing.T) {
			spec, ok := c.Get(name)
			if !ok {
				t.Fatalf("missing %s", name)
			}
			if spec.Category != want.category || !slices.Equal(spec.InputPositions, want.inputPositions) {
				t.Fatalf("category/input_positions = %q/%v, want %q/%v", spec.Category, spec.InputPositions, want.category, want.inputPositions)
			}
			if want.defaultVariant != "" {
				field := fieldByName(spec.Fields.Optional, "variant")
				if field.Default != want.defaultVariant {
					t.Fatalf("variant default = %q, want %q", field.Default, want.defaultVariant)
				}
				if got := variantNames(spec.Variants); !slices.Equal(got, want.variants) {
					t.Fatalf("variants = %v, want %v", got, want.variants)
				}
			}
			if want.defaultSymbol != "" {
				if got := fieldByName(spec.Fields.Optional, "symbol").Default; got != want.defaultSymbol {
					t.Fatalf("symbol default = %q, want %q", got, want.defaultSymbol)
				}
			}
			if name == "hero" || name == "section-title" || name == "epilogue" || name == "closing" {
				if got := strings.Join(fieldByName(spec.Fields.Optional, "symbol").Enum, ","); got != sharedSymbols {
					t.Fatalf("symbol enum = %q, want %q", got, sharedSymbols)
				}
			}
			switch name {
			case "hero":
				if got := fieldByName(spec.Fields.Optional, "symbol"); !slices.Equal(got.AppliesTo, []string{"masthead"}) || got.Default != "spark-solid" {
					t.Fatalf("masthead symbol = %#v", got)
				}
				for _, field := range []string{"kicker", "points", "image", "tags"} {
					if got := fieldByName(spec.Fields.Optional, field).AppliesTo; !slices.Equal(got, []string{"editorial", "briefing", "story"}) {
						t.Fatalf("%s applicability = %v", field, got)
					}
				}
			case "section-title":
				if got := fieldByName(spec.Fields.Optional, "symbol"); !slices.Equal(got.AppliesTo, []string{"marker", "divider", "focus", "vertical"}) || got.Default != "diamond-outline" {
					t.Fatalf("section symbol = %#v", got)
				}
				if got := fieldByName(spec.Fields.Optional, "index"); !slices.Equal(got.AppliesTo, []string{"numbered"}) || got.MinRunes != 1 || got.MaxRunes != 4 {
					t.Fatalf("numbered index = %#v", got)
				}
				for variant, wantDefaults := range map[string]map[string]string{
					"divider":  {"symbol": "spark-outline"},
					"focus":    {"symbol": "double-circle"},
					"vertical": {"symbol": "diamond-solid"},
				} {
					if got := variantDefaults(spec.Variants, variant); !reflect.DeepEqual(got, wantDefaults) {
						t.Fatalf("%s defaults = %#v, want %#v", variant, got, wantDefaults)
					}
				}
			case "cta":
				if got := fieldByName(spec.Fields.Optional, "points").AppliesTo; !slices.Equal(got, []string{"trial"}) {
					t.Fatalf("points applicability = %v", got)
				}
				if got := spec.Fields.Shapes; len(got) != 1 || got[0].Field != "points" || got[0].MaxParts != 3 {
					t.Fatalf("points shape = %#v", got)
				}
				for field, wantDefault := range map[string]string{"primary": "收藏这篇", "secondary": "关注更新", "tertiary": "转给同事"} {
					if got := fieldByName(spec.Fields.Optional, field).Default; got != wantDefault {
						t.Fatalf("%s default = %q, want %q", field, got, wantDefault)
					}
				}
				for variant, wantDefaults := range map[string]map[string]string{
					"consult": {"primary": "回复「排版」", "secondary": "查看案例", "tertiary": "收藏这篇"},
					"trial":   {"primary": "试一版高级稿", "secondary": "查看工作流", "tertiary": "获取 API Key"},
				} {
					if got := variantDefaults(spec.Variants, variant); !reflect.DeepEqual(got, wantDefaults) {
						t.Fatalf("%s defaults = %#v, want %#v", variant, got, wantDefaults)
					}
				}
			}
		})
	}
}

func fieldByName(fields []FieldSpec, name string) FieldSpec {
	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}
	return FieldSpec{}
}

func variantNames(variants []VariantSpec) []string {
	names := make([]string, 0, len(variants))
	for _, variant := range variants {
		names = append(names, variant.Name)
	}
	return names
}

func variantDefaults(variants []VariantSpec, name string) map[string]string {
	for _, variant := range variants {
		if variant.Name == name {
			return variant.Defaults
		}
	}
	return nil
}

func fieldNames(fields []FieldSpec) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}
