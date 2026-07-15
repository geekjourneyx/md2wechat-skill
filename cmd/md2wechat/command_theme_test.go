package main

import (
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/spf13/cobra"
)

func TestApplyEffectiveCommandThemeMatrix(t *testing.T) {
	oldCfg := cfg
	oldConvertTheme, oldInspectTheme, oldPreviewTheme := convertTheme, inspectTheme, previewTheme
	t.Cleanup(func() {
		cfg = oldCfg
		convertTheme, inspectTheme, previewTheme = oldConvertTheme, oldInspectTheme, oldPreviewTheme
	})

	commands := []struct {
		name  string
		cmd   *cobra.Command
		theme *string
	}{
		{name: "convert", cmd: convertCmd, theme: &convertTheme},
		{name: "inspect", cmd: inspectCmd, theme: &inspectTheme},
		{name: "preview", cmd: previewCmd, theme: &previewTheme},
	}
	tests := []struct {
		name       string
		configured string
		current    string
		explicit   bool
		want       string
	}{
		{name: "unset uses configured default", configured: "chinese", current: "default", want: "chinese"},
		{name: "unset with blank config falls back", configured: "  ", current: "default", want: "default"},
		{name: "explicit default wins", configured: "chinese", current: "default", explicit: true, want: "default"},
		{name: "explicit other wins", configured: "chinese", current: "autumn-warm", explicit: true, want: "autumn-warm"},
	}

	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			flag := command.cmd.Flags().Lookup("theme")
			if flag == nil {
				t.Fatal("theme flag is missing")
			}
			oldChanged := flag.Changed
			t.Cleanup(func() { flag.Changed = oldChanged })

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					cfg = &config.Config{DefaultTheme: tt.configured}
					*command.theme = tt.current
					flag.Changed = tt.explicit

					applyEffectiveCommandTheme(command.cmd, command.theme)

					if got := *command.theme; got != tt.want {
						t.Fatalf("theme = %q, want %q", got, tt.want)
					}
				})
			}
		})
	}
}
