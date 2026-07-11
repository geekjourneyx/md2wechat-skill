package layoutcatalog

import (
	"slices"
	"testing"
)

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
