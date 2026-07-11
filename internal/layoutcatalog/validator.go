package layoutcatalog

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ValidationIssue struct {
	Module  string `json:"module"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

type ValidationReport struct {
	Errors   []ValidationIssue `json:"errors"`
	Warnings []ValidationIssue `json:"warnings"`
}

func (c *Catalog) Validate(markdown string) ValidationReport {
	var r ValidationReport
	lines := strings.Split(markdown, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimRight(lines[i], "\r")
		opener, err := parseBlockOpener(line)
		if err != nil {
			if strings.HasPrefix(strings.TrimSpace(line), ":::") && strings.TrimSpace(line) != ":::" {
				r.Errors = append(r.Errors, ValidationIssue{
					Line:    i + 1,
					Message: "invalid layout block opener",
				})
			}
			i++
			continue
		}
		moduleName := opener.Name
		startLine := i + 1
		j := i + 1
		body := []string{}
		for j < len(lines) && strings.TrimRight(lines[j], "\r") != ":::" {
			body = append(body, lines[j])
			j++
		}
		if j >= len(lines) {
			r.Errors = append(r.Errors, ValidationIssue{
				Module:  moduleName,
				Line:    startLine,
				Message: "unterminated :::" + moduleName + " block",
			})
			break
		}
		c.validateBlock(opener, body, startLine, &r)
		i = j + 1
	}
	return r
}

func (c *Catalog) validateBlock(opener ParsedOpener, body []string, line int, r *ValidationReport) {
	name := opener.Name
	spec, ok := c.Get(name)
	if !ok {
		r.Warnings = append(r.Warnings, ValidationIssue{
			Module:  name,
			Line:    line,
			Message: "unknown layout module (CLI catalog may be older than the API)",
		})
		return
	}
	if _, err := validateOpener(opener, spec.Opener); err != nil {
		r.Errors = append(r.Errors, ValidationIssue{
			Module:  name,
			Line:    line,
			Message: err.Error(),
		})
		return
	}
	for _, issue := range validateBlockBody(spec, body) {
		r.Errors = append(r.Errors, ValidationIssue{
			Module:  name,
			Field:   issue.field,
			Line:    line,
			Message: issue.message,
		})
	}
}

type jsonItemFacts struct {
	values map[string][]string
	types  map[string][]string
}

func parseJSONBodyData(body []string, bodyFormat string) (map[string]string, map[string][]string, []jsonItemFacts, error) {
	rawLines := make([]string, 0, len(body))
	for _, ln := range body {
		trimmed := strings.TrimSpace(strings.TrimRight(ln, "\r"))
		if trimmed != "" {
			rawLines = append(rawLines, trimmed)
		}
	}
	if len(rawLines) == 0 {
		return nil, nil, nil, nil
	}
	raw := strings.Join(rawLines, "\n")
	switch bodyFormat {
	case BodyFormatJSONObject:
		if !strings.HasPrefix(raw, "{") {
			return nil, nil, nil, fmt.Errorf("%w: expected JSON object body", ErrInvalidFieldValue)
		}
	case BodyFormatJSONArray:
		if !strings.HasPrefix(raw, "[") {
			return nil, nil, nil, fmt.Errorf("%w: expected JSON array body", ErrInvalidFieldValue)
		}
	}

	fields := map[string]string{}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, nil, nil, err
	}
	collectJSONFields(fields, "", decoded)
	types := map[string][]string{}
	collectJSONTypes(types, "", decoded)
	var items []jsonItemFacts
	if array, ok := decoded.([]any); ok {
		items = make([]jsonItemFacts, 0, len(array))
		for _, item := range array {
			flat := map[string]string{}
			collectJSONFields(flat, "", item)
			values := map[string][]string{}
			for name, value := range flat {
				values[name] = []string{value}
			}
			itemTypes := map[string][]string{}
			collectJSONTypes(itemTypes, "", item)
			items = append(items, jsonItemFacts{values: values, types: itemTypes})
		}
	}
	return fields, types, items, nil
}

func collectJSONTypes(types map[string][]string, prefix string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			name := key
			if prefix != "" {
				name = prefix + "." + key
			}
			types[name] = append(types[name], jsonValueType(item))
			switch nested := item.(type) {
			case map[string]any:
				collectJSONTypes(types, name, nested)
			case []any:
				for _, child := range nested {
					if object, ok := child.(map[string]any); ok {
						collectJSONTypes(types, name, object)
					}
				}
			}
		}
	case []any:
		for _, item := range typed {
			collectJSONTypes(types, prefix, item)
		}
	}
}

func jsonValueType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func collectJSONFields(fields map[string]string, prefix string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			collectJSONFields(fields, key, v)
		}
	case []any:
		for _, item := range typed {
			collectJSONFields(fields, prefix, item)
		}
	case string:
		if prefix != "" {
			fields[prefix] = typed
		}
	case float64, bool:
		if prefix != "" {
			fields[prefix] = fmt.Sprint(typed)
		}
	default:
		if prefix != "" {
			fields[prefix] = "present"
		}
	}
}
