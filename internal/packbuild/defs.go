package packbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PromptDef is one prompt definition from packs/defs/*.yaml.
type PromptDef struct {
	ID       string `yaml:"id"`
	Category string `yaml:"category"`
	Text     string `yaml:"text"`
	SPARQL   string `yaml:"sparql"`
}

// DefFile is the on-disk YAML shape (single prompt or a list under prompts:).
type DefFile struct {
	ID       string      `yaml:"id"`
	Category string      `yaml:"category"`
	Text     string      `yaml:"text"`
	SPARQL   string      `yaml:"sparql"`
	Prompts  []PromptDef `yaml:"prompts"`
}

// LoadDefs reads all *.yaml / *.yml files under dir.
func LoadDefs(dir string) ([]PromptDef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read defs dir: %w", err)
	}
	var out []PromptDef
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".yaml") && !strings.HasSuffix(lower, ".yml") {
			continue
		}
		path := filepath.Join(dir, name)
		defs, err := loadDefFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, defs...)
	}
	return out, nil
}

func loadDefFile(path string) ([]PromptDef, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f DefFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(f.Prompts) > 0 {
		for i, p := range f.Prompts {
			if err := validateDef(p); err != nil {
				return nil, fmt.Errorf("%s prompts[%d]: %w", path, i, err)
			}
		}
		return f.Prompts, nil
	}
	p := PromptDef{ID: f.ID, Category: f.Category, Text: f.Text, SPARQL: f.SPARQL}
	if err := validateDef(p); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return []PromptDef{p}, nil
}

func validateDef(p PromptDef) error {
	if p.ID == "" {
		return fmt.Errorf("missing id")
	}
	if p.Category == "" {
		return fmt.Errorf("missing category")
	}
	if p.Text == "" {
		return fmt.Errorf("missing text")
	}
	if strings.TrimSpace(p.SPARQL) == "" {
		return fmt.Errorf("missing sparql")
	}
	return nil
}
