package artifact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

func TestLoad_PlainModelContract(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "model.json")
	mdl := testModel()
	if err := model.WriteToFile(path, mdl); err != nil {
		t.Fatalf("write model: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Metadata.Kind != modelcontract.KindModel {
		t.Fatalf("expected model kind, got %s", loaded.Metadata.Kind)
	}
	if loaded.Metadata.Contract.Name != modelcontract.BeringModelV100Name {
		t.Fatalf("unexpected contract: %+v", loaded.Metadata.Contract)
	}
}

func TestLoad_SnapshotContract(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "snapshot.json")
	snapshot := SnapshotEnvelope{
		Schema: modelcontract.SchemaRef{
			Name:    modelcontract.BeringSnapshotV100Name,
			Version: modelcontract.BeringSnapshotV100Version,
			URI:     modelcontract.BeringSnapshotV100URI,
			Digest:  modelcontract.BeringSnapshotV100Digest,
		},
		ArtifactID:      "snapshot-1",
		ProducedAt:      "2026-03-11T08:00:00Z",
		SourceType:      "bering",
		SourceRef:       "bering://snapshot/1",
		TopologyVersion: "topology-1",
		Model:           testModel(),
		Predicates: map[string]predicates.Definition{
			"frontend:GET /health": {
				Type:     predicates.TypeAllOf,
				Services: []string{"frontend"},
			},
		},
		EndpointWeights: map[string]float64{
			"frontend:GET /health": 2,
		},
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Metadata.Kind != modelcontract.KindSnapshot {
		t.Fatalf("expected snapshot kind, got %s", loaded.Metadata.Kind)
	}
	if loaded.PredicateSource != ProvenanceSnapshot {
		t.Fatalf("expected snapshot predicate provenance, got %s", loaded.PredicateSource)
	}
	if loaded.WeightsSource != ProvenanceSnapshot {
		t.Fatalf("expected snapshot weight provenance, got %s", loaded.WeightsSource)
	}
	if loaded.Metadata.TopologyVersion != "topology-1" {
		t.Fatalf("unexpected topology version: %s", loaded.Metadata.TopologyVersion)
	}
}

func TestLoad_UnsupportedContract(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "unsupported.json")
	raw := `{"metadata":{"schema":{"name":"io.mb3r.bering.model","version":"9.9.9","uri":"https://example.invalid/model.json","digest":"sha256:deadbeef"}}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write unsupported artifact: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected unsupported contract error")
	}
	if !strings.Contains(err.Error(), "supported contracts") {
		t.Fatalf("expected supported contracts hint, got %v", err)
	}
}

func testModel() model.ResilienceModel {
	return model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"},
		},
		Metadata: model.Metadata{
			SourceType:   "bering",
			SourceRef:    "bering://fixture",
			DiscoveredAt: "2026-03-03T00:00:00Z",
			Confidence:   0.8,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
}
