package layoutcatalog

import (
	"fmt"
	"regexp"
	"strings"
)

var markdownImageLineRE = regexp.MustCompile(`!\[[^\]]*\]\((https?://[^)\s]+|<https?://[^>]+>)\)`)

type bodyValidationIssue struct {
	field   string
	message string
	cause   error
}

type bodyField struct {
	name  string
	value string
}

type bodyFacts struct {
	fields      []bodyField
	fieldValues map[string][]string
	imageCount  int
	itemCount   int
}

func validateBlockBody(spec *LayoutSpec, body []string) []bodyValidationIssue {
	formats := append([]string{spec.BodyFormat}, spec.CompatibleBodyFormats...)
	var primary []bodyValidationIssue
	for i, format := range formats {
		facts, parseIssues := parseBodyFacts(spec, format, body)
		issues := append(parseIssues, validateBodyFacts(spec, format, facts)...)
		if len(issues) == 0 {
			return nil
		}
		if i == 0 {
			primary = issues
		}
	}
	if len(primary) == 0 {
		primary = []bodyValidationIssue{{message: "body did not match an accepted format"}}
	}
	if len(formats) > 1 {
		primary[0].message += fmt.Sprintf(" (accepted formats: %s)", strings.Join(formats, ", "))
	}
	return primary
}

func parseBodyFacts(spec *LayoutSpec, format string, body []string) (bodyFacts, []bodyValidationIssue) {
	facts := newBodyFacts()
	switch format {
	case BodyFormatFields, BodyFormatMarkdownFields, "":
		facts.addDeclaredFields(spec.Fields, body)
		facts.imageCount = countMarkdownImages(body)
		return facts, nil
	case BodyFormatJSONObject, BodyFormatJSONArray:
		fields, err := parseJSONBodyFields(body, format)
		if err != nil {
			return facts, []bodyValidationIssue{{message: err.Error()}}
		}
		for name, value := range fields {
			facts.addField(name, value)
		}
		return facts, nil
	case BodyFormatRows:
		return parseRowsBody(spec, body)
	case BodyFormatMarkdownImages:
		facts.imageCount = countMarkdownImages(body)
		facts.itemCount = facts.imageCount
		return facts, nil
	case BodyFormatSplit:
		return parseSplitBody(spec.Body, body)
	case BodyFormatLines:
		return parseLinesBody(spec.Body, body)
	case BodyFormatDialogue:
		return parseDialogueBody(spec.Body, body)
	default:
		return facts, []bodyValidationIssue{{message: fmt.Sprintf("unsupported body_format %q", format)}}
	}
}

func newBodyFacts() bodyFacts {
	return bodyFacts{fieldValues: map[string][]string{}}
}

func (facts *bodyFacts) addField(name, value string) {
	facts.fields = append(facts.fields, bodyField{name: name, value: value})
	facts.fieldValues[name] = append(facts.fieldValues[name], value)
}

func (facts *bodyFacts) addDeclaredFields(fields *FieldsSpec, body []string) {
	for _, line := range body {
		name, value, ok := parseDeclaredField(fields, line)
		if ok {
			facts.addField(name, value)
		}
	}
}

func parseDeclaredField(fields *FieldsSpec, line string) (string, string, bool) {
	if fields == nil {
		return "", "", false
	}
	line = strings.TrimSpace(strings.TrimRight(line, "\r"))
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	name := strings.TrimSpace(line[:idx])
	if !isDeclaredField(fields, name) {
		return "", "", false
	}
	return name, strings.TrimSpace(line[idx+1:]), true
}

func isDeclaredField(fields *FieldsSpec, name string) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.Required {
		if field.Name == name {
			return true
		}
	}
	for _, field := range fields.Optional {
		if field.Name == name {
			return true
		}
	}
	return false
}

func parseRowsBody(spec *LayoutSpec, body []string) (bodyFacts, []bodyValidationIssue) {
	facts := newBodyFacts()
	if spec.Rows == nil {
		return facts, []bodyValidationIssue{{message: "rows body requires rows schema"}}
	}
	delimiter := spec.Rows.Delimiter
	if delimiter == "" {
		delimiter = "|"
	}
	for _, line := range body {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		if name, value, ok := parseDeclaredField(spec.Fields, line); ok {
			facts.addField(name, value)
			continue
		}
		facts.itemCount++
		cells := strings.Split(line, delimiter)
		if len(cells) < spec.Rows.MinColumns {
			return facts, []bodyValidationIssue{{message: fmt.Sprintf("row requires at least %d columns", spec.Rows.MinColumns), cause: ErrMissingRequiredField}}
		}
		for i := 0; i < spec.Rows.MinColumns; i++ {
			if strings.TrimSpace(cells[i]) == "" {
				return facts, []bodyValidationIssue{{message: fmt.Sprintf("row column %d must not be empty", i+1)}}
			}
		}
	}
	if facts.itemCount == 0 {
		return facts, []bodyValidationIssue{{message: "rows module requires at least one data row"}}
	}
	return facts, nil
}

func parseSplitBody(bodySpec *BodySpec, body []string) (bodyFacts, []bodyValidationIssue) {
	facts := newBodyFacts()
	separator := "---"
	if bodySpec != nil && bodySpec.Separator != "" {
		separator = bodySpec.Separator
	}
	separatorIndex := -1
	for i, line := range body {
		if strings.TrimSpace(strings.TrimRight(line, "\r")) == separator {
			if separatorIndex >= 0 {
				return facts, []bodyValidationIssue{{message: "split body requires exactly one standalone separator"}}
			}
			separatorIndex = i
		}
	}
	if separatorIndex < 0 {
		return facts, []bodyValidationIssue{{message: "split body requires a standalone separator"}}
	}
	if !hasNonEmptyLine(body[:separatorIndex]) || !hasNonEmptyLine(body[separatorIndex+1:]) {
		return facts, []bodyValidationIssue{{message: "split body requires two non-empty sides"}}
	}
	facts.itemCount = 2
	return facts, nil
}

func parseLinesBody(bodySpec *BodySpec, body []string) (bodyFacts, []bodyValidationIssue) {
	facts := newBodyFacts()
	separator := ""
	var allowedPrefixes []string
	if bodySpec != nil {
		separator = bodySpec.Separator
		allowedPrefixes = bodySpec.AllowedPrefixes
	}
	for _, line := range body {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		if len(allowedPrefixes) > 0 {
			matched := false
			for _, prefix := range allowedPrefixes {
				if strings.HasPrefix(line, prefix) && strings.TrimSpace(strings.TrimPrefix(line, prefix)) != "" {
					matched = true
					break
				}
			}
			if !matched {
				return facts, []bodyValidationIssue{{message: fmt.Sprintf("line must use one of prefixes %v", allowedPrefixes)}}
			}
		}
		parts := []string{line}
		if separator != "" {
			parts = strings.Split(line, separator)
		}
		for _, part := range parts {
			if strings.TrimSpace(part) != "" {
				facts.itemCount++
			}
		}
	}
	return facts, nil
}

func parseDialogueBody(bodySpec *BodySpec, body []string) (bodyFacts, []bodyValidationIssue) {
	facts := newBodyFacts()
	for _, line := range body {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		separatorWidth := len(":")
		fullWidthSeparator := false
		if fullWidth := strings.Index(line, "："); idx < 0 || (fullWidth >= 0 && fullWidth < idx) {
			idx = fullWidth
			separatorWidth = len("：")
			fullWidthSeparator = true
		}
		if idx < 0 {
			return facts, []bodyValidationIssue{{message: "dialogue line requires a speaker prefix"}}
		}
		prefix := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+separatorWidth:])
		if prefix == "" {
			return facts, []bodyValidationIssue{{message: "dialogue speaker prefix must not be empty"}}
		}
		if value == "" {
			return facts, []bodyValidationIssue{{field: prefix, message: "dialogue line must not be empty"}}
		}
		if !containsString(normalizedDialoguePrefixes(bodySpec), prefix) {
			if bodySpec == nil || !bodySpec.AllowNamedSpeakers {
				return facts, []bodyValidationIssue{{field: prefix, message: fmt.Sprintf("dialogue prefix %q is not allowed", prefix)}}
			}
			if !fullWidthSeparator {
				return facts, []bodyValidationIssue{{field: prefix, message: "legacy named speakers require a full-width colon"}}
			}
		}
		facts.addField(prefix, value)
		facts.itemCount++
	}
	return facts, nil
}

func normalizedDialoguePrefixes(bodySpec *BodySpec) []string {
	if bodySpec == nil {
		return nil
	}
	prefixes := make([]string, 0, len(bodySpec.AllowedPrefixes))
	for _, prefix := range bodySpec.AllowedPrefixes {
		if normalized := normalizeDialoguePrefix(prefix); normalized != "" {
			prefixes = append(prefixes, normalized)
		}
	}
	return prefixes
}

func normalizeDialoguePrefix(prefix string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(prefix), ":"), "："))
}

func validateBodyFacts(spec *LayoutSpec, format string, facts bodyFacts) []bodyValidationIssue {
	var issues []bodyValidationIssue
	if spec.Body != nil {
		if format == BodyFormatMarkdownFields || format == BodyFormatDialogue {
			issues = append(issues, validateRequiredPairs(spec.Body.RequiredPairs, facts)...)
		}
		if format == BodyFormatMarkdownFields && spec.Body.Group != nil {
			issues = append(issues, validateFieldGroups(spec.Body.Group, facts.fields)...)
		}
		if format == BodyFormatMarkdownImages || format == BodyFormatMarkdownFields {
			if facts.imageCount < spec.Body.MinImages {
				issues = append(issues, bodyValidationIssue{message: fmt.Sprintf("body requires at least %d image(s)", spec.Body.MinImages)})
			}
			if spec.Body.MaxImages > 0 && facts.imageCount > spec.Body.MaxImages {
				issues = append(issues, bodyValidationIssue{message: fmt.Sprintf("body allows at most %d image(s)", spec.Body.MaxImages)})
			}
		}
		if (format == BodyFormatMarkdownImages || format == BodyFormatLines || format == BodyFormatDialogue) && facts.itemCount < spec.Body.MinItems {
			issues = append(issues, bodyValidationIssue{message: fmt.Sprintf("body requires at least %d items", spec.Body.MinItems)})
		}
	}
	if formatSupportsDeclaredFields(format) {
		issues = append(issues, validateFields(spec.Fields, facts.fieldValues)...)
	}
	return issues
}

func formatSupportsDeclaredFields(format string) bool {
	switch format {
	case BodyFormatFields, BodyFormatMarkdownFields, BodyFormatJSONObject, BodyFormatJSONArray, BodyFormatRows, "":
		return true
	default:
		return false
	}
}

func validateRequiredPairs(pairs [][2]string, facts bodyFacts) []bodyValidationIssue {
	for _, pair := range pairs {
		left := hasNonEmptyValue(facts.fieldValues[pair[0]])
		right := hasNonEmptyValue(facts.fieldValues[pair[1]])
		if !left {
			return []bodyValidationIssue{{field: pair[0], message: fmt.Sprintf("paired field %s is required", pair[0]), cause: ErrMissingRequiredField}}
		}
		if !right {
			return []bodyValidationIssue{{field: pair[1], message: fmt.Sprintf("paired field %s is required", pair[1]), cause: ErrMissingRequiredField}}
		}
	}
	return nil
}

func validateFieldGroups(group *FieldGroupSpec, fields []bodyField) []bodyValidationIssue {
	var groups []map[string]string
	for _, field := range fields {
		if field.name == group.Start {
			groups = append(groups, map[string]string{})
		}
		if len(groups) > 0 {
			groups[len(groups)-1][field.name] = field.value
		}
	}
	complete := 0
	for _, values := range groups {
		missing := ""
		for _, required := range group.Required {
			if strings.TrimSpace(values[required]) == "" {
				missing = required
				break
			}
		}
		if missing != "" {
			return []bodyValidationIssue{{field: missing, message: fmt.Sprintf("group field %s is required", missing), cause: ErrMissingRequiredField}}
		}
		complete++
	}
	if complete < group.Min {
		return []bodyValidationIssue{{message: fmt.Sprintf("body requires at least %d complete group(s)", group.Min), cause: ErrMissingRequiredField}}
	}
	return nil
}

func validateFields(fields *FieldsSpec, values map[string][]string) []bodyValidationIssue {
	if fields == nil {
		return nil
	}
	var issues []bodyValidationIssue
	for _, field := range fields.Required {
		if !hasNonEmptyValue(values[field.Name]) {
			issues = append(issues, bodyValidationIssue{field: field.Name, message: "required field missing", cause: ErrMissingRequiredField})
			continue
		}
		for _, value := range values[field.Name] {
			if value != "" {
				if err := checkEnum(field, value); err != nil {
					issues = append(issues, bodyValidationIssue{field: field.Name, message: err.Error()})
				}
			}
		}
	}
	for _, field := range fields.Optional {
		for _, value := range values[field.Name] {
			if value != "" {
				if err := checkEnum(field, value); err != nil {
					issues = append(issues, bodyValidationIssue{field: field.Name, message: err.Error()})
				}
			}
		}
	}
	for _, group := range fields.RequiredAny {
		found := false
		for _, name := range group {
			if hasNonEmptyValue(values[name]) {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, bodyValidationIssue{message: fmt.Sprintf("at least one of %v is required", group), cause: ErrMissingRequiredField})
		}
	}
	return issues
}

func countMarkdownImages(body []string) int {
	count := 0
	for _, line := range body {
		count += len(markdownImageLineRE.FindAllString(line, -1))
	}
	return count
}

func hasNonEmptyLine(lines []string) bool {
	for _, line := range lines {
		if strings.TrimSpace(strings.TrimRight(line, "\r")) != "" {
			return true
		}
	}
	return false
}

func hasNonEmptyValue(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
