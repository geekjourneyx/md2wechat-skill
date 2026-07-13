package layoutcatalog

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/geekjourneyx/md2wechat-skill/internal/assets"
)

type Catalog struct {
	mu      sync.RWMutex
	modules map[string]*LayoutSpec
}

var (
	defaultCatalog     *Catalog
	defaultCatalogOnce sync.Once
	defaultCatalogErr  error
)

func NewCatalog() *Catalog {
	return &Catalog{modules: map[string]*LayoutSpec{}}
}

func DefaultCatalog() (*Catalog, error) {
	defaultCatalogOnce.Do(func() {
		cat := NewCatalog()
		defaultCatalogErr = cat.Load()
		if defaultCatalogErr == nil {
			defaultCatalog = cat
		}
	})
	return defaultCatalog, defaultCatalogErr
}

func ResetDefaultCatalogForTests() {
	defaultCatalog = nil
	defaultCatalogOnce = sync.Once{}
	defaultCatalogErr = nil
}

func (c *Catalog) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modules = map[string]*LayoutSpec{}

	if err := c.loadBuiltin(); err != nil {
		return fmt.Errorf("load builtin layout: %w", err)
	}
	return nil
}

func (c *Catalog) loadBuiltin() error {
	cats, err := assets.ListBuiltinLayoutCategories()
	if err != nil {
		return err
	}
	for _, cat := range cats {
		names, err := assets.ListBuiltinLayouts(cat)
		if err != nil {
			return err
		}
		for _, name := range names {
			data, err := assets.ReadBuiltinLayout(cat, name)
			if err != nil {
				return err
			}
			spec, err := parseLayoutSpec(data)
			if err != nil {
				return fmt.Errorf("parse builtin %s/%s: %w", cat, name, err)
			}
			if err := validateLoadedWitnesses(spec); err != nil {
				return fmt.Errorf("validate builtin %s/%s: %w", cat, name, err)
			}
			c.modules[spec.Name] = spec
		}
	}
	return nil
}

func parseLayoutSpec(data []byte) (*LayoutSpec, error) {
	var spec LayoutSpec
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&spec); err != nil {
		return nil, err
	}
	if spec.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported schema_version %q (expected %q)", spec.SchemaVersion, SchemaVersion)
	}
	if spec.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := validateLayoutModuleName(spec.Name); err != nil {
		return nil, err
	}
	if spec.Name == reservedModuleName {
		return nil, fmt.Errorf("layout module name %q is reserved", spec.Name)
	}
	if spec.Lifecycle == "" {
		spec.Lifecycle = LifecycleRecommended
	}
	if !ValidLifecycles[spec.Lifecycle] {
		return nil, fmt.Errorf("invalid lifecycle %q", spec.Lifecycle)
	}
	if err := validateOpenerSpec(spec.Opener); err != nil {
		return nil, err
	}
	normalizeBodyFormat(&spec)
	if !ValidBodyFormats[spec.BodyFormat] {
		return nil, fmt.Errorf("invalid body_format %q", spec.BodyFormat)
	}
	seenBodyFormats := map[string]bool{spec.BodyFormat: true}
	for _, format := range spec.CompatibleBodyFormats {
		if !ValidBodyFormats[format] {
			return nil, fmt.Errorf("invalid compatible body_format %q", format)
		}
		if seenBodyFormats[format] {
			return nil, fmt.Errorf("duplicate compatible body_format %q", format)
		}
		seenBodyFormats[format] = true
	}
	if spec.Category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if len(spec.Serves) == 0 {
		return nil, fmt.Errorf("serves must contain at least one value")
	}
	for _, s := range spec.Serves {
		if !ValidServes[s] {
			return nil, fmt.Errorf("invalid serves value: %q", s)
		}
	}
	declaredFields, err := validateFieldsSpec(spec.Fields)
	if err != nil {
		return nil, err
	}
	if err := validateBodySpec(spec.Body, declaredFields, seenBodyFormats); err != nil {
		return nil, err
	}
	if seenBodyFormats[BodyFormatRows] && spec.Rows == nil {
		return nil, fmt.Errorf("body_format rows requires rows")
	}
	if !seenBodyFormats[BodyFormatRows] && spec.Rows != nil {
		return nil, fmt.Errorf("rows requires body_format rows")
	}
	if err := validateRowsSpec(spec.Rows); err != nil {
		return nil, err
	}
	if spec.Metadata.Author == "" || spec.Metadata.Provenance == "" {
		return nil, fmt.Errorf("metadata.author and metadata.provenance are required")
	}
	if err := validateWitnessSpecs(&spec, declaredFields); err != nil {
		return nil, err
	}
	return &spec, nil
}

func validateLoadedWitnesses(spec *LayoutSpec) error {
	if spec.Lifecycle != LifecycleRecommended {
		return nil
	}
	if strings.TrimSpace(spec.Example) == "" {
		return fmt.Errorf("recommended layout %q requires a canonical example", spec.Name)
	}
	temporary := NewCatalog()
	temporary.modules[spec.Name] = spec
	if err := temporary.ValidateWitness(WitnessContract{
		Module: spec.Name, Example: spec.Example, AssertContains: spec.ExampleAssertContains,
	}); err != nil {
		return fmt.Errorf("canonical example: %w", err)
	}
	for _, variant := range spec.Variants {
		if strings.TrimSpace(variant.UseWhen) == "" {
			return fmt.Errorf("variant %q requires use_when", variant.Name)
		}
		if strings.TrimSpace(variant.Example) == "" {
			return fmt.Errorf("variant %q requires an executable example", variant.Name)
		}
		if err := temporary.ValidateWitness(WitnessContract{
			Module: spec.Name, Variant: variant.Name, VariantAliases: variant.Aliases,
			Example: variant.Example, AssertContains: variant.AssertContains,
		}); err != nil {
			return fmt.Errorf("variant %q example: %w", variant.Name, err)
		}
	}
	return nil
}

func validateRowsSpec(rows *RowsSpec) error {
	if rows == nil {
		return nil
	}
	if rows.MinColumns <= 0 {
		return fmt.Errorf("rows.min_columns must be greater than zero")
	}
	if len(rows.Schema) == 0 {
		return nil
	}
	if rows.MinColumns > len(rows.Schema) {
		return fmt.Errorf("rows.min_columns must not exceed rows.schema length")
	}
	seen := map[string]bool{}
	for _, field := range rows.Schema {
		if strings.TrimSpace(field.Name) == "" || strings.TrimSpace(field.Name) != field.Name {
			return fmt.Errorf("rows.schema field name %q is invalid", field.Name)
		}
		if seen[field.Name] {
			return fmt.Errorf("duplicate rows.schema field %q", field.Name)
		}
		seen[field.Name] = true
		if field.ValueType != "" && field.ValueType != "string" {
			return fmt.Errorf("rows.schema field %q has invalid value_type %q", field.Name, field.ValueType)
		}
	}
	return nil
}

func validateWitnessSpecs(spec *LayoutSpec, declaredFields map[string]bool) error {
	if spec.ExampleAssertContains != "" && !strings.Contains(spec.Example, spec.ExampleAssertContains) {
		return fmt.Errorf("example_assert_contains %q is absent from example", spec.ExampleAssertContains)
	}
	identities := make(map[string]string)
	for _, variant := range spec.Variants {
		name := strings.TrimSpace(variant.Name)
		if name == "" {
			return fmt.Errorf("variant name must not be empty")
		}
		if name != variant.Name {
			return fmt.Errorf("variant name %q must not have surrounding whitespace", variant.Name)
		}
		for _, identity := range append([]string{variant.Name}, variant.Aliases...) {
			normalized := strings.TrimSpace(identity)
			if normalized == "" {
				return fmt.Errorf("variant alias must not be empty")
			}
			if normalized != identity {
				return fmt.Errorf("variant name or alias %q must not have surrounding whitespace", identity)
			}
			if previous, exists := identities[normalized]; exists {
				return fmt.Errorf("duplicate variant name or alias %q (already declared by %q)", normalized, previous)
			}
			identities[normalized] = name
		}
	}
	for _, variant := range spec.Variants {
		if variant.AssertContains != "" && !strings.Contains(variant.Example, variant.AssertContains) {
			return fmt.Errorf("variant %q assert_contains %q is absent from its example", variant.Name, variant.AssertContains)
		}
		for _, field := range variant.Required {
			if !declaredFields[field] {
				return fmt.Errorf("variant %q required field %q is not declared", variant.Name, field)
			}
		}
		for _, group := range variant.RequiredAny {
			if len(group) == 0 {
				return fmt.Errorf("variant %q required_any group must not be empty", variant.Name)
			}
			for _, field := range group {
				if !declaredFields[field] {
					return fmt.Errorf("variant %q required_any field %q is not declared", variant.Name, field)
				}
			}
		}
		if err := validateFieldShapeSpecs(variant.Shapes, declaredFields, fmt.Sprintf("variant %q", variant.Name)); err != nil {
			return err
		}
	}
	return nil
}

func validateFieldsSpec(fields *FieldsSpec) (map[string]bool, error) {
	declared := map[string]bool{}
	if fields == nil {
		return declared, nil
	}
	for _, field := range append(append([]FieldSpec{}, fields.Required...), fields.Optional...) {
		if strings.TrimSpace(field.Name) == "" {
			return nil, fmt.Errorf("field name must not be empty")
		}
		if declared[field.Name] {
			return nil, fmt.Errorf("duplicate field %q", field.Name)
		}
		if field.ValueType != "" && field.ValueType != "string" {
			return nil, fmt.Errorf("field %q has invalid value_type %q", field.Name, field.ValueType)
		}
		declared[field.Name] = true
	}
	seenGroups := map[string]bool{}
	for _, group := range fields.RequiredAny {
		if len(group) == 0 {
			return nil, fmt.Errorf("required_any group must not be empty")
		}
		seenFields := map[string]bool{}
		for _, name := range group {
			if strings.TrimSpace(name) == "" {
				return nil, fmt.Errorf("required_any fields must not be empty")
			}
			if !declared[name] {
				return nil, fmt.Errorf("required_any field %q is not declared", name)
			}
			if seenFields[name] {
				return nil, fmt.Errorf("duplicate required_any field %q", name)
			}
			seenFields[name] = true
		}
		canonical := append([]string(nil), group...)
		sort.Strings(canonical)
		key := strings.Join(canonical, "\x00")
		if seenGroups[key] {
			return nil, fmt.Errorf("duplicate required_any group")
		}
		seenGroups[key] = true
	}
	seenDefaultRequired := map[string]bool{}
	for _, name := range fields.RequiredWhenNoVariant {
		if !declared[name] {
			return nil, fmt.Errorf("required_when_no_variant field %q is not declared", name)
		}
		if seenDefaultRequired[name] {
			return nil, fmt.Errorf("duplicate required_when_no_variant field %q", name)
		}
		seenDefaultRequired[name] = true
	}
	for _, group := range fields.RequiredAnyWhenNoVariant {
		if len(group) == 0 {
			return nil, fmt.Errorf("required_any_when_no_variant group must not be empty")
		}
		seen := map[string]bool{}
		for _, name := range group {
			if !declared[name] {
				return nil, fmt.Errorf("required_any_when_no_variant field %q is not declared", name)
			}
			if seen[name] {
				return nil, fmt.Errorf("duplicate required_any_when_no_variant field %q", name)
			}
			seen[name] = true
		}
	}
	seenOutput := map[string]bool{}
	for _, name := range fields.OutputOrder {
		if !declared[name] {
			return nil, fmt.Errorf("output_order field %q is not declared", name)
		}
		if seenOutput[name] {
			return nil, fmt.Errorf("duplicate output_order field %q", name)
		}
		seenOutput[name] = true
	}
	if err := validateFieldShapeSpecs(fields.Shapes, declared, "fields"); err != nil {
		return nil, err
	}
	return declared, nil
}

func validateFieldShapeSpecs(shapes []FieldShapeSpec, declared map[string]bool, owner string) error {
	for _, shape := range shapes {
		if !declared[shape.Field] {
			return fmt.Errorf("%s shape field %q is not declared", owner, shape.Field)
		}
		if shape.Separator == "" {
			return fmt.Errorf("%s shape separator must not be empty", owner)
		}
		if shape.MinParts <= 1 {
			return fmt.Errorf("%s shape min_parts must be greater than 1", owner)
		}
		if shape.MaxOccurrences < 0 {
			return fmt.Errorf("%s shape max_occurrences must be nonnegative", owner)
		}
		for _, rule := range shape.PartRules {
			if rule.MinParts < 0 || rule.MaxParts < 0 {
				return fmt.Errorf("%s shape part rule bounds must be nonnegative", owner)
			}
			if rule.MinParts == 0 && rule.MaxParts == 0 {
				return fmt.Errorf("%s shape part rule requires min_parts or max_parts", owner)
			}
			if rule.MinParts > 0 && rule.MaxParts > 0 && rule.MinParts > rule.MaxParts {
				return fmt.Errorf("%s shape part rule min_parts must not exceed max_parts", owner)
			}
			if len(rule.RequiredPositions) == 0 {
				return fmt.Errorf("%s shape part rule requires required_positions", owner)
			}
			seenPositions := map[int]bool{}
			for _, position := range rule.RequiredPositions {
				if position <= 0 {
					return fmt.Errorf("%s shape part rule positions must be positive", owner)
				}
				if rule.MaxParts > 0 && position > rule.MaxParts {
					return fmt.Errorf("%s shape part rule position %d exceeds max_parts", owner, position)
				}
				if seenPositions[position] {
					return fmt.Errorf("%s shape part rule position %d is duplicated", owner, position)
				}
				seenPositions[position] = true
			}
		}
		if shape.ItemSeparator == "" && shape.ItemMinParts != 0 {
			return fmt.Errorf("%s shape item_separator is required with item_min_parts", owner)
		}
		if shape.ItemSeparator != "" && shape.ItemMinParts <= 1 {
			return fmt.Errorf("%s shape item_min_parts must be greater than 1", owner)
		}
	}
	return nil
}

func validateBodySpec(body *BodySpec, declared, acceptedFormats map[string]bool) error {
	if body == nil {
		return nil
	}
	if body.MinImages < 0 {
		return fmt.Errorf("min_images must be nonnegative")
	}
	if body.MaxImages < 0 {
		return fmt.Errorf("max_images must be nonnegative")
	}
	if body.MinItems < 0 {
		return fmt.Errorf("min_items must be nonnegative")
	}
	if body.MaxImages != 0 && body.MaxImages < body.MinImages {
		return fmt.Errorf("max_images must be at least min_images when nonzero")
	}
	seenPairs := map[string]bool{}
	dialoguePrefixes := map[string]bool{}
	for _, prefix := range body.AllowedPrefixes {
		normalized := normalizeDialoguePrefix(prefix)
		if normalized != "" {
			dialoguePrefixes[normalized] = true
		}
	}
	for _, pair := range body.RequiredPairs {
		if strings.TrimSpace(pair[0]) == "" || strings.TrimSpace(pair[1]) == "" {
			return fmt.Errorf("required_pairs fields must not be empty")
		}
		if pair[0] == pair[1] {
			return fmt.Errorf("required_pairs fields must be distinct")
		}
		applicableFormat := false
		if acceptedFormats[BodyFormatDialogue] {
			applicableFormat = true
			for _, name := range pair {
				if !dialoguePrefixes[name] {
					return fmt.Errorf("required_pairs dialogue prefix %q is not configured", name)
				}
			}
		}
		if acceptedFormats[BodyFormatMarkdownFields] {
			applicableFormat = true
			for _, name := range pair {
				if !declared[name] {
					return fmt.Errorf("required_pairs field %q is not declared", name)
				}
			}
		}
		if !applicableFormat {
			return fmt.Errorf("required_pairs requires dialogue or markdown_fields body format")
		}
		canonical := pair
		if canonical[1] < canonical[0] {
			canonical[0], canonical[1] = canonical[1], canonical[0]
		}
		key := canonical[0] + "\x00" + canonical[1]
		if seenPairs[key] {
			return fmt.Errorf("duplicate required_pairs pair")
		}
		seenPairs[key] = true
	}
	if body.Group == nil {
		return nil
	}
	group := body.Group
	if strings.TrimSpace(group.Start) == "" {
		return fmt.Errorf("group.start must not be empty")
	}
	if !declared[group.Start] {
		return fmt.Errorf("group.start field %q is not declared", group.Start)
	}
	if len(group.Required) == 0 {
		return fmt.Errorf("group.required must not be empty")
	}
	seenRequired := map[string]bool{}
	for _, name := range group.Required {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("group.required fields must not be empty")
		}
		if !declared[name] {
			return fmt.Errorf("group.required field %q is not declared", name)
		}
		if seenRequired[name] {
			return fmt.Errorf("duplicate group.required field %q", name)
		}
		seenRequired[name] = true
	}
	if group.Min < 0 {
		return fmt.Errorf("group.min must be nonnegative")
	}
	return nil
}

func validateOpenerSpec(opener *OpenerSpec) error {
	if opener == nil {
		return nil
	}
	validStyles := map[string]bool{
		"":                true,
		ParamStyleBraces:  true,
		ParamStyleTokens:  true,
		ParamStyleBracket: true,
		ParamStyleToken:   true,
	}
	if !validStyles[opener.ParamStyle] {
		return fmt.Errorf("invalid opener param_style %q", opener.ParamStyle)
	}
	if opener.Caption && opener.ParamStyle == ParamStyleBracket {
		return fmt.Errorf("opener caption and bracket param_style are mutually exclusive")
	}
	if opener.Caption && len(opener.Params) > 0 {
		return fmt.Errorf("opener caption and opener params are mutually exclusive")
	}
	if len(opener.Params) > 0 && opener.ParamStyle == "" {
		return fmt.Errorf("opener params require param_style")
	}
	seen := make(map[string]bool, len(opener.Params))
	for _, param := range opener.Params {
		if !isValidOpenerParamName(param.Name) {
			return fmt.Errorf("invalid opener param name %q", param.Name)
		}
		if seen[param.Name] {
			return fmt.Errorf("duplicate opener param %q", param.Name)
		}
		seen[param.Name] = true
	}
	if (opener.ParamStyle == ParamStyleBracket || opener.ParamStyle == ParamStyleToken) && len(opener.Params) != 1 {
		return fmt.Errorf("opener param_style %q requires exactly one parameter", opener.ParamStyle)
	}
	return nil
}

func normalizeBodyFormat(spec *LayoutSpec) {
	if spec.BodyFormat != "" {
		return
	}
	if spec.Rows != nil {
		spec.BodyFormat = BodyFormatRows
		return
	}
	if bodyKind := exampleJSONBodyKind(spec.Example); bodyKind == "object" {
		spec.BodyFormat = BodyFormatJSONObject
		return
	} else if bodyKind == "array" {
		spec.BodyFormat = BodyFormatJSONArray
		return
	}
	spec.BodyFormat = BodyFormatFields
}

func (c *Catalog) Get(name string) (*LayoutSpec, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	spec, ok := c.modules[name]
	return spec, ok
}

func (c *Catalog) ListFiltered(f ListFilter) []*LayoutSpec {
	c.mu.RLock()
	defer c.mu.RUnlock()
	lifecycle := f.Lifecycle
	if lifecycle == "" {
		lifecycle = LifecycleRecommended
	}
	out := make([]*LayoutSpec, 0, len(c.modules))
	for _, m := range c.modules {
		if m.Lifecycle != lifecycle {
			continue
		}
		if f.Category != "" && m.Category != f.Category {
			continue
		}
		if f.Serves != "" && !contains(m.Serves, f.Serves) {
			continue
		}
		if f.ContentType != "" && !contains(m.ContentTypes, f.ContentType) {
			continue
		}
		if f.Industry != "" && !contains(m.Industry, f.Industry) {
			continue
		}
		if f.Tag != "" && !contains(m.Tags, f.Tag) {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
