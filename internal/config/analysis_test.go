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

func writeConfigFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}
