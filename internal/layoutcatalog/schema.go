package layoutcatalog

const SchemaVersion = "1"

const (
	BodyFormatFields     = "fields"
	BodyFormatRows       = "rows"
	BodyFormatJSONObject = "json_object"
	BodyFormatJSONArray  = "json_array"
)

const (
	LifecycleRecommended   = "recommended"
	LifecycleCompatibility = "compatibility"
)

var ValidServes = map[string]bool{
	"attention":    true,
	"readability":  true,
	"memorability": true,
	"conversion":   true,
}

var ValidBodyFormats = map[string]bool{
	BodyFormatFields:     true,
	BodyFormatRows:       true,
	BodyFormatJSONObject: true,
	BodyFormatJSONArray:  true,
}

var ValidLifecycles = map[string]bool{
	LifecycleRecommended:   true,
	LifecycleCompatibility: true,
}

type FieldSpec struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Enum        []string `yaml:"enum,omitempty"`
	Example     string   `yaml:"example,omitempty"`
}

type RowsSpec struct {
	Delimiter   string      `yaml:"delimiter"`
	MinColumns  int         `yaml:"min_columns"`
	Schema      []FieldSpec `yaml:"schema"`
	Description string      `yaml:"description,omitempty"`
}

type FieldsSpec struct {
	Required []FieldSpec `yaml:"required,omitempty"`
	Optional []FieldSpec `yaml:"optional,omitempty"`
}

type LayoutMetadata struct {
	Author     string `yaml:"author,omitempty"`
	Provenance string `yaml:"provenance,omitempty"`
	InspiredBy string `yaml:"inspired_by,omitempty"`
}

type LayoutSpec struct {
	SchemaVersion      string         `yaml:"schema_version"`
	Name               string         `yaml:"name"`
	Lifecycle          string         `yaml:"lifecycle,omitempty"`
	BodyFormat         string         `yaml:"body_format,omitempty" json:"body_format,omitempty"`
	Version            string         `yaml:"version"`
	Since              string         `yaml:"since,omitempty"`
	Category           string         `yaml:"category"`
	Serves             []string       `yaml:"serves"`
	ContentTypes       []string       `yaml:"content_types,omitempty"`
	Industry           []string       `yaml:"industry,omitempty"`
	Tags               []string       `yaml:"tags,omitempty"`
	Position           string         `yaml:"position,omitempty"`
	WhenToUse          string         `yaml:"when_to_use,omitempty"`
	WhenNotToUse       string         `yaml:"when_not_to_use,omitempty"`
	PairsWellWith      []string       `yaml:"pairs_well_with,omitempty"`
	AvoidCombiningWith []string       `yaml:"avoid_combining_with,omitempty"`
	AntiPattern        string         `yaml:"anti_pattern,omitempty"`
	Fields             *FieldsSpec    `yaml:"fields,omitempty"`
	Rows               *RowsSpec      `yaml:"rows,omitempty"`
	Example            string         `yaml:"example,omitempty"`
	Metadata           LayoutMetadata `yaml:"metadata"`
}

type ListFilter struct {
	Category    string
	Serves      string
	ContentType string
	Industry    string
	Tag         string
	Lifecycle   string
}
