package simulation

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

func importanceFixture() artifact.Loaded {
	loaded := edgeAwareFixtureLoaded([]model.Endpoint{{ID: "checkout", EntryService: "frontend", SuccessPredicateRef: "checkout"}})
	loaded.Model.Services = loaded.Model.Services[:2]
	loaded.Model.Edges = loaded.Model.Edges[:1]
	return loaded
}

func importanceRequest(profile ProfileParams, endpoints ...string) ImportanceParams {
	spec := config.Importance{Name: "criticality", Profile: profile.Name, Method: config.ImportanceMethodBirnbaum,
		ConfidenceLevel: 0.95, Components: config.ImportanceComponents{Kinds: []string{"service", "edge"}}}
	for _, id := range endpoints {
		spec.Targets = append(spec.Targets, config.ImportanceTarget{EndpointID: id})
	}
	return ImportanceParams{Spec: spec, BaseProfile: profile}
}

func runTestImportance(t *testing.T, loaded artifact.Loaded, request ImportanceParams) ImportanceOutput {
	t.Helper()
	out, err := RunArtifactProfiles(loaded, AnalysisParams{Seed: 42, Importance: []ImportanceParams{request}})
	if err != nil {
		t.Fatal(err)
	}
	return out.Importance[0]
}

func importanceComponent(t *testing.T, output ImportanceOutput, endpoint, kind, id string) ComponentImportance {
	t.Helper()
	for _, result := range output.Endpoints {
		if result.EndpointID != endpoint {
			continue
		}
		for _, component := range result.Components {
			if component.Kind == kind && component.ID == id {
				return component
			}
		}
	}
	t.Fatalf("missing %s/%s for %s", kind, id, endpoint)
	return ComponentImportance{}
}

func TestBirnbaumSeriesMatchesDerivativeAndReplicaSemantics(t *testing.T) {
	t.Parallel()
	loaded := importanceFixture()
	loaded.Model.Services[0].Replicas = 2
	edge := loaded.Model.Edges[0].ID
	profile := ProfileParams{Name: "steady", Trials: 16000, SamplingMode: config.SamplingModeIndependentReplica,
		Reliability: config.ReliabilityConfig{Services: map[string]float64{"frontend": 0.8, "checkout": 0.9}, Edges: map[string]float64{edge: 0.7}}}
	request := importanceRequest(profile, "checkout")
	out := runTestImportance(t, loaded, request)
	for _, expected := range []struct {
		kind, id                string
		importance, probability float64
	}{
		{"service", "frontend", 0.9 * 0.7, 0.96}, {"service", "checkout", 0.96 * 0.7, 0.9}, {"edge", edge, 0.96 * 0.9, 0.7},
	} {
		got := importanceComponent(t, out, "checkout", expected.kind, expected.id)
		if math.Abs(got.BirnbaumImportance-expected.importance) > 0.016 {
			t.Fatalf("%s importance=%f want %f", expected.id, got.BirnbaumImportance, expected.importance)
		}
		if math.Abs(got.LiveProbability-expected.probability) > 1e-12 {
			t.Fatalf("%s probability=%f want %f", expected.id, got.LiveProbability, expected.probability)
		}
		if math.Abs(got.AvailabilityIfUp-got.AvailabilityIfDown-got.BirnbaumImportance) > 1e-12 {
			t.Fatal("paired estimate must equal the up/down difference")
		}
		if got.Confidence.LowerBound > got.BirnbaumImportance || got.Confidence.UpperBound < got.BirnbaumImportance {
			t.Fatal("interval does not contain estimate")
		}
		if math.Abs(got.ForcedDownImpact+got.PerfectComponentGain-got.BirnbaumImportance) > 1e-12 {
			t.Fatal("impact/gain decomposition failed")
		}
	}
	if out.Endpoints[0].Components[0].ID != edge {
		t.Fatal("edge bottleneck must rank first")
	}
	if repeated := runTestImportance(t, loaded, request); !reflect.DeepEqual(out, repeated) {
		t.Fatal("importance must be repeatable")
	}
	// Birnbaum is independent of the selected component's own probability.
	request.BaseProfile.Reliability.Services = map[string]float64{"frontend": 0.2, "checkout": 0.9}
	changed := runTestImportance(t, loaded, request)
	a := importanceComponent(t, out, "checkout", "service", "frontend")
	b := importanceComponent(t, changed, "checkout", "service", "frontend")
	if a.BirnbaumImportance != b.BirnbaumImportance {
		t.Fatal("own probability changed Birnbaum estimate")
	}
}

func TestBirnbaumParallelFallbackAndIrrelevantComponents(t *testing.T) {
	t.Parallel()
	loaded := edgeAwareFixtureLoaded([]model.Endpoint{{ID: "fallback", EntryService: "frontend", SuccessPredicateRef: "fallback", SuccessPredicate: &predicates.Definition{
		Type: predicates.TypeAnyOf, Services: []string{"checkout", "payment"},
	}}})
	profile := ProfileParams{Name: "steady", Trials: 16000, SamplingMode: config.SamplingModeIndependentService,
		Reliability: config.ReliabilityConfig{Services: map[string]float64{"checkout": 0.8, "payment": 0.9}}}
	out := runTestImportance(t, loaded, importanceRequest(profile, "fallback"))
	a := importanceComponent(t, out, "fallback", "service", "checkout")
	b := importanceComponent(t, out, "fallback", "service", "payment")
	if math.Abs(a.BirnbaumImportance-0.1) > 0.012 || math.Abs(b.BirnbaumImportance-0.2) > 0.012 {
		t.Fatalf("parallel derivatives mismatch: %+v %+v", a, b)
	}
	for _, component := range out.Endpoints[0].Components {
		if component.Kind == "edge" || component.ID == "frontend" {
			if component.BirnbaumImportance != 0 || component.AvailabilityIfUp != component.AvailabilityIfDown {
				t.Fatal("irrelevant components must have exactly zero importance")
			}
		}
	}
}

func TestBirnbaumImmediateAndEventualAsyncImportance(t *testing.T) {
	t.Parallel()
	loaded := edgeAwareFixtureLoaded([]model.Endpoint{
		{ID: "immediate", EntryService: "frontend", SuccessPredicateRef: "immediate", SuccessPredicate: &predicates.Definition{Type: predicates.TypeEdgeAware, EntryService: "frontend", MandatoryTargets: []string{"checkout"}, EdgeModes: []string{"sync"}}},
		{ID: "eventual", EntryService: "frontend", SuccessPredicateRef: "eventual", SuccessPredicate: &predicates.Definition{Type: predicates.TypeEdgeAware, EntryService: "frontend", MandatoryTargets: []string{"payment"}, EdgeModes: []string{"sync", "async"}}},
	})
	one := 1.0
	profile := ProfileParams{Name: "steady", Trials: 100, SamplingMode: config.SamplingModeIndependentService,
		Reliability: config.ReliabilityConfig{NodeLiveProbability: &one, EdgeLiveProbability: &one}}
	out := runTestImportance(t, loaded, importanceRequest(profile, "immediate", "eventual"))
	asyncEdge := loaded.Model.Edges[1].ID
	if got := importanceComponent(t, out, "immediate", "edge", asyncEdge); got.BirnbaumImportance != 0 {
		t.Fatal("async edge changed immediate response")
	}
	if got := importanceComponent(t, out, "eventual", "edge", asyncEdge); got.BirnbaumImportance != 1 {
		t.Fatal("eventual completion must depend on async edge")
	}
}

func TestBirnbaumPreservesObservedRetriesAndExcludedServiceCounterfactual(t *testing.T) {
	t.Parallel()
	loaded := importanceFixture()
	ineligible := false
	loaded.Model.Services[0].Metadata = &model.ServiceMetadata{FailureEligible: &ineligible}
	errorRate := 0.5
	loaded.Model.Edges[0].Observed = &model.ObservedEdge{ErrorRate: &errorRate}
	loaded.Model.Edges[0].Resilience = &model.ResiliencePolicy{Retry: &model.RetryPolicy{MaxAttempts: 2}}
	one := 1.0
	profile := ProfileParams{Name: "steady", Trials: 16000, SamplingMode: config.SamplingModeIndependentReplica,
		Reliability: config.ReliabilityConfig{NodeLiveProbability: &one}}
	out := runTestImportance(t, loaded, importanceRequest(profile, "checkout"))
	got := importanceComponent(t, out, "checkout", "service", "frontend")
	if math.Abs(got.BirnbaumImportance-0.75) > 0.016 || got.LiveProbability != 1 || got.PerfectComponentGain != 0 {
		t.Fatalf("retry/exclusion semantics mismatch: %+v", got)
	}
}

func TestBirnbaumKeepsDistinctEdgePathsBetweenEndpoints(t *testing.T) {
	t.Parallel()
	loaded := importanceFixture()
	one := 1.0
	loaded.Model.Edges = append(loaded.Model.Edges, model.Edge{
		ID: "frontend->checkout:async", From: "frontend", To: "checkout", Kind: model.EdgeKindAsync,
		Observed: &model.ObservedEdge{ErrorRate: &one},
	})
	loaded.Model.Endpoints = []model.Endpoint{
		{ID: "async", EntryService: "frontend", SuccessPredicateRef: "async", SuccessPredicate: &predicates.Definition{
			Type: predicates.TypeEdgeAware, EntryService: "frontend", MandatoryTargets: []string{"checkout"}, EdgeModes: []string{"async"},
		}},
		{ID: "sync", EntryService: "frontend", SuccessPredicateRef: "sync", SuccessPredicate: &predicates.Definition{
			Type: predicates.TypeEdgeAware, EntryService: "frontend", MandatoryTargets: []string{"checkout"}, EdgeModes: []string{"sync"},
		}},
	}
	profile := ProfileParams{Name: "steady", Trials: 20, SamplingMode: config.SamplingModeIndependentService,
		Reliability: config.ReliabilityConfig{NodeLiveProbability: &one, EdgeLiveProbability: &one}}
	out := runTestImportance(t, loaded, importanceRequest(profile, "async", "sync"))
	for _, endpoint := range []string{"async", "sync"} {
		want := 1.0
		if endpoint == "async" {
			want = 0
		}
		got := importanceComponent(t, out, endpoint, "service", "frontend")
		if got.BirnbaumImportance != want {
			t.Fatalf("%s importance=%f want %f; paths with different edges must not share execution viability", endpoint, got.BirnbaumImportance, want)
		}
	}
}

func TestBirnbaumSelectionTopKAndInvalidInputs(t *testing.T) {
	t.Parallel()
	loaded := importanceFixture()
	profile := ProfileParams{Name: "steady", Trials: 10, SamplingMode: config.SamplingModeIndependentService}
	request := importanceRequest(profile, "checkout")
	full := runTestImportance(t, loaded, request)
	request.Spec.TopK = 1
	top := runTestImportance(t, loaded, request)
	if top.EvaluatedComponents != 3 || len(top.Endpoints[0].Components) != 1 || !reflect.DeepEqual(top.Endpoints[0].Components[0], full.Endpoints[0].Components[0]) {
		t.Fatal("top_k must truncate after complete ranking")
	}
	for _, test := range []struct {
		name, message string
		mutate        func(*ImportanceParams)
	}{
		{"unknown component", "unknown service", func(r *ImportanceParams) { r.Spec.Components.ServiceIDs = []string{"missing"} }},
		{"unknown endpoint", "not present", func(r *ImportanceParams) { r.Spec.Targets[0].EndpointID = "missing" }},
		{"fixed-k", "requires independent", func(r *ImportanceParams) { r.BaseProfile.SamplingMode = config.SamplingModeFixedKServiceSet }},
		{"fault overlay", "without fault_profile", func(r *ImportanceParams) { r.BaseProfile.FaultProfile = "zone-down" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := importanceRequest(profile, "checkout")
			test.mutate(&candidate)
			_, err := RunArtifactProfiles(loaded, AnalysisParams{Seed: 42, Importance: []ImportanceParams{candidate}})
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("wanted %q error, got %v", test.message, err)
			}
		})
	}
}
