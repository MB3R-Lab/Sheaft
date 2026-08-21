package simulation

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

func TestRun_DeterministicWithSameSeed(t *testing.T) {
	t.Parallel()

	mdl := model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkout", Name: "checkout", Replicas: 1},
		},
		Edges: []model.Edge{
			{ID: "frontend|checkout|sync|true", From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
		},
		Metadata: model.Metadata{
			SourceType:   "test",
			SourceRef:    "fixture",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.7,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
	params := Params{
		Trials:             2000,
		Seed:               7,
		FailureProbability: 0.1,
	}

	outA, err := Run(mdl, params)
	if err != nil {
		t.Fatalf("Run A failed: %v", err)
	}
	outB, err := Run(mdl, params)
	if err != nil {
		t.Fatalf("Run B failed: %v", err)
	}

	avA := outA.EndpointAvailability["frontend:GET /checkout"]
	avB := outB.EndpointAvailability["frontend:GET /checkout"]
	if avA != avB {
		t.Fatalf("expected deterministic availability, got A=%v B=%v", avA, avB)
	}
}

func TestRun_UsesJourneyAnyPathSemantics(t *testing.T) {
	t.Parallel()

	mdl := model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkoutA", Name: "checkoutA", Replicas: 1},
			{ID: "checkoutB", Name: "checkoutB", Replicas: 1},
		},
		Edges: []model.Edge{
			{ID: "frontend|checkoutA|sync|true", From: "frontend", To: "checkoutA", Kind: model.EdgeKindSync, Blocking: true},
			{ID: "frontend|checkoutB|sync|true", From: "frontend", To: "checkoutB", Kind: model.EdgeKindSync, Blocking: true},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
		},
		Metadata: model.Metadata{
			SourceType:   "test",
			SourceRef:    "fixture",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.7,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
	params := Params{
		Trials:             200000,
		Seed:               42,
		FailureProbability: 0.5,
	}

	out, err := Run(mdl, params)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	got := out.EndpointAvailability["frontend:GET /checkout"]
	expected := 0.375 // P(frontend alive) * P(checkoutA alive OR checkoutB alive) = 0.5 * 0.75
	if math.Abs(got-expected) > 0.02 {
		t.Fatalf("journey semantics mismatch: got=%f expected~=%f", got, expected)
	}
}

func TestRun_UsesManualJourneyOverrides(t *testing.T) {
	t.Parallel()

	mdl := model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkoutA", Name: "checkoutA", Replicas: 1},
			{ID: "checkoutB", Name: "checkoutB", Replicas: 1},
		},
		Edges: []model.Edge{
			{ID: "frontend|checkoutA|sync|true", From: "frontend", To: "checkoutA", Kind: model.EdgeKindSync, Blocking: true},
			{ID: "frontend|checkoutB|sync|true", From: "frontend", To: "checkoutB", Kind: model.EdgeKindSync, Blocking: true},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
		},
		Metadata: model.Metadata{
			SourceType:   "test",
			SourceRef:    "fixture",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.7,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
	params := Params{
		Trials:             200000,
		Seed:               42,
		FailureProbability: 0.5,
		JourneyOverrides: map[string][][]string{
			"frontend:GET /checkout": {
				{"frontend", "checkoutA"},
			},
		},
	}

	out, err := Run(mdl, params)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	got := out.EndpointAvailability["frontend:GET /checkout"]
	expected := 0.25 // P(frontend alive AND checkoutA alive) = 0.5 * 0.5
	if math.Abs(got-expected) > 0.02 {
		t.Fatalf("manual override semantics mismatch: got=%f expected~=%f", got, expected)
	}
}

func TestRun_FailsOnUnknownJourneyOverrideEndpoint(t *testing.T) {
	t.Parallel()

	mdl := model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
		},
		Edges: []model.Edge{},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"},
		},
		Metadata: model.Metadata{
			SourceType:   "test",
			SourceRef:    "fixture",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.7,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
	params := Params{
		Trials:             1000,
		Seed:               42,
		FailureProbability: 0.1,
		JourneyOverrides: map[string][][]string{
			"frontend:GET /checkout": {
				{"frontend"},
			},
		},
	}

	_, err := Run(mdl, params)
	if err == nil {
		t.Fatal("expected error for unknown override endpoint")
	}
	if !strings.Contains(err.Error(), "endpoint not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProfiles_MultiProfileDeterministic(t *testing.T) {
	t.Parallel()

	mdl := testModel(
		[]model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkoutA", Name: "checkoutA", Replicas: 1},
			{ID: "checkoutB", Name: "checkoutB", Replicas: 1},
		},
		[]model.Edge{
			{From: "frontend", To: "checkoutA", Kind: model.EdgeKindSync, Blocking: true},
			{From: "frontend", To: "checkoutB", Kind: model.EdgeKindSync, Blocking: true},
		},
		[]model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
		},
	)
	params := AnalysisParams{
		Seed: 99,
		Profiles: []ProfileParams{
			{Name: "steady", Trials: 100000, SamplingMode: "independent_replica", FailureProbability: 0.1},
			{Name: "stress", Trials: 100000, SamplingMode: "independent_replica", FailureProbability: 0.5},
		},
	}

	outA, err := RunProfiles(mdl, params)
	if err != nil {
		t.Fatalf("RunProfiles A failed: %v", err)
	}
	outB, err := RunProfiles(mdl, params)
	if err != nil {
		t.Fatalf("RunProfiles B failed: %v", err)
	}

	if len(outA.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(outA.Profiles))
	}
	for i := range outA.Profiles {
		if outA.Profiles[i].WeightedAggregate != outB.Profiles[i].WeightedAggregate {
			t.Fatalf("expected deterministic profile aggregate for %s", outA.Profiles[i].Name)
		}
	}
	if !(outA.Profiles[0].WeightedAggregate > outA.Profiles[1].WeightedAggregate) {
		t.Fatalf("expected steady profile to outperform stress profile: %+v", outA.Profiles)
	}
}

func TestRunProfiles_SamplingModes(t *testing.T) {
	t.Parallel()

	replicaModel := testModel(
		[]model.Service{{ID: "frontend", Name: "frontend", Replicas: 2}},
		nil,
		[]model.Endpoint{{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"}},
	)

	replicaOut, err := RunProfiles(replicaModel, AnalysisParams{
		Seed: 7,
		Profiles: []ProfileParams{
			{Name: "replica", Trials: 200000, SamplingMode: "independent_replica", FailureProbability: 0.5},
			{Name: "service", Trials: 200000, SamplingMode: "independent_service", FailureProbability: 0.5},
		},
	})
	if err != nil {
		t.Fatalf("RunProfiles failed: %v", err)
	}
	if math.Abs(replicaOut.Profiles[0].WeightedAggregate-0.75) > 0.02 {
		t.Fatalf("expected replica availability ~0.75, got %f", replicaOut.Profiles[0].WeightedAggregate)
	}
	if math.Abs(replicaOut.Profiles[1].WeightedAggregate-0.5) > 0.02 {
		t.Fatalf("expected service availability ~0.5, got %f", replicaOut.Profiles[1].WeightedAggregate)
	}

	fixedKModel := testModel(
		[]model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkout", Name: "checkout", Replicas: 1},
			{ID: "payment", Name: "payment", Replicas: 1},
		},
		[]model.Edge{
			{From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
			{From: "checkout", To: "payment", Kind: model.EdgeKindSync, Blocking: true},
		},
		[]model.Endpoint{{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"}},
	)
	fixedKOut, err := RunProfiles(fixedKModel, AnalysisParams{
		Seed: 11,
		Profiles: []ProfileParams{
			{Name: "fixed-k", Trials: 1000, SamplingMode: "fixed_k_service_set", FixedKFailures: 1},
		},
	})
	if err != nil {
		t.Fatalf("fixed-k RunProfiles failed: %v", err)
	}
	if fixedKOut.Profiles[0].WeightedAggregate != 0 {
		t.Fatalf("expected fixed-k path availability to be 0, got %f", fixedKOut.Profiles[0].WeightedAggregate)
	}

	replicaSlotModel := testModel(
		[]model.Service{{ID: "frontend", Name: "frontend", Replicas: 2}},
		nil,
		[]model.Endpoint{{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"}},
	)
	replicaSlotOut, err := RunProfiles(replicaSlotModel, AnalysisParams{
		Seed: 13,
		Profiles: []ProfileParams{
			{Name: "fixed-replica-slots", Trials: 1000, SamplingMode: "fixed_k_replica_slots", FailureProbability: 0.5},
			{Name: "fixed-replica-slots-explicit", Trials: 1000, SamplingMode: "fixed_k_replica_slots", FixedKFailures: 2},
		},
	})
	if err != nil {
		t.Fatalf("fixed replica slots RunProfiles failed: %v", err)
	}
	if replicaSlotOut.Profiles[0].FixedKFailures != 1 {
		t.Fatalf("expected derived fixed_k_failures=1, got %d", replicaSlotOut.Profiles[0].FixedKFailures)
	}
	if replicaSlotOut.Profiles[0].WeightedAggregate != 1 {
		t.Fatalf("expected one failed replica slot to leave service alive, got %f", replicaSlotOut.Profiles[0].WeightedAggregate)
	}
	if replicaSlotOut.Profiles[1].WeightedAggregate != 0 {
		t.Fatalf("expected two failed replica slots to kill service, got %f", replicaSlotOut.Profiles[1].WeightedAggregate)
	}
}

func TestRunProfiles_ExplicitlyIneligibleServiceIsExcludedFromBaselineSampling(t *testing.T) {
	t.Parallel()

	ineligible := false
	mdl := testModel(
		[]model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1, Metadata: &model.ServiceMetadata{FailureEligible: &ineligible}},
			{ID: "worker", Name: "worker", Replicas: 2},
		},
		nil,
		[]model.Endpoint{{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"}},
	)

	out, err := RunProfiles(mdl, AnalysisParams{
		Seed: 17,
		PredicateSet: map[string]predicates.Definition{
			"frontend:GET /health": {Type: predicates.TypeAllOf, Services: []string{"frontend"}},
		},
		Profiles: []ProfileParams{
			{Name: "independent-replica", Trials: 20, SamplingMode: config.SamplingModeIndependentReplica, FailureProbability: 1},
			{Name: "independent-service", Trials: 20, SamplingMode: config.SamplingModeIndependentService, FailureProbability: 1},
			{Name: "fixed-service", Trials: 20, SamplingMode: config.SamplingModeFixedKServiceSet, FixedKFailures: 1},
			{Name: "fixed-slots", Trials: 20, SamplingMode: config.SamplingModeFixedKReplicaSlots, FailureProbability: 0.5},
		},
	})
	if err != nil {
		t.Fatalf("RunProfiles failed: %v", err)
	}
	for _, profile := range out.Profiles {
		if profile.WeightedAggregate != 1 {
			t.Fatalf("expected excluded frontend to remain live in %s, got %f", profile.Name, profile.WeightedAggregate)
		}
	}
	if out.Profiles[3].FixedKFailures != 1 {
		t.Fatalf("expected derived k from two eligible worker slots, got %d", out.Profiles[3].FixedKFailures)
	}
}

func TestRunProfiles_WeightedAggregate(t *testing.T) {
	t.Parallel()

	mdl := testModel(
		[]model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkoutA", Name: "checkoutA", Replicas: 1},
			{ID: "checkoutB", Name: "checkoutB", Replicas: 1},
		},
		[]model.Edge{
			{From: "frontend", To: "checkoutA", Kind: model.EdgeKindSync, Blocking: true},
			{From: "frontend", To: "checkoutB", Kind: model.EdgeKindSync, Blocking: true},
		},
		[]model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
			{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"},
		},
	)

	out, err := RunProfiles(mdl, AnalysisParams{
		Seed: 42,
		PredicateSet: map[string]predicates.Definition{
			"frontend:GET /checkout": journeysToPredicate([][]string{
				{"frontend", "checkoutA"},
				{"frontend", "checkoutB"},
			}),
			"frontend:GET /health": {
				Type:     predicates.TypeAllOf,
				Services: []string{"frontend"},
			},
		},
		Profiles: []ProfileParams{
			{
				Name:               "weighted",
				Trials:             200000,
				SamplingMode:       "independent_replica",
				FailureProbability: 0.5,
				EndpointWeights: map[string]float64{
					"frontend:GET /checkout": 0.1,
					"frontend:GET /health":   0.9,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunProfiles failed: %v", err)
	}
	got := out.Profiles[0]
	if math.Abs(got.UnweightedAggregate-0.4375) > 0.03 {
		t.Fatalf("expected unweighted aggregate ~0.4375, got %f", got.UnweightedAggregate)
	}
	if math.Abs(got.WeightedAggregate-0.4875) > 0.03 {
		t.Fatalf("expected weighted aggregate ~0.4875, got %f", got.WeightedAggregate)
	}
}

func TestRunProfiles_PredicateEvaluation(t *testing.T) {
	t.Parallel()

	mdl := testModel(
		[]model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkout", Name: "checkout", Replicas: 1},
			{ID: "searchA", Name: "searchA", Replicas: 1},
			{ID: "searchB", Name: "searchB", Replicas: 1},
			{ID: "cacheA", Name: "cacheA", Replicas: 1},
			{ID: "cacheB", Name: "cacheB", Replicas: 1},
			{ID: "cacheC", Name: "cacheC", Replicas: 1},
		},
		nil,
		[]model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "checkout_success"},
			{ID: "frontend:GET /search", EntryService: "frontend", SuccessPredicateRef: "search_success"},
			{ID: "frontend:GET /cache", EntryService: "frontend", SuccessPredicateRef: "cache_quorum"},
		},
	)

	out, err := RunProfiles(mdl, AnalysisParams{
		Seed: 5,
		PredicateSet: map[string]predicates.Definition{
			"checkout_success": {
				Type:     predicates.TypeAllOf,
				Services: []string{"frontend", "checkout"},
			},
			"search_success": {
				Type:     predicates.TypeAnyOf,
				Services: []string{"searchA", "searchB"},
			},
			"cache_quorum": {
				Type:     predicates.TypeKOfN,
				K:        2,
				Services: []string{"cacheA", "cacheB", "cacheC"},
			},
		},
		Profiles: []ProfileParams{
			{Name: "predicates", Trials: 200000, SamplingMode: "independent_service", FailureProbability: 0.5},
		},
	})
	if err != nil {
		t.Fatalf("RunProfiles failed: %v", err)
	}

	got := out.Profiles[0].EndpointAvailability
	if math.Abs(got["frontend:GET /checkout"]-0.25) > 0.02 {
		t.Fatalf("expected all_of availability ~0.25, got %f", got["frontend:GET /checkout"])
	}
	if math.Abs(got["frontend:GET /search"]-0.75) > 0.02 {
		t.Fatalf("expected any_of availability ~0.75, got %f", got["frontend:GET /search"])
	}
	if math.Abs(got["frontend:GET /cache"]-0.5) > 0.02 {
		t.Fatalf("expected k_of_n availability ~0.5, got %f", got["frontend:GET /cache"])
	}
}

func TestRunProfiles_RejectsEdgeAwarePredicate(t *testing.T) {
	t.Parallel()

	mdl := testModel(
		[]model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkout", Name: "checkout", Replicas: 1},
		},
		[]model.Edge{
			{From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
		},
		[]model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "catalog.checkout.edge"},
		},
	)
	_, err := RunProfiles(mdl, AnalysisParams{
		Seed: 3,
		PredicateSet: map[string]predicates.Definition{
			"catalog.checkout.edge": {
				Type:             predicates.TypeEdgeAware,
				EntryService:     "frontend",
				MandatoryTargets: []string{"checkout"},
				EdgeModes:        []string{predicates.EdgeModeSync},
			},
		},
		Profiles: []ProfileParams{
			{Name: "legacy", Trials: 10, SamplingMode: "independent_replica", FailureProbability: 0},
		},
	})
	if err == nil {
		t.Fatal("expected edge-aware predicate to require artifact path-aware analysis")
	}
	if !strings.Contains(err.Error(), "edge-aware predicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testModel(services []model.Service, edges []model.Edge, endpoints []model.Endpoint) model.ResilienceModel {
	edges = append([]model.Edge(nil), edges...)
	for i := range edges {
		if strings.TrimSpace(edges[i].ID) == "" {
			edges[i].ID = fmt.Sprintf("%s|%s|%s|%t", edges[i].From, edges[i].To, edges[i].Kind, edges[i].Blocking)
		}
	}
	return model.ResilienceModel{
		Services:  services,
		Edges:     edges,
		Endpoints: endpoints,
		Metadata: model.Metadata{
			SourceType:   "test",
			SourceRef:    "fixture",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.7,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
}
