package model

import (
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

func TestValidate_StructuralAndContractValidation(t *testing.T) {
	t.Parallel()

	mdl := ResilienceModel{
		Services: []Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
		},
		Edges: []Edge{},
		Endpoints: []Endpoint{
			{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"},
		},
		Metadata: Metadata{
			SourceType:   "bering",
			SourceRef:    "artifact",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.8,
			Schema: Schema{
				Name:    modelcontract.BeringModelV130Name,
				Version: modelcontract.BeringModelV130Version,
				URI:     modelcontract.BeringModelV130URI,
				Digest:  modelcontract.BeringModelV130Digest,
			},
		},
	}

	if err := mdl.Validate(); err != nil {
		t.Fatalf("expected valid model, got error: %v", err)
	}
	if err := modelcontract.ValidateStrict(modelcontract.SchemaRef{
		Name:    mdl.Metadata.Schema.Name,
		Version: mdl.Metadata.Schema.Version,
		URI:     mdl.Metadata.Schema.URI,
		Digest:  mdl.Metadata.Schema.Digest,
	}); err != nil {
		t.Fatalf("expected supported contract, got error: %v", err)
	}

	mdl.Metadata.Schema.Version = "1.2.0"
	if err := mdl.Validate(); err != nil {
		t.Fatalf("expected structural validation to ignore supported-contract resolution, got error: %v", err)
	}
	if err := modelcontract.ValidateStrict(modelcontract.SchemaRef{
		Name:    mdl.Metadata.Schema.Name,
		Version: mdl.Metadata.Schema.Version,
		URI:     mdl.Metadata.Schema.URI,
		Digest:  mdl.Metadata.Schema.Digest,
	}); err == nil {
		t.Fatal("expected contract validation to reject unsupported version, got nil")
	}
}

func TestValidate_VersionedContractsRequireEdgeIDs(t *testing.T) {
	t.Parallel()

	refs := []modelcontract.SchemaRef{
		modelcontract.ExpectedModelV130Ref(),
	}
	for _, ref := range refs {
		ref := ref
		t.Run(ref.Version, func(t *testing.T) {
			t.Parallel()

			mdl := ResilienceModel{
				Services: []Service{
					{ID: "gateway", Name: "gateway", Replicas: 1},
					{ID: "checkout", Name: "checkout", Replicas: 1},
				},
				Edges: []Edge{
					{From: "gateway", To: "checkout", Kind: EdgeKindSync, Blocking: true},
				},
				Endpoints: []Endpoint{
					{ID: "gateway:GET /checkout", EntryService: "gateway", SuccessPredicateRef: "gateway:GET /checkout"},
				},
				Metadata: Metadata{
					SourceType:   "bering",
					SourceRef:    "artifact",
					DiscoveredAt: "2026-03-22T00:00:00Z",
					Confidence:   0.9,
					Schema: Schema{
						Name:    ref.Name,
						Version: ref.Version,
						URI:     ref.URI,
						Digest:  ref.Digest,
					},
				},
			}

			if err := mdl.Validate(); err == nil {
				t.Fatalf("expected %s model without edge ids to fail validation", ref.Version)
			}

			mdl.Edges[0].ID = "gateway|checkout|sync|true"
			if err := mdl.Validate(); err != nil {
				t.Fatalf("expected %s model with edge ids to validate, got %v", ref.Version, err)
			}
		})
	}
}

func TestValidate_RejectsZeroReplicas(t *testing.T) {
	t.Parallel()

	ref := modelcontract.ExpectedModelV130Ref()
	base := ResilienceModel{
		Services: []Service{
			{ID: "gateway", Name: "gateway", Replicas: 1},
		},
		Edges: []Edge{},
		Endpoints: []Endpoint{
			{ID: "gateway:GET /health", EntryService: "gateway", SuccessPredicateRef: "gateway:GET /health"},
		},
		Metadata: Metadata{
			SourceType:   "bering",
			SourceRef:    "artifact",
			DiscoveredAt: "2026-03-22T00:00:00Z",
			Confidence:   0.9,
			Schema: Schema{
				Name:    ref.Name,
				Version: ref.Version,
				URI:     ref.URI,
				Digest:  ref.Digest,
			},
		},
	}

	serviceZero := base
	serviceZero.Services = []Service{{ID: "gateway", Name: "gateway", Replicas: 0}}
	if err := serviceZero.Validate(); err == nil {
		t.Fatal("expected zero service replicas to fail validation")
	}

	placementZero := base
	placementZero.Services = []Service{{
		ID:       "gateway",
		Name:     "gateway",
		Replicas: 1,
		Metadata: &ServiceMetadata{
			Placements: []Placement{{Replicas: 0, Labels: map[string]string{"zone": "a"}}},
		},
	}}
	if err := placementZero.Validate(); err == nil {
		t.Fatal("expected zero placement replicas to fail validation")
	}
}

func TestValidate_EndpointSemantics(t *testing.T) {
	t.Parallel()

	confidence := 0.9
	mdl := ResilienceModel{
		Services: []Service{
			{ID: "gateway", Name: "gateway", Replicas: 1},
			{ID: "checkout", Name: "checkout", Replicas: 1},
		},
		Edges: []Edge{},
		Endpoints: []Endpoint{{
			ID:                  "gateway:GET /checkout",
			EntryService:        "gateway",
			SuccessPredicateRef: "catalog.checkout.immediate",
			Metadata: &EndpointMetadata{
				Semantics: &EndpointSemantics{
					PredicateMode:    EndpointPredicateModeImmediate,
					MandatoryTargets: []string{"checkout"},
					DependencyModes:  []string{string(EdgeKindSync)},
					Source:           "bering",
					Confidence:       &confidence,
				},
			},
		}},
		Metadata: Metadata{
			SourceType:   "bering",
			SourceRef:    "artifact",
			DiscoveredAt: "2026-03-22T00:00:00Z",
			Confidence:   0.9,
			Schema: Schema{
				Name:    modelcontract.BeringModelV130Name,
				Version: modelcontract.BeringModelV130Version,
				URI:     modelcontract.BeringModelV130URI,
				Digest:  modelcontract.BeringModelV130Digest,
			},
		},
	}

	if err := mdl.Validate(); err != nil {
		t.Fatalf("expected endpoint semantics to validate, got %v", err)
	}

	mdl.Endpoints[0].Metadata.Semantics.MandatoryTargets = []string{"missing"}
	if err := mdl.Validate(); err == nil {
		t.Fatal("expected unknown mandatory target to fail validation")
	}
}
