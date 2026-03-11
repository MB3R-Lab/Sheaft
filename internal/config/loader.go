package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadStructured(path string, dst any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(raw, dst); err != nil {
			return fmt.Errorf("decode json %s: %w", path, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, dst); err != nil {
			return fmt.Errorf("decode yaml %s: %w", path, err)
		}
	default:
		return fmt.Errorf("unsupported config extension %q", filepath.Ext(path))
	}
	return nil
}

func loadStructuredFile(path string, dst any) error {
	return LoadStructured(path, dst)
}
