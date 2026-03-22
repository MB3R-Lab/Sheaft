package analyzer

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/config"
)

func TestAnalyzeFile_BaselineArtifactComparisonAcrossContractLines(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	primary := filepath.Join(root, "examples", "outputs", "snapshot-v1.1.0.sample.json")
	baseline := filepath.Join(root, "examples", "outputs", "snapshot-v1.0.0.sample.json")

	result, err := AnalyzeFile(primary, config.AnalysisConfig{
		SchemaVersion:      config.AnalysisSchemaVersionV110,
		Seed:               42,
		Trials:             4000,
		SamplingMode:       config.SamplingModeIndependentReplica,
		FailureProbability: 0,
		Baselines: []config.BaselineRef{
			{Name: "bering-1.0.0", Path: baseline},
		},
		Profiles: []config.Profile{
			{Name: "steady", Trials: 4000, SamplingMode: config.SamplingModeIndependentReplica, FailureProbability: 0},
		},
		Gate: config.GateConfig{
			Mode:          config.ModeWarn,
			DefaultAction: config.ModeWarn,
		},
	}, nil)
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	if len(result.Report.Diffs.Baselines) != 1 {
		t.Fatalf("expected one baseline diff, got %+v", result.Report.Diffs.Baselines)
	}
	diff := result.Report.Diffs.Baselines[0]
	if diff.Name != "bering-1.0.0" {
		t.Fatalf("unexpected baseline diff name: %+v", diff)
	}
	if len(diff.Profiles) != 1 {
		t.Fatalf("expected one profile diff, got %+v", diff.Profiles)
	}
	if len(diff.Profiles[0].Endpoints) == 0 {
		t.Fatalf("expected overlapping endpoint metrics to produce diffs, got %+v", diff.Profiles[0])
	}
	foundNonComparable := false
	for _, metric := range diff.Profiles[0].AdvancedMetrics {
		if metric.Status == "non_comparable" {
			foundNonComparable = true
			break
		}
	}
	if !foundNonComparable {
		t.Fatalf("expected advanced metrics missing on the v1.0.0 baseline to be marked non-comparable, got %+v", diff.Profiles[0].AdvancedMetrics)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}
