package layoutcatalog

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

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
				{name: "columns", enum: []string{"1", "2", "3"}, defaultValue: "2"},
				{name: "variant", enum: []string{"card", "plain"}, defaultValue: "card"},
			},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#gallery-grid"},
			example:  galleryGridGuideSnippet,
		},
		"gallery-story": {
			format: BodyFormatMarkdownImages, category: "evidence", lifecycle: LifecycleRecommended, minImages: 1,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "variant", enum: []string{"card", "plain"}, defaultValue: "card"},
			},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#gallery-story"},
			example:  galleryStoryGuideSnippet,
		},
		"image-phone-shot": {
			format: BodyFormatMarkdownImages, category: "evidence", lifecycle: LifecycleRecommended, minImages: 1,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "columns", enum: []string{"1", "2", "3"}, defaultValue: "2"},
				{name: "image_shape", enum: []string{"phone"}, defaultValue: "phone"},
			},
			metadata: LayoutMetadata{Author: "md2wechat", Provenance: "builtin", InspiredBy: "advanced-layout-modules-guide.md#image-phone-shot"},
			example:  imagePhoneShotGuideSnippet,
		},
		"figure-caption": {
			format: BodyFormatMarkdownFields, category: "evidence", lifecycle: LifecycleRecommended, minImages: 1, maxImages: 1,
			paramStyle: ParamStyleBraces,
			params: []imageModuleParamContract{
				{name: "caption_style", enum: []string{"minimal", "numbered"}, defaultValue: "minimal"},
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
				{name: "accent", enum: []string{"brand", "muted"}, defaultValue: "brand"},
				{name: "svg_fallback", enum: []string{"first-layer", "static"}, defaultValue: "first-layer"},
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
				{name: "svg_fallback", enum: []string{"first-layer", "static"}, defaultValue: "first-layer"},
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
	"hero", "image-annotate", "image-compare", "image-phone-shot", "image-steps",
	"image-text", "infographic", "label-title", "logos", "manifesto", "matrix",
	"metrics", "myth-fact", "notice", "part", "people", "pricing", "question",
	"quote", "quote-card", "resource-list", "series", "specs", "split", "stat-row",
	"steps", "subscribe", "summary", "svg-reveal", "svg-swipe-gallery", "timeline",
	"toc", "toolbox", "tweet", "verdict",
}

var compatibilityModuleNames = []string{"dialogue", "gallery", "longimage"}

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

func TestKnownDriftContractsAreCalibrated(t *testing.T) {
	calibratedFields := map[string]struct {
		required []string
		any      [][]string
	}{
		"hero":           {required: []string{"title"}},
		"audience-fit":   {any: [][]string{{"fit", "avoid"}}},
		"verdict":        {any: [][]string{{"title", "body"}}},
		"bridge":         {any: [][]string{{"title", "body", "next"}}},
		"manifesto":      {any: [][]string{{"title", "believe", "against"}}},
		"quote":          {any: [][]string{{"quote", "text"}}},
		"image-text":     {required: []string{"image"}, any: [][]string{{"title", "body"}}},
		"image-annotate": {required: []string{"image", "point"}},
		"author-card":    {required: []string{"name", "bio"}},
		"series":         {required: []string{"name", "title"}},
		"subscribe":      {required: []string{"title"}},
		"cta":            {required: []string{"title"}},
		"tweet":          {required: []string{"name", "text"}},
		"resource-list":  {required: []string{"name"}},
	}
	calibratedFormats := map[string]string{
		"callout": BodyFormatLines, "image-steps": BodyFormatMarkdownFields, "question": BodyFormatDialogue,
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

func fieldNames(fields []FieldSpec) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}
