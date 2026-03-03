package model

import "testing"

func TestValidate_StrictSchemaBinding(t *testing.T) {
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
				URI:     "https://schemas.mb3r.dev/bering/model/v1.0.0/model.schema.json",
				Digest:  "sha256:7dc733936a9d3f94ab92f46a30d4c8d0f5c05d60670c4247786c59a3fe7630f7",
			},
		},
	}

	if err := mdl.Validate(); err != nil {
		t.Fatalf("expected valid model, got error: %v", err)
	}

	mdl.Metadata.Schema.Version = "1.0.1"
	if err := mdl.Validate(); err == nil {
		t.Fatal("expected strict schema version mismatch error, got nil")
	}
}
