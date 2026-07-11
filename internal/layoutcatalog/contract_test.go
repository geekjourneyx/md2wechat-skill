package layoutcatalog

import (
	"slices"
	"testing"
)

func TestNewImageModuleContracts(t *testing.T) {
	contracts := map[string]struct {
		format, category string
		minImages        int
		maxImages        int
		params           []string
		requiredFields   []string
		optionalFields   []string
	}{
		"gallery-grid": {
			format: BodyFormatMarkdownImages, category: "evidence", minImages: 1,
			params: []string{"columns", "variant"},
		},
		"gallery-story": {
			format: BodyFormatMarkdownImages, category: "evidence", minImages: 1,
			params: []string{"variant"},
		},
		"image-phone-shot": {
			format: BodyFormatMarkdownImages, category: "evidence", minImages: 1,
			params: []string{"columns", "image_shape"},
		},
		"figure-caption": {
			format: BodyFormatMarkdownFields, category: "evidence", minImages: 1, maxImages: 1,
			params: []string{"caption_style"}, requiredFields: []string{"caption"}, optionalFields: []string{"source"},
		},
		"svg-reveal": {
			format: BodyFormatFields, category: "interactive",
			params: []string{"accent", "svg_fallback"}, requiredFields: []string{"question", "answer"},
		},
		"svg-swipe-gallery": {
			format: BodyFormatMarkdownImages, category: "interactive", minImages: 2,
			params: []string{"svg_fallback"},
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
			if spec.Body == nil {
				if contract.minImages != 0 || contract.maxImages != 0 {
					t.Fatal("missing body constraints")
				}
			} else if spec.Body.MinImages != contract.minImages || spec.Body.MaxImages != contract.maxImages {
				t.Errorf("image limits = (%d, %d), want (%d, %d)", spec.Body.MinImages, spec.Body.MaxImages, contract.minImages, contract.maxImages)
			}
			if spec.Opener == nil || spec.Opener.ParamStyle != ParamStyleBraces {
				t.Fatalf("opener = %+v, want braces", spec.Opener)
			}
			if got := paramNames(spec.Opener.Params); !slices.Equal(got, contract.params) {
				t.Errorf("opener params = %v, want %v", got, contract.params)
			}
			if got := fieldNames(spec.Fields, true); !slices.Equal(got, contract.requiredFields) {
				t.Errorf("required fields = %v, want %v", got, contract.requiredFields)
			}
			if got := fieldNames(spec.Fields, false); !slices.Equal(got, contract.optionalFields) {
				t.Errorf("optional fields = %v, want %v", got, contract.optionalFields)
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

func paramNames(params []ParamSpec) []string {
	names := make([]string, 0, len(params))
	for _, param := range params {
		names = append(names, param.Name)
	}
	return names
}

func fieldNames(fields *FieldsSpec, required bool) []string {
	if fields == nil {
		return nil
	}
	items := fields.Optional
	if required {
		items = fields.Required
	}
	names := make([]string, 0, len(items))
	for _, field := range items {
		names = append(names, field.Name)
	}
	return names
}

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
