package artifact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
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
		SnapshotID:      "snapshot-1",
		TopologyVersion: "topology-1",
		WindowStart:     "2026-03-11T08:00:00Z",
		WindowEnd:       "2026-03-11T08:05:00Z",
		Ingest: SnapshotIngest{
			Spans:  10,
			Traces: 2,
		},
		Counts: SnapshotCounts{
			Services:  1,
			Edges:     0,
			Endpoints: 1,
		},
		Coverage: SnapshotCoverage{
			Confidence:         0.8,
			ServiceSupportMin:  1,
			EdgeSupportMin:     0,
			EndpointSupportMin: 1,
		},
		Sources: []SnapshotSource{
			{
				Type:         "traces",
				Connector:    "trace_file",
				Ref:          "bering://snapshot/1",
				Observations: 10,
			},
		},
		Diff: SnapshotDiff{},
		Discovery: SnapshotDiscovery{
			Endpoints: []SnapshotDiscoveryEndpoint{
				{
					ID: "frontend:GET /health",
					Metadata: SnapshotEndpointMetadata{
						Weight: floatPtr(2),
					},
				},
			},
		},
		Model: testModel(),
		Metadata: SnapshotMetadata{
			SourceType: "bering",
			SourceRef:  "bering://snapshot/1",
			EmittedAt:  "2026-03-11T08:00:00Z",
			Confidence: 0.8,
			Schema: modelcontract.SchemaRef{
				Name:    modelcontract.BeringSnapshotV100Name,
				Version: modelcontract.BeringSnapshotV100Version,
				URI:     modelcontract.BeringSnapshotV100URI,
				Digest:  modelcontract.BeringSnapshotV100Digest,
			},
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
	if loaded.PredicateSource != ProvenanceDefault {
		t.Fatalf("expected default predicate provenance, got %s", loaded.PredicateSource)
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

func TestLoad_CheckedInSnapshotSample(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "examples", "outputs", "snapshot.sample.json")

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load checked-in snapshot sample failed: %v", err)
	}
	if loaded.Metadata.Kind != modelcontract.KindSnapshot {
		t.Fatalf("expected snapshot kind, got %s", loaded.Metadata.Kind)
	}
	if loaded.Metadata.Contract.Digest != modelcontract.BeringSnapshotV100Digest {
		t.Fatalf("unexpected snapshot digest: got %s want %s", loaded.Metadata.Contract.Digest, modelcontract.BeringSnapshotV100Digest)
	}
	if loaded.Metadata.TopologyVersion == "" {
		t.Fatal("expected topology version from snapshot sample")
	}
	if len(loaded.EndpointWeights) == 0 {
		t.Fatal("expected endpoint weights extracted from snapshot discovery metadata")
	}
}

func TestLoad_CheckedInSnapshotV110Sample(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "examples", "outputs", "snapshot-v1.1.0.sample.json")

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load checked-in snapshot v1.1.0 sample failed: %v", err)
	}
	if loaded.Metadata.Kind != modelcontract.KindSnapshot {
		t.Fatalf("expected snapshot kind, got %s", loaded.Metadata.Kind)
	}
	if loaded.Metadata.Contract.Version != modelcontract.BeringSnapshotV110Version {
		t.Fatalf("unexpected snapshot contract version: %+v", loaded.Metadata.Contract)
	}
	if loaded.Snapshot == nil {
		t.Fatal("expected typed snapshot metadata for v1.1.0 artifact")
	}
	if loaded.Model.Edges[0].ID == "" {
		t.Fatal("expected edge ids to be preserved for v1.1.0 model")
	}
}

func TestLoad_CheckedInModelV110Sample(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "examples", "outputs", "model-v1.1.0.sample.json")

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load checked-in model v1.1.0 sample failed: %v", err)
	}
	if loaded.Metadata.Kind != modelcontract.KindModel {
		t.Fatalf("expected model kind, got %s", loaded.Metadata.Kind)
	}
	if loaded.Metadata.Contract.Version != modelcontract.BeringModelV110Version {
		t.Fatalf("unexpected model contract version: %+v", loaded.Metadata.Contract)
	}
}

func TestLoad_RejectsDigestMismatchForSupportedVersion(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "model.json")
	mdl := testModel()
	mdl.Metadata.Schema.Digest = "sha256:badbadbad"
	if err := model.WriteToFile(path, mdl); err != nil {
		t.Fatalf("write model: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected digest mismatch error")
	}
	if !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %v", err)
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

func floatPtr(value float64) *float64 {
	return &value
}
