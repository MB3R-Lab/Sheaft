package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

const (
	ProvenanceModel    = "model"
	ProvenanceSnapshot = "snapshot"
	ProvenanceExternal = "external_overlay"
	ProvenanceDefault  = "default"
)

type SnapshotEnvelope struct {
	SnapshotID      string                `json:"snapshot_id"`
	TopologyVersion string                `json:"topology_version"`
	WindowStart     string                `json:"window_start"`
	WindowEnd       string                `json:"window_end"`
	Ingest          SnapshotIngest        `json:"ingest"`
	Counts          SnapshotCounts        `json:"counts"`
	Coverage        SnapshotCoverage      `json:"coverage"`
	Sources         []SnapshotSource      `json:"sources"`
	Diff            SnapshotDiff          `json:"diff"`
	Discovery       SnapshotDiscovery     `json:"discovery"`
	Model           model.ResilienceModel `json:"model"`
	Metadata        SnapshotMetadata      `json:"metadata"`
}

type SnapshotIngest struct {
	Spans        int `json:"spans"`
	Traces       int `json:"traces"`
	DroppedSpans int `json:"dropped_spans"`
	LateSpans    int `json:"late_spans"`
}

type SnapshotCounts struct {
	Services  int `json:"services"`
	Edges     int `json:"edges"`
	Endpoints int `json:"endpoints"`
}

type SnapshotCoverage struct {
	Confidence         float64 `json:"confidence"`
	ServiceSupportMin  int     `json:"service_support_min"`
	EdgeSupportMin     int     `json:"edge_support_min"`
	EndpointSupportMin int     `json:"endpoint_support_min"`
}

type SnapshotSource struct {
	Type         string `json:"type"`
	Connector    string `json:"connector,omitempty"`
	Ref          string `json:"ref,omitempty"`
	Observations int    `json:"observations,omitempty"`
}

type SnapshotDiff struct {
	AddedServices    int `json:"added_services"`
	RemovedServices  int `json:"removed_services"`
	ChangedServices  int `json:"changed_services"`
	AddedEdges       int `json:"added_edges"`
	RemovedEdges     int `json:"removed_edges"`
	ChangedEdges     int `json:"changed_edges"`
	AddedEndpoints   int `json:"added_endpoints"`
	RemovedEndpoints int `json:"removed_endpoints"`
	ChangedEndpoints int `json:"changed_endpoints"`
}

type SnapshotDiscovery struct {
	Services  []json.RawMessage           `json:"services"`
	Edges     []json.RawMessage           `json:"edges"`
	Endpoints []SnapshotDiscoveryEndpoint `json:"endpoints"`
}

type SnapshotDiscoveryEndpoint struct {
	ID       string                   `json:"id"`
	Metadata SnapshotEndpointMetadata `json:"metadata"`
}

type SnapshotEndpointMetadata struct {
	Weight       *float64 `json:"weight,omitempty"`
	PredicateRef string   `json:"predicate_ref,omitempty"`
}

type SnapshotMetadata struct {
	SourceType string                  `json:"source_type"`
	SourceRef  string                  `json:"source_ref"`
	EmittedAt  string                  `json:"emitted_at"`
	Confidence float64                 `json:"confidence"`
	Schema     modelcontract.SchemaRef `json:"schema"`
}

type Metadata struct {
	Path            string                          `json:"path"`
	Digest          string                          `json:"digest"`
	ArtifactID      string                          `json:"artifact_id,omitempty"`
	ProducedAt      string                          `json:"produced_at,omitempty"`
	Kind            modelcontract.ArtifactKind      `json:"kind"`
	Contract        modelcontract.SupportedContract `json:"contract"`
	SourceType      string                          `json:"source_type"`
	SourceRef       string                          `json:"source_ref"`
	TopologyVersion string                          `json:"topology_version,omitempty"`
}

type Loaded struct {
	Metadata        Metadata                         `json:"metadata"`
	Model           model.ResilienceModel            `json:"model"`
	Predicates      map[string]predicates.Definition `json:"predicates,omitempty"`
	EndpointWeights map[string]float64               `json:"endpoint_weights,omitempty"`
	PredicateSource string                           `json:"predicate_source"`
	WeightsSource   string                           `json:"weights_source"`
}

func Load(path string) (Loaded, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Loaded{}, fmt.Errorf("read artifact: %w", err)
	}

	sum := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(sum[:])

	var probe struct {
		Schema   modelcontract.SchemaRef `json:"schema"`
		Metadata struct {
			Schema modelcontract.SchemaRef `json:"schema"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && strings.TrimSpace(probe.Schema.Name) != "" {
		contract, err := modelcontract.Resolve(probe.Schema)
		if err != nil {
			return Loaded{}, err
		}
		if contract.Kind != modelcontract.KindSnapshot {
			return Loaded{}, fmt.Errorf("artifact declares snapshot schema %s@%s but supported kind is %s", contract.Name, contract.Version, contract.Kind)
		}
		return loadSnapshot(path, raw, digest, contract)
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return Loaded{}, fmt.Errorf("decode artifact probe: %w", err)
	}
	if strings.TrimSpace(probe.Metadata.Schema.Name) == "" {
		return Loaded{}, fmt.Errorf("artifact %s is missing supported schema metadata", filepath.Base(path))
	}
	contract, err := modelcontract.Resolve(probe.Metadata.Schema)
	if err != nil {
		return Loaded{}, err
	}
	switch contract.Kind {
	case modelcontract.KindSnapshot:
		return loadSnapshot(path, raw, digest, contract)
	case modelcontract.KindModel:
		return loadModel(path, raw, digest, contract)
	default:
		return Loaded{}, fmt.Errorf("artifact declares model schema %s@%s but supported kind is %s", contract.Name, contract.Version, contract.Kind)
	}
}

func loadModel(path string, raw []byte, digest string, contract modelcontract.SupportedContract) (Loaded, error) {
	var mdl model.ResilienceModel
	if err := json.Unmarshal(raw, &mdl); err != nil {
		return Loaded{}, fmt.Errorf("decode model json: %w", err)
	}
	if err := mdl.Validate(); err != nil {
		return Loaded{}, fmt.Errorf("validate model: %w", err)
	}
	return Loaded{
		Metadata: Metadata{
			Path:            path,
			Digest:          digest,
			ArtifactID:      mdl.Metadata.SourceRef,
			ProducedAt:      mdl.Metadata.DiscoveredAt,
			Kind:            contract.Kind,
			Contract:        contract,
			SourceType:      mdl.Metadata.SourceType,
			SourceRef:       mdl.Metadata.SourceRef,
			TopologyVersion: mdl.Metadata.TopologyVersion,
		},
		Model:           mdl,
		Predicates:      clonePredicates(mdl.Predicates),
		EndpointWeights: cloneWeights(mdl.EndpointWeights),
		PredicateSource: sourceOrDefault(len(mdl.Predicates) > 0, ProvenanceModel),
		WeightsSource:   sourceOrDefault(len(mdl.EndpointWeights) > 0, ProvenanceModel),
	}, nil
}

func loadSnapshot(path string, raw []byte, digest string, contract modelcontract.SupportedContract) (Loaded, error) {
	var snapshot SnapshotEnvelope
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return Loaded{}, fmt.Errorf("decode snapshot json: %w", err)
	}
	mdl := snapshot.Model
	if mdl.Metadata.TopologyVersion == "" && strings.TrimSpace(snapshot.TopologyVersion) != "" {
		mdl.Metadata.TopologyVersion = snapshot.TopologyVersion
	}
	if err := mdl.Validate(); err != nil {
		return Loaded{}, fmt.Errorf("validate snapshot model: %w", err)
	}

	predicateSource := ProvenanceModel
	preds := clonePredicates(mdl.Predicates)
	weightSource := ProvenanceModel
	weights := cloneWeights(mdl.EndpointWeights)
	snapshotWeights, err := extractSnapshotWeights(snapshot.Discovery)
	if err != nil {
		return Loaded{}, err
	}
	if len(snapshotWeights) > 0 {
		weights = snapshotWeights
		weightSource = ProvenanceSnapshot
	}

	return Loaded{
		Metadata: Metadata{
			Path:            path,
			Digest:          digest,
			ArtifactID:      firstNonEmpty(snapshot.SnapshotID, snapshot.Metadata.SourceRef, mdl.Metadata.SourceRef),
			ProducedAt:      firstNonEmpty(snapshot.Metadata.EmittedAt, mdl.Metadata.DiscoveredAt),
			Kind:            contract.Kind,
			Contract:        contract,
			SourceType:      firstNonEmpty(snapshot.Metadata.SourceType, mdl.Metadata.SourceType),
			SourceRef:       firstNonEmpty(snapshot.Metadata.SourceRef, mdl.Metadata.SourceRef),
			TopologyVersion: firstNonEmpty(snapshot.TopologyVersion, mdl.Metadata.TopologyVersion),
		},
		Model:           mdl,
		Predicates:      preds,
		EndpointWeights: weights,
		PredicateSource: sourceOrDefault(len(preds) > 0, predicateSource),
		WeightsSource:   sourceOrDefault(len(weights) > 0, weightSource),
	}, nil
}

func extractSnapshotWeights(discovery SnapshotDiscovery) (map[string]float64, error) {
	if len(discovery.Endpoints) == 0 {
		return map[string]float64{}, nil
	}
	weights := make(map[string]float64, len(discovery.Endpoints))
	for _, endpoint := range discovery.Endpoints {
		if strings.TrimSpace(endpoint.ID) == "" || endpoint.Metadata.Weight == nil {
			continue
		}
		if *endpoint.Metadata.Weight < 0 {
			return nil, fmt.Errorf("snapshot discovery endpoint weight[%s] must be >= 0", endpoint.ID)
		}
		weights[endpoint.ID] = *endpoint.Metadata.Weight
	}
	return weights, nil
}

func clonePredicates(in map[string]predicates.Definition) map[string]predicates.Definition {
	if len(in) == 0 {
		return map[string]predicates.Definition{}
	}
	out := make(map[string]predicates.Definition, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneWeights(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sourceOrDefault(found bool, source string) string {
	if found {
		return source
	}
	return ProvenanceDefault
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
