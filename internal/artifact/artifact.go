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
	Schema          modelcontract.SchemaRef          `json:"schema"`
	ArtifactID      string                           `json:"artifact_id"`
	ProducedAt      string                           `json:"produced_at"`
	SourceType      string                           `json:"source_type"`
	SourceRef       string                           `json:"source_ref"`
	TopologyVersion string                           `json:"topology_version,omitempty"`
	Model           model.ResilienceModel            `json:"model"`
	Predicates      map[string]predicates.Definition `json:"predicates,omitempty"`
	EndpointWeights map[string]float64               `json:"endpoint_weights,omitempty"`
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

	var snapshotProbe struct {
		Schema modelcontract.SchemaRef `json:"schema"`
	}
	if err := json.Unmarshal(raw, &snapshotProbe); err == nil && strings.TrimSpace(snapshotProbe.Schema.Name) != "" {
		contract, err := modelcontract.Resolve(snapshotProbe.Schema)
		if err != nil {
			return Loaded{}, err
		}
		if contract.Kind != modelcontract.KindSnapshot {
			return Loaded{}, fmt.Errorf("artifact declares snapshot schema %s@%s but supported kind is %s", contract.Name, contract.Version, contract.Kind)
		}
		return loadSnapshot(path, raw, digest, contract)
	}

	var modelProbe struct {
		Metadata struct {
			Schema modelcontract.SchemaRef `json:"schema"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &modelProbe); err != nil {
		return Loaded{}, fmt.Errorf("decode artifact probe: %w", err)
	}
	if strings.TrimSpace(modelProbe.Metadata.Schema.Name) == "" {
		return Loaded{}, fmt.Errorf("artifact %s is missing supported schema metadata", filepath.Base(path))
	}
	contract, err := modelcontract.Resolve(modelProbe.Metadata.Schema)
	if err != nil {
		return Loaded{}, err
	}
	if contract.Kind != modelcontract.KindModel {
		return Loaded{}, fmt.Errorf("artifact declares model schema %s@%s but supported kind is %s", contract.Name, contract.Version, contract.Kind)
	}
	return loadModel(path, raw, digest, contract)
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
	if err := snapshot.Model.Validate(); err != nil {
		return Loaded{}, fmt.Errorf("validate snapshot model: %w", err)
	}
	for name, def := range snapshot.Predicates {
		if strings.TrimSpace(name) == "" {
			return Loaded{}, fmt.Errorf("snapshot predicate key cannot be empty")
		}
		if err := def.Validate(); err != nil {
			return Loaded{}, fmt.Errorf("snapshot predicate %q: %w", name, err)
		}
	}
	for endpoint, weight := range snapshot.EndpointWeights {
		if strings.TrimSpace(endpoint) == "" {
			return Loaded{}, fmt.Errorf("snapshot endpoint_weights key cannot be empty")
		}
		if weight < 0 {
			return Loaded{}, fmt.Errorf("snapshot endpoint_weights[%s] must be >= 0", endpoint)
		}
	}

	predicateSource := ProvenanceModel
	preds := clonePredicates(snapshot.Model.Predicates)
	if len(snapshot.Predicates) > 0 {
		preds = clonePredicates(snapshot.Predicates)
		predicateSource = ProvenanceSnapshot
	}
	weightSource := ProvenanceModel
	weights := cloneWeights(snapshot.Model.EndpointWeights)
	if len(snapshot.EndpointWeights) > 0 {
		weights = cloneWeights(snapshot.EndpointWeights)
		weightSource = ProvenanceSnapshot
	}

	return Loaded{
		Metadata: Metadata{
			Path:            path,
			Digest:          digest,
			ArtifactID:      firstNonEmpty(snapshot.ArtifactID, snapshot.SourceRef, snapshot.Model.Metadata.SourceRef),
			ProducedAt:      firstNonEmpty(snapshot.ProducedAt, snapshot.Model.Metadata.DiscoveredAt),
			Kind:            contract.Kind,
			Contract:        contract,
			SourceType:      firstNonEmpty(snapshot.SourceType, snapshot.Model.Metadata.SourceType),
			SourceRef:       firstNonEmpty(snapshot.SourceRef, snapshot.Model.Metadata.SourceRef),
			TopologyVersion: firstNonEmpty(snapshot.TopologyVersion, snapshot.Model.Metadata.TopologyVersion),
		},
		Model:           snapshot.Model,
		Predicates:      preds,
		EndpointWeights: weights,
		PredicateSource: sourceOrDefault(len(preds) > 0, predicateSource),
		WeightsSource:   sourceOrDefault(len(weights) > 0, weightSource),
	}, nil
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
