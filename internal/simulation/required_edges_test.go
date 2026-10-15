package simulation

import (
	"math"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

var mandatoryCallIDs = []string{"frontend|checkout|sync|true", "checkout|payment|sync|true", "frontend|payment|sync|true"}

func requiredCallsFixture(required bool) artifact.Loaded {
	def := predicates.Definition{Type: predicates.TypeEdgeAware, EntryService: "frontend", MandatoryTargets: []string{"checkout", "payment"}, EdgeModes: []string{"sync"}}
	if required {
		def.RequiredEdges = append([]string{}, mandatoryCallIDs...)
	}
	loaded := edgeAwareFixtureLoaded([]model.Endpoint{{ID: "operation", EntryService: "frontend", SuccessPredicateRef: "operation", SuccessPredicate: &def}})
	loaded.Model.Edges = []model.Edge{
		{ID: mandatoryCallIDs[0], From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
		{ID: mandatoryCallIDs[1], From: "checkout", To: "payment", Kind: model.EdgeKindSync, Blocking: true},
		{ID: mandatoryCallIDs[2], From: "frontend", To: "payment", Kind: model.EdgeKindSync, Blocking: true},
	}
	return loaded
}

func TestRequiredCallsDistinguishReachabilityForEveryEdgeState(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		edges := map[string]float64{}
		for i, id := range mandatoryCallIDs {
			if mask&(1<<i) != 0 {
				edges[id] = 1
			} else {
				edges[id] = 0
			}
		}
		p := ProfileParams{Name: "fixed-state", Trials: 1, SamplingMode: config.SamplingModeIndependentReplica,
			Reliability: config.ReliabilityConfig{Services: map[string]float64{"frontend": 1, "checkout": 1, "payment": 1}, Edges: edges}}
		for _, required := range []bool{false, true} {
			out, err := RunArtifactProfiles(requiredCallsFixture(required), AnalysisParams{Seed: 42, Profiles: []ProfileParams{p}})
			if err != nil {
				t.Fatal(err)
			}
			want := 0.0
			if required && mask == 7 || !required && mask&1 != 0 && mask&6 != 0 {
				want = 1
			}
			if got := out.Profiles[0].EndpointAvailability["operation"]; got != want {
				t.Fatalf("required=%v mask=%03b got %v want %v", required, mask, got, want)
			}
		}
	}
}

func TestRequiredCallsBirnbaumAgainstIndependentSeriesOracle(t *testing.T) {
	p := ProfileParams{Name: "series", Trials: 40000, SamplingMode: config.SamplingModeIndependentReplica,
		Reliability: config.ReliabilityConfig{Services: map[string]float64{"frontend": 1, "checkout": 0.8, "payment": 0.9},
			Edges: map[string]float64{mandatoryCallIDs[0]: 0.7, mandatoryCallIDs[1]: 0.85, mandatoryCallIDs[2]: 0.95}}}
	out := runTestImportance(t, requiredCallsFixture(true), importanceRequest(p, "operation"))
	probability := 0.8 * 0.9 * 0.7 * 0.85 * 0.95
	if math.Abs(out.Endpoints[0].BaselineAvailability-probability) > 0.008 {
		t.Fatal("availability disagrees with series oracle")
	}
	for _, v := range []struct {
		kind, id string
		q        float64
	}{
		{"service", "checkout", 0.8}, {"service", "payment", 0.9},
		{"edge", mandatoryCallIDs[0], 0.7}, {"edge", mandatoryCallIDs[1], 0.85}, {"edge", mandatoryCallIDs[2], 0.95},
	} {
		got := importanceComponent(t, out, "operation", v.kind, v.id)
		if math.Abs(got.BirnbaumImportance-probability/v.q) > 0.008 {
			t.Fatalf("%s importance=%v want=%v", v.id, got.BirnbaumImportance, probability/v.q)
		}
	}
}

func TestRequiredCallsRejectUnknownEdges(t *testing.T) {
	loaded := requiredCallsFixture(true)
	loaded.Model.Endpoints[0].SuccessPredicate.RequiredEdges = []string{"absent"}
	_, err := RunArtifactProfiles(loaded, AnalysisParams{Profiles: []ProfileParams{{Name: "fixed", Trials: 1}}})
	if err == nil || !strings.Contains(err.Error(), "required edge") {
		t.Fatalf("got %v", err)
	}
}
