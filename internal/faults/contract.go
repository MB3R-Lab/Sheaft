package faults

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
)

const SchemaVersion = "1.0"

const (
	TypeCorrelatedFailureDomain   = "correlated_failure_domain"
	TypeEdgeFailStop              = "edge_fail_stop"
	TypeEdgePartialDegradation    = "edge_partial_degradation"
	TypeServicePartialDegradation = "service_partial_degradation"
)

const (
	MetricExpectedSuccessRate      = "expected_success_rate"
	MetricMaxAmplificationFactor   = "max_amplification_factor"
	MetricTimeoutMismatchCount     = "timeout_mismatch_count"
	MetricBlastRadiusServiceCount  = "blast_radius_service_count"
	MetricBlastRadiusEndpointCount = "blast_radius_endpoint_count"
)

const (
	TargetEndpoint = "endpoint"
	TargetPath     = "path"
	TargetEdge     = "edge"
	TargetProfile  = "profile"
)

type Contract struct {
	SchemaVersion string             `json:"schema_version" yaml:"schema_version"`
	Profiles      map[string]Profile `json:"profiles" yaml:"profiles"`
}

type Profile struct {
	Faults     []Fault     `json:"faults,omitempty" yaml:"faults,omitempty"`
	Assertions []Assertion `json:"assertions,omitempty" yaml:"assertions,omitempty"`
}

type Fault struct {
	Type                string                `json:"type" yaml:"type"`
	Selector            Selector              `json:"selector" yaml:"selector"`
	OnlyFailureEligible bool                  `json:"only_failure_eligible,omitempty" yaml:"only_failure_eligible,omitempty"`
	ErrorRate           *float64              `json:"error_rate,omitempty" yaml:"error_rate,omitempty"`
	LatencyMS           *model.LatencySummary `json:"latency_ms,omitempty" yaml:"latency_ms,omitempty"`
}

type Selector struct {
	ServiceIDs         []string          `json:"service_ids,omitempty" yaml:"service_ids,omitempty"`
	ServiceLabels      map[string]string `json:"service_labels,omitempty" yaml:"service_labels,omitempty"`
	PlacementLabels    map[string]string `json:"placement_labels,omitempty" yaml:"placement_labels,omitempty"`
	SharedResourceRefs []string          `json:"shared_resource_refs,omitempty" yaml:"shared_resource_refs,omitempty"`
	EdgeIDs            []string          `json:"edge_ids,omitempty" yaml:"edge_ids,omitempty"`
}

type Assertion struct {
	Metric string          `json:"metric" yaml:"metric"`
	Target AssertionTarget `json:"target" yaml:"target"`
	Op     string          `json:"op" yaml:"op"`
	Value  float64         `json:"value" yaml:"value"`
}

type AssertionTarget struct {
	Type       string   `json:"type" yaml:"type"`
	EndpointID string   `json:"endpoint_id,omitempty" yaml:"endpoint_id,omitempty"`
	Services   []string `json:"services,omitempty" yaml:"services,omitempty"`
	EdgeID     string   `json:"edge_id,omitempty" yaml:"edge_id,omitempty"`
}

func Load(path string) (Contract, error) {
	var contract Contract
	if err := config.LoadStructured(path, &contract); err != nil {
		return Contract{}, fmt.Errorf("read fault contract: %w", err)
	}
	if strings.TrimSpace(contract.SchemaVersion) == "" {
		contract.SchemaVersion = SchemaVersion
	}
	if err := contract.Validate(); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

func (c Contract) Validate() error {
	if c.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported fault contract schema_version: got %q want %q", c.SchemaVersion, SchemaVersion)
	}
	if len(c.Profiles) == 0 {
		return errors.New("fault contract must define at least one profile")
	}
	for name, profile := range c.Profiles {
		if strings.TrimSpace(name) == "" {
			return errors.New("fault profile name cannot be empty")
		}
		if len(profile.Faults) == 0 && len(profile.Assertions) == 0 {
			return fmt.Errorf("fault profile %q must define faults or assertions", name)
		}
		for idx, fault := range profile.Faults {
			if err := fault.Validate(); err != nil {
				return fmt.Errorf("profile %q fault %d: %w", name, idx, err)
			}
		}
		for idx, assertion := range profile.Assertions {
			if err := assertion.Validate(); err != nil {
				return fmt.Errorf("profile %q assertion %d: %w", name, idx, err)
			}
		}
	}
	return nil
}

func (f Fault) Validate() error {
	switch f.Type {
	case TypeCorrelatedFailureDomain:
		if f.Selector.emptyServiceSelector() {
			return errors.New("correlated_failure_domain requires a service, placement, or shared resource selector")
		}
	case TypeEdgeFailStop:
		if len(f.Selector.EdgeIDs) == 0 {
			return errors.New("edge_fail_stop requires selector.edge_ids")
		}
	case TypeEdgePartialDegradation:
		if len(f.Selector.EdgeIDs) == 0 {
			return errors.New("edge_partial_degradation requires selector.edge_ids")
		}
		if f.ErrorRate == nil {
			return errors.New("edge_partial_degradation requires error_rate")
		}
	case TypeServicePartialDegradation:
		if f.Selector.emptyServiceSelector() {
			return errors.New("service_partial_degradation requires a service selector")
		}
		if f.ErrorRate == nil {
			return errors.New("service_partial_degradation requires error_rate")
		}
	default:
		return fmt.Errorf("unsupported fault type %q", f.Type)
	}
	if f.ErrorRate != nil && (*f.ErrorRate < 0 || *f.ErrorRate > 1) {
		return errors.New("error_rate must be in range [0,1]")
	}
	if f.LatencyMS != nil {
		values := []float64{f.LatencyMS.P50, f.LatencyMS.P90, f.LatencyMS.P95, f.LatencyMS.P99}
		active := false
		for _, value := range values {
			if value < 0 {
				return errors.New("latency_ms values must be >= 0")
			}
			if value > 0 {
				active = true
			}
		}
		if !active {
			return errors.New("latency_ms must set at least one percentile")
		}
	}
	return f.Selector.Validate()
}

func (s Selector) Validate() error {
	for idx, id := range s.ServiceIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("service_ids[%d] cannot be empty", idx)
		}
	}
	for key, value := range s.ServiceLabels {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return errors.New("service_labels cannot contain empty keys or values")
		}
	}
	for key, value := range s.PlacementLabels {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return errors.New("placement_labels cannot contain empty keys or values")
		}
	}
	for idx, ref := range s.SharedResourceRefs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("shared_resource_refs[%d] cannot be empty", idx)
		}
	}
	for idx, edgeID := range s.EdgeIDs {
		if strings.TrimSpace(edgeID) == "" {
			return fmt.Errorf("edge_ids[%d] cannot be empty", idx)
		}
	}
	return nil
}

func (s Selector) emptyServiceSelector() bool {
	return len(s.ServiceIDs) == 0 &&
		len(s.ServiceLabels) == 0 &&
		len(s.PlacementLabels) == 0 &&
		len(s.SharedResourceRefs) == 0
}

func (a Assertion) Validate() error {
	switch a.Metric {
	case MetricExpectedSuccessRate,
		MetricMaxAmplificationFactor,
		MetricTimeoutMismatchCount,
		MetricBlastRadiusServiceCount,
		MetricBlastRadiusEndpointCount:
	default:
		return fmt.Errorf("unsupported assertion metric %q", a.Metric)
	}
	switch a.Op {
	case ">=", "<=", "==":
	default:
		return fmt.Errorf("unsupported assertion op %q", a.Op)
	}
	return a.Target.Validate()
}

func (t AssertionTarget) Validate() error {
	switch t.Type {
	case TargetEndpoint:
		if strings.TrimSpace(t.EndpointID) == "" {
			return errors.New("endpoint target requires endpoint_id")
		}
	case TargetPath:
		if len(t.Services) == 0 {
			return errors.New("path target requires services")
		}
		for idx, serviceID := range t.Services {
			if strings.TrimSpace(serviceID) == "" {
				return fmt.Errorf("path target services[%d] cannot be empty", idx)
			}
		}
	case TargetEdge:
		if strings.TrimSpace(t.EdgeID) == "" {
			return errors.New("edge target requires edge_id")
		}
	case TargetProfile:
	default:
		return fmt.Errorf("unsupported assertion target type %q", t.Type)
	}
	return nil
}
