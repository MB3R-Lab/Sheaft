package simulation

import (
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/model"
)

func TestRun_DeterministicWithSameSeed(t *testing.T) {
	t.Parallel()

	mdl := model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkout", Name: "checkout", Replicas: 1},
		},
		Edges: []model.Edge{
			{From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
		},
		Metadata: model.Metadata{
			SourceType:   "test",
			SourceRef:    "fixture",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.7,
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
