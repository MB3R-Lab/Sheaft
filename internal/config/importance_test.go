package config

import (
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportanceConfigDefaultsAndSchemaVersion(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "analysis.yaml")
	writeConfigFile(t, path, `schema_version: "1.3"
profiles:
  - name: steady
importance:
  - name: criticality
    profile: steady
    targets:
      - endpoint_id: checkout
gate:
  mode: report
  default_action: report
`)
	cfg, err := LoadAnalysis(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Importance[0].Method != ImportanceMethodBirnbaum || cfg.Importance[0].ConfidenceLevel != 0.95 || len(cfg.Importance[0].Components.Kinds) != 2 {
		t.Fatalf("defaults missing: %+v", cfg.Importance)
	}
	for _, version := range []string{"1.0", "1.1", "1.2"} {
		cfg.SchemaVersion = version
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "does not support importance") {
			t.Fatalf("version %s accepted importance: %v", version, err)
		}
	}
}

func TestImportanceRejectsInvalidDefinitions(t *testing.T) {
	t.Parallel()
	for _, mutate := range []func(*AnalysisConfig){
		func(c *AnalysisConfig) { c.Importance[0].Profile = "missing" },
		func(c *AnalysisConfig) { c.Importance[0].Method = "blast_radius" },
		func(c *AnalysisConfig) { c.Importance[0].ConfidenceLevel = math.NaN() },
		func(c *AnalysisConfig) { c.Importance[0].TopK = -1 },
		func(c *AnalysisConfig) { c.Importance[0].Targets = nil },
		func(c *AnalysisConfig) { c.Importance[0].Components.Kinds = []string{"zone"} },
		func(c *AnalysisConfig) { c.Importance[0].Components.Kinds = []string{"service", "service"} },
		func(c *AnalysisConfig) {
			c.Importance[0].Components.Kinds = []string{"edge"}
			c.Importance[0].Components.ServiceIDs = []string{"a"}
		},
		func(c *AnalysisConfig) { c.Profiles[0].FaultProfile = "zone-down" },
		func(c *AnalysisConfig) { c.Profiles[0].SamplingMode = SamplingModeFixedKReplicaSlots },
		func(c *AnalysisConfig) { c.Importance = append(c.Importance, c.Importance[0]) },
	} {
		cfg := AnalysisConfig{SchemaVersion: AnalysisSchemaVersionV130, Profiles: []Profile{{Name: "steady"}}, Importance: []Importance{{Name: "criticality", Profile: "steady", Targets: []ImportanceTarget{{EndpointID: "checkout"}}}}}.Normalized()
		mutate(&cfg)
		if err := cfg.Validate(); err == nil {
			t.Fatalf("invalid definition accepted: %+v", cfg.Importance)
		}
	}
}
