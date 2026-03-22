package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

type EdgeKind string

const (
	EdgeKindSync  EdgeKind = "sync"
	EdgeKindAsync EdgeKind = "async"
)

type CommonMetadata struct {
	Labels     map[string]string `json:"labels,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	SLORefs    []string          `json:"slo_refs,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Placement struct {
	Replicas int               `json:"replicas"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type ServiceMetadata struct {
	CommonMetadata
	FailureEligible    *bool       `json:"failure_eligible,omitempty"`
	Placements         []Placement `json:"placements,omitempty"`
	SharedResourceRefs []string    `json:"shared_resource_refs,omitempty"`
}

type EdgeMetadata struct {
	CommonMetadata
	Weight *float64 `json:"weight,omitempty"`
}

type EndpointMetadata struct {
	CommonMetadata
	Weight *float64 `json:"weight,omitempty"`
}

type BackoffPolicy struct {
	InitialMS  int     `json:"initial_ms,omitempty"`
	MaxMS      int     `json:"max_ms,omitempty"`
	Multiplier float64 `json:"multiplier,omitempty"`
	Jitter     string  `json:"jitter,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int            `json:"max_attempts,omitempty"`
	BudgetCap   float64        `json:"budget_cap,omitempty"`
	RetryOn     []string       `json:"retry_on,omitempty"`
	Backoff     *BackoffPolicy `json:"backoff,omitempty"`
}

type CircuitBreakerPolicy struct {
	Enabled            *bool `json:"enabled,omitempty"`
	MaxPendingRequests int   `json:"max_pending_requests,omitempty"`
	MaxRequests        int   `json:"max_requests,omitempty"`
	MaxConnections     int   `json:"max_connections,omitempty"`
	Consecutive5xx     int   `json:"consecutive_5xx,omitempty"`
	IntervalMS         int   `json:"interval_ms,omitempty"`
	BaseEjectionTimeMS int   `json:"base_ejection_time_ms,omitempty"`
}

type ResiliencePolicy struct {
	RequestTimeoutMS int                   `json:"request_timeout_ms,omitempty"`
	PerTryTimeoutMS  int                   `json:"per_try_timeout_ms,omitempty"`
	Retry            *RetryPolicy          `json:"retry,omitempty"`
	CircuitBreaker   *CircuitBreakerPolicy `json:"circuit_breaker,omitempty"`
}

type LatencySummary struct {
	P50 float64 `json:"p50,omitempty"`
	P90 float64 `json:"p90,omitempty"`
	P95 float64 `json:"p95,omitempty"`
	P99 float64 `json:"p99,omitempty"`
}

type ObservedEdge struct {
	LatencyMS *LatencySummary `json:"latency_ms,omitempty"`
	ErrorRate *float64        `json:"error_rate,omitempty"`
}

type PolicyScope struct {
	SourceEndpointID string `json:"source_endpoint_id,omitempty"`
	SourceRoute      string `json:"source_route,omitempty"`
	Method           string `json:"method,omitempty"`
	Operation        string `json:"operation,omitempty"`
}

type Service struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Replicas int              `json:"replicas"`
	Metadata *ServiceMetadata `json:"metadata,omitempty"`
}

type Edge struct {
	ID          string            `json:"id,omitempty"`
	From        string            `json:"from"`
	To          string            `json:"to"`
	Kind        EdgeKind          `json:"kind"`
	Blocking    bool              `json:"blocking"`
	Metadata    *EdgeMetadata     `json:"metadata,omitempty"`
	Resilience  *ResiliencePolicy `json:"resilience,omitempty"`
	Observed    *ObservedEdge     `json:"observed,omitempty"`
	PolicyScope *PolicyScope      `json:"policy_scope,omitempty"`
}

type Endpoint struct {
	ID                  string                 `json:"id"`
	EntryService        string                 `json:"entry_service"`
	SuccessPredicateRef string                 `json:"success_predicate_ref"`
	SuccessPredicate    *predicates.Definition `json:"success_predicate,omitempty"`
	Method              string                 `json:"method,omitempty"`
	Path                string                 `json:"path,omitempty"`
	Metadata            *EndpointMetadata      `json:"metadata,omitempty"`
}

type Metadata struct {
	SourceType      string  `json:"source_type"`
	SourceRef       string  `json:"source_ref"`
	DiscoveredAt    string  `json:"discovered_at"`
	Confidence      float64 `json:"confidence"`
	TopologyVersion string  `json:"topology_version,omitempty"`
	Schema          Schema  `json:"schema"`
}

type Schema struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URI     string `json:"uri"`
	Digest  string `json:"digest"`
}

type ResilienceModel struct {
	Services        []Service                        `json:"services"`
	Edges           []Edge                           `json:"edges"`
	Endpoints       []Endpoint                       `json:"endpoints"`
	Predicates      map[string]predicates.Definition `json:"predicates,omitempty"`
	EndpointWeights map[string]float64               `json:"endpoint_weights,omitempty"`
	Metadata        Metadata                         `json:"metadata"`
}

func (m ResilienceModel) Validate() error {
	if len(m.Services) == 0 {
		return errors.New("model has no services")
	}

	serviceSet := make(map[string]struct{}, len(m.Services))
	for _, svc := range m.Services {
		if strings.TrimSpace(svc.ID) == "" {
			return errors.New("service id cannot be empty")
		}
		if strings.TrimSpace(svc.Name) == "" {
			return fmt.Errorf("service %q has empty name", svc.ID)
		}
		if svc.Replicas < 0 {
			return fmt.Errorf("service %q has negative replicas", svc.ID)
		}
		if err := validateServiceMetadata(svc.ID, svc.Metadata); err != nil {
			return err
		}
		serviceSet[svc.ID] = struct{}{}
	}

	for _, edge := range m.Edges {
		if strings.TrimSpace(m.Metadata.Schema.Version) == "1.1.0" && strings.TrimSpace(edge.ID) == "" {
			return fmt.Errorf("edge %s -> %s requires id for schema version 1.1.0", edge.From, edge.To)
		}
		if _, ok := serviceSet[edge.From]; !ok {
			return fmt.Errorf("edge.from service not found: %s", edge.From)
		}
		if _, ok := serviceSet[edge.To]; !ok {
			return fmt.Errorf("edge.to service not found: %s", edge.To)
		}
		if edge.Kind != EdgeKindSync && edge.Kind != EdgeKindAsync {
			return fmt.Errorf("unsupported edge kind: %s", edge.Kind)
		}
		if err := validateEdgeMetadata(edge.ID, edge.Metadata); err != nil {
			return err
		}
		if err := validateResiliencePolicy(edge.ID, edge.Resilience); err != nil {
			return err
		}
		if err := validateObservedEdge(edge.ID, edge.Observed); err != nil {
			return err
		}
	}

	for _, ep := range m.Endpoints {
		if strings.TrimSpace(ep.ID) == "" {
			return errors.New("endpoint id cannot be empty")
		}
		if _, ok := serviceSet[ep.EntryService]; !ok {
			return fmt.Errorf("endpoint %q entry service not found: %s", ep.ID, ep.EntryService)
		}
		if strings.TrimSpace(ep.SuccessPredicateRef) == "" {
			return fmt.Errorf("endpoint %q has empty success_predicate_ref", ep.ID)
		}
		if ep.SuccessPredicate != nil {
			if err := ep.SuccessPredicate.Validate(); err != nil {
				return fmt.Errorf("endpoint %q success_predicate: %w", ep.ID, err)
			}
		}
		if err := validateEndpointMetadata(ep.ID, ep.Metadata); err != nil {
			return err
		}
	}
	for name, def := range m.Predicates {
		if strings.TrimSpace(name) == "" {
			return errors.New("predicates key cannot be empty")
		}
		if err := def.Validate(); err != nil {
			return fmt.Errorf("predicate %q: %w", name, err)
		}
	}
	for endpoint, weight := range m.EndpointWeights {
		if strings.TrimSpace(endpoint) == "" {
			return errors.New("endpoint_weights key cannot be empty")
		}
		if weight < 0 {
			return fmt.Errorf("endpoint_weights[%s] must be >= 0", endpoint)
		}
	}

	if strings.TrimSpace(m.Metadata.SourceType) == "" {
		return errors.New("metadata.source_type cannot be empty")
	}
	if strings.TrimSpace(m.Metadata.SourceRef) == "" {
		return errors.New("metadata.source_ref cannot be empty")
	}
	if m.Metadata.Confidence < 0 || m.Metadata.Confidence > 1 {
		return errors.New("metadata.confidence must be in range [0,1]")
	}
	if strings.TrimSpace(m.Metadata.Schema.Name) == "" {
		return errors.New("metadata.schema.name cannot be empty")
	}
	if strings.TrimSpace(m.Metadata.Schema.Version) == "" {
		return errors.New("metadata.schema.version cannot be empty")
	}
	if strings.TrimSpace(m.Metadata.Schema.URI) == "" {
		return errors.New("metadata.schema.uri cannot be empty")
	}
	if strings.TrimSpace(m.Metadata.Schema.Digest) == "" {
		return errors.New("metadata.schema.digest cannot be empty")
	}

	return nil
}

func validateServiceMetadata(serviceID string, metadata *ServiceMetadata) error {
	if metadata == nil {
		return nil
	}
	if err := validateCommonMetadata(fmt.Sprintf("service %q", serviceID), metadata.CommonMetadata); err != nil {
		return err
	}
	for idx, placement := range metadata.Placements {
		if placement.Replicas < 0 {
			return fmt.Errorf("service %q placement %d has negative replicas", serviceID, idx)
		}
		if err := validateLabelMap(fmt.Sprintf("service %q placement %d labels", serviceID, idx), placement.Labels); err != nil {
			return err
		}
	}
	for idx, ref := range metadata.SharedResourceRefs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("service %q shared_resource_refs[%d] cannot be empty", serviceID, idx)
		}
	}
	return nil
}

func validateEdgeMetadata(edgeID string, metadata *EdgeMetadata) error {
	if metadata == nil {
		return nil
	}
	if err := validateCommonMetadata(fmt.Sprintf("edge %q", edgeID), metadata.CommonMetadata); err != nil {
		return err
	}
	if metadata.Weight != nil && *metadata.Weight < 0 {
		return fmt.Errorf("edge %q metadata.weight must be >= 0", edgeID)
	}
	return nil
}

func validateEndpointMetadata(endpointID string, metadata *EndpointMetadata) error {
	if metadata == nil {
		return nil
	}
	if err := validateCommonMetadata(fmt.Sprintf("endpoint %q", endpointID), metadata.CommonMetadata); err != nil {
		return err
	}
	if metadata.Weight != nil && *metadata.Weight < 0 {
		return fmt.Errorf("endpoint %q metadata.weight must be >= 0", endpointID)
	}
	return nil
}

func validateCommonMetadata(label string, metadata CommonMetadata) error {
	if err := validateLabelMap(label+" labels", metadata.Labels); err != nil {
		return err
	}
	if err := validateLabelMap(label+" attributes", metadata.Attributes); err != nil {
		return err
	}
	for idx, tag := range metadata.Tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("%s tags[%d] cannot be empty", label, idx)
		}
	}
	for idx, ref := range metadata.SLORefs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("%s slo_refs[%d] cannot be empty", label, idx)
		}
	}
	return nil
}

func validateLabelMap(label string, values map[string]string) error {
	for key, value := range values {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s key cannot be empty", label)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s[%s] value cannot be empty", label, key)
		}
	}
	return nil
}

func validateResiliencePolicy(edgeID string, policy *ResiliencePolicy) error {
	if policy == nil {
		return nil
	}
	if policy.RequestTimeoutMS < 0 {
		return fmt.Errorf("edge %q resilience.request_timeout_ms must be >= 0", edgeID)
	}
	if policy.PerTryTimeoutMS < 0 {
		return fmt.Errorf("edge %q resilience.per_try_timeout_ms must be >= 0", edgeID)
	}
	if policy.Retry != nil {
		if policy.Retry.MaxAttempts < 0 {
			return fmt.Errorf("edge %q resilience.retry.max_attempts must be >= 0", edgeID)
		}
		if policy.Retry.BudgetCap < 0 {
			return fmt.Errorf("edge %q resilience.retry.budget_cap must be >= 0", edgeID)
		}
		if policy.Retry.Backoff != nil {
			if policy.Retry.Backoff.InitialMS < 0 || policy.Retry.Backoff.MaxMS < 0 || policy.Retry.Backoff.Multiplier < 0 {
				return fmt.Errorf("edge %q resilience.retry.backoff values must be >= 0", edgeID)
			}
		}
	}
	if policy.CircuitBreaker != nil {
		if policy.CircuitBreaker.MaxPendingRequests < 0 ||
			policy.CircuitBreaker.MaxRequests < 0 ||
			policy.CircuitBreaker.MaxConnections < 0 ||
			policy.CircuitBreaker.Consecutive5xx < 0 ||
			policy.CircuitBreaker.IntervalMS < 0 ||
			policy.CircuitBreaker.BaseEjectionTimeMS < 0 {
			return fmt.Errorf("edge %q resilience.circuit_breaker values must be >= 0", edgeID)
		}
	}
	return nil
}

func validateObservedEdge(edgeID string, observed *ObservedEdge) error {
	if observed == nil {
		return nil
	}
	if observed.LatencyMS != nil {
		for name, value := range map[string]float64{
			"p50": observed.LatencyMS.P50,
			"p90": observed.LatencyMS.P90,
			"p95": observed.LatencyMS.P95,
			"p99": observed.LatencyMS.P99,
		} {
			if value < 0 {
				return fmt.Errorf("edge %q observed.latency_ms.%s must be >= 0", edgeID, name)
			}
		}
	}
	if observed.ErrorRate != nil && (*observed.ErrorRate < 0 || *observed.ErrorRate > 1) {
		return fmt.Errorf("edge %q observed.error_rate must be in range [0,1]", edgeID)
	}
	return nil
}

func (m ResilienceModel) SortedEndpointIDs() []string {
	ids := make([]string, 0, len(m.Endpoints))
	for _, ep := range m.Endpoints {
		ids = append(ids, ep.ID)
	}
	slices.Sort(ids)
	return ids
}

func LoadFromFile(path string) (ResilienceModel, error) {
	var mdl ResilienceModel

	raw, err := os.ReadFile(path)
	if err != nil {
		return mdl, fmt.Errorf("read model file: %w", err)
	}

	if err := json.Unmarshal(raw, &mdl); err != nil {
		return mdl, fmt.Errorf("decode model json: %w", err)
	}

	if err := mdl.Validate(); err != nil {
		return mdl, fmt.Errorf("validate model: %w", err)
	}

	return mdl, nil
}

func WriteToFile(path string, mdl ResilienceModel) error {
	if err := mdl.Validate(); err != nil {
		return fmt.Errorf("validate model: %w", err)
	}

	raw, err := json.MarshalIndent(mdl, "", "  ")
	if err != nil {
		return fmt.Errorf("encode model json: %w", err)
	}

	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write model file: %w", err)
	}

	return nil
}
