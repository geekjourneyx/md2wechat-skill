package layoutcatalog

import (
	"fmt"
	"strings"
)

func parseBlockOpener(line string) (ParsedOpener, error) {
	line = strings.TrimRight(line, " \t\r")
	if !strings.HasPrefix(line, ":::") {
		return ParsedOpener{}, fmt.Errorf("layout opener must start with :::")
	}
	rest := strings.TrimPrefix(line, ":::")
	nameEnd := 0
	for nameEnd < len(rest) && isModuleNameChar(rest[nameEnd], nameEnd == 0) {
		nameEnd++
	}
	if nameEnd == 0 {
		return ParsedOpener{}, fmt.Errorf("layout opener requires a valid module name")
	}
	parsed := ParsedOpener{Name: rest[:nameEnd], Params: map[string]string{}}
	suffix := rest[nameEnd:]
	if suffix == "" {
		return parsed, nil
	}

	switch suffix[0] {
	case '[':
		if !strings.HasSuffix(suffix, "]") || strings.Contains(suffix[1:len(suffix)-1], "]") {
			return ParsedOpener{}, fmt.Errorf("invalid bracket opener suffix")
		}
		parsed.Caption = suffix[1 : len(suffix)-1]
		parsed.bracket = true
		return parsed, nil
	case '{':
		if !strings.HasSuffix(suffix, "}") || strings.ContainsAny(suffix[1:len(suffix)-1], "{}") {
			return ParsedOpener{}, fmt.Errorf("invalid braced opener suffix")
		}
		params, err := parseAssignments(suffix[1 : len(suffix)-1])
		if err != nil {
			return ParsedOpener{}, err
		}
		parsed.RawParams = suffix
		parsed.Params = params
		return parsed, nil
	case ' ', '\t':
		raw := strings.TrimSpace(suffix)
		if raw == "" || strings.ContainsAny(raw, "＝：") {
			return ParsedOpener{}, fmt.Errorf("invalid token opener suffix")
		}
		if parsed.Name == "block" {
			return ParsedOpener{}, fmt.Errorf(":::block is not a valid layout module opener")
		}
		parsed.RawParams = raw
		if strings.Contains(raw, "=") {
			params, err := parseAssignments(raw)
			if err != nil {
				return ParsedOpener{}, err
			}
			parsed.Params = params
			return parsed, nil
		}
		if len(strings.Fields(raw)) != 1 {
			return ParsedOpener{}, fmt.Errorf("raw opener suffix must be one token")
		}
		return parsed, nil
	default:
		return ParsedOpener{}, fmt.Errorf("invalid layout opener suffix")
	}
}

func isModuleNameChar(ch byte, first bool) bool {
	if first {
		return ch >= 'a' && ch <= 'z'
	}
	return ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '-'
}

func parseAssignments(raw string) (map[string]string, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil, fmt.Errorf("opener parameters must not be empty")
	}
	params := make(map[string]string, len(fields))
	for _, field := range fields {
		if strings.Count(field, "=") != 1 {
			return nil, fmt.Errorf("invalid opener parameter %q", field)
		}
		parts := strings.SplitN(field, "=", 2)
		if parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("opener parameter keys and values must not be empty")
		}
		if _, exists := params[parts[0]]; exists {
			return nil, fmt.Errorf("duplicate opener parameter %q", parts[0])
		}
		params[parts[0]] = parts[1]
	}
	return params, nil
}

func validateOpener(parsed ParsedOpener, spec *OpenerSpec) (ParsedOpener, error) {
	if spec == nil {
		if parsed.RawParams != "" || len(parsed.Params) != 0 {
			return ParsedOpener{}, fmt.Errorf("module does not support opener parameters")
		}
		return parsed, nil
	}

	style := openerParamStyle(parsed)
	if parsed.bracket {
		if spec.Caption {
			if spec.ParamStyle == ParamStyleBracket {
				return ParsedOpener{}, fmt.Errorf("bracket content is reserved for the opener caption")
			}
			return parsed, nil
		}
		if spec.ParamStyle != ParamStyleBracket {
			return ParsedOpener{}, fmt.Errorf("module does not support an opener caption")
		}
		parsed.Params[spec.Params[0].Name] = parsed.Caption
		parsed.Caption = ""
		style = ParamStyleBracket
	}
	if parsed.RawParams == "" && len(parsed.Params) == 0 {
		return parsed, nil
	}
	if style != spec.ParamStyle {
		return ParsedOpener{}, fmt.Errorf("module expects opener param_style %q", spec.ParamStyle)
	}
	if style == ParamStyleToken {
		parsed.Params[spec.Params[0].Name] = parsed.RawParams
	}
	declared := make(map[string]ParamSpec, len(spec.Params))
	for _, param := range spec.Params {
		declared[param.Name] = param
	}
	for name, value := range parsed.Params {
		param, ok := declared[name]
		if !ok {
			return ParsedOpener{}, fmt.Errorf("undeclared opener parameter %q", name)
		}
		if !stringAllowed(value, param.Enum) {
			return ParsedOpener{}, fmt.Errorf("opener parameter %s must be one of %v, got %q", name, param.Enum, value)
		}
	}
	return parsed, nil
}

func openerParamStyle(parsed ParsedOpener) string {
	if strings.HasPrefix(parsed.RawParams, "{") {
		return ParamStyleBraces
	}
	if len(parsed.Params) != 0 {
		return ParamStyleTokens
	}
	if parsed.RawParams != "" {
		return ParamStyleToken
	}
	return ""
}

func stringAllowed(value string, enum []string) bool {
	if len(enum) == 0 {
		return true
	}
	for _, allowed := range enum {
		if value == allowed {
			return true
		}
	}
	return false
}

func renderOpener(spec *LayoutSpec, vars map[string]any) (string, error) {
	base := ":::" + spec.Name
	if spec.Opener == nil {
		if caption, ok := lookupString(vars, "caption"); ok && caption != "" {
			return renderBracket(base, caption)
		}
		return base, nil
	}
	if spec.Opener.Caption {
		if caption, ok := lookupString(vars, "caption"); ok && caption != "" {
			return renderBracket(base, caption)
		}
	}

	values := make([]string, 0, len(spec.Opener.Params))
	for _, param := range spec.Opener.Params {
		value, ok := lookupString(vars, param.Name)
		if !ok || value == "" {
			value = param.Default
		}
		if value == "" {
			continue
		}
		if strings.ContainsAny(value, " \t\r\n") {
			return "", fmt.Errorf("opener parameter %s must be one token", param.Name)
		}
		if !stringAllowed(value, param.Enum) {
			return "", fmt.Errorf("opener parameter %s must be one of %v, got %q", param.Name, param.Enum, value)
		}
		values = append(values, param.Name+"="+value)
	}
	if len(values) == 0 {
		return base, nil
	}
	switch spec.Opener.ParamStyle {
	case ParamStyleBraces:
		return base + "{" + strings.Join(values, " ") + "}", nil
	case ParamStyleTokens:
		return base + " " + strings.Join(values, " "), nil
	case ParamStyleBracket, ParamStyleToken:
		if len(values) != 1 {
			return "", fmt.Errorf("opener param_style %q requires exactly one parameter", spec.Opener.ParamStyle)
		}
		value := strings.SplitN(values[0], "=", 2)[1]
		if spec.Opener.ParamStyle == ParamStyleBracket {
			return renderBracket(base, value)
		}
		return base + " " + value, nil
	default:
		return base, nil
	}
}

func renderBracket(base, value string) (string, error) {
	if strings.ContainsAny(value, "[]\r\n") {
		return "", fmt.Errorf("bracket opener value contains unsupported punctuation")
	}
	return base + "[" + value + "]", nil
}
