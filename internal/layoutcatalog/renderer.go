package layoutcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownModule        = errors.New("layout module not found")
	ErrMissingRequiredField = errors.New("missing required field")
	ErrInvalidFieldValue    = errors.New("invalid field value")
)

type RenderInput struct {
	Fields  map[string]any
	Params  map[string]string
	Caption string
	Body    string
}

func (c *Catalog) Render(name string, vars map[string]any) (string, error) {
	return c.renderBlock(name, RenderInput{Fields: vars}, openerParamOrderDeclared)
}

func (c *Catalog) RenderBlock(name string, input RenderInput) (string, error) {
	return c.renderBlock(name, input, openerParamOrderLexical)
}

func (c *Catalog) renderBlock(name string, input RenderInput, order openerParamOrder) (string, error) {
	if err := validateLayoutModuleName(name); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidFieldValue, err)
	}
	if name == reservedModuleName {
		return "", fmt.Errorf("%w: layout module name %q is reserved", ErrInvalidFieldValue, name)
	}
	spec, ok := c.Get(name)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownModule, name)
	}
	openerVars, err := renderOpenerVars(spec, input)
	if err != nil {
		return "", err
	}
	opener, err := renderOpenerWithOrder(spec, openerVars, order)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidFieldValue, err)
	}

	rawBody := input.Body
	if rawBody == "" && spec.Body != nil {
		rawBody, _ = lookupString(input.Fields, "body")
	}
	if rawBody != "" {
		out := renderRawBody(opener, rawBody)
		if err := validateRenderedBody(spec, rawBody); err != nil {
			return "", err
		}
		return out, nil
	}

	formats := append([]string{spec.BodyFormat}, spec.CompatibleBodyFormats...)
	for _, format := range formats {
		var out string
		var renderErr error
		switch format {
		case BodyFormatRows:
			if _, ok := input.Fields["rows"]; !ok {
				continue
			}
			out, renderErr = renderRows(spec, input.Fields, opener)
		case BodyFormatJSONObject:
			out, renderErr = renderJSONFields(spec, input.Fields, "object", opener)
		case BodyFormatJSONArray:
			out, renderErr = renderJSONFields(spec, input.Fields, "array", opener)
		case BodyFormatFields, BodyFormatMarkdownFields, "":
			out, renderErr = renderFields(spec, input.Fields, opener)
		default:
			continue
		}
		if renderErr != nil {
			return "", renderErr
		}
		selected := *spec
		selected.BodyFormat = format
		selected.CompatibleBodyFormats = nil
		if err := validateRenderedOutput(&selected, out); err != nil {
			return "", err
		}
		return out, nil
	}
	if !isStructuredBodyFormat(spec.BodyFormat) {
		primary := *spec
		primary.CompatibleBodyFormats = nil
		if err := validateRenderedBody(&primary, ""); err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("%w: no accepted body format supports structured rendering", ErrInvalidFieldValue)
}

func renderOpenerVars(spec *LayoutSpec, input RenderInput) (map[string]any, error) {
	vars := make(map[string]any, len(input.Fields)+len(input.Params)+1)
	for name, value := range input.Fields {
		vars[name] = value
	}
	if len(input.Params) > 0 {
		if spec.Opener == nil {
			return nil, fmt.Errorf("%w: %s does not support opener parameters", ErrInvalidFieldValue, spec.Name)
		}
		declared := make(map[string]bool, len(spec.Opener.Params))
		for _, param := range spec.Opener.Params {
			declared[param.Name] = true
		}
		for name, value := range input.Params {
			if !declared[name] {
				return nil, fmt.Errorf("%w: %s has no opener parameter %q", ErrInvalidFieldValue, spec.Name, name)
			}
			vars[name] = value
		}
	}
	if input.Caption != "" {
		if spec.Opener == nil || !spec.Opener.Caption {
			return nil, fmt.Errorf("%w: %s does not support an opener caption", ErrInvalidFieldValue, spec.Name)
		}
		vars["caption"] = input.Caption
	}
	return vars, nil
}

func isStructuredBodyFormat(format string) bool {
	switch format {
	case BodyFormatFields, BodyFormatMarkdownFields, BodyFormatRows, BodyFormatJSONObject, BodyFormatJSONArray, "":
		return true
	default:
		return false
	}
}

func renderFields(spec *LayoutSpec, vars map[string]any, opener string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", opener)

	if spec.Fields != nil {
		for _, f := range orderedFieldSpecs(spec.Fields) {
			val, ok := lookupString(vars, f.Name)
			if !ok || val == "" {
				continue
			}
			fmt.Fprintf(&b, "%s: %s\n", f.Name, val)
		}
	}
	b.WriteString(":::\n")
	return b.String(), nil
}

func renderJSONFields(spec *LayoutSpec, vars map[string]any, bodyKind, opener string) (string, error) {
	obj := map[string]any{}
	if spec.Fields != nil {
		for _, f := range spec.Fields.Required {
			val, ok := lookupString(vars, f.Name)
			if ok {
				setJSONField(obj, f.Name, parseJSONFieldValue(val))
			}
		}
		for _, f := range spec.Fields.Optional {
			val, ok := lookupString(vars, f.Name)
			if !ok || val == "" {
				continue
			}
			setJSONField(obj, f.Name, parseJSONFieldValue(val))
		}
	}

	var body any = obj
	if bodyKind == "array" {
		body = []any{obj}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", opener)
	fmt.Fprintf(&b, "%s\n", encoded)
	b.WriteString(":::\n")
	return b.String(), nil
}

func renderRows(spec *LayoutSpec, vars map[string]any, opener string) (string, error) {
	rowsRaw, ok := vars["rows"]
	if !ok {
		return "", fmt.Errorf("%w: %s.rows", ErrMissingRequiredField, spec.Name)
	}
	rows, ok := rowsRaw.([]any)
	if !ok {
		return "", fmt.Errorf("%w: %s.rows must be a list", ErrInvalidFieldValue, spec.Name)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", opener)

	if spec.Fields != nil {
		for _, f := range orderedFieldSpecs(spec.Fields) {
			val, ok := lookupString(vars, f.Name)
			if !ok || val == "" {
				continue
			}
			fmt.Fprintf(&b, "%s: %s\n", f.Name, val)
		}
	}

	delim := spec.Rows.Delimiter
	if delim == "" {
		delim = "|"
	}
	for i, row := range rows {
		cells, ok := row.([]any)
		if !ok {
			return "", fmt.Errorf("%w: %s.rows[%d] must be a list", ErrInvalidFieldValue, spec.Name, i)
		}
		strCells := make([]string, len(cells))
		for j, cell := range cells {
			strCells[j] = fmt.Sprintf("%v", cell)
		}
		fmt.Fprintln(&b, strings.Join(strCells, delim))
	}
	b.WriteString(":::\n")
	return b.String(), nil
}

func orderedFieldSpecs(fields *FieldsSpec) []FieldSpec {
	declared := append(append([]FieldSpec{}, fields.Required...), fields.Optional...)
	if len(fields.OutputOrder) == 0 {
		return declared
	}
	byName := make(map[string]FieldSpec, len(declared))
	for _, field := range declared {
		byName[field.Name] = field
	}
	ordered := make([]FieldSpec, 0, len(declared))
	seen := make(map[string]bool, len(declared))
	for _, name := range fields.OutputOrder {
		ordered = append(ordered, byName[name])
		seen[name] = true
	}
	for _, field := range declared {
		if !seen[field.Name] {
			ordered = append(ordered, field)
		}
	}
	return ordered
}

func renderRawBody(opener, body string) string {
	return opener + "\n" + strings.TrimRight(body, "\r\n") + "\n:::\n"
}

func validateRenderedOutput(spec *LayoutSpec, out string) error {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("%w: rendered block is incomplete", ErrInvalidFieldValue)
	}
	return validateRenderedBody(spec, strings.Join(lines[1:len(lines)-1], "\n"))
}

func validateRenderedBody(spec *LayoutSpec, rawBody string) error {
	issues := validateBlockBody(spec, strings.Split(rawBody, "\n"))
	if len(issues) == 0 {
		return nil
	}
	issue := issues[0]
	cause := issue.cause
	if cause == nil {
		cause = ErrInvalidFieldValue
	}
	if issue.field != "" {
		return fmt.Errorf("%w: %s.%s: %s", cause, spec.Name, issue.field, issue.message)
	}
	return fmt.Errorf("%w: %s: %s", cause, spec.Name, issue.message)
}

func exampleJSONBodyKind(example string) string {
	body := strings.Split(example, "\n")
	for _, ln := range body {
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" || strings.HasPrefix(trimmed, ":::") {
			continue
		}
		switch trimmed[0] {
		case '{':
			return "object"
		case '[':
			return "array"
		default:
			return ""
		}
	}
	return ""
}

func parseJSONFieldValue(val string) any {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return val
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded
		}
	}
	return val
}

func setJSONField(obj map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	current := obj
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func lookupString(vars map[string]any, key string) (string, bool) {
	v, ok := vars[key]
	if !ok {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	case fmt.Stringer:
		return t.String(), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func checkEnum(f FieldSpec, val string) error {
	if len(f.Enum) == 0 {
		return nil
	}
	for _, allowed := range f.Enum {
		if allowed == val {
			return nil
		}
	}
	return fmt.Errorf("%w: %s must be one of %v, got %q", ErrInvalidFieldValue, f.Name, f.Enum, val)
}
