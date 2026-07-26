package simulation

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/faults"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

func TestRunArtifactProfiles_StochasticConnectivityChainMatchesClosedForm(t *testing.T) {
	t.Parallel()

	loaded := artifact.Loaded{
		Metadata: artifact.Metadata{
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
				{ID: "frontend", Name: "frontend", Replicas: 1},
				{ID: "checkout", Name: "checkout", Replicas: 2},
				{ID: "payment", Name: "payment", Replicas: 1},
			},
			Edges: []model.Edge{
				{ID: "frontend|checkout|sync|true", From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
				{ID: "checkout|payment|sync|true", From: "checkout", To: "payment", Kind: model.EdgeKindSync, Blocking: true},
			},
			Endpoints: []model.Endpoint{
				{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
			},
			Metadata: model.Metadata{
				SourceType:   "test",
				SourceRef:    "fixture",
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

	theta := 0.90
	rho := 0.80
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed: 99,
		Profiles: []ProfileParams{
			{
				Name:         "chain",
				Trials:       200000,
				SamplingMode: config.SamplingModeIndependentReplica,
				Reliability: config.ReliabilityConfig{
					NodeLiveProbability: &theta,
					EdgeLiveProbability: &rho,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	checkoutGroupLive := 1 - (1-theta)*(1-theta)
	expected := theta * checkoutGroupLive * theta * rho * rho
	got := out.Profiles[0].EndpointAvailability["frontend:GET /checkout"]
	if diff := math.Abs(got - expected); diff > 0.01 {
		t.Fatalf("chain availability mismatch: got=%f expected=%f diff=%f", got, expected, diff)
	}
}

func TestRunArtifactProfiles_ReplicatedTargetSaturatesAtEdgeBottleneck(t *testing.T) {
	t.Parallel()

	loaded := artifact.Loaded{
		Metadata: artifact.Metadata{
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
				{ID: "frontend", Name: "frontend", Replicas: 1},
				{ID: "target", Name: "target", Replicas: 12},
			},
			Edges: []model.Edge{
				{ID: "frontend|target|sync|true", From: "frontend", To: "target", Kind: model.EdgeKindSync, Blocking: true},
			},
			Endpoints: []model.Endpoint{
				{ID: "frontend:GET /target", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /target"},
			},
			Metadata: model.Metadata{
				SourceType:   "test",
				SourceRef:    "fixture",
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

	rho := 0.82
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed: 100,
		Profiles: []ProfileParams{
			{
				Name:         "bottleneck",
				Trials:       200000,
				SamplingMode: config.SamplingModeIndependentReplica,
				Reliability: config.ReliabilityConfig{
					Services: map[string]float64{
						"frontend": 1,
						"target":   0.80,
					},
					Edges: map[string]float64{
						"frontend|target|sync|true": rho,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	targetGroupLive := 1 - math.Pow(1-0.80, 12)
	expected := targetGroupLive * rho
	got := out.Profiles[0].EndpointAvailability["frontend:GET /target"]
	if got > rho+0.01 {
		t.Fatalf("replicated target exceeded edge bottleneck: got=%f rho=%f", got, rho)
	}
	if diff := math.Abs(got - expected); diff > 0.01 {
		t.Fatalf("bottleneck availability mismatch: got=%f expected=%f diff=%f", got, expected, diff)
	}
}

func TestRunArtifactProfiles_FixedKReplicaSlots(t *testing.T) {
	t.Parallel()

	loaded := artifact.Loaded{
		Metadata: artifact.Metadata{
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
				{ID: "frontend", Name: "frontend", Replicas: 2},
			},
			Endpoints: []model.Endpoint{
				{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"},
			},
			Metadata: model.Metadata{
				SourceType:   "test",
				SourceRef:    "fixture",
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

	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed: 13,
		Profiles: []ProfileParams{
			{Name: "derived", Trials: 1000, SamplingMode: config.SamplingModeFixedKReplicaSlots, FailureProbability: 0.5},
			{Name: "explicit", Trials: 1000, SamplingMode: config.SamplingModeFixedKReplicaSlots, FixedKFailures: 2},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}
	if out.Profiles[0].FixedKFailures != 1 {
		t.Fatalf("expected derived fixed_k_failures=1, got %d", out.Profiles[0].FixedKFailures)
	}
	if got := out.Profiles[0].EndpointAvailability["frontend:GET /health"]; got != 1 {
		t.Fatalf("expected one failed replica slot to leave service alive, got %f", got)
	}
	if got := out.Profiles[1].EndpointAvailability["frontend:GET /health"]; got != 0 {
		t.Fatalf("expected two failed replica slots to kill service, got %f", got)
	}
}

func TestRunArtifactProfiles_FailureToleranceSweeps(t *testing.T) {
	t.Parallel()

	loaded := artifact.Loaded{
		Metadata: artifact.Metadata{
			Contract: modelcontract.SupportedContract{
				Name:    modelcontract.BeringModelV130Name,
				Version: modelcontract.BeringModelV130Version,
				URI:     modelcontract.BeringModelV130URI,
				Digest:  modelcontract.BeringModelV130Digest,
				Kind:    modelcontract.KindModel,
			},
		},
		Model: model.ResilienceModel{
			Services:  []model.Service{{ID: "frontend", Name: "frontend", Replicas: 2}},
			Endpoints: []model.Endpoint{{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"}},
			Metadata: model.Metadata{
				SourceType: "test", SourceRef: "fixture", DiscoveredAt: "2026-07-01T00:00:00Z", Confidence: 1,
				Schema: model.Schema{Name: modelcontract.BeringModelV130Name, Version: modelcontract.BeringModelV130Version, URI: modelcontract.BeringModelV130URI, Digest: modelcontract.BeringModelV130Digest},
			},
		},
	}
	base := ProfileParams{Name: "steady", Trials: 50000, SamplingMode: config.SamplingModeIndependentReplica, FailureProbability: 0.1}
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed:     31,
		Profiles: []ProfileParams{base},
		Sweeps: []SweepParams{
			{
				Name: "probability", BaseProfile: base,
				Axis:    config.SweepAxis{Type: config.SweepAxisIndependentReplicaFailureProbability, Values: []float64{0, 0.5, 1}},
				Targets: []config.SweepTarget{{EndpointID: "frontend:GET /health", SLO: 0.8}},
			},
			{
				Name: "slots", BaseProfile: base,
				Axis:    config.SweepAxis{Type: config.SweepAxisFailedReplicaSlots, Values: []float64{0, 1, 2}},
				Targets: []config.SweepTarget{{EndpointID: "frontend:GET /health", SLO: 0.5}},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}
	if len(out.Sweeps) != 2 {
		t.Fatalf("expected two sweeps, got %+v", out.Sweeps)
	}
	probability := out.Sweeps[0]
	if got := probability.Points[0].EndpointAvailability["frontend:GET /health"]; got != 1 {
		t.Fatalf("expected explicit zero-failure point availability 1, got %f", got)
	}
	if got := probability.Points[1].EndpointAvailability["frontend:GET /health"]; math.Abs(got-0.75) > 0.02 {
		t.Fatalf("expected two-replica availability near 0.75, got %f", got)
	}
	if boundary := probability.Boundaries[0]; boundary.Status != SweepBoundaryCrossed || boundary.CrossingBracket == nil || boundary.CrossingBracket.LowerAxisValue != 0 || boundary.CrossingBracket.UpperAxisValue != 0.5 {
		t.Fatalf("unexpected probability boundary: %+v", boundary)
	}
	if interval := probability.Points[1].EndpointConfidence["frontend:GET /health"]; interval.ConfidenceLevel != 0.95 || interval.LowerBound >= interval.Estimate || interval.UpperBound <= interval.Estimate {
		t.Fatalf("unexpected probability confidence interval: %+v", interval)
	}
	if boundary := probability.Boundaries[0]; boundary.CertifiedTolerance == nil || boundary.CertifiedTolerance.AxisValue != 0 || boundary.CertificationStatus != SweepCertificationCertified {
		t.Fatalf("unexpected certified probability tolerance: %+v", boundary)
	}
	slots := out.Sweeps[1]
	if got := slots.Points[1].EndpointAvailability["frontend:GET /health"]; got != 1 {
		t.Fatalf("expected one lost slot to leave the service live, got %f", got)
	}
	if slots.Points[2].FailedReplicaSlots == nil || *slots.Points[2].FailedReplicaSlots != 2 || slots.Points[2].FailedReplicaFraction == nil || *slots.Points[2].FailedReplicaFraction != 1 {
		t.Fatalf("unexpected fixed-slot point metadata: %+v", slots.Points[2])
	}
	if boundary := slots.Boundaries[0]; boundary.Status != SweepBoundaryCrossed || boundary.CrossingBracket == nil || boundary.CrossingBracket.LowerAxisValue != 1 || boundary.CrossingBracket.UpperAxisValue != 2 {
		t.Fatalf("unexpected fixed-slot boundary: %+v", boundary)
	}
	if boundary := slots.Boundaries[0]; boundary.CertifiedTolerance == nil || boundary.CertifiedTolerance.AxisValue != 1 {
		t.Fatalf("unexpected certified fixed-slot tolerance: %+v", boundary)
	}
	if len(out.Profiles) != 1 {
		t.Fatalf("sweeps must not be added to profile aggregation: %+v", out.Profiles)
	}
}

func TestRunArtifactProfiles_EndpointSemanticsImmediateIgnoresAsyncAndEventualUsesIt(t *testing.T) {
	t.Parallel()

	loaded := edgeAwareFixtureLoaded([]model.Endpoint{
		{
			ID:                  "frontend:GET /checkout",
			EntryService:        "frontend",
			SuccessPredicateRef: "catalog.checkout.immediate",
			Metadata: &model.EndpointMetadata{
				Semantics: &model.EndpointSemantics{
					PredicateMode:    model.EndpointPredicateModeImmediate,
					MandatoryTargets: []string{"checkout"},
					DependencyModes:  []string{string(model.EdgeKindSync), string(model.EdgeKindAsync)},
				},
			},
		},
		{
			ID:                  "frontend:POST /checkout",
			EntryService:        "frontend",
			SuccessPredicateRef: "catalog.checkout.eventual",
			Metadata: &model.EndpointMetadata{
				Semantics: &model.EndpointSemantics{
					PredicateMode:    model.EndpointPredicateModeEventual,
					MandatoryTargets: []string{"payment"},
					DependencyModes:  []string{string(model.EdgeKindSync), string(model.EdgeKindAsync)},
				},
			},
		},
	})

	rhoAsync := 0.25
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed: 77,
		Profiles: []ProfileParams{
			{
				Name:         "semantic-hints",
				Trials:       100000,
				SamplingMode: config.SamplingModeIndependentReplica,
				Reliability: config.ReliabilityConfig{
					Services: map[string]float64{
						"frontend": 1,
						"checkout": 1,
						"payment":  1,
					},
					Edges: map[string]float64{
						"frontend|checkout|sync|true":  rhoOne,
						"checkout|payment|async|false": rhoAsync,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	gotImmediate := out.Profiles[0].EndpointAvailability["frontend:GET /checkout"]
	if gotImmediate != 1 {
		t.Fatalf("expected immediate endpoint to ignore async dependency mode, got %f", gotImmediate)
	}
	gotEventual := out.Profiles[0].EndpointAvailability["frontend:POST /checkout"]
	if diff := math.Abs(gotEventual - rhoAsync); diff > 0.02 {
		t.Fatalf("expected eventual endpoint to include async edge reliability: got=%f expected~=%f diff=%f", gotEventual, rhoAsync, diff)
	}
}

func TestRunArtifactProfiles_EdgeAwarePredicateContractUsesEdgesButLegacyPredicateDoesNot(t *testing.T) {
	t.Parallel()

	loaded := edgeAwareFixtureLoaded([]model.Endpoint{
		{ID: "frontend:POST /checkout", EntryService: "frontend", SuccessPredicateRef: "catalog.checkout.eventual"},
		{ID: "frontend:GET /legacy", EntryService: "frontend", SuccessPredicateRef: "catalog.checkout.legacy"},
	})
	contract := faults.Contract{
		SchemaVersion: faults.SchemaVersion,
		Profiles: map[string]faults.Profile{
			"async-cut": {
				Faults: []faults.Fault{
					{
						Type: faults.TypeEdgeFailStop,
						Selector: faults.Selector{
							EdgeIDs: []string{"checkout|payment|async|false"},
						},
					},
				},
			},
		},
	}

	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed:          88,
		FaultContract: &contract,
		PredicateSet: map[string]predicates.Definition{
			"catalog.checkout.eventual": {
				Type:             predicates.TypeEdgeAware,
				EntryService:     "frontend",
				MandatoryTargets: []string{"payment"},
				EdgeModes:        []string{predicates.EdgeModeSync, predicates.EdgeModeAsync},
			},
			"catalog.checkout.legacy": {
				Type:     predicates.TypeAllOf,
				Services: []string{"frontend", "checkout", "payment"},
			},
		},
		Profiles: []ProfileParams{
			{
				Name:         "async-cut",
				FaultProfile: "async-cut",
				Trials:       1000,
				SamplingMode: config.SamplingModeIndependentReplica,
				Reliability: config.ReliabilityConfig{
					Services: map[string]float64{
						"frontend": 1,
						"checkout": 1,
						"payment":  1,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	got := out.Profiles[0].EndpointAvailability
	if got["frontend:POST /checkout"] != 0 {
		t.Fatalf("expected edge-aware predicate to fail under async edge cut, got %f", got["frontend:POST /checkout"])
	}
	if got["frontend:GET /legacy"] != 1 {
		t.Fatalf("expected legacy service predicate to ignore async edge cut, got %f", got["frontend:GET /legacy"])
	}
}

func TestRunArtifactProfiles_PlacementFaultReducesReplicasWithoutKillingService(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.3.0.sample.json")
	contract := faults.Contract{
		SchemaVersion: faults.SchemaVersion,
		Profiles: map[string]faults.Profile{
			"az-outage": {
				Faults: []faults.Fault{
					{
						Type: faults.TypeCorrelatedFailureDomain,
						Selector: faults.Selector{
							PlacementLabels: map[string]string{"az": "us-east-1a"},
						},
						OnlyFailureEligible: true,
					},
				},
			},
		},
	}
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed:           42,
		DefaultWeights: loaded.EndpointWeights,
		FaultContract:  &contract,
		Profiles: []ProfileParams{
			{Name: "az-outage", FaultProfile: "az-outage", Trials: 2000, SamplingMode: "independent_replica", FailureProbability: 0},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	profile := out.Profiles[0]
	if profile.EndpointAvailability["gateway:POST /checkout"] != 1 {
		t.Fatalf("expected placement outage to preserve journey endpoint because one checkout bucket survives, got %f", profile.EndpointAvailability["gateway:POST /checkout"])
	}
	if profile.Advanced == nil || profile.Advanced.BlastRadius == nil {
		t.Fatal("expected blast radius diagnostics")
	}
	if profile.Advanced.BlastRadius.ServiceCount.Value != 1 {
		t.Fatalf("expected one impacted service, got %+v", profile.Advanced.BlastRadius)
	}
}

func TestRunArtifactProfiles_SharedResourceFaultWorks(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.3.0.sample.json")
	contract := faults.Contract{
		SchemaVersion: faults.SchemaVersion,
		Profiles: map[string]faults.Profile{
			"shared-db": {
				Faults: []faults.Fault{
					{
						Type: faults.TypeCorrelatedFailureDomain,
						Selector: faults.Selector{
							SharedResourceRefs: []string{"db:payments"},
						},
					},
				},
			},
		},
	}
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed:           42,
		DefaultWeights: loaded.EndpointWeights,
		FaultContract:  &contract,
		Profiles: []ProfileParams{
			{Name: "shared-db", FaultProfile: "shared-db", Trials: 1000, SamplingMode: "independent_replica", FailureProbability: 0},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	profile := out.Profiles[0]
	if profile.Advanced.BlastRadius.ServiceCount.Value != 2 {
		t.Fatalf("expected two services in shared-resource blast radius, got %+v", profile.Advanced.BlastRadius)
	}
	if profile.EndpointAvailability["gateway:POST /checkout"] != 0 || profile.EndpointAvailability["gateway:GET /explicit"] != 0 {
		t.Fatalf("expected shared-resource outage to fail both endpoints, got %+v", profile.EndpointAvailability)
	}
}

func TestRunArtifactProfiles_EdgeFailStopDistinguishesImmediateAndEventualSemantics(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.3.0.sample.json")
	one := 1.0
	contract := faults.Contract{
		SchemaVersion: faults.SchemaVersion,
		Profiles: map[string]faults.Profile{
			"cut": {
				Faults: []faults.Fault{
					{
						Type: faults.TypeEdgeFailStop,
						Selector: faults.Selector{
							EdgeIDs: []string{"checkout|payment|sync|true|protocol=http|operation=POST|route=%2Fcharge"},
						},
					},
				},
			},
		},
	}
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed:           42,
		DefaultWeights: loaded.EndpointWeights,
		FaultContract:  &contract,
		Profiles: []ProfileParams{
			{
				Name:         "cut",
				FaultProfile: "cut",
				Trials:       1000,
				SamplingMode: "independent_replica",
				Reliability: config.ReliabilityConfig{
					NodeLiveProbability: &one,
					EdgeLiveProbability: &one,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	profile := out.Profiles[0]
	if profile.EndpointAvailability["gateway:POST /checkout"] < 0.98 {
		t.Fatalf("expected immediate endpoint to ignore downstream payment edge cut, got %f", profile.EndpointAvailability["gateway:POST /checkout"])
	}
	if profile.EndpointAvailability["gateway:GET /explicit"] != 0 {
		t.Fatalf("expected eventual endpoint to fail under payment edge cut, got %f", profile.EndpointAvailability["gateway:GET /explicit"])
	}
}

func TestRunArtifactProfiles_EdgePartialDegradationChangesOutputsAndTimeoutMismatch(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.3.0.sample.json")
	errorRate := 0.35
	one := 1.0
	contract := faults.Contract{
		SchemaVersion: faults.SchemaVersion,
		Profiles: map[string]faults.Profile{
			"brownout": {
				Faults: []faults.Fault{
					{
						Type:      faults.TypeEdgePartialDegradation,
						ErrorRate: &errorRate,
						LatencyMS: &model.LatencySummary{P90: 5000},
						Selector: faults.Selector{
							EdgeIDs: []string{"checkout|payment|sync|true|protocol=http|operation=POST|route=%2Fcharge"},
						},
					},
				},
			},
		},
	}
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed:           42,
		DefaultWeights: loaded.EndpointWeights,
		FaultContract:  &contract,
		Profiles: []ProfileParams{
			{
				Name:         "brownout",
				FaultProfile: "brownout",
				Trials:       6000,
				SamplingMode: "independent_replica",
				Reliability: config.ReliabilityConfig{
					NodeLiveProbability: &one,
					EdgeLiveProbability: &one,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	profile := out.Profiles[0]
	if profile.EndpointAvailability["gateway:POST /checkout"] < 0.98 {
		t.Fatalf("expected immediate endpoint to ignore downstream payment edge brownout, got %f", profile.EndpointAvailability["gateway:POST /checkout"])
	}
	if profile.EndpointAvailability["gateway:GET /explicit"] >= profile.EndpointAvailability["gateway:POST /checkout"] {
		t.Fatalf("expected eventual endpoint to be reduced by payment edge brownout, got %f", profile.EndpointAvailability["gateway:GET /explicit"])
	}
	pathMetric := findPathMetric(profile.Advanced.Paths, []string{"gateway", "checkout", "payment"})
	if pathMetric == nil || !pathMetric.TimeoutMismatchCount.Available {
		t.Fatalf("expected timeout mismatch metric to be available, got %+v", pathMetric)
	}
	if pathMetric.TimeoutMismatchCount.Value == 0 {
		t.Fatalf("expected timeout mismatch count to increase under latency injection, got %+v", pathMetric.TimeoutMismatchCount)
	}
}

func TestRunArtifactProfiles_RetryAmplificationExposed(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.3.0.sample.json")
	out, err := RunArtifactProfiles(loaded, AnalysisParams{
		Seed:           42,
		DefaultWeights: loaded.EndpointWeights,
		Profiles: []ProfileParams{
			{Name: "steady", Trials: 1000, SamplingMode: "independent_replica", FailureProbability: 0},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	pathMetric := findPathMetric(out.Profiles[0].Advanced.Paths, []string{"gateway", "checkout", "payment"})
	if pathMetric == nil || !pathMetric.MaxAmplificationFactor.Available {
		t.Fatalf("expected amplification metric to be available, got %+v", pathMetric)
	}
	if pathMetric.MaxAmplificationFactor.Value <= 1 {
		t.Fatalf("expected amplification > 1, got %+v", pathMetric.MaxAmplificationFactor)
	}
}

func loadExampleArtifact(t *testing.T, name string) artifact.Loaded {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "examples", "outputs", name)
	loaded, err := artifact.Load(path)
	if err != nil {
		t.Fatalf("artifact.Load(%s): %v", name, err)
	}
	return loaded
}

const rhoOne = 1.0

func edgeAwareFixtureLoaded(endpoints []model.Endpoint) artifact.Loaded {
	return artifact.Loaded{
		Metadata: artifact.Metadata{
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
				{ID: "frontend", Name: "frontend", Replicas: 1},
				{ID: "checkout", Name: "checkout", Replicas: 1},
				{ID: "payment", Name: "payment", Replicas: 1},
			},
			Edges: []model.Edge{
				{ID: "frontend|checkout|sync|true", From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
				{ID: "checkout|payment|async|false", From: "checkout", To: "payment", Kind: model.EdgeKindAsync, Blocking: false},
			},
			Endpoints: endpoints,
			Metadata: model.Metadata{
				SourceType:   "test",
				SourceRef:    "fixture",
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
}
