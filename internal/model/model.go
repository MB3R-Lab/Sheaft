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

type Service struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
}

type Edge struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Kind     EdgeKind `json:"kind"`
	Blocking bool     `json:"blocking"`
}

type Endpoint struct {
	ID                  string                 `json:"id"`
	EntryService        string                 `json:"entry_service"`
	SuccessPredicateRef string                 `json:"success_predicate_ref"`
	SuccessPredicate    *predicates.Definition `json:"success_predicate,omitempty"`
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
		serviceSet[svc.ID] = struct{}{}
	}

	for _, edge := range m.Edges {
		if _, ok := serviceSet[edge.From]; !ok {
			return fmt.Errorf("edge.from service not found: %s", edge.From)
		}
		if _, ok := serviceSet[edge.To]; !ok {
			return fmt.Errorf("edge.to service not found: %s", edge.To)
		}
		if edge.Kind != EdgeKindSync && edge.Kind != EdgeKindAsync {
			return fmt.Errorf("unsupported edge kind: %s", edge.Kind)
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
