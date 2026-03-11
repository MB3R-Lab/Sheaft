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
