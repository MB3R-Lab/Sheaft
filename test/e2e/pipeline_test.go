package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/app"
	"github.com/MB3R-Lab/Sheaft/internal/report"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

func TestRunPipelineGeneratesArtifacts(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	modelPath := filepath.Join(root, "test", "fixtures", "model.disconnected.json")
	policyPath := filepath.Join(root, "test", "fixtures", "policy.fixture.yaml")
	outDir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)
	if code := runner.Run([]string{"run", "--model", modelPath, "--policy", policyPath, "--out-dir", outDir, "--seed", "42"}); code != app.ExitOK {
		t.Fatalf("run failed code=%d stderr=%s", code, stderr.String())
	}

	for _, fileName := range []string{"model.json", "report.json", "summary.md"} {
		target := filepath.Join(outDir, fileName)
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("expected %s to exist: %v", target, err)
		}
	}
}

func TestRunPipelineReportsBirnbaumImportance(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", ".."))
	outDir := t.TempDir()
	analysisPath := filepath.Join(outDir, "analysis.yaml")
	analysis := `schema_version: "1.3"
seed: 42
trials: 100
profiles:
  - name: steady
importance:
  - name: health-criticality
    profile: steady
    targets:
      - endpoint_id: frontend:GET /health
    components:
      kinds: [service]
gate:
  mode: report
  default_action: report
`
	if err := os.WriteFile(analysisPath, []byte(analysis), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)
	args := []string{"run", "--model", filepath.Join(root, "test", "fixtures", "model.disconnected.json"), "--analysis", analysisPath, "--out-dir", outDir}
	if code := runner.Run(args); code != app.ExitOK {
		t.Fatalf("run code=%d stderr=%s", code, stderr.String())
	}
	rep, err := report.Load(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Importance) != 1 || rep.Importance[0].EvaluatedComponents != 2 {
		t.Fatalf("importance did not survive report serialization: %+v", rep.Importance)
	}
	components := rep.Importance[0].Endpoints[0].Components
	if components[0].ID != "frontend" || components[0].BirnbaumImportance != 1 || components[1].BirnbaumImportance != 0 {
		t.Fatalf("unexpected disconnected-model importance: %+v", components)
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Component criticality (Birnbaum importance)") || !strings.Contains(string(raw), "service/frontend") {
		t.Fatal("summary missing importance ranking")
	}
	// Importance is explanatory evidence; reading it must preserve the normal gate.
	if code := runner.Run([]string{"gate", "--report", filepath.Join(outDir, "report.json"), "--analysis", analysisPath}); code != app.ExitOK {
		t.Fatalf("gate code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunPipelineReportsFailureToleranceSweep(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	modelPath := filepath.Join(root, "test", "fixtures", "model.disconnected.json")
	outDir := t.TempDir()
	analysisPath := filepath.Join(outDir, "analysis.yaml")
	analysis := `
schema_version: "1.2"
seed: 42
trials: 5000
profiles:
  - name: steady
    sampling_mode: independent_replica
sweeps:
  - name: health-boundary
    profile: steady
    axis:
      type: independent_replica_failure_probability
      values: [0, 0.5, 1]
    targets:
      - endpoint_id: frontend:GET /health
        slo: 0.75
gate:
  mode: report
  default_action: report
`
	if err := os.WriteFile(analysisPath, []byte(analysis), 0o644); err != nil {
		t.Fatalf("write analysis config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)
	if code := runner.Run([]string{"run", "--model", modelPath, "--analysis", analysisPath, "--out-dir", outDir}); code != app.ExitOK {
		t.Fatalf("run failed code=%d stderr=%s", code, stderr.String())
	}

	rep, err := report.Load(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatalf("load report: %v", err)
	}
	if len(rep.Sweeps) != 1 || rep.Sweeps[0].Boundaries[0].Status != simulation.SweepBoundaryCrossed {
		t.Fatalf("unexpected sweep report: %+v", rep.Sweeps)
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "summary.md"))
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if !strings.Contains(string(raw), "health-boundary") || !strings.Contains(string(raw), "bracket=`[0.0000, 0.5000]`") {
		t.Fatalf("summary does not contain the boundary:\n%s", raw)
	}
}
