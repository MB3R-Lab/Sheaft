package report

import (
	"math"
	"testing"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

func TestCompare_BaselineDiffs(t *testing.T) {
	t.Parallel()

	current := Report{
		InputArtifact: &InputArtifact{
			Digest:          "sha256:current",
			TopologyVersion: "topology-2",
		},
		Summary: Summary{
			CrossProfileAvailability:         0.8,
			CrossProfileWeightedAvailability: 0.7,
		},
		Profiles: []ProfileSummary{
			{
				Name: "steady",
				Simulation: simulation.ProfileOutput{
					Name:                "steady",
					WeightedAggregate:   0.7,
					UnweightedAggregate: 0.8,
				},
				Decision: "warn",
				EndpointResults: []gate.EndpointResult{
					{EndpointID: "frontend:GET /checkout", Availability: 0.65, Status: "warn"},
				},
			},
		},
	}
	reference := Report{
		InputArtifact: &InputArtifact{
			Digest:          "sha256:baseline",
			TopologyVersion: "topology-1",
		},
		Summary: Summary{
			CrossProfileAvailability:         0.85,
			CrossProfileWeightedAvailability: 0.9,
		},
		Profiles: []ProfileSummary{
			{
				Name: "steady",
				Simulation: simulation.ProfileOutput{
					Name:                "steady",
					WeightedAggregate:   0.9,
					UnweightedAggregate: 0.85,
				},
				Decision: "pass",
				EndpointResults: []gate.EndpointResult{
					{EndpointID: "frontend:GET /checkout", Availability: 0.95, Status: "pass"},
				},
			},
		},
	}

	diff := Compare(current, reference, "baseline-a")

	if diff.Name != "baseline-a" {
		t.Fatalf("unexpected diff name: %s", diff.Name)
	}
	if diff.CurrentDigest != "sha256:current" || diff.ReferenceDigest != "sha256:baseline" {
		t.Fatalf("unexpected digest tracking: %+v", diff)
	}
	if diff.CrossProfileWeighted.Signed >= 0 {
		t.Fatalf("expected negative weighted delta, got %+v", diff.CrossProfileWeighted)
	}
	if math.Abs(diff.CrossProfileUnweighted.Absolute-0.05) > 1e-9 {
		t.Fatalf("expected absolute unweighted delta 0.05, got %+v", diff.CrossProfileUnweighted)
	}
	if len(diff.Profiles) != 1 {
		t.Fatalf("expected one profile diff, got %d", len(diff.Profiles))
	}
	if !diff.Profiles[0].Decision.Changed {
		t.Fatalf("expected decision status change, got %+v", diff.Profiles[0].Decision)
	}
	if diff.Profiles[0].Endpoints[0].Availability.Signed >= 0 {
		t.Fatalf("expected endpoint availability to regress, got %+v", diff.Profiles[0].Endpoints[0].Availability)
	}
}

func TestComposeAnalysis_IncludesParameterSources(t *testing.T) {
	t.Parallel()

	cfg := config.AnalysisConfig{
		SchemaVersion:      config.AnalysisSchemaVersion,
		Seed:               42,
		EndpointWeights:    map[string]float64{"frontend:GET /health": 1},
		PredicateContract:  "configs/predicate-contract.example.yaml",
		Profiles: []config.Profile{
			{
				Name:               "steady",
				Trials:             100,
				SamplingMode:       config.SamplingModeIndependentReplica,
				FailureProbability: 0.05,
			},
		},
		Sources: config.ParameterSources{
			ConfigSource:       config.ParameterSourceOverride,
			Seed:               config.ParameterSourceDefault,
			Trials:             config.ParameterSourceDefault,
			SamplingMode:       config.ParameterSourceDefault,
			FailureProbability: config.ParameterSourceDefault,
			EndpointWeights:    config.ParameterSourceOverride,
			Journeys:           config.ParameterSourceDefault,
			PredicateContract:  config.ParameterSourceExternal,
			Baselines:          config.ParameterSourceDefault,
			Profiles: map[string]config.ProfileParameterSources{
				"steady": {
					Trials:             config.ParameterSourceOverride,
					SamplingMode:       config.ParameterSourceOverride,
					FailureProbability: config.ParameterSourceOverride,
					FixedKFailures:     config.ParameterSourceDefault,
					EndpointWeights:    config.ParameterSourceOverride,
				},
			},
		},
	}

	rep := ComposeAnalysis(
		artifact.Loaded{
			Metadata: artifact.Metadata{
				Kind: modelcontract.KindSnapshot,
				Contract: modelcontract.SupportedContract{
					Name:    modelcontract.BeringSnapshotV100Name,
					Version: modelcontract.BeringSnapshotV100Version,
				},
			},
			Model: model.ResilienceModel{
				Metadata: model.Metadata{Confidence: 0.8},
			},
			PredicateSource: artifact.ProvenanceExternal,
			WeightsSource:   artifact.ProvenanceSnapshot,
		},
		simulation.AnalysisOutput{
			Profiles: []simulation.ProfileOutput{
				{
					Name:                 "steady",
					Trials:               100,
					Seed:                 2620229120648183554,
					SamplingMode:         config.SamplingModeIndependentReplica,
					FailureProbability:   0.05,
					EndpointAvailability: map[string]float64{"frontend:GET /health": 0.99},
					EndpointWeights:      map[string]float64{"frontend:GET /health": 1},
					WeightedAggregate:    0.99,
					UnweightedAggregate:  0.99,
				},
			},
			CrossProfileWeighted:   0.99,
			CrossProfileUnweighted: 0.99,
		},
		gate.Evaluation{
			Mode: config.ModeWarn,
			ProfileEvaluations: []gate.ProfileEvaluation{
				{
					Profile: "steady",
					Decision: "pass",
					EndpointResults: []gate.EndpointResult{
						{Profile: "steady", EndpointID: "frontend:GET /health", Availability: 0.99, Threshold: 0.98, Status: "pass"},
					},
				},
			},
		},
		cfg,
		0.8,
		time.Unix(0, 0).UTC(),
		0,
	)

	if rep.Parameters == nil {
		t.Fatal("expected parameters section to be present")
	}
	if rep.Parameters.ConfigSource != string(config.ParameterSourceOverride) {
		t.Fatalf("unexpected config source: %+v", rep.Parameters)
	}
	if rep.Parameters.Profiles[0].Trials.Source != string(config.ParameterSourceOverride) {
		t.Fatalf("unexpected trials source: %+v", rep.Parameters.Profiles[0])
	}
	if rep.Parameters.Profiles[0].EndpointWeights.Source != string(config.ParameterSourceOverride) {
		t.Fatalf("unexpected weight source: %+v", rep.Parameters.Profiles[0].EndpointWeights)
	}
	if !rep.Parameters.Calibration.PredicateOverlay.Active || rep.Parameters.Calibration.PredicateOverlay.Source != string(config.ParameterSourceExternal) {
		t.Fatalf("unexpected predicate overlay calibration: %+v", rep.Parameters.Calibration.PredicateOverlay)
	}
	if rep.Parameters.Calibration.HistoricalSignals.Fallback == "" {
		t.Fatalf("expected historical signal fallback marker, got %+v", rep.Parameters.Calibration.HistoricalSignals)
	}
}
