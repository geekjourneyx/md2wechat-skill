package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed builtin/themes/*.yaml builtin/writers/*.yaml builtin/prompts/*/*.yaml builtin/layout/*/*.yaml builtin/skills/*/SKILL.md
var builtinFS embed.FS

func listYAMLNames(dir string) ([]string, error) {
	entries, err := fs.ReadDir(builtinFS, dir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		names = append(names, strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml"))
	}
	sort.Strings(names)
	return names, nil
}

func readYAMLFile(dir, name string) ([]byte, error) {
	if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
		name += ".yaml"
	}
	return builtinFS.ReadFile(path.Join(dir, name))
}

func ListBuiltinThemes() ([]string, error) {
	return listYAMLNames("builtin/themes")
}

func ReadBuiltinTheme(name string) ([]byte, error) {
	return readYAMLFile("builtin/themes", name)
}

func ListBuiltinWriters() ([]string, error) {
	return listYAMLNames("builtin/writers")
}

func ReadBuiltinWriter(name string) ([]byte, error) {
	return readYAMLFile("builtin/writers", name)
}

func ListBuiltinPrompts(kind string) ([]string, error) {
	return listYAMLNames(path.Join("builtin/prompts", kind))
}

func ReadBuiltinPrompt(kind, name string) ([]byte, error) {
	return readYAMLFile(path.Join("builtin/prompts", kind), name)
}

func ListBuiltinLayoutCategories() ([]string, error) {
	entries, err := fs.ReadDir(builtinFS, "builtin/layout")
	if err != nil {
		return nil, err
	}
	cats := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			cats = append(cats, e.Name())
		}
	}
	sort.Strings(cats)
	return cats, nil
}

func ListBuiltinLayouts(category string) ([]string, error) {
	return listYAMLNames(path.Join("builtin/layout", category))
}

func ReadBuiltinLayout(category, name string) ([]byte, error) {
	return readYAMLFile(path.Join("builtin/layout", category), name)
}

func ListBuiltinSkills() ([]string, error) {
	entries, err := fs.ReadDir(builtinFS, "builtin/skills")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func ReadBuiltinSkill(name string) ([]byte, error) {
	if strings.TrimSpace(name) == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return nil, fmt.Errorf("invalid builtin skill name: %q", name)
	}
	return builtinFS.ReadFile(path.Join("builtin/skills", name, "SKILL.md"))
}
