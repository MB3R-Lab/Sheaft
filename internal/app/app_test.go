package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

func TestLoadExecutionConfig_PreservesAnalysisSeedUnlessOverridden(t *testing.T) {
	t.Parallel()

	analysisPath := filepath.Join(t.TempDir(), "analysis.yaml")
	writeFile(t, analysisPath, `
schema_version: "1.0"
seed: 12345
profiles:
  - name: default
    trials: 100
    sampling_mode: independent_replica
    failure_probability: 0.1
gate:
  mode: warn
  default_action: warn
`)

	cfg, err := loadExecutionConfig("", analysisPath, nil, "")
	if err != nil {
		t.Fatalf("loadExecutionConfig without override failed: %v", err)
	}
	if cfg.Seed != 12345 {
		t.Fatalf("expected analysis seed to be preserved, got %d", cfg.Seed)
	}
	if cfg.Sources.ConfigSource != config.ParameterSourceOverride {
		t.Fatalf("expected analysis config source override, got %s", cfg.Sources.ConfigSource)
	}
	if cfg.Sources.Seed != config.ParameterSourceOverride {
		t.Fatalf("expected analysis seed source override, got %s", cfg.Sources.Seed)
	}

	override := int64(42)
	cfg, err = loadExecutionConfig("", analysisPath, &override, "")
	if err != nil {
		t.Fatalf("loadExecutionConfig with override failed: %v", err)
	}
	if cfg.Seed != 42 {
		t.Fatalf("expected seed override 42, got %d", cfg.Seed)
	}
	if cfg.Sources.Seed != config.ParameterSourceOverride {
		t.Fatalf("expected override seed source, got %s", cfg.Sources.Seed)
	}
}

func TestLoadExecutionConfig_PolicySources(t *testing.T) {
	t.Parallel()

	policyPath := filepath.Join(t.TempDir(), "policy.yaml")
	writeFile(t, policyPath, `
mode: warn
default_action: warn
global_threshold: 0.99
failure_probability: 0.1
trials: 120
`)

	cfg, err := loadExecutionConfig(policyPath, "", nil, "")
	if err != nil {
		t.Fatalf("loadExecutionConfig from policy failed: %v", err)
	}
	if cfg.Sources.ConfigSource != config.ParameterSourcePolicy {
		t.Fatalf("expected policy config source, got %s", cfg.Sources.ConfigSource)
	}
	if cfg.Sources.Trials != config.ParameterSourcePolicy {
		t.Fatalf("expected policy trials source, got %s", cfg.Sources.Trials)
	}
	if cfg.Sources.Profiles["default"].FailureProbability != config.ParameterSourcePolicy {
		t.Fatalf("expected policy profile source, got %+v", cfg.Sources.Profiles["default"])
	}
}

func TestRunServe_InvalidPollIntervalReturnsExitError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifactPath := filepath.Join(root, "model.json")
	if err := model.WriteToFile(artifactPath, testModel()); err != nil {
		t.Fatalf("write model: %v", err)
	}
	analysisPath := filepath.Join(root, "analysis.yaml")
	writeFile(t, analysisPath, `
schema_version: "1.0"
profiles:
  - name: default
    trials: 100
    sampling_mode: independent_replica
    failure_probability: 0.1
gate:
  mode: warn
  default_action: warn
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRunner(&stdout, &stderr)
	code := runner.Run([]string{
		"serve",
		"--artifact", artifactPath,
		"--analysis", analysisPath,
		"--poll-interval", "not-a-duration",
	})
	if code != ExitError {
		t.Fatalf("expected serve to fail fast, got code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "poll_interval") {
		t.Fatalf("expected poll interval validation error, got %s", stderr.String())
	}
}

func testModel() model.ResilienceModel {
	return model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"},
		},
		Metadata: model.Metadata{
			SourceType:   "bering",
			SourceRef:    "bering://app-test",
			DiscoveredAt: "2026-03-11T08:00:00Z",
			Confidence:   0.8,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
