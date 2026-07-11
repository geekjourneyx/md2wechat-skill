package layoutcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/geekjourneyx/md2wechat-skill/internal/assets"
)

const layoutDirEnvVar = "MD2WECHAT_LAYOUT_DIR"

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
	for _, dir := range overrideDirs() {
		if dir == "" {
			continue
		}
		if err := c.loadFromDir(dir); err != nil {
			return fmt.Errorf("load layout dir %s: %w", dir, err)
		}
	}
	return nil
}

func overrideDirs() []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "md2wechat", "layout"))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "layout"))
	}
	if envDir := strings.TrimSpace(os.Getenv(layoutDirEnvVar)); envDir != "" {
		dirs = append(dirs, envDir)
	}
	return dirs
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
			c.modules[spec.Name] = spec
		}
	}
	return nil
}

func (c *Catalog) loadFromDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.Walk(dir, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		if !strings.HasSuffix(fi.Name(), ".yaml") && !strings.HasSuffix(fi.Name(), ".yml") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		spec, err := parseLayoutSpec(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", p, err)
		}
		c.modules[spec.Name] = spec
		return nil
	})
}

func parseLayoutSpec(data []byte) (*LayoutSpec, error) {
	var spec LayoutSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	if spec.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported schema_version %q (expected %q)", spec.SchemaVersion, SchemaVersion)
	}
	if spec.Name == "" {
		return nil, fmt.Errorf("name is required")
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
	if spec.Metadata.Author == "" || spec.Metadata.Provenance == "" {
		return nil, fmt.Errorf("metadata.author and metadata.provenance are required")
	}
	if err := validateWitnessSpecs(&spec, declaredFields); err != nil {
		return nil, err
	}
	return &spec, nil
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
	return declared, nil
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
