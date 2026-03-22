package simulation

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/faults"
	"github.com/MB3R-Lab/Sheaft/internal/model"
)

func TestRunArtifactProfiles_V100MatchesLegacyWithoutFaultContract(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.0.0.sample.json")
	params := AnalysisParams{
		Seed:           42,
		DefaultWeights: loaded.EndpointWeights,
		Profiles: []ProfileParams{
			{Name: "steady", Trials: 5000, SamplingMode: "independent_replica", FailureProbability: 0.05},
		},
	}

	legacy, err := RunProfiles(loaded.Model, params)
	if err != nil {
		t.Fatalf("RunProfiles failed: %v", err)
	}
	advanced, err := RunArtifactProfiles(loaded, params)
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	if legacy.Profiles[0].WeightedAggregate != advanced.Profiles[0].WeightedAggregate {
		t.Fatalf("expected v1.0.0 advanced runner to preserve weighted aggregate: legacy=%f advanced=%f", legacy.Profiles[0].WeightedAggregate, advanced.Profiles[0].WeightedAggregate)
	}
	if legacy.Profiles[0].EndpointAvailability["gateway:POST /checkout"] != advanced.Profiles[0].EndpointAvailability["gateway:POST /checkout"] {
		t.Fatalf("expected v1.0.0 advanced runner to preserve endpoint availability")
	}
}

func TestRunArtifactProfiles_PlacementFaultReducesReplicasWithoutKillingService(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.1.0.sample.json")
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

	loaded := loadExampleArtifact(t, "snapshot-v1.1.0.sample.json")
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

func TestRunArtifactProfiles_EdgeFailStopBreaksJourneyButNotExplicitPredicate(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.1.0.sample.json")
	contract := faults.Contract{
		SchemaVersion: faults.SchemaVersion,
		Profiles: map[string]faults.Profile{
			"cut": {
				Faults: []faults.Fault{
					{
						Type: faults.TypeEdgeFailStop,
						Selector: faults.Selector{
							EdgeIDs: []string{"checkout|payment|sync|true"},
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
			{Name: "cut", FaultProfile: "cut", Trials: 1000, SamplingMode: "independent_replica", FailureProbability: 0},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	profile := out.Profiles[0]
	if profile.EndpointAvailability["gateway:POST /checkout"] != 0 {
		t.Fatalf("expected journey endpoint to fail under edge cut, got %f", profile.EndpointAvailability["gateway:POST /checkout"])
	}
	if profile.EndpointAvailability["gateway:GET /explicit"] != 1 {
		t.Fatalf("expected explicit predicate endpoint to ignore edge cut, got %f", profile.EndpointAvailability["gateway:GET /explicit"])
	}
}

func TestRunArtifactProfiles_EdgePartialDegradationChangesOutputsAndTimeoutMismatch(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.1.0.sample.json")
	errorRate := 0.35
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
							EdgeIDs: []string{"checkout|payment|sync|true"},
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
			{Name: "brownout", FaultProfile: "brownout", Trials: 6000, SamplingMode: "independent_replica", FailureProbability: 0},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	profile := out.Profiles[0]
	if profile.EndpointAvailability["gateway:POST /checkout"] >= 1 {
		t.Fatalf("expected brownout to reduce journey endpoint availability, got %f", profile.EndpointAvailability["gateway:POST /checkout"])
	}
	if profile.EndpointAvailability["gateway:GET /explicit"] != 1 {
		t.Fatalf("expected explicit predicate endpoint to ignore edge brownout, got %f", profile.EndpointAvailability["gateway:GET /explicit"])
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

	loaded := loadExampleArtifact(t, "snapshot-v1.1.0.sample.json")
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

func TestRunArtifactProfiles_V100AdvancedMetricsUnavailableWhenMetadataMissing(t *testing.T) {
	t.Parallel()

	loaded := loadExampleArtifact(t, "snapshot-v1.0.0.sample.json")
	errorRate := 0.10
	contract := faults.Contract{
		SchemaVersion: faults.SchemaVersion,
		Profiles: map[string]faults.Profile{
			"service-brownout": {
				Faults: []faults.Fault{
					{
						Type:      faults.TypeServicePartialDegradation,
						ErrorRate: &errorRate,
						Selector: faults.Selector{
							ServiceIDs: []string{"checkout"},
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
			{Name: "service-brownout", FaultProfile: "service-brownout", Trials: 1000, SamplingMode: "independent_replica", FailureProbability: 0},
		},
	})
	if err != nil {
		t.Fatalf("RunArtifactProfiles failed: %v", err)
	}

	pathMetric := findPathMetric(out.Profiles[0].Advanced.Paths, []string{"gateway", "checkout", "payment"})
	if pathMetric == nil {
		t.Fatal("expected path diagnostics")
	}
	if pathMetric.MaxAmplificationFactor.Available {
		t.Fatalf("expected amplification to be unavailable on v1.0.0 artifact, got %+v", pathMetric.MaxAmplificationFactor)
	}
	if !strings.Contains(pathMetric.MaxAmplificationFactor.Reason, "retry metadata unavailable") {
		t.Fatalf("expected explicit unavailable reason, got %+v", pathMetric.MaxAmplificationFactor)
	}
	if pathMetric.TimeoutMismatchCount.Available {
		t.Fatalf("expected timeout mismatch metric to be unavailable on v1.0.0 artifact, got %+v", pathMetric.TimeoutMismatchCount)
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
