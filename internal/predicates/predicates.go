package predicates

import (
	"fmt"
	"os"
	"strings"
)

type Predicate struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type Set map[string]Predicate

// Load parses a small YAML subset used by examples.
func Load(path string) (Set, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read predicates file: %w", err)
	}

	out := Set{}
	lines := strings.Split(string(raw), "\n")
	inPredicates := false
	currentKey := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))

		if indent == 0 && trimmed == "predicates:" {
			inPredicates = true
			continue
		}
		if !inPredicates {
			continue
		}

		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			currentKey = strings.TrimSuffix(trimmed, ":")
			out[currentKey] = Predicate{}
			continue
		}
		if indent == 4 && currentKey != "" {
			idx := strings.Index(trimmed, ":")
			if idx <= 0 {
				continue
			}
			key := strings.TrimSpace(trimmed[:idx])
			val := strings.TrimSpace(trimmed[idx+1:])
			p := out[currentKey]
			switch key {
			case "type":
				p.Type = val
			case "description":
				p.Description = val
			}
			out[currentKey] = p
		}
	}
	return out, nil
}
