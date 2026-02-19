package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/riricardoMa/claude-grid/internal/pathutil"
)

type Manifest struct {
	Name      string     `yaml:"name"`
	Instances []Instance `yaml:"instances"`
}

type Instance struct {
	Dir    string `yaml:"dir"`
	Prompt string `yaml:"prompt"`
	Branch string `yaml:"branch"`
}

func Parse(manifestPath string) (Manifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %q: %w", manifestPath, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %q: %w", manifestPath, err)
	}

	if len(m.Instances) == 0 {
		return Manifest{}, fmt.Errorf("manifest %q: instances list is required and must not be empty", manifestPath)
	}

	if len(m.Instances) > 16 {
		return Manifest{}, fmt.Errorf("manifest %q: too many instances (%d); maximum is 16", manifestPath, len(m.Instances))
	}

	manifestDir := filepath.Dir(manifestPath)

	for i, inst := range m.Instances {
		if inst.Dir == "" {
			return Manifest{}, fmt.Errorf("manifest %q: instance %d is missing required field \"dir\"", manifestPath, i)
		}

		expanded, err := pathutil.ExpandTilde(inst.Dir)
		if err != nil {
			return Manifest{}, fmt.Errorf("manifest %q: instance %d dir %q: %w", manifestPath, i, inst.Dir, err)
		}

		if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(manifestDir, expanded)
		}

		m.Instances[i].Dir = expanded
	}

	return m, nil
}
