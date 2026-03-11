package predicates

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/config"
)

const (
	SchemaVersion = "1.0"
	TypeAllOf     = "all_of"
	TypeAnyOf     = "any_of"
	TypeKOfN      = "k_of_n"
)

type Definition struct {
	Type        string       `json:"type" yaml:"type"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	Services    []string     `json:"services,omitempty" yaml:"services,omitempty"`
	Children    []Definition `json:"children,omitempty" yaml:"children,omitempty"`
	K           int          `json:"k,omitempty" yaml:"k,omitempty"`
}

type Set map[string]Definition

type Contract struct {
	SchemaVersion   string             `json:"schema_version" yaml:"schema_version"`
	Predicates      Set                `json:"predicates" yaml:"predicates"`
	EndpointWeights map[string]float64 `json:"endpoint_weights,omitempty" yaml:"endpoint_weights,omitempty"`
}

func Load(path string) (Contract, error) {
	var contract Contract
	if err := configLoad(path, &contract); err != nil {
		return Contract{}, fmt.Errorf("read predicates file: %w", err)
	}
	if contract.SchemaVersion == "" {
		contract.SchemaVersion = SchemaVersion
	}
	if contract.EndpointWeights == nil {
		contract.EndpointWeights = map[string]float64{}
	}
	if err := contract.Validate(); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

func (c Contract) Validate() error {
	if c.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported predicate contract schema_version: got %q want %q", c.SchemaVersion, SchemaVersion)
	}
	if len(c.Predicates) == 0 && len(c.EndpointWeights) == 0 {
		return errors.New("predicate contract must define predicates or endpoint_weights")
	}
	for name, def := range c.Predicates {
		if strings.TrimSpace(name) == "" {
			return errors.New("predicate key cannot be empty")
		}
		if err := def.Validate(); err != nil {
			return fmt.Errorf("predicate %q: %w", name, err)
		}
	}
	for endpoint, weight := range c.EndpointWeights {
		if strings.TrimSpace(endpoint) == "" {
			return errors.New("endpoint_weights key cannot be empty")
		}
		if weight < 0 {
			return fmt.Errorf("endpoint_weights[%s] must be >= 0", endpoint)
		}
	}
	return nil
}

func (d Definition) Validate() error {
	if strings.TrimSpace(d.Type) == "" {
		return errors.New("type cannot be empty")
	}
	operandCount := len(d.Services) + len(d.Children)
	switch d.Type {
	case TypeAllOf, TypeAnyOf:
		if operandCount == 0 {
			return fmt.Errorf("%s requires at least one service or child predicate", d.Type)
		}
	case TypeKOfN:
		if operandCount == 0 {
			return errors.New("k_of_n requires at least one service or child predicate")
		}
		if d.K <= 0 {
			return errors.New("k_of_n requires k > 0")
		}
		if d.K > operandCount {
			return fmt.Errorf("k_of_n k=%d exceeds operand count %d", d.K, operandCount)
		}
	default:
		return fmt.Errorf("unsupported predicate type %q", d.Type)
	}

	for idx, service := range d.Services {
		if strings.TrimSpace(service) == "" {
			return fmt.Errorf("services[%d] cannot be empty", idx)
		}
	}
	for idx, child := range d.Children {
		if err := child.Validate(); err != nil {
			return fmt.Errorf("children[%d]: %w", idx, err)
		}
	}
	return nil
}

func Evaluate(def Definition, alive func(string) bool) bool {
	switch def.Type {
	case TypeAllOf:
		for _, service := range def.Services {
			if !alive(service) {
				return false
			}
		}
		for _, child := range def.Children {
			if !Evaluate(child, alive) {
				return false
			}
		}
		return true
	case TypeAnyOf:
		for _, service := range def.Services {
			if alive(service) {
				return true
			}
		}
		for _, child := range def.Children {
			if Evaluate(child, alive) {
				return true
			}
		}
		return false
	case TypeKOfN:
		successes := 0
		for _, service := range def.Services {
			if alive(service) {
				successes++
			}
		}
		for _, child := range def.Children {
			if Evaluate(child, alive) {
				successes++
			}
		}
		return successes >= def.K
	default:
		return false
	}
}

func configLoad(path string, dst any) error {
	return config.LoadStructured(path, dst)
}
