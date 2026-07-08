package image

import "testing"

func TestLookupProviderMetaSupportsVolcengineAlias(t *testing.T) {
	meta, ok := LookupProviderMeta("volc")
	if !ok {
		t.Fatal("expected volc alias to resolve")
	}
	if meta.Name != "volcengine" {
		t.Fatalf("Name = %q", meta.Name)
	}
	if meta.DefaultModel != "doubao-seedream-5-0-pro-260628" {
		t.Fatalf("DefaultModel = %q", meta.DefaultModel)
	}
}

func TestProviderRegistryDefaultModelIsInSupportedModels(t *testing.T) {
	for _, meta := range SupportedProviders() {
		if meta.DefaultModel == "" {
			continue
		}

		foundDefault := false
		for _, model := range meta.SupportedModels {
			if model.Name != meta.DefaultModel {
				continue
			}
			foundDefault = true
			if !model.Default {
				t.Fatalf("%s default model %q is not marked default", meta.Name, model.Name)
			}
		}

		if !foundDefault {
			t.Fatalf("%s default model %q missing from supported_models", meta.Name, meta.DefaultModel)
		}
	}
}
