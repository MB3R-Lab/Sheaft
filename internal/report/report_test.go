package report

import (
	"math"
	"os"
	"path/filepath"
	"strings"
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
		Sweeps: []simulation.SweepOutput{{
			Name: "checkout-boundary", AxisType: config.SweepAxisIndependentReplicaFailureProbability, Fingerprint: "sha256:same",
			Boundaries: []simulation.SweepBoundary{{EndpointID: "frontend:GET /checkout", CertifiedTolerance: &simulation.BoundaryPoint{AxisValue: 0.1}}},
		}},
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
		Sweeps: []simulation.SweepOutput{{
			Name: "checkout-boundary", AxisType: config.SweepAxisIndependentReplicaFailureProbability, Fingerprint: "sha256:same",
			Boundaries: []simulation.SweepBoundary{{EndpointID: "frontend:GET /checkout", CertifiedTolerance: &simulation.BoundaryPoint{AxisValue: 0.2}}},
		}},
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
	if len(diff.SweepBoundaries) != 1 || diff.SweepBoundaries[0].Tolerance == nil || diff.SweepBoundaries[0].Tolerance.Signed >= 0 {
		t.Fatalf("expected comparable boundary regression, got %+v", diff.SweepBoundaries)
	}
}

func TestCompare_OlderReportWithoutSweepsIsNonComparable(t *testing.T) {
	t.Parallel()

	current := Report{Sweeps: []simulation.SweepOutput{{
		Name: "checkout-boundary", AxisType: config.SweepAxisIndependentReplicaFailureProbability, Fingerprint: "sha256:current",
		Boundaries: []simulation.SweepBoundary{{EndpointID: "frontend:GET /checkout", CertifiedTolerance: &simulation.BoundaryPoint{AxisValue: 0.1}}},
	}}}

	diff := Compare(current, Report{}, "pre-v1.2")
	if len(diff.SweepBoundaries) != 1 {
		t.Fatalf("expected one non-comparable boundary, got %+v", diff.SweepBoundaries)
	}
	boundary := diff.SweepBoundaries[0]
	if boundary.Status != "non_comparable" || boundary.Reason != "reference report does not contain the sweep" || boundary.Tolerance != nil {
		t.Fatalf("expected an explicit non-comparable result, got %+v", boundary)
	}
}

func TestComposeAnalysis_IncludesParameterSources(t *testing.T) {
	t.Parallel()

	cfg := config.AnalysisConfig{
		SchemaVersion:     config.AnalysisSchemaVersion,
		Seed:              42,
		EndpointWeights:   map[string]float64{"frontend:GET /health": 1},
		PredicateContract: "configs/predicate-contract.example.yaml",
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
					Name:    modelcontract.BeringSnapshotV130Name,
					Version: modelcontract.BeringSnapshotV130Version,
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
			Sweeps: []simulation.SweepOutput{
				{
					Name: "health-boundary", BaseProfile: "steady", AxisType: config.SweepAxisIndependentReplicaFailureProbability, Trials: 100,
					Boundaries: []simulation.SweepBoundary{{EndpointID: "frontend:GET /health", SLO: 0.98, Status: simulation.SweepBoundaryCrossed}},
				},
			},
		},
		gate.Evaluation{
			Mode: config.ModeWarn,
			Reasons: []gate.DecisionReason{
				{
					ID:      "gate_pass",
					Scope:   "gate",
					Status:  "pass",
					Message: "all evaluated endpoints, assertions, and aggregates satisfy the configured gate",
				},
			},
			ProfileEvaluations: []gate.ProfileEvaluation{
				{
					Profile:  "steady",
					Decision: "pass",
					EndpointResults: []gate.EndpointResult{
						{Profile: "steady", EndpointID: "frontend:GET /health", Availability: 0.99, Threshold: 0.98, Status: "pass"},
					},
					Reasons: []gate.DecisionReason{
						{
							ID:      "gate_pass",
							Scope:   "gate",
							Status:  "pass",
							Message: "all evaluated endpoints, assertions, and aggregates satisfy the configured gate",
						},
					},
				},
			},
		},
		cfg,
		config.ContractPolicyDecision{
			Status: config.ContractPolicyStatusDeprecated,
			Action: string(config.ContractPolicyActionWarn),
		},
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
	if rep.ContractPolicy == nil || rep.ContractPolicy.Status != config.ContractPolicyStatusDeprecated || rep.ContractPolicy.Action != string(config.ContractPolicyActionWarn) {
		t.Fatalf("expected contract policy status in report, got %+v", rep.ContractPolicy)
	}
	if len(rep.PolicyEvaluation.Reasons) != 1 || rep.PolicyEvaluation.Reasons[0].ID != "gate_pass" {
		t.Fatalf("expected policy evaluation why reasons in report, got %+v", rep.PolicyEvaluation.Reasons)
	}
	if len(rep.Sweeps) != 1 || rep.Sweeps[0].Name != "health-boundary" {
		t.Fatalf("expected sweep output in report, got %+v", rep.Sweeps)
	}
}

func TestWriteSummaryMarkdown_IncludesFailureToleranceBoundary(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "summary.md")
	rep := Report{
		PolicyEvaluation: PolicyEvaluation{Decision: "report", Mode: "report"},
		Sweeps: []simulation.SweepOutput{
			{
				Name: "checkout-boundary", BaseProfile: "steady", AxisType: config.SweepAxisIndependentReplicaFailureProbability, Trials: 1000,
				Boundaries: []simulation.SweepBoundary{
					{
						EndpointID: "frontend:GET /checkout", SLO: 0.99, Status: simulation.SweepBoundaryCrossed,
						LastMeetingSLO:    &simulation.BoundaryPoint{AxisValue: 0.1, Availability: 0.995},
						FirstViolatingSLO: &simulation.BoundaryPoint{AxisValue: 0.2, Availability: 0.97},
						CrossingBracket:   &simulation.CrossingBracket{LowerAxisValue: 0.1, UpperAxisValue: 0.2},
					},
				},
			},
		},
	}
	if err := WriteSummaryMarkdown(path, rep); err != nil {
		t.Fatalf("WriteSummaryMarkdown failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	text := string(raw)
	for _, want := range []string{"Failure-tolerance sweeps", "checkout-boundary", "first-violating=`0.2000`", "bracket=`[0.1000, 0.2000]`"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}
