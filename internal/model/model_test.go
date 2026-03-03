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
				URI:     "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json",
				Digest:  "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7",
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
