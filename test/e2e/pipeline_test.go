package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/app"
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
