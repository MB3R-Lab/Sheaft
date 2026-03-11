package config

import (
	"errors"
	"fmt"
	"strings"
)

type PolicyMode string

const (
	ModeWarn   PolicyMode = "warn"
	ModeFail   PolicyMode = "fail"
	ModeReport PolicyMode = "report"
)

type Policy struct {
	Mode               PolicyMode         `json:"mode" yaml:"mode"`
	DefaultAction      PolicyMode         `json:"default_action" yaml:"default_action"`
	GlobalThreshold    float64            `json:"global_threshold" yaml:"global_threshold"`
	FailureProbability float64            `json:"failure_probability" yaml:"failure_probability"`
	Trials             int                `json:"trials" yaml:"trials"`
	EndpointThresholds map[string]float64 `json:"endpoint_thresholds" yaml:"endpoint_thresholds"`
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
	if !isValidPolicyMode(p.Mode) {
		return fmt.Errorf("unsupported mode: %q", p.Mode)
	}
	if !isValidPolicyMode(p.DefaultAction) {
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
	var p Policy
	if err := loadStructuredFile(path, &p); err != nil {
		return Policy{}, err
	}
	p = p.Normalized()
	if err := p.Validate(); err != nil {
		return Policy{}, fmt.Errorf("validate policy: %w", err)
	}
	return p, nil
}

func isValidPolicyMode(mode PolicyMode) bool {
	return mode == ModeWarn || mode == ModeFail || mode == ModeReport
}
