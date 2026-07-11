package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
)

const (
	codeLayoutShown     = "LAYOUT_SHOWN"
	codeLayoutRendered  = "LAYOUT_RENDERED"
	codeLayoutValidated = "LAYOUT_VALIDATED"

	recommendedLayoutScenarioCount = 68
	baseLayoutEnhancementCount     = 4
	renderLayoutSyntaxCount        = 60
)

var (
	layoutListFilters struct {
		category    string
		serves      string
		contentType string
		industry    string
		tag         string
		lifecycle   string
	}
	layoutRenderVars     []string
	layoutRenderParams   []string
	layoutRenderCaption  string
	layoutRenderBodyFile string
	layoutValidateFile   string
	layoutValidateStdin  bool
)

var layoutCmd = &cobra.Command{
	Use:   "layout",
	Short: "Discover and render advanced layout modules (:::module syntax)",
}

var layoutListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available layout modules",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := layoutcatalog.DefaultCatalog()
		if err != nil {
			return wrapCLIError(codeError, err, "load layout catalog")
		}
		mods := c.ListFiltered(layoutcatalog.ListFilter{
			Category:    layoutListFilters.category,
			Serves:      layoutListFilters.serves,
			ContentType: layoutListFilters.contentType,
			Industry:    layoutListFilters.industry,
			Tag:         layoutListFilters.tag,
			Lifecycle:   layoutListFilters.lifecycle,
		})
		summaries := make([]map[string]any, 0, len(mods))
		for _, m := range mods {
			summaries = append(summaries, map[string]any{
				"name":          m.Name,
				"lifecycle":     m.Lifecycle,
				"body_format":   m.BodyFormat,
				"category":      m.Category,
				"serves":        m.Serves,
				"content_types": m.ContentTypes,
				"industry":      m.Industry,
				"tags":          m.Tags,
				"version":       m.Version,
			})
		}
		responseSuccessWith(codeLayoutShown, "layout modules", map[string]any{
			"count":   len(summaries),
			"modules": summaries,
		})
		return nil
	},
}

var layoutShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show full spec of one layout module",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := layoutcatalog.DefaultCatalog()
		if err != nil {
			return wrapCLIError(codeError, err, "load layout catalog")
		}
		spec, ok := c.Get(args[0])
		if !ok {
			return wrapCLIError(codeLayoutModuleNotFound,
				fmt.Errorf("module %q not found", args[0]),
				"layout module not found")
		}
		responseSuccessWith(codeLayoutShown, "layout module", map[string]any{"spec": spec})
		return nil
	},
}

var layoutRenderCmd = &cobra.Command{
	Use:   "render <name>",
	Short: "Render a :::module ... ::: block",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := layoutcatalog.DefaultCatalog()
		if err != nil {
			return wrapCLIError(codeError, err, "load layout catalog")
		}
		vars := map[string]any{}
		for _, kv := range layoutRenderVars {
			key, value, ok := strings.Cut(kv, "=")
			if !ok {
				return wrapCLIError(codeLayoutInvalidFieldValue,
					fmt.Errorf("invalid --var %q (want KEY=VALUE)", kv),
					"invalid var")
			}
			vars[key] = value
		}
		if raw, ok := vars["rows"]; ok {
			if s, isStr := raw.(string); isStr && strings.HasPrefix(s, "[") {
				var parsed []any
				if err := json.Unmarshal([]byte(s), &parsed); err == nil {
					vars["rows"] = parsed
				}
			}
		}
		params := map[string]string{}
		for _, kv := range layoutRenderParams {
			key, value, ok := strings.Cut(kv, "=")
			if !ok {
				return wrapCLIError(codeLayoutInvalidFieldValue,
					fmt.Errorf("invalid --param %q (want KEY=VALUE)", kv),
					"invalid param")
			}
			params[key] = value
		}

		var body string
		if layoutRenderBodyFile != "" {
			var content []byte
			var readErr error
			if layoutRenderBodyFile == "-" {
				content, readErr = io.ReadAll(stdinReader)
			} else {
				content, readErr = os.ReadFile(layoutRenderBodyFile)
			}
			if readErr != nil {
				return wrapCLIError(codeError, readErr, "read layout body")
			}
			body = string(content)
		}

		out, err := c.RenderBlock(args[0], layoutcatalog.RenderInput{
			Fields:  vars,
			Params:  params,
			Caption: layoutRenderCaption,
			Body:    body,
		})
		if err != nil {
			switch {
			case errors.Is(err, layoutcatalog.ErrUnknownModule):
				return wrapCLIError(codeLayoutModuleNotFound, err, "module not found")
			case errors.Is(err, layoutcatalog.ErrMissingRequiredField):
				return wrapCLIError(codeLayoutMissingRequiredField, err, "missing required field")
			case errors.Is(err, layoutcatalog.ErrInvalidFieldValue):
				return wrapCLIError(codeLayoutInvalidFieldValue, err, "invalid field value")
			default:
				return wrapCLIError(codeError, err, "render failed")
			}
		}
		responseSuccessWith(codeLayoutRendered, "rendered", map[string]any{"block": out})
		return nil
	},
}

var layoutValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate :::module usage in a Markdown file",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := layoutcatalog.DefaultCatalog()
		if err != nil {
			return wrapCLIError(codeError, err, "load layout catalog")
		}
		var content []byte
		var readErr error
		switch {
		case layoutValidateStdin:
			content, readErr = io.ReadAll(stdinReader)
			if readErr != nil {
				return wrapCLIError(codeError, readErr, "read stdin")
			}
		case layoutValidateFile != "":
			content, readErr = os.ReadFile(layoutValidateFile)
			if readErr != nil {
				return wrapCLIError(codeError, readErr, "read file")
			}
		default:
			return wrapCLIError(codeLayoutInvalidFilter,
				fmt.Errorf("either --file or --stdin required"),
				"no input source")
		}
		report := c.Validate(string(content))
		if len(report.Errors) > 0 {
			responseSuccessWith(codeLayoutValidateHasErrors, "validation failed", map[string]any{
				"errors":   report.Errors,
				"warnings": report.Warnings,
			})
			exitFunc(1)
			return nil
		}
		responseSuccessWith(codeLayoutValidated, "validation passed", map[string]any{
			"errors":   report.Errors,
			"warnings": report.Warnings,
		})
		return nil
	},
}

func init() {
	layoutListCmd.Flags().StringVar(&layoutListFilters.category, "category", "", "filter by category")
	layoutListCmd.Flags().StringVar(&layoutListFilters.serves, "serves", "", "filter by serves (attention/readability/memorability/conversion)")
	layoutListCmd.Flags().StringVar(&layoutListFilters.contentType, "content-type", "", "filter by content_type")
	layoutListCmd.Flags().StringVar(&layoutListFilters.industry, "industry", "", "filter by industry")
	layoutListCmd.Flags().StringVar(&layoutListFilters.tag, "tag", "", "filter by tag")
	layoutListCmd.Flags().StringVar(
		&layoutListFilters.lifecycle, "lifecycle", "",
		"filter by lifecycle (recommended/compatibility; default recommended)",
	)

	layoutRenderCmd.Flags().StringArrayVar(&layoutRenderVars, "var", nil, "field as KEY=VALUE (repeatable)")
	layoutRenderCmd.Flags().StringArrayVar(&layoutRenderParams, "param", nil, "opener parameter as KEY=VALUE (repeatable)")
	layoutRenderCmd.Flags().StringVar(&layoutRenderCaption, "caption", "", "bracket caption")
	layoutRenderCmd.Flags().StringVar(&layoutRenderBodyFile, "body-file", "", "raw module body file; use - for stdin")

	layoutValidateCmd.Flags().StringVar(&layoutValidateFile, "file", "", "path to markdown file")
	layoutValidateCmd.Flags().BoolVar(&layoutValidateStdin, "stdin", false, "read markdown from stdin")

	layoutCmd.AddCommand(layoutListCmd, layoutShowCmd, layoutRenderCmd, layoutValidateCmd)
}
