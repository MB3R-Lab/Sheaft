package integration_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/app"
)

func TestRoundtripSimulateGateFromBeringModel(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	modelPath := filepath.Join(root, "test", "fixtures", "model.disconnected.json")
	policyPath := filepath.Join(root, "test", "fixtures", "policy.fixture.yaml")
	schemaModel := filepath.Join(root, "api", "schema", "model.schema.json")
	schemaReport := filepath.Join(root, "api", "schema", "report.schema.json")

	outDir := t.TempDir()
	reportOut := filepath.Join(outDir, "report.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.NewRunner(&stdout, &stderr)

	validateJSONAgainstSchemaRequired(t, schemaModel, modelPath)

	if code := runner.Run([]string{"simulate", "--model", modelPath, "--policy", policyPath, "--out", reportOut, "--seed", "42"}); code != app.ExitOK {
		t.Fatalf("simulate failed code=%d stderr=%s", code, stderr.String())
	}
	validateJSONAgainstSchemaRequired(t, schemaReport, reportOut)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"gate", "--report", reportOut, "--policy", policyPath, "--mode", "fail"}); code != app.ExitGateDeny {
		t.Fatalf("gate expected deny code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}
