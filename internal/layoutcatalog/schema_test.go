package layoutcatalog

import "testing"

func TestValidServesContainsFourValues(t *testing.T) {
	want := []string{"attention", "readability", "memorability", "conversion"}
	for _, v := range want {
		if !ValidServes[v] {
			t.Errorf("expected %q to be a valid serve, missing", v)
		}
	}
	if len(ValidServes) != 4 {
		t.Errorf("ValidServes should contain exactly 4 values, got %d", len(ValidServes))
	}
}

func TestSchemaVersionConstant(t *testing.T) {
	if SchemaVersion != "1" {
		t.Errorf("SchemaVersion = %q, want %q", SchemaVersion, "1")
	}
}

func TestValidBodyFormats(t *testing.T) {
	want := []string{
		BodyFormatFields,
		BodyFormatRows,
		BodyFormatJSONObject,
		BodyFormatJSONArray,
		BodyFormatMarkdownImages,
		BodyFormatMarkdownFields,
		BodyFormatSplit,
		BodyFormatLines,
		BodyFormatDialogue,
	}
	for _, v := range want {
		if !ValidBodyFormats[v] {
			t.Errorf("expected %q to be a valid body_format, missing", v)
		}
	}
	if len(ValidBodyFormats) != len(want) {
		t.Errorf("ValidBodyFormats should contain exactly %d values, got %d", len(want), len(ValidBodyFormats))
	}
}
