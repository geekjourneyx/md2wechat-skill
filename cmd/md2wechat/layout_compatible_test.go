package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
)

func TestLayoutRenderBodyFileAcceptsQuestionCompatibleJSONArray(t *testing.T) {
	oldJSON := jsonOutput
	oldVars := append([]string(nil), layoutRenderVars...)
	oldParams := append([]string(nil), layoutRenderParams...)
	oldCaption := layoutRenderCaption
	oldBodyFile := layoutRenderBodyFile
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutRenderVars = oldVars
		layoutRenderParams = oldParams
		layoutRenderCaption = oldCaption
		layoutRenderBodyFile = oldBodyFile
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	body := `[{"q":"支持吗？","a":"支持。"}]`
	bodyPath := filepath.Join(t.TempDir(), "question.json")
	if err := os.WriteFile(bodyPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonOutput = true
	layoutRenderVars = nil
	layoutRenderParams = nil
	layoutRenderCaption = ""
	layoutRenderBodyFile = bodyPath
	layoutcatalog.ResetDefaultCatalogForTests()

	stdout := captureStdout(t, func() {
		if err := layoutRenderCmd.RunE(layoutRenderCmd, []string{"question"}); err != nil {
			t.Fatalf("layoutRenderCmd.RunE() error = %v", err)
		}
	})
	if !strings.Contains(string(stdout), `:::question`) || !strings.Contains(string(stdout), `支持吗？`) {
		t.Fatalf("compatible question body missing from output:\n%s", stdout)
	}
}

func TestLayoutRenderDistinguishesReservedAndUnknownModuleNames(t *testing.T) {
	oldVars := append([]string(nil), layoutRenderVars...)
	oldParams := append([]string(nil), layoutRenderParams...)
	oldCaption := layoutRenderCaption
	oldBodyFile := layoutRenderBodyFile
	t.Cleanup(func() {
		layoutRenderVars = oldVars
		layoutRenderParams = oldParams
		layoutRenderCaption = oldCaption
		layoutRenderBodyFile = oldBodyFile
		layoutcatalog.ResetDefaultCatalogForTests()
	})
	layoutRenderVars = nil
	layoutRenderParams = nil
	layoutRenderCaption = ""
	layoutRenderBodyFile = ""
	layoutcatalog.ResetDefaultCatalogForTests()

	tests := []struct {
		name     string
		module   string
		wantCode string
	}{
		{name: "reserved", module: "block", wantCode: codeLayoutInvalidFieldValue},
		{name: "unknown", module: "well-formed-unknown", wantCode: codeLayoutModuleNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := layoutRenderCmd.RunE(layoutRenderCmd, []string{tt.module})
			cliErr, ok := err.(*cliError)
			if !ok || cliErr.Code != tt.wantCode {
				t.Fatalf("layout render %s error = %#v, want code %s", tt.module, err, tt.wantCode)
			}
		})
	}
}
