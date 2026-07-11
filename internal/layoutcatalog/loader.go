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
	if spec.Lifecycle == "" {
		spec.Lifecycle = LifecycleRecommended
	}
	if !ValidLifecycles[spec.Lifecycle] {
		return nil, fmt.Errorf("invalid lifecycle %q", spec.Lifecycle)
	}
	normalizeBodyFormat(&spec)
	if !ValidBodyFormats[spec.BodyFormat] {
		return nil, fmt.Errorf("invalid body_format %q", spec.BodyFormat)
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
	if spec.Fields != nil && spec.Rows != nil {
		return nil, fmt.Errorf("fields and rows are mutually exclusive")
	}
	if spec.BodyFormat == BodyFormatRows && spec.Rows == nil {
		return nil, fmt.Errorf("body_format rows requires rows")
	}
	if spec.BodyFormat != BodyFormatRows && spec.Rows != nil {
		return nil, fmt.Errorf("rows requires body_format rows")
	}
	if spec.Metadata.Author == "" || spec.Metadata.Provenance == "" {
		return nil, fmt.Errorf("metadata.author and metadata.provenance are required")
	}
	return &spec, nil
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
