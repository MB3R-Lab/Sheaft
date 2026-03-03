package simulation

import (
	"math"
	"strings"
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
			Schema: model.Schema{
				Name:    "io.mb3r.bering.model",
				Version: "1.0.0",
				URI:     "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json",
				Digest:  "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7",
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
			{From: "frontend", To: "checkoutA", Kind: model.EdgeKindSync, Blocking: true},
			{From: "frontend", To: "checkoutB", Kind: model.EdgeKindSync, Blocking: true},
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
				Name:    "io.mb3r.bering.model",
				Version: "1.0.0",
				URI:     "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json",
				Digest:  "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7",
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
			{From: "frontend", To: "checkoutA", Kind: model.EdgeKindSync, Blocking: true},
			{From: "frontend", To: "checkoutB", Kind: model.EdgeKindSync, Blocking: true},
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
				Name:    "io.mb3r.bering.model",
				Version: "1.0.0",
				URI:     "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json",
				Digest:  "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7",
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
				Name:    "io.mb3r.bering.model",
				Version: "1.0.0",
				URI:     "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json",
				Digest:  "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7",
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
