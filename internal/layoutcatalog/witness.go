package layoutcatalog

import (
	"fmt"
	"strings"
)

// WitnessContract describes one canonical or variant example that must be
// structurally bound to a catalog module.
type WitnessContract struct {
	Module         string
	Variant        string
	VariantAliases []string
	Example        string
	AssertContains string
}

// ValidateWitness checks an example using the same opener and body parsers as
// normal catalog validation. Examples remain optional at schema load time.
func (c *Catalog) ValidateWitness(contract WitnessContract) error {
	if strings.TrimSpace(contract.Example) == "" {
		return fmt.Errorf("%s witness example is empty", contract.Module)
	}
	lines := strings.Split(strings.TrimSpace(contract.Example), "\n")
	opener, err := parseBlockOpener(strings.TrimRight(lines[0], "\r"))
	if err != nil {
		return fmt.Errorf("parse %s witness opener: %w", contract.Module, err)
	}
	if opener.Name != contract.Module {
		return fmt.Errorf("witness opener %q does not match module %q", opener.Name, contract.Module)
	}
	spec, ok := c.Get(contract.Module)
	if !ok {
		return fmt.Errorf("module %q not found", contract.Module)
	}
	validatedOpener, err := validateOpener(opener, spec.Opener)
	if err != nil {
		return fmt.Errorf("validate %s witness opener: %w", contract.Module, err)
	}
	if report := c.Validate(contract.Example); len(report.Errors) != 0 {
		return fmt.Errorf("%s witness validation errors: %v", contract.Module, report.Errors)
	}
	body, err := witnessBody(lines)
	if err != nil {
		return fmt.Errorf("parse %s witness body: %w", contract.Module, err)
	}
	facts, issues := parseBodyFacts(spec, spec.BodyFormat, body)
	if len(issues) != 0 {
		return fmt.Errorf("parse %s witness facts: %v", contract.Module, issues)
	}
	addWitnessControlFacts(&facts, spec.BodyFormat, body)
	if contract.Variant != "" && !witnessSelectsVariant(validatedOpener, facts, contract.Variant, contract.VariantAliases) {
		return fmt.Errorf("%s witness does not select variant %q or a declared alias", contract.Module, contract.Variant)
	}
	if contract.AssertContains != "" && !strings.Contains(contract.Example, contract.AssertContains) {
		return fmt.Errorf("%s witness does not contain assertion %q", contract.Module, contract.AssertContains)
	}
	return nil
}

func witnessBody(lines []string) ([]string, error) {
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(strings.TrimRight(lines[i], "\r")) == ":::" {
			return lines[1:i], nil
		}
	}
	return nil, fmt.Errorf("missing closing fence")
}

func addWitnessControlFacts(facts *bodyFacts, format string, body []string) {
	if format != BodyFormatFields && format != BodyFormatMarkdownFields {
		return
	}
	for _, line := range body {
		name, value, ok := strings.Cut(strings.TrimSpace(strings.TrimRight(line, "\r")), ":")
		name = strings.TrimSpace(name)
		if ok && (name == "variant" || name == "type") {
			facts.addField(name, strings.TrimSpace(value))
		}
	}
}

func witnessSelectsVariant(opener ParsedOpener, facts bodyFacts, name string, aliases []string) bool {
	accepted := make(map[string]bool, 1+len(aliases))
	accepted[name] = true
	for _, alias := range aliases {
		accepted[alias] = true
	}
	selector := lastNonEmptyValue(facts.fieldValues["type"])
	if selector == "" {
		selector = strings.TrimSpace(opener.Params["type"])
	}
	if selector == "" {
		selector = lastNonEmptyValue(facts.fieldValues["variant"])
	}
	if selector == "" {
		selector = strings.TrimSpace(opener.Params["variant"])
	}
	return accepted[selector]
}
