package analyzer

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

func TestAnalyzeFile_BaselineArtifactComparisonAcrossContractLines(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	primary := filepath.Join(root, "examples", "outputs", "snapshot-v1.3.0.sample.json")
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

func TestAnalyzeLoaded_UsesArtifactReliabilityAsProfileDefaults(t *testing.T) {
	t.Parallel()

	gatewayLive := 1.0
	checkoutLive := 0.5
	edgeLive := 0.8
	loaded := artifact.Loaded{
		Metadata: artifact.Metadata{
			Kind: modelcontract.KindModel,
			Contract: modelcontract.SupportedContract{
				Name:    modelcontract.BeringModelV130Name,
				Version: modelcontract.BeringModelV130Version,
				URI:     modelcontract.BeringModelV130URI,
				Digest:  modelcontract.BeringModelV130Digest,
				Kind:    modelcontract.KindModel,
			},
		},
		Model: model.ResilienceModel{
			Services: []model.Service{
				{
					ID:       "gateway",
					Name:     "gateway",
					Replicas: 1,
					Metadata: &model.ServiceMetadata{
						Reliability: &model.ReliabilityEvidence{LiveProbability: &gatewayLive},
					},
				},
				{
					ID:       "checkout",
					Name:     "checkout",
					Replicas: 1,
					Metadata: &model.ServiceMetadata{
						Reliability: &model.ReliabilityEvidence{LiveProbability: &checkoutLive},
					},
				},
			},
			Edges: []model.Edge{
				{
					ID:       "gateway|checkout|sync|true",
					From:     "gateway",
					To:       "checkout",
					Kind:     model.EdgeKindSync,
					Blocking: true,
					Metadata: &model.EdgeMetadata{
						Reliability: &model.ReliabilityEvidence{LiveProbability: &edgeLive},
					},
				},
			},
			Endpoints: []model.Endpoint{
				{ID: "gateway:GET /checkout", EntryService: "gateway", SuccessPredicateRef: "gateway:GET /checkout"},
			},
			Metadata: model.Metadata{
				SourceType:   "bering",
				SourceRef:    "test",
				DiscoveredAt: "2026-07-01T00:00:00Z",
				Confidence:   1,
				Schema: model.Schema{
					Name:    modelcontract.BeringModelV130Name,
					Version: modelcontract.BeringModelV130Version,
					URI:     modelcontract.BeringModelV130URI,
					Digest:  modelcontract.BeringModelV130Digest,
				},
			},
		},
	}

	result, err := AnalyzeLoaded(loaded, config.AnalysisConfig{
		SchemaVersion: config.AnalysisSchemaVersionV110,
		Seed:          7,
		Trials:        20000,
		Profiles: []config.Profile{
			{Name: "steady", Trials: 20000, SamplingMode: config.SamplingModeIndependentReplica},
		},
		Gate: config.GateConfig{
			Mode:          config.ModeWarn,
			DefaultAction: config.ModeWarn,
		},
	}, nil)
	if err != nil {
		t.Fatalf("AnalyzeLoaded failed: %v", err)
	}

	got := result.Simulation.Profiles[0].EndpointAvailability["gateway:GET /checkout"]
	want := gatewayLive * checkoutLive * edgeLive
	if math.Abs(got-want) > 0.02 {
		t.Fatalf("artifact reliability availability mismatch: got=%f want=%f", got, want)
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
