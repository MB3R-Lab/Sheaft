package config

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

const (
	ImportanceMethodBirnbaum = "birnbaum"
	ImportanceKindService    = "service"
	ImportanceKindEdge       = "edge"
)

type Importance struct {
	Name            string               `json:"name" yaml:"name"`
	Profile         string               `json:"profile" yaml:"profile"`
	Method          string               `json:"method" yaml:"method"`
	Targets         []ImportanceTarget   `json:"targets" yaml:"targets"`
	Components      ImportanceComponents `json:"components" yaml:"components"`
	ConfidenceLevel float64              `json:"confidence_level,omitempty" yaml:"confidence_level,omitempty"`
	TopK            int                  `json:"top_k,omitempty" yaml:"top_k,omitempty"`
}

type ImportanceTarget struct {
	EndpointID string `json:"endpoint_id" yaml:"endpoint_id"`
}

type ImportanceComponents struct {
	Kinds      []string `json:"kinds" yaml:"kinds"`
	ServiceIDs []string `json:"service_ids,omitempty" yaml:"service_ids,omitempty"`
	EdgeIDs    []string `json:"edge_ids,omitempty" yaml:"edge_ids,omitempty"`
}

func validateImportance(cfg AnalysisConfig) error {
	if len(cfg.Importance) > 0 && cfg.SchemaVersion != AnalysisSchemaVersionV130 {
		return fmt.Errorf("analysis schema_version %q does not support importance; use %q", cfg.SchemaVersion, AnalysisSchemaVersionV130)
	}
	names := map[string]bool{}
	for _, spec := range cfg.Importance {
		if strings.TrimSpace(spec.Name) == "" || names[spec.Name] {
			return fmt.Errorf("importance name must be non-empty and unique: %q", spec.Name)
		}
		names[spec.Name] = true
		if spec.Method != ImportanceMethodBirnbaum {
			return fmt.Errorf("importance %q has unsupported method %q", spec.Name, spec.Method)
		}
		profileIndex := slices.IndexFunc(cfg.Profiles, func(p Profile) bool { return p.Name == spec.Profile })
		if profileIndex < 0 {
			return fmt.Errorf("importance %q references unknown profile %q", spec.Name, spec.Profile)
		}
		profile := cfg.Profiles[profileIndex]
		if profile.SamplingMode != SamplingModeIndependentService && profile.SamplingMode != SamplingModeIndependentReplica {
			return fmt.Errorf("importance %q requires independent_service or independent_replica sampling", spec.Name)
		}
		if profile.FaultProfile != "" {
			return fmt.Errorf("importance %q requires a base profile without fault_profile", spec.Name)
		}
		if math.IsNaN(spec.ConfidenceLevel) || spec.ConfidenceLevel <= 0 || spec.ConfidenceLevel >= 1 {
			return fmt.Errorf("importance %q confidence_level must be in range (0,1)", spec.Name)
		}
		if spec.TopK < 0 {
			return fmt.Errorf("importance %q top_k must be >= 0", spec.Name)
		}
		if len(spec.Targets) == 0 {
			return fmt.Errorf("importance %q requires at least one target", spec.Name)
		}
		targets := map[string]bool{}
		for _, target := range spec.Targets {
			if strings.TrimSpace(target.EndpointID) == "" || targets[target.EndpointID] {
				return fmt.Errorf("importance %q target endpoint_id must be non-empty and unique: %q", spec.Name, target.EndpointID)
			}
			targets[target.EndpointID] = true
		}
		if len(spec.Components.Kinds) == 0 {
			return fmt.Errorf("importance %q requires component kinds", spec.Name)
		}
		kinds := map[string]bool{}
		for _, kind := range spec.Components.Kinds {
			if (kind != ImportanceKindService && kind != ImportanceKindEdge) || kinds[kind] {
				return fmt.Errorf("importance %q component kinds must be unique service/edge values", spec.Name)
			}
			kinds[kind] = true
		}
		for _, selection := range []struct {
			kind string
			ids  []string
		}{
			{ImportanceKindService, spec.Components.ServiceIDs}, {ImportanceKindEdge, spec.Components.EdgeIDs},
		} {
			if len(selection.ids) > 0 && !kinds[selection.kind] {
				return fmt.Errorf("importance %q selects %s ids without enabling that kind", spec.Name, selection.kind)
			}
			ids := map[string]bool{}
			for _, id := range selection.ids {
				if strings.TrimSpace(id) == "" || ids[id] {
					return fmt.Errorf("importance %q %s ids must be non-empty and unique", spec.Name, selection.kind)
				}
				ids[id] = true
			}
		}
	}
	return nil
}
