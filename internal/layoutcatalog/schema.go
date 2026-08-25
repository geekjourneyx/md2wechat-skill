package layoutcatalog

const SchemaVersion = "1"

const (
	BodyFormatFields         = "fields"
	BodyFormatRows           = "rows"
	BodyFormatJSONObject     = "json_object"
	BodyFormatJSONArray      = "json_array"
	BodyFormatMarkdownImages = "markdown_images"
	BodyFormatMarkdownFields = "markdown_fields"
	BodyFormatSplit          = "split"
	BodyFormatLines          = "lines"
	BodyFormatDialogue       = "dialogue"
)

type AgentInputPosition string

const (
	InputBodyKV       AgentInputPosition = "body-kv"
	InputRowDSL       AgentInputPosition = "row-dsl"
	InputJSON         AgentInputPosition = "json"
	InputMarkdownBody AgentInputPosition = "markdown-body"
	InputHeaderAttrs  AgentInputPosition = "header-attrs"
	InputPrefixDSL    AgentInputPosition = "prefix-dsl"
)

var ValidAgentInputPositions = map[AgentInputPosition]bool{
	InputBodyKV:       true,
	InputRowDSL:       true,
	InputJSON:         true,
	InputMarkdownBody: true,
	InputHeaderAttrs:  true,
	InputPrefixDSL:    true,
}

const (
	LifecycleRecommended   = "recommended"
	LifecycleCompatibility = "compatibility"
)

const (
	ParamStyleBraces  = "braces"
	ParamStyleTokens  = "tokens"
	ParamStyleBracket = "bracket"
	ParamStyleToken   = "token"
)

var ValidServes = map[string]bool{
	"attention":    true,
	"readability":  true,
	"memorability": true,
	"conversion":   true,
}

var ValidBodyFormats = map[string]bool{
	BodyFormatFields:         true,
	BodyFormatRows:           true,
	BodyFormatJSONObject:     true,
	BodyFormatJSONArray:      true,
	BodyFormatMarkdownImages: true,
	BodyFormatMarkdownFields: true,
	BodyFormatSplit:          true,
	BodyFormatLines:          true,
	BodyFormatDialogue:       true,
}

var ValidLifecycles = map[string]bool{
	LifecycleRecommended:   true,
	LifecycleCompatibility: true,
}

type FieldSpec struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Enum        []string `yaml:"enum,omitempty"`
	Default     string   `yaml:"default,omitempty"`
	Example     string   `yaml:"example,omitempty"`
	ValueType   string   `yaml:"value_type,omitempty"`
	MinRunes    int      `yaml:"min_runes,omitempty"`
	MaxRunes    int      `yaml:"max_runes,omitempty"`
	AppliesTo   []string `yaml:"applies_to,omitempty"`
}

type RowsSpec struct {
	Delimiter   string      `yaml:"delimiter"`
	MinColumns  int         `yaml:"min_columns"`
	MaxColumns  int         `yaml:"max_columns,omitempty"`
	Schema      []FieldSpec `yaml:"schema"`
	Description string      `yaml:"description,omitempty"`
}

type FieldsSpec struct {
	Required                 []FieldSpec      `yaml:"required,omitempty"`
	Optional                 []FieldSpec      `yaml:"optional,omitempty"`
	RequiredAny              [][]string       `yaml:"required_any,omitempty"`
	RequiredWhenNoVariant    []string         `yaml:"required_when_no_variant,omitempty"`
	RequiredAnyWhenNoVariant [][]string       `yaml:"required_any_when_no_variant,omitempty"`
	OutputOrder              []string         `yaml:"output_order,omitempty"`
	Shapes                   []FieldShapeSpec `yaml:"shapes,omitempty"`
}

type FieldShapeSpec struct {
	Field          string              `yaml:"field"`
	Separator      string              `yaml:"separator"`
	MinParts       int                 `yaml:"min_parts"`
	MaxParts       int                 `yaml:"max_parts,omitempty" json:"MaxParts,omitempty"`
	MaxOccurrences int                 `yaml:"max_occurrences,omitempty" json:"MaxOccurrences,omitempty"`
	PartRules      []FieldPartRuleSpec `yaml:"part_rules,omitempty" json:"-"`
	ItemSeparator  string              `yaml:"item_separator,omitempty"`
	ItemMinParts   int                 `yaml:"item_min_parts,omitempty"`
	ItemMaxParts   int                 `yaml:"item_max_parts,omitempty" json:"ItemMaxParts,omitempty"`
}

// FieldPartRuleSpec requires specific 1-based positions to be non-empty for
// values whose raw separator-delimited part count matches the configured range.
type FieldPartRuleSpec struct {
	MinParts          int   `yaml:"min_parts,omitempty" json:"MinParts,omitempty"`
	MaxParts          int   `yaml:"max_parts,omitempty" json:"MaxParts,omitempty"`
	RequiredPositions []int `yaml:"required_positions" json:"RequiredPositions"`
}

type FieldGroupSpec struct {
	Start    string   `yaml:"start"`
	Required []string `yaml:"required"`
	Min      int      `yaml:"min"`
}

type BodySpec struct {
	MinImages          int             `yaml:"min_images,omitempty"`
	MaxImages          int             `yaml:"max_images,omitempty"`
	MinItems           int             `yaml:"min_items,omitempty"`
	Separator          string          `yaml:"separator,omitempty"`
	AllowedPrefixes    []string        `yaml:"allowed_prefixes,omitempty"`
	RequiredPairs      [][2]string     `yaml:"required_pairs,omitempty"`
	AllowNamedSpeakers bool            `yaml:"allow_named_speakers,omitempty"`
	Group              *FieldGroupSpec `yaml:"group,omitempty"`
}

type LayoutMetadata struct {
	Author     string `yaml:"author,omitempty"`
	Provenance string `yaml:"provenance,omitempty"`
	InspiredBy string `yaml:"inspired_by,omitempty"`
}

type VariantSpec struct {
	Name           string           `yaml:"name"`
	Aliases        []string         `yaml:"aliases,omitempty"`
	Description    string           `yaml:"description,omitempty"`
	UseWhen        string           `yaml:"use_when"`
	Required       []string         `yaml:"required,omitempty"`
	RequiredAny    [][]string       `yaml:"required_any,omitempty"`
	Shapes         []FieldShapeSpec `yaml:"shapes,omitempty"`
	Example        string           `yaml:"example"`
	AssertContains string           `yaml:"assert_contains,omitempty"`
}

type ParamSpec struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Enum        []string `yaml:"enum,omitempty"`
	Default     string   `yaml:"default,omitempty"`
	Example     string   `yaml:"example,omitempty"`
}

type OpenerSpec struct {
	Caption    bool        `yaml:"caption,omitempty"`
	ParamStyle string      `yaml:"param_style,omitempty"`
	Params     []ParamSpec `yaml:"params,omitempty"`
}

type ParsedOpener struct {
	Name      string
	Caption   string
	RawParams string
	Params    map[string]string
	bracket   bool
}

type LayoutSpec struct {
	SchemaVersion         string               `yaml:"schema_version"`
	Name                  string               `yaml:"name"`
	Lifecycle             string               `yaml:"lifecycle,omitempty"`
	BodyFormat            string               `yaml:"body_format,omitempty" json:"body_format,omitempty"`
	CompatibleBodyFormats []string             `yaml:"compatible_body_formats,omitempty" json:"compatible_body_formats,omitempty"`
	Version               string               `yaml:"version"`
	Since                 string               `yaml:"since,omitempty"`
	Category              string               `yaml:"category"`
	Serves                []string             `yaml:"serves"`
	ContentTypes          []string             `yaml:"content_types,omitempty"`
	Industry              []string             `yaml:"industry,omitempty"`
	Tags                  []string             `yaml:"tags,omitempty"`
	Position              string               `yaml:"position,omitempty"`
	InputPositions        []AgentInputPosition `yaml:"input_positions,omitempty" json:"input_positions,omitempty"`
	WhenToUse             string               `yaml:"when_to_use,omitempty"`
	WhenNotToUse          string               `yaml:"when_not_to_use,omitempty"`
	PairsWellWith         []string             `yaml:"pairs_well_with,omitempty"`
	AvoidCombiningWith    []string             `yaml:"avoid_combining_with,omitempty"`
	AntiPattern           string               `yaml:"anti_pattern,omitempty"`
	Opener                *OpenerSpec          `yaml:"opener,omitempty"`
	Fields                *FieldsSpec          `yaml:"fields,omitempty"`
	Rows                  *RowsSpec            `yaml:"rows,omitempty"`
	Body                  *BodySpec            `yaml:"body,omitempty"`
	Example               string               `yaml:"example,omitempty"`
	ExampleAssertContains string               `yaml:"example_assert_contains,omitempty"`
	Variants              []VariantSpec        `yaml:"variants,omitempty"`
	Metadata              LayoutMetadata       `yaml:"metadata"`
}

type ListFilter struct {
	Category    string
	Serves      string
	ContentType string
	Industry    string
	Tag         string
	Lifecycle   string
}
