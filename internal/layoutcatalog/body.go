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
	fieldTypes  map[string][]string
	jsonItems   []jsonItemFacts
	imageCount  int
	itemCount   int
}

func validateBlockBody(spec *LayoutSpec, body []string) []bodyValidationIssue {
	formats := append([]string{spec.BodyFormat}, spec.CompatibleBodyFormats...)
	var best []bodyValidationIssue
	bestParseIssues := 0
	for i, format := range formats {
		facts, parseIssues := parseBodyFacts(spec, format, body)
		issues := parseIssues
		if len(parseIssues) == 0 {
			issues = append(issues, validateBodyFacts(spec, format, facts)...)
		}
		if len(issues) == 0 {
			return nil
		}
		if i == 0 || len(parseIssues) < bestParseIssues {
			best = issues
			bestParseIssues = len(parseIssues)
		}
	}
	if len(best) == 0 {
		best = []bodyValidationIssue{{message: "body did not match an accepted format"}}
	}
	if len(formats) > 1 {
		best[0].message += fmt.Sprintf(" (accepted formats: %s)", strings.Join(formats, ", "))
	}
	return best
}

func parseBodyFacts(spec *LayoutSpec, format string, body []string) (bodyFacts, []bodyValidationIssue) {
	facts := newBodyFacts()
	switch format {
	case BodyFormatFields, "":
		return parseFieldsBody(spec.Fields, body)
	case BodyFormatMarkdownFields:
		facts.addDeclaredFields(spec.Fields, body)
		facts.imageCount = countMarkdownImages(body)
		return facts, nil
	case BodyFormatJSONObject, BodyFormatJSONArray:
		fields, types, items, err := parseJSONBodyData(body, format)
		if err != nil {
			return facts, []bodyValidationIssue{{message: err.Error()}}
		}
		for name, value := range fields {
			facts.fields = append(facts.fields, bodyField{name: name, value: value})
			facts.fieldValues[name] = append(facts.fieldValues[name], value)
		}
		facts.fieldTypes = types
		facts.jsonItems = items
		facts.itemCount = len(items)
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

func parseFieldsBody(fields *FieldsSpec, body []string) (bodyFacts, []bodyValidationIssue) {
	facts := newBodyFacts()
	for _, raw := range body {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			return facts, []bodyValidationIssue{{message: "fields body lines must use key: value syntax", cause: ErrInvalidFieldValue}}
		}
		name := strings.TrimSpace(line[:idx])
		if !isDeclaredField(fields, name) {
			return facts, []bodyValidationIssue{{field: name, message: fmt.Sprintf("unknown field %q", name), cause: ErrInvalidFieldValue}}
		}
		facts.addField(name, strings.TrimSpace(line[idx+1:]))
	}
	return facts, nil
}

func newBodyFacts() bodyFacts {
	return bodyFacts{fieldValues: map[string][]string{}, fieldTypes: map[string][]string{}}
}

func (facts *bodyFacts) addField(name, value string) {
	facts.fields = append(facts.fields, bodyField{name: name, value: value})
	facts.fieldValues[name] = append(facts.fieldValues[name], value)
	facts.fieldTypes[name] = append(facts.fieldTypes[name], "string")
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
		if len(spec.Rows.Schema) > 0 {
			for i, cell := range cells {
				if i >= len(spec.Rows.Schema) {
					break
				}
				field := spec.Rows.Schema[i]
				value := strings.TrimSpace(cell)
				if err := checkEnum(field, value); err != nil {
					return facts, []bodyValidationIssue{{field: field.Name, message: err.Error(), cause: ErrInvalidFieldValue}}
				}
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
		configured := containsString(normalizedDialoguePrefixes(bodySpec), prefix)
		if configured && fullWidthSeparator {
			return facts, []bodyValidationIssue{{field: prefix, message: "configured dialogue prefixes require an ASCII colon"}}
		}
		if !configured {
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
			issues = append(issues, validateRequiredPairs(spec.Body.RequiredPairs, facts, format == BodyFormatDialogue)...)
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
		if format == BodyFormatJSONArray && len(facts.jsonItems) > 0 {
			for index, item := range facts.jsonItems {
				itemIssues := validateStructuredFields(spec, item.values, item.types)
				for _, issue := range itemIssues {
					issue.message = fmt.Sprintf("item %d: %s", index+1, issue.message)
					issues = append(issues, issue)
				}
			}
			return issues
		}
		issues = append(issues, validateStructuredFields(spec, facts.fieldValues, facts.fieldTypes)...)
	}
	return issues
}

func validateStructuredFields(spec *LayoutSpec, values, types map[string][]string) []bodyValidationIssue {
	var issues []bodyValidationIssue
	active, selectorPresent, selectorIssues := resolveVariant(spec.Variants, values)
	issues = append(issues, selectorIssues...)
	fieldIssues := validateFields(spec.Fields, spec.Variants, values, types, !selectorPresent)
	for _, issue := range fieldIssues {
		duplicateSelectorError := false
		for _, selectorIssue := range selectorIssues {
			if issue.field == selectorIssue.field && strings.Contains(issue.message, "must be one of") {
				duplicateSelectorError = true
				break
			}
		}
		if !duplicateSelectorError {
			issues = append(issues, issue)
		}
	}
	if len(selectorIssues) == 0 && active != nil {
		issues = append(issues, validateVariantFields(*active, values)...)
	}
	return issues
}

func validateVariantFields(variant VariantSpec, values map[string][]string) []bodyValidationIssue {
	var issues []bodyValidationIssue
	for _, name := range variant.Required {
		if !hasNonEmptyValue(values[name]) {
			issues = append(issues, bodyValidationIssue{field: name, message: fmt.Sprintf("variant %s requires field %s", variant.Name, name), cause: ErrMissingRequiredField})
		}
	}
	for _, group := range variant.RequiredAny {
		found := false
		for _, name := range group {
			if hasNonEmptyValue(values[name]) {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, bodyValidationIssue{message: fmt.Sprintf("variant %s requires at least one of %v", variant.Name, group), cause: ErrMissingRequiredField})
		}
	}
	issues = append(issues, validateFieldShapes(variant.Shapes, values)...)
	return issues
}

func resolveVariant(variants []VariantSpec, values map[string][]string) (*VariantSpec, bool, []bodyValidationIssue) {
	if len(variants) == 0 {
		return nil, false, nil
	}
	present := false
	for _, field := range []string{"type", "variant"} {
		for _, value := range values[field] {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			present = true
			if !isVariantIdentity(variants, value) {
				return nil, true, []bodyValidationIssue{{field: field, message: fmt.Sprintf("unknown %s %q", field, value), cause: ErrInvalidFieldValue}}
			}
		}
	}
	for _, field := range []string{"type", "variant"} {
		selector := lastNonEmptyValue(values[field])
		if selector == "" {
			continue
		}
		for i := range variants {
			if selector == variants[i].Name || containsString(variants[i].Aliases, selector) {
				return &variants[i], true, nil
			}
		}
	}
	return nil, present, nil
}

func formatSupportsDeclaredFields(format string) bool {
	switch format {
	case BodyFormatFields, BodyFormatMarkdownFields, BodyFormatJSONObject, BodyFormatJSONArray, BodyFormatRows, "":
		return true
	default:
		return false
	}
}

func validateRequiredPairs(pairs [][2]string, facts bodyFacts, ordered bool) []bodyValidationIssue {
	if ordered && len(pairs) > 0 {
		for i := 0; i < len(facts.fields); i += 2 {
			if i+1 >= len(facts.fields) {
				for _, pair := range pairs {
					if facts.fields[i].name == pair[0] {
						return []bodyValidationIssue{{field: pair[1], message: fmt.Sprintf("paired field %s is required", pair[1]), cause: ErrMissingRequiredField}}
					}
				}
				return []bodyValidationIssue{{field: facts.fields[i].name, message: "dialogue pair is incomplete", cause: ErrMissingRequiredField}}
			}
			matched := false
			for _, pair := range pairs {
				if facts.fields[i].name == pair[0] && facts.fields[i+1].name == pair[1] {
					matched = true
					break
				}
			}
			if !matched {
				return []bodyValidationIssue{{field: facts.fields[i].name, message: fmt.Sprintf("dialogue fields must form ordered pairs %v", pairs), cause: ErrInvalidFieldValue}}
			}
		}
		if len(facts.fields) == 0 {
			return []bodyValidationIssue{{message: "dialogue requires at least one ordered pair", cause: ErrMissingRequiredField}}
		}
		return nil
	}
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

func validateFields(fields *FieldsSpec, variants []VariantSpec, values, types map[string][]string, applyDefaultRules bool) []bodyValidationIssue {
	if fields == nil {
		return nil
	}
	var issues []bodyValidationIssue
	for _, field := range fields.Required {
		value := lastNonEmptyValue(values[field.Name])
		if value == "" {
			issues = append(issues, bodyValidationIssue{field: field.Name, message: "required field missing", cause: ErrMissingRequiredField})
			continue
		}
		issues = append(issues, validateFieldValue(field, variants, value, lastNonEmptyValue(types[field.Name]))...)
	}
	for _, field := range fields.Optional {
		if value := lastNonEmptyValue(values[field.Name]); value != "" {
			issues = append(issues, validateFieldValue(field, variants, value, lastNonEmptyValue(types[field.Name]))...)
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
	if applyDefaultRules {
		for _, name := range fields.RequiredWhenNoVariant {
			if !hasNonEmptyValue(values[name]) {
				issues = append(issues, bodyValidationIssue{field: name, message: "required field missing when no variant is selected", cause: ErrMissingRequiredField})
			}
		}
		for _, group := range fields.RequiredAnyWhenNoVariant {
			found := false
			for _, name := range group {
				if hasNonEmptyValue(values[name]) {
					found = true
					break
				}
			}
			if !found {
				issues = append(issues, bodyValidationIssue{message: fmt.Sprintf("at least one of %v is required when no variant is selected", group), cause: ErrMissingRequiredField})
			}
		}
	}
	issues = append(issues, validateFieldShapes(fields.Shapes, values)...)
	return issues
}

func validateFieldValue(field FieldSpec, variants []VariantSpec, value, actualType string) []bodyValidationIssue {
	var issues []bodyValidationIssue
	if err := checkFieldEnum(field, variants, value); err != nil {
		issues = append(issues, bodyValidationIssue{field: field.Name, message: err.Error()})
	}
	if field.ValueType == "string" {
		if actualType == "" {
			actualType = "string"
		}
		if actualType != "string" {
			issues = append(issues, bodyValidationIssue{field: field.Name, message: fmt.Sprintf("field %s must be a string, got %s", field.Name, actualType), cause: ErrInvalidFieldValue})
		}
	}
	return issues
}

func validateFieldShapes(shapes []FieldShapeSpec, values map[string][]string) []bodyValidationIssue {
	var issues []bodyValidationIssue
	for _, shape := range shapes {
		for _, value := range values[shape.Field] {
			if strings.TrimSpace(value) == "" {
				continue
			}
			var parts []string
			for _, part := range strings.Split(value, shape.Separator) {
				if strings.TrimSpace(part) != "" {
					parts = append(parts, strings.TrimSpace(part))
				}
			}
			if len(parts) < shape.MinParts {
				issues = append(issues, bodyValidationIssue{field: shape.Field, message: fmt.Sprintf("field %s requires at least %d parts separated by %q", shape.Field, shape.MinParts, shape.Separator), cause: ErrInvalidFieldValue})
				continue
			}
			if shape.ItemSeparator != "" {
				for _, part := range parts {
					nested := strings.Split(part, shape.ItemSeparator)
					valid := len(nested) >= shape.ItemMinParts
					for i := 0; valid && i < shape.ItemMinParts; i++ {
						valid = strings.TrimSpace(nested[i]) != ""
					}
					if !valid {
						issues = append(issues, bodyValidationIssue{field: shape.Field, message: fmt.Sprintf("each %s entry requires at least %d non-empty parts separated by %q", shape.Field, shape.ItemMinParts, shape.ItemSeparator), cause: ErrInvalidFieldValue})
						break
					}
				}
			}
		}
	}
	return issues
}

func lastNonEmptyValue(values []string) string {
	for i := len(values) - 1; i >= 0; i-- {
		if value := strings.TrimSpace(values[i]); value != "" {
			return value
		}
	}
	return ""
}

func checkFieldEnum(field FieldSpec, variants []VariantSpec, value string) error {
	if (field.Name == "variant" || field.Name == "type") && isVariantIdentity(variants, value) {
		return nil
	}
	return checkEnum(field, value)
}

func isVariantIdentity(variants []VariantSpec, value string) bool {
	for _, variant := range variants {
		if value == variant.Name || containsString(variant.Aliases, value) {
			return true
		}
	}
	return false
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
