package integration_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/app"
)

func TestDiscoverFailsOnEmptyTraces(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	tracePath := filepath.Join(root, "test", "fixtures", "traces.empty.json")
	modelOut := filepath.Join(t.TempDir(), "model.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)
	if code := runner.Run([]string{"discover", "--input", tracePath, "--out", modelOut}); code != app.ExitError {
		t.Fatalf("expected discover failure code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunFailsOnInvalidPolicy(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	tracePath := filepath.Join(root, "test", "fixtures", "traces.fixture.json")
	policyPath := filepath.Join(root, "test", "fixtures", "policy.invalid.yaml")
	outDir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)
	if code := runner.Run([]string{"run", "--input", tracePath, "--policy", policyPath, "--out-dir", outDir}); code != app.ExitError {
		t.Fatalf("expected run failure code=%d stderr=%s", code, stderr.String())
	}
}

func TestSimulateWorksWithDisconnectedGraph(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	modelPath := filepath.Join(root, "test", "fixtures", "model.disconnected.json")
	policyPath := filepath.Join(root, "test", "fixtures", "policy.fixture.yaml")
	reportOut := filepath.Join(t.TempDir(), "report.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)
	if code := runner.Run([]string{"simulate", "--model", modelPath, "--policy", policyPath, "--out", reportOut, "--seed", "7"}); code != app.ExitOK {
		t.Fatalf("simulate on disconnected graph failed code=%d stderr=%s", code, stderr.String())
	}
}
