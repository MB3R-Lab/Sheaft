package report

import (
	"math"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/gate"
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
