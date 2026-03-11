package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

type ContractPolicyAction string

const (
	ContractPolicyActionAllow ContractPolicyAction = "allow"
	ContractPolicyActionWarn  ContractPolicyAction = "warn"
	ContractPolicyActionFail  ContractPolicyAction = "fail"
)

type ContractPolicy struct {
	AllowedKinds        []string                    `json:"allowed_kinds,omitempty" yaml:"allowed_kinds,omitempty"`
	AllowedContracts    []ContractPolicySelector    `json:"allowed_contracts,omitempty" yaml:"allowed_contracts,omitempty"`
	DeprecatedAction    ContractPolicyAction        `json:"deprecated_action,omitempty" yaml:"deprecated_action,omitempty"`
	DeprecatedContracts []DeprecatedContractSelector `json:"deprecated_contracts,omitempty" yaml:"deprecated_contracts,omitempty"`
}

type ContractPolicySelector struct {
	Kind     string   `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name     string   `json:"name" yaml:"name"`
	Versions []string `json:"versions,omitempty" yaml:"versions,omitempty"`
}

type DeprecatedContractSelector struct {
	Kind     string               `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name     string               `json:"name" yaml:"name"`
	Versions []string             `json:"versions,omitempty" yaml:"versions,omitempty"`
	Action   ContractPolicyAction `json:"action,omitempty" yaml:"action,omitempty"`
	Message  string               `json:"message,omitempty" yaml:"message,omitempty"`
}

type ContractPolicyDecision struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Message string `json:"message,omitempty"`
}

const (
	ContractPolicyStatusCurrent    = "current"
	ContractPolicyStatusDeprecated = "deprecated"
)

func LoadContractPolicy(path string) (ContractPolicy, error) {
	var policy ContractPolicy
	if err := loadStructuredFile(path, &policy); err != nil {
		return ContractPolicy{}, err
	}
	policy = policy.Normalized()
	if err := policy.Validate(); err != nil {
		return ContractPolicy{}, fmt.Errorf("validate contract policy: %w", err)
	}
	return policy, nil
}

func (p ContractPolicy) Normalized() ContractPolicy {
	out := p
	if out.DeprecatedAction == "" {
		out.DeprecatedAction = ContractPolicyActionWarn
	}
	out.AllowedKinds = cloneStringSlice(out.AllowedKinds)
	out.AllowedContracts = cloneContractSelectors(out.AllowedContracts)
	out.DeprecatedContracts = cloneDeprecatedSelectors(out.DeprecatedContracts)
	return out
}

func (p ContractPolicy) Validate() error {
	normalized := p.Normalized()
	supported := modelcontract.Supported()
	supportedKinds := make(map[string]struct{}, len(supported))
	for _, contract := range supported {
		supportedKinds[string(contract.Kind)] = struct{}{}
	}

	seenKinds := map[string]struct{}{}
	for _, kind := range normalized.AllowedKinds {
		trimmed := strings.TrimSpace(kind)
		if trimmed == "" {
			return fmt.Errorf("contract_policy.allowed_kinds entries cannot be empty")
		}
		if _, ok := supportedKinds[trimmed]; !ok {
			return fmt.Errorf("contract_policy.allowed_kinds contains unsupported kind %q", trimmed)
		}
		if _, exists := seenKinds[trimmed]; exists {
			return fmt.Errorf("contract_policy.allowed_kinds contains duplicate kind %q", trimmed)
		}
		seenKinds[trimmed] = struct{}{}
	}

	for idx, selector := range normalized.AllowedContracts {
		if err := validateSelector(fmt.Sprintf("contract_policy.allowed_contracts[%d]", idx), selector.Kind, selector.Name, selector.Versions, supported); err != nil {
			return err
		}
	}

	if normalized.DeprecatedAction != ContractPolicyActionWarn && normalized.DeprecatedAction != ContractPolicyActionFail {
		return fmt.Errorf("contract_policy.deprecated_action must be %q or %q", ContractPolicyActionWarn, ContractPolicyActionFail)
	}

	for idx, selector := range normalized.DeprecatedContracts {
		if err := validateSelector(fmt.Sprintf("contract_policy.deprecated_contracts[%d]", idx), selector.Kind, selector.Name, selector.Versions, supported); err != nil {
			return err
		}
		if selector.Action != "" && selector.Action != ContractPolicyActionWarn && selector.Action != ContractPolicyActionFail {
			return fmt.Errorf("contract_policy.deprecated_contracts[%d].action must be %q or %q", idx, ContractPolicyActionWarn, ContractPolicyActionFail)
		}
	}

	return nil
}

func (p ContractPolicy) Evaluate(contract modelcontract.SupportedContract) (ContractPolicyDecision, error) {
	p = p.Normalized()

	if len(p.AllowedKinds) > 0 && !slices.Contains(p.AllowedKinds, string(contract.Kind)) {
		return ContractPolicyDecision{}, fmt.Errorf(
			"project contract policy rejected %s %s@%s: allowed kinds: %s",
			contract.Kind,
			contract.Name,
			contract.Version,
			strings.Join(p.AllowedKinds, ", "),
		)
	}

	if len(p.AllowedContracts) > 0 {
		allowed := false
		for _, selector := range p.AllowedContracts {
			if selectorMatches(selector.Kind, selector.Name, selector.Versions, contract) {
				allowed = true
				break
			}
		}
		if !allowed {
			return ContractPolicyDecision{}, fmt.Errorf(
				"project contract policy rejected %s %s@%s: allowed contracts: %s",
				contract.Kind,
				contract.Name,
				contract.Version,
				describeSelectors(p.AllowedContracts),
			)
		}
	}

	for _, selector := range p.DeprecatedContracts {
		if !selectorMatches(selector.Kind, selector.Name, selector.Versions, contract) {
			continue
		}
		action := selector.Action
		if action == "" {
			action = p.DeprecatedAction
		}
		message := strings.TrimSpace(selector.Message)
		if message == "" {
			message = fmt.Sprintf("contract %s %s@%s is deprecated by project policy", contract.Kind, contract.Name, contract.Version)
		}
		if action == ContractPolicyActionFail {
			return ContractPolicyDecision{
				Status:  ContractPolicyStatusDeprecated,
				Action:  string(action),
				Message: message,
			}, fmt.Errorf(message)
		}
		return ContractPolicyDecision{
			Status:  ContractPolicyStatusDeprecated,
			Action:  string(action),
			Message: message,
		}, nil
	}

	return ContractPolicyDecision{
		Status: ContractPolicyStatusCurrent,
		Action: string(ContractPolicyActionAllow),
	}, nil
}

func validateSelector(label, kind, name string, versions []string, supported []modelcontract.SupportedContract) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s.name cannot be empty", label)
	}
	if kind != "" {
		if _, ok := supportedKindSet(supported)[kind]; !ok {
			return fmt.Errorf("%s.kind %q is not in the supported-contract registry", label, kind)
		}
	}
	if len(versions) == 0 {
		for _, contract := range supported {
			if selectorMatches(kind, name, nil, contract) {
				return nil
			}
		}
		return fmt.Errorf("%s does not match any supported contract", label)
	}
	seenVersions := map[string]struct{}{}
	for _, version := range versions {
		trimmed := strings.TrimSpace(version)
		if trimmed == "" {
			return fmt.Errorf("%s.versions entries cannot be empty", label)
		}
		if _, exists := seenVersions[trimmed]; exists {
			return fmt.Errorf("%s contains duplicate version %q", label, trimmed)
		}
		seenVersions[trimmed] = struct{}{}
	}
	for _, contract := range supported {
		if selectorMatches(kind, name, versions, contract) {
			return nil
		}
	}
	return fmt.Errorf("%s does not match any supported contract", label)
}

func selectorMatches(kind, name string, versions []string, contract modelcontract.SupportedContract) bool {
	if kind != "" && kind != string(contract.Kind) {
		return false
	}
	if name != contract.Name {
		return false
	}
	if len(versions) == 0 {
		return true
	}
	return slices.Contains(versions, contract.Version)
}

func describeSelectors(selectors []ContractPolicySelector) string {
	values := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		prefix := selector.Name
		if selector.Kind != "" {
			prefix = selector.Kind + ":" + prefix
		}
		if len(selector.Versions) == 0 {
			values = append(values, prefix+"@*")
			continue
		}
		for _, version := range selector.Versions {
			values = append(values, prefix+"@"+version)
		}
	}
	slices.Sort(values)
	return strings.Join(values, ", ")
}

func supportedKindSet(supported []modelcontract.SupportedContract) map[string]struct{} {
	out := make(map[string]struct{}, len(supported))
	for _, contract := range supported {
		out[string(contract.Kind)] = struct{}{}
	}
	return out
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := slices.Clone(values)
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

func cloneContractSelectors(values []ContractPolicySelector) []ContractPolicySelector {
	if len(values) == 0 {
		return nil
	}
	out := make([]ContractPolicySelector, 0, len(values))
	for _, value := range values {
		out = append(out, ContractPolicySelector{
			Kind:     strings.TrimSpace(value.Kind),
			Name:     strings.TrimSpace(value.Name),
			Versions: cloneStringSlice(value.Versions),
		})
	}
	return out
}

func cloneDeprecatedSelectors(values []DeprecatedContractSelector) []DeprecatedContractSelector {
	if len(values) == 0 {
		return nil
	}
	out := make([]DeprecatedContractSelector, 0, len(values))
	for _, value := range values {
		out = append(out, DeprecatedContractSelector{
			Kind:     strings.TrimSpace(value.Kind),
			Name:     strings.TrimSpace(value.Name),
			Versions: cloneStringSlice(value.Versions),
			Action:   value.Action,
			Message:  strings.TrimSpace(value.Message),
		})
	}
	return out
}
