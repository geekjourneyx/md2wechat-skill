package layoutcatalog

import (
	"reflect"
	"slices"
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
