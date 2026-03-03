package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PolicyMode string

const (
	ModeWarn   PolicyMode = "warn"
	ModeFail   PolicyMode = "fail"
	ModeReport PolicyMode = "report"
)

type Policy struct {
	Mode               PolicyMode         `json:"mode"`
	DefaultAction      PolicyMode         `json:"default_action"`
	GlobalThreshold    float64            `json:"global_threshold"`
	FailureProbability float64            `json:"failure_probability"`
	Trials             int                `json:"trials"`
	EndpointThresholds map[string]float64 `json:"endpoint_thresholds"`
}

type RuntimeConfig struct {
	Model      string `json:"model"`
	Journeys   string `json:"journeys"`
	OutputDir  string `json:"output_dir"`
	Seed       int64  `json:"seed"`
	Simulation struct {
		Trials             int     `json:"trials"`
		FailureProbability float64 `json:"failure_probability"`
	} `json:"simulation"`
	Policy struct {
		File string `json:"file"`
	} `json:"policy"`
}

func (p Policy) Normalized() Policy {
	out := p
	if out.Mode == "" {
		out.Mode = ModeWarn
	}
	if out.DefaultAction == "" {
		out.DefaultAction = ModeWarn
	}
	if out.GlobalThreshold == 0 {
		out.GlobalThreshold = 0.99
	}
	if out.FailureProbability == 0 {
		out.FailureProbability = 0.05
	}
	if out.Trials <= 0 {
		out.Trials = 10000
	}
	if out.EndpointThresholds == nil {
		out.EndpointThresholds = map[string]float64{}
	}
	return out
}

func (p Policy) Validate() error {
	if p.Mode != ModeWarn && p.Mode != ModeFail && p.Mode != ModeReport {
		return fmt.Errorf("unsupported mode: %q", p.Mode)
	}
	if p.DefaultAction != ModeWarn && p.DefaultAction != ModeFail && p.DefaultAction != ModeReport {
		return fmt.Errorf("unsupported default action: %q", p.DefaultAction)
	}
	if p.GlobalThreshold < 0 || p.GlobalThreshold > 1 {
		return errors.New("global_threshold must be in range [0,1]")
	}
	if p.FailureProbability < 0 || p.FailureProbability > 1 {
		return errors.New("failure_probability must be in range [0,1]")
	}
	if p.Trials <= 0 {
		return errors.New("trials must be > 0")
	}
	for endpoint, threshold := range p.EndpointThresholds {
		if strings.TrimSpace(endpoint) == "" {
			return errors.New("endpoint threshold key cannot be empty")
		}
		if threshold < 0 || threshold > 1 {
			return fmt.Errorf("endpoint threshold out of range [0,1]: %s", endpoint)
		}
	}
	return nil
}

func LoadPolicy(path string) (Policy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read policy file: %w", err)
	}

	var p Policy
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(raw, &p); err != nil {
			return Policy{}, fmt.Errorf("decode policy json: %w", err)
		}
	case ".yml", ".yaml":
		p, err = parsePolicyYAML(string(raw))
		if err != nil {
			return Policy{}, err
		}
	default:
		return Policy{}, fmt.Errorf("unsupported policy extension %q", ext)
	}

	p = p.Normalized()
	if err := p.Validate(); err != nil {
		return Policy{}, fmt.Errorf("validate policy: %w", err)
	}
	return p, nil
}

func LoadRuntimeConfig(path string) (RuntimeConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var cfg RuntimeConfig
	switch ext {
	case ".json":
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return RuntimeConfig{}, fmt.Errorf("decode config json: %w", err)
		}
	case ".yml", ".yaml":
		cfg, err = parseRuntimeYAML(string(raw))
		if err != nil {
			return RuntimeConfig{}, err
		}
	default:
		return RuntimeConfig{}, fmt.Errorf("unsupported config extension %q", ext)
	}

	return cfg, nil
}

func parsePolicyYAML(data string) (Policy, error) {
	p := Policy{
		EndpointThresholds: make(map[string]float64),
	}

	section := ""
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 {
			if strings.HasSuffix(trimmed, ":") {
				section = strings.TrimSuffix(trimmed, ":")
				continue
			}
			key, val, ok := splitYAMLKeyValue(trimmed)
			if !ok {
				return Policy{}, fmt.Errorf("invalid yaml line: %q", line)
			}
			switch key {
			case "mode":
				p.Mode = PolicyMode(val)
			case "default_action":
				p.DefaultAction = PolicyMode(val)
			case "global_threshold":
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return Policy{}, fmt.Errorf("invalid global_threshold %q", val)
				}
				p.GlobalThreshold = f
			case "failure_probability":
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return Policy{}, fmt.Errorf("invalid failure_probability %q", val)
				}
				p.FailureProbability = f
			case "trials":
				i, err := strconv.Atoi(val)
				if err != nil {
					return Policy{}, fmt.Errorf("invalid trials %q", val)
				}
				p.Trials = i
			default:
				// Keep parser forward-compatible by ignoring unknown keys.
			}
			continue
		}

		if section == "endpoint_thresholds" {
			idx := strings.LastIndex(trimmed, ":")
			if idx < 0 {
				return Policy{}, fmt.Errorf("invalid endpoint_thresholds line: %q", line)
			}
			key := strings.TrimSpace(trimmed[:idx])
			val := strings.TrimSpace(trimmed[idx+1:])
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return Policy{}, fmt.Errorf("invalid endpoint threshold %q for %q", val, key)
			}
			p.EndpointThresholds[key] = f
		}
	}

	return p, nil
}

func parseRuntimeYAML(data string) (RuntimeConfig, error) {
	cfg := RuntimeConfig{}
	section := ""

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 {
			if strings.HasSuffix(trimmed, ":") {
				section = strings.TrimSuffix(trimmed, ":")
				continue
			}
			key, val, ok := splitYAMLKeyValue(trimmed)
			if !ok {
				return RuntimeConfig{}, fmt.Errorf("invalid yaml line: %q", line)
			}
			switch key {
			case "model":
				cfg.Model = val
			case "journeys":
				cfg.Journeys = val
			case "output_dir":
				cfg.OutputDir = val
			case "seed":
				i, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return RuntimeConfig{}, fmt.Errorf("invalid seed %q", val)
				}
				cfg.Seed = i
			default:
			}
			continue
		}

		key, val, ok := splitYAMLKeyValue(trimmed)
		if !ok {
			return RuntimeConfig{}, fmt.Errorf("invalid nested yaml line: %q", line)
		}
		switch section {
		case "simulation":
			switch key {
			case "trials":
				i, err := strconv.Atoi(val)
				if err != nil {
					return RuntimeConfig{}, fmt.Errorf("invalid simulation.trials %q", val)
				}
				cfg.Simulation.Trials = i
			case "failure_probability":
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return RuntimeConfig{}, fmt.Errorf("invalid simulation.failure_probability %q", val)
				}
				cfg.Simulation.FailureProbability = f
			}
		case "policy":
			if key == "file" {
				cfg.Policy.File = val
			}
		}
	}
	return cfg, nil
}

func splitYAMLKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	return key, val, true
}
