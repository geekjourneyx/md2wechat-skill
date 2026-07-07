package advise

const (
	SchemaVersion = "v1"

	DecisionNoEnhancementNeeded = "no_enhancement_needed"
	DecisionEnhanceMinimally    = "enhance_minimally"

	ArticleTypeOpinion      = "opinion"
	ArticleTypeTutorial     = "tutorial"
	ArticleTypeCaseStudy    = "case_study"
	ArticleTypeAnnouncement = "announcement"
	ArticleTypeReport       = "report"
	ArticleTypeEssay        = "essay"
	ArticleTypeUnknown      = "unknown"

	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"

	ActionStateRecommended = "recommended"
	ActionStateOptional    = "optional"
	ActionStateSkip        = "skip"

	ToolTitle     = "title"
	ToolCover     = "cover"
	ToolLayout    = "layout"
	ToolMicroEdit = "micro_edit"
)

type Input struct {
	SourceFile string
	Markdown   string
}

type Result struct {
	SchemaVersion string         `json:"schema_version"`
	Command       string         `json:"command"`
	Source        string         `json:"source"`
	Decision      string         `json:"decision"`
	Article       ArticleProfile `json:"article"`
	Actions       []Action       `json:"actions"`
	Layout        LayoutAdvice   `json:"layout"`
	MicroEdits    []MicroEdit    `json:"micro_edits"`
	DoNotDo       []string       `json:"do_not_do"`
}

type ArticleProfile struct {
	Type          string   `json:"type"`
	Confidence    string   `json:"confidence"`
	Stats         Stats    `json:"stats"`
	Signals       []string `json:"signals"`
	Uncertainties []string `json:"uncertainties"`
}

type Stats struct {
	WordCount        int `json:"word_count"`
	H1Count          int `json:"h1_count"`
	H2Count          int `json:"h2_count"`
	ParagraphCount   int `json:"paragraph_count"`
	ImageCount       int `json:"image_count"`
	BlockquoteCount  int `json:"blockquote_count"`
	OrderedListCount int `json:"ordered_list_count"`
}

type Action struct {
	Tool                 string     `json:"tool"`
	State                string     `json:"state"`
	Reason               string     `json:"reason"`
	Evidence             []Evidence `json:"evidence"`
	CommandHint          string     `json:"command_hint,omitempty"`
	RequiresConfirmation bool       `json:"requires_confirmation"`
}

type Evidence struct {
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	Location *Location `json:"location,omitempty"`
}

type Location struct {
	Line    int    `json:"line,omitempty"`
	Section string `json:"section,omitempty"`
}

type LayoutAdvice struct {
	MaxModules  int            `json:"max_modules"`
	Recommended []ModuleAdvice `json:"recommended"`
	Rejected    []ModuleAdvice `json:"rejected"`
}

type ModuleAdvice struct {
	Name     string     `json:"name"`
	State    string     `json:"state"`
	Reason   string     `json:"reason"`
	Evidence []Evidence `json:"evidence"`
}

type MicroEdit struct {
	Kind                 string     `json:"kind"`
	Reason               string     `json:"reason"`
	Evidence             []Evidence `json:"evidence"`
	RequiresConfirmation bool       `json:"requires_confirmation"`
}
