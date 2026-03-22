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
				Name:    "io.mb3r.bering.model",
				Version: "1.0.0",
				URI:     "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json",
				Digest:  "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7",
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

	mdl.Metadata.Schema.Version = "1.0.1"
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

func TestValidate_V110RequiresEdgeIDs(t *testing.T) {
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
				Name:    modelcontract.BeringModelV110Name,
				Version: modelcontract.BeringModelV110Version,
				URI:     modelcontract.BeringModelV110URI,
				Digest:  modelcontract.BeringModelV110Digest,
			},
		},
	}

	if err := mdl.Validate(); err == nil {
		t.Fatal("expected v1.1.0 model without edge ids to fail validation")
	}

	mdl.Edges[0].ID = "gateway|checkout|sync|true"
	if err := mdl.Validate(); err != nil {
		t.Fatalf("expected v1.1.0 model with edge ids to validate, got %v", err)
	}
}
