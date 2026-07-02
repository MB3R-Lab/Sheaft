package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAnalysis_V110SupportsFaultContract(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "analysis.yaml")
	writeConfigFile(t, path, `
schema_version: "1.1"
fault_contract: configs/fault-contract.example.yaml
profiles:
  - name: steady
    fault_profile: payment-brownout
gate:
  mode: warn
  default_action: warn
`)

	cfg, err := LoadAnalysis(path)
	if err != nil {
		t.Fatalf("LoadAnalysis failed: %v", err)
	}
	if cfg.SchemaVersion != AnalysisSchemaVersionV110 || cfg.FaultContract == "" || cfg.Profiles[0].FaultProfile == "" {
		t.Fatalf("expected v1.1 config to preserve advanced fields, got %+v", cfg)
	}
}

func TestLoadAnalysis_V100RejectsFaultContract(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "analysis.yaml")
	writeConfigFile(t, path, `
schema_version: "1.0"
fault_contract: configs/fault-contract.example.yaml
profiles:
  - name: steady
gate:
  mode: warn
  default_action: warn
`)

	if _, err := LoadAnalysis(path); err == nil {
		t.Fatal("expected v1.0 analysis config to reject fault_contract")
	}
}

func TestLoadAnalysis_ReliabilityInheritsAndOverrides(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "analysis.yaml")
	writeConfigFile(t, path, `
schema_version: "1.1"
reliability:
  node_live_probability: 0.97
  edge_live_probability: 0.95
  services:
    checkout: 0.99
  edges:
    frontend|checkout|sync|true: 0.98
profiles:
  - name: steady
  - name: stress
    reliability:
      edge_live_probability: 0.90
      services:
        payment: 0.80
gate:
  mode: warn
  default_action: warn
`)

	cfg, err := LoadAnalysis(path)
	if err != nil {
		t.Fatalf("LoadAnalysis failed: %v", err)
	}
	if got := *cfg.Profiles[0].Reliability.NodeLiveProbability; got != 0.97 {
		t.Fatalf("expected inherited node reliability, got %f", got)
	}
	if got := *cfg.Profiles[1].Reliability.EdgeLiveProbability; got != 0.90 {
		t.Fatalf("expected profile edge reliability override, got %f", got)
	}
	if got := cfg.Profiles[1].Reliability.Services["checkout"]; got != 0.99 {
		t.Fatalf("expected inherited service reliability, got %f", got)
	}
	if got := cfg.Profiles[1].Reliability.Services["payment"]; got != 0.80 {
		t.Fatalf("expected profile service reliability override, got %f", got)
	}
	if cfg.Sources.Profiles["stress"].Reliability != ParameterSourceOverride {
		t.Fatalf("expected stress reliability source override, got %s", cfg.Sources.Profiles["stress"].Reliability)
	}
}

func TestLoadAnalysis_ReliabilityRejectsInvalidProbability(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "analysis.yaml")
	writeConfigFile(t, path, `
schema_version: "1.1"
profiles:
  - name: bad
    reliability:
      edges:
        frontend|checkout|sync|true: 1.1
gate:
  mode: warn
  default_action: warn
`)

	if _, err := LoadAnalysis(path); err == nil {
		t.Fatal("expected invalid reliability probability to fail")
	}
}

func TestLoadAnalysis_AcceptsFixedKReplicaSlots(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "analysis.yaml")
	writeConfigFile(t, path, `
schema_version: "1.0"
profiles:
  - name: experiment
    sampling_mode: fixed_k_replica_slots
    failure_probability: 0.30
gate:
  mode: report
  default_action: report
`)

	cfg, err := LoadAnalysis(path)
	if err != nil {
		t.Fatalf("LoadAnalysis failed: %v", err)
	}
	if got := cfg.Profiles[0].SamplingMode; got != SamplingModeFixedKReplicaSlots {
		t.Fatalf("expected fixed replica slot sampling mode, got %q", got)
	}
	if got := cfg.Profiles[0].FailureProbability; got != 0.30 {
		t.Fatalf("expected failure_probability 0.30, got %f", got)
	}
}

func writeConfigFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}
