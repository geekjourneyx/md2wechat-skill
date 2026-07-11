package main

import (
	"os"
	"strings"
	"testing"
)

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

	requirePhrases := map[string][]string{
		"../../README.md": {
			"68 个主推",
			"53 个主推",
			"3 个兼容模块",
			"60 项渲染层语法能力",
		},
		"../../docs/DISCOVERY.md": {
			"68 个主推",
			"53 个主推",
			"3 个兼容模块",
			"60 项渲染层语法能力",
		},
	}
	for path, phrases := range requirePhrases {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, phrase := range phrases {
			if !strings.Contains(string(data), phrase) {
				t.Errorf("%s must define %q", path, phrase)
			}
		}
	}
}
