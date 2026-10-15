package app

import (
	"bytes"
	"testing"
)

func TestPackagedRequiredCallsExample(t *testing.T) {
	t.Chdir("../..")
	var stdout, stderr bytes.Buffer
	runner := NewRunner(&stdout, &stderr)
	code := runner.Run([]string{"run", "--model", "examples/outputs/model.required-calls.sample.json", "--analysis", "configs/analysis.required-calls.example.yaml", "--out-dir", t.TempDir()})
	if code != 0 {
		t.Fatalf("packaged required-calls example exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}
