package journeys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/model"
)

func TestLoadJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "journeys.json")
	payload := `{
  "schema_version": "1.0",
  "journeys": {
    "frontend:GET /checkout": [
      ["frontend", "checkout", "payment"],
      ["frontend", "checkout", "inventory"]
    ]
  }
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write journeys file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	paths := got["frontend:GET /checkout"]
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
}

func TestValidateAgainstModel_InvalidHop(t *testing.T) {
	t.Parallel()

	mdl := model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "checkout", Name: "checkout", Replicas: 1},
			{ID: "payment", Name: "payment", Replicas: 1},
		},
		Edges: []model.Edge{
			{From: "frontend", To: "checkout", Kind: model.EdgeKindSync, Blocking: true},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"},
		},
	}

	err := ValidateAgainstModel(map[string][][]string{
		"frontend:GET /checkout": {
			{"frontend", "payment"},
		},
	}, mdl)
	if err == nil {
		t.Fatal("expected invalid hop error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid hop") {
		t.Fatalf("expected invalid hop error, got: %v", err)
	}
}

