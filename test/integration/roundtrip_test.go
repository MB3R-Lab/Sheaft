package integration_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/app"
)

func TestRoundtripDiscoverSimulateGate(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	tracePath := filepath.Join(root, "test", "fixtures", "traces.fixture.json")
	policyPath := filepath.Join(root, "test", "fixtures", "policy.fixture.yaml")
	schemaModel := filepath.Join(root, "api", "schema", "model.schema.json")
	schemaReport := filepath.Join(root, "api", "schema", "report.schema.json")

	outDir := t.TempDir()
	modelOut := filepath.Join(outDir, "model.json")
	reportOut := filepath.Join(outDir, "report.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)

	if code := runner.Run([]string{"discover", "--input", tracePath, "--out", modelOut}); code != app.ExitOK {
		t.Fatalf("discover failed code=%d stderr=%s", code, stderr.String())
	}
	validateJSONAgainstSchemaRequired(t, schemaModel, modelOut)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"simulate", "--model", modelOut, "--policy", policyPath, "--out", reportOut, "--seed", "42"}); code != app.ExitOK {
		t.Fatalf("simulate failed code=%d stderr=%s", code, stderr.String())
	}
	validateJSONAgainstSchemaRequired(t, schemaReport, reportOut)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"gate", "--report", reportOut, "--policy", policyPath, "--mode", "fail"}); code != app.ExitGateDeny {
		t.Fatalf("gate expected deny code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}
