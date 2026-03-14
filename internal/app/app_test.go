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

	cfg, err := loadExecutionConfig("", analysisPath, "", nil, "")
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
	cfg, err = loadExecutionConfig("", analysisPath, "", &override, "")
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

	cfg, err := loadExecutionConfig(policyPath, "", "", nil, "")
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

func TestLoadExecutionConfig_ContractPolicyOverride(t *testing.T) {
	t.Parallel()

	analysisPath := filepath.Join(t.TempDir(), "analysis.yaml")
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
	contractPolicyPath := filepath.Join(filepath.Dir(analysisPath), "contract-policy.yaml")
	writeFile(t, contractPolicyPath, `
allowed_kinds:
  - snapshot
deprecated_action: warn
deprecated_contracts:
  - kind: snapshot
    name: io.mb3r.bering.snapshot
    versions: ["1.0.0"]
`)

	cfg, err := loadExecutionConfig("", analysisPath, contractPolicyPath, nil, "")
	if err != nil {
		t.Fatalf("loadExecutionConfig with contract policy failed: %v", err)
	}
	if len(cfg.ContractPolicy.AllowedKinds) != 1 || cfg.ContractPolicy.AllowedKinds[0] != "snapshot" {
		t.Fatalf("expected contract policy to be loaded, got %+v", cfg.ContractPolicy)
	}
}

func TestRun_DeprecatedContractPolicyFailReturnsExitError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifactPath := filepath.Join(root, "snapshot.json")
	writeFile(t, artifactPath, `
{
  "snapshot_id": "snapshot-1",
  "topology_version": "topology-1",
  "window_start": "2026-03-11T08:00:00Z",
  "window_end": "2026-03-11T08:05:00Z",
  "ingest": {
    "spans": 10,
    "traces": 2,
    "dropped_spans": 0,
    "late_spans": 0
  },
  "counts": {
    "services": 1,
    "edges": 0,
    "endpoints": 1
  },
  "coverage": {
    "confidence": 0.8,
    "service_support_min": 1,
    "edge_support_min": 0,
    "endpoint_support_min": 1
  },
  "sources": [
    {
      "type": "traces",
      "connector": "trace_file",
      "ref": "bering://snapshot/1",
      "observations": 10
    }
  ],
  "diff": {
    "added_services": 1,
    "removed_services": 0,
    "changed_services": 0,
    "added_edges": 0,
    "removed_edges": 0,
    "changed_edges": 0,
    "added_endpoints": 1,
    "removed_endpoints": 0,
    "changed_endpoints": 0
  },
  "discovery": {
    "services": [],
    "edges": [],
    "endpoints": []
  },
  "model": {
    "services": [{ "id": "frontend", "name": "frontend", "replicas": 1 }],
    "endpoints": [{ "id": "frontend:GET /health", "entry_service": "frontend", "success_predicate_ref": "frontend:GET /health" }],
    "metadata": {
      "source_type": "bering",
      "source_ref": "bering://snapshot/1",
      "discovered_at": "2026-03-11T08:00:00Z",
      "confidence": 0.8,
      "schema": {
        "name": "io.mb3r.bering.model",
        "version": "1.0.0",
        "uri": "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json",
        "digest": "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7"
      }
    }
  },
  "metadata": {
    "source_type": "bering",
    "source_ref": "bering://snapshot/1",
    "emitted_at": "2026-03-11T08:00:00Z",
    "confidence": 0.8,
    "schema": {
      "name": "io.mb3r.bering.snapshot",
      "version": "1.0.0",
      "uri": "https://mb3r-lab.github.io/Bering/schema/snapshot/v1.0.0/snapshot.schema.json",
      "digest": "sha256:87e4e887ed4a37b72f6136e268b73552eccb92941c4de2c6f3a514dd066ea972"
    }
  }
}
`)
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
	contractPolicyPath := filepath.Join(root, "contract-policy.yaml")
	writeFile(t, contractPolicyPath, `
deprecated_action: fail
deprecated_contracts:
  - kind: snapshot
    name: io.mb3r.bering.snapshot
    versions: ["1.0.0"]
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRunner(&stdout, &stderr)
	code := runner.Run([]string{
		"run",
		"--model", artifactPath,
		"--analysis", analysisPath,
		"--contract-policy", contractPolicyPath,
		"--out-dir", filepath.Join(root, "out"),
	})
	if code != ExitError {
		t.Fatalf("expected run to fail on deprecated contract policy, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "contract policy") {
		t.Fatalf("expected contract policy error, got %s", stderr.String())
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
