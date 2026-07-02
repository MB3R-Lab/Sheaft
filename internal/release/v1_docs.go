package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

const DefaultV1SemanticsDocPath = "docs/v1-major-semantics.md"

type v1SampleModel struct {
	Services  []map[string]any `json:"services"`
	Edges     []map[string]any `json:"edges"`
	Endpoints []map[string]any `json:"endpoints"`
	Metadata  struct {
		Schema modelcontract.SchemaRef `json:"schema"`
	} `json:"metadata"`
}

type v1SampleSnapshot struct {
	Discovery struct {
		Services  []map[string]any `json:"services"`
		Edges     []map[string]any `json:"edges"`
		Endpoints []map[string]any `json:"endpoints"`
	} `json:"discovery"`
	Model    v1SampleModel `json:"model"`
	Metadata struct {
		Schema modelcontract.SchemaRef `json:"schema"`
	} `json:"metadata"`
}

func ValidateV1ReleaseDocs(root string) error {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	requiredPaths := []string{
		DefaultV1SemanticsDocPath,
		"README.md",
		"RELEASING.md",
		"CHANGELOG.md",
		"docs/assumptions-and-limitations.md",
		"docs/compatibility-matrix.md",
		"docs/configuration.md",
		"docs/consumer-semantics-v2.md",
		"docs/methodology.md",
		"docs/oracle-suite.md",
		"docs/release-assets.md",
		"api/schema/analysis.schema.json",
		"api/schema/model.v1.3.0.schema.json",
		"api/schema/oracle-report.schema.json",
		"api/schema/predicate-contract.schema.json",
		"api/schema/snapshot.v1.3.0.schema.json",
		"configs/analysis.v1.1.example.yaml",
		"configs/fault-contract.example.yaml",
		"configs/predicate-contract.example.yaml",
		"examples/outputs/model-v1.3.0.sample.json",
		"examples/outputs/snapshot-v1.3.0.sample.json",
		"compatibility-manifest.json",
	}
	for _, path := range requiredPaths {
		if err := requireExistingPath(root, path); err != nil {
			return err
		}
	}

	v1Doc, err := readReleaseText(root, DefaultV1SemanticsDocPath)
	if err != nil {
		return err
	}
	if err := requireContains(DefaultV1SemanticsDocPath, v1Doc, []string{
		"G=(V,E,tau)",
		"replication map `R`",
		"P_Node * P_Edge",
		"endpoint success predicates `Phi`",
		"`theta`",
		"`rho`",
		"immediate_response",
		"eventual_completion",
		"failure_probability",
		"service-only predicates",
		"automatic probability calibration",
		"arbitrary non-product `P`",
		"live chaos execution",
		"rich temporal workflow models",
		"oracle suite",
		"validate-v1-release-docs",
	}); err != nil {
		return err
	}

	for _, path := range []string{
		"api/schema/model.v1.3.0.schema.json",
		"api/schema/snapshot.v1.3.0.schema.json",
		"api/schema/predicate-contract.schema.json",
		"api/schema/oracle-report.schema.json",
		"configs/analysis.v1.1.example.yaml",
		"configs/predicate-contract.example.yaml",
		"examples/outputs/model-v1.3.0.sample.json",
		"examples/outputs/snapshot-v1.3.0.sample.json",
	} {
		if !strings.Contains(v1Doc, path) {
			return fmt.Errorf("%s must reference %s", DefaultV1SemanticsDocPath, path)
		}
	}

	if err := validateV1LinkedDocs(root, v1Doc); err != nil {
		return err
	}
	if err := validateV1CompatibilityDocs(root); err != nil {
		return err
	}
	if err := validateV1Examples(root); err != nil {
		return err
	}
	if err := validateV1DefaultPack(root); err != nil {
		return err
	}
	if err := validateV1Makefile(root); err != nil {
		return err
	}
	return nil
}

func validateV1LinkedDocs(root string, v1Doc string) error {
	readme, err := readReleaseText(root, "README.md")
	if err != nil {
		return err
	}
	if err := requireContains("README.md", readme, []string{"docs/v1-major-semantics.md", "P_Node * P_Edge"}); err != nil {
		return err
	}

	releasing, err := readReleaseText(root, "RELEASING.md")
	if err != nil {
		return err
	}
	if err := requireContains("RELEASING.md", releasing, []string{
		"validate-v1-release-docs",
		"docs/v1-major-semantics.md",
		"automatic probability calibration",
		"arbitrary non-product `P`",
		"live chaos execution",
		"rich temporal workflow models",
	}); err != nil {
		return err
	}

	releaseAssets, err := readReleaseText(root, "docs/release-assets.md")
	if err != nil {
		return err
	}
	if err := requireContains("docs/release-assets.md", releaseAssets, []string{
		"V1 Major Release Checklist",
		"validate-v1-release-docs",
		"schema sync",
		"fixed benchmark slice",
		"synthetic oracle suite",
	}); err != nil {
		return err
	}

	assumptions, err := readReleaseText(root, "docs/assumptions-and-limitations.md")
	if err != nil {
		return err
	}
	if err := requireContains("docs/assumptions-and-limitations.md", assumptions, []string{
		"automatic probability calibration",
		"arbitrary non-product `P`",
		"P_Node * P_Edge",
		"rich temporal workflow models",
	}); err != nil {
		return err
	}

	if !strings.Contains(v1Doc, "Pre-v1 Bering preview lines `1.0.0`, `1.1.0`, and `1.2.0` are retired") {
		return fmt.Errorf("%s must document retired pre-v1 Bering contract lines", DefaultV1SemanticsDocPath)
	}
	return nil
}

func validateV1CompatibilityDocs(root string) error {
	matrix, err := readReleaseText(root, "docs/compatibility-matrix.md")
	if err != nil {
		return err
	}
	manifestText, err := readReleaseText(root, DefaultCompatibilityManifestPath)
	if err != nil {
		return err
	}
	v1Doc, err := readReleaseText(root, DefaultV1SemanticsDocPath)
	if err != nil {
		return err
	}

	for _, contract := range modelcontract.Supported() {
		needles := []string{
			fmt.Sprintf("%s@%s", contract.Name, contract.Version),
			contract.URI,
			contract.Digest,
		}
		if err := requireContains("docs/compatibility-matrix.md", matrix, needles); err != nil {
			return err
		}
		if err := requireContains(DefaultCompatibilityManifestPath, manifestText, []string{
			contract.Name,
			contract.Version,
			contract.URI,
			contract.Digest,
		}); err != nil {
			return err
		}
	}
	for _, contract := range []modelcontract.SupportedContract{
		{
			Name:    modelcontract.BeringModelV130Name,
			Version: modelcontract.BeringModelV130Version,
			URI:     modelcontract.BeringModelV130URI,
			Digest:  modelcontract.BeringModelV130Digest,
		},
		{
			Name:    modelcontract.BeringSnapshotV130Name,
			Version: modelcontract.BeringSnapshotV130Version,
			URI:     modelcontract.BeringSnapshotV130URI,
			Digest:  modelcontract.BeringSnapshotV130Digest,
		},
	} {
		if err := requireContains(DefaultV1SemanticsDocPath, v1Doc, []string{
			fmt.Sprintf("%s@%s", contract.Name, contract.Version),
			contract.URI,
			contract.Digest,
		}); err != nil {
			return err
		}
	}
	if err := requireContains("docs/compatibility-matrix.md", matrix, []string{"Bering v1 contract line", "v1-major-semantics.md"}); err != nil {
		return err
	}
	return nil
}

func validateV1Examples(root string) error {
	var sample v1SampleModel
	if err := readReleaseJSON(root, "examples/outputs/model-v1.3.0.sample.json", &sample); err != nil {
		return err
	}
	if err := requireSchemaRef("model-v1.3.0 sample", sample.Metadata.Schema, modelcontract.ExpectedModelV130Ref()); err != nil {
		return err
	}
	if err := requireV1ExampleFeatures("model-v1.3.0 sample", sample.Services, sample.Edges, sample.Endpoints); err != nil {
		return err
	}

	var snapshot v1SampleSnapshot
	if err := readReleaseJSON(root, "examples/outputs/snapshot-v1.3.0.sample.json", &snapshot); err != nil {
		return err
	}
	if err := requireSchemaRef("snapshot-v1.3.0 sample", snapshot.Metadata.Schema, modelcontract.ExpectedSnapshotV130Ref()); err != nil {
		return err
	}
	if err := requireSchemaRef("snapshot-v1.3.0 embedded model", snapshot.Model.Metadata.Schema, modelcontract.ExpectedModelV130Ref()); err != nil {
		return err
	}
	if err := requireV1ExampleFeatures("snapshot-v1.3.0 embedded model", snapshot.Model.Services, snapshot.Model.Edges, snapshot.Model.Endpoints); err != nil {
		return err
	}
	if err := requireV1ExampleFeatures("snapshot-v1.3.0 discovery", snapshot.Discovery.Services, snapshot.Discovery.Edges, snapshot.Discovery.Endpoints); err != nil {
		return err
	}
	return nil
}

func validateV1DefaultPack(root string) error {
	pack, err := readReleaseText(root, DefaultConfigPackSourceListPath)
	if err != nil {
		return err
	}
	return requireContains(DefaultConfigPackSourceListPath, pack, []string{
		"configs/analysis.v1.1.example.yaml",
		"configs/fault-contract.example.yaml",
		"configs/predicate-contract.example.yaml",
		"examples/outputs/model-v1.3.0.sample.json",
		"examples/outputs/snapshot-v1.3.0.sample.json",
		"benchmarks/fixed-slice/manifest.json",
	})
}

func validateV1Makefile(root string) error {
	makefile, err := readReleaseText(root, "Makefile")
	if err != nil {
		return err
	}
	return requireContains("Makefile", makefile, []string{
		"validate-v1-release-docs",
		"release-dry-run:",
	})
}

func requireV1ExampleFeatures(label string, services, edges, endpoints []map[string]any) error {
	if !hasReliabilityEvidence(services) {
		return fmt.Errorf("%s must include service reliability evidence", label)
	}
	if !hasPlacementReliability(services) {
		return fmt.Errorf("%s must include placement reliability evidence", label)
	}
	if !hasEdgeIdentity(edges) {
		return fmt.Errorf("%s must include operation-aware edge identity", label)
	}
	if !hasReliabilityEvidence(edges) {
		return fmt.Errorf("%s must include edge reliability evidence", label)
	}
	modes := endpointSemanticModes(endpoints)
	if !modes["immediate_response"] {
		return fmt.Errorf("%s must include immediate_response endpoint semantics", label)
	}
	if !modes["eventual_completion"] {
		return fmt.Errorf("%s must include eventual_completion endpoint semantics", label)
	}
	return nil
}

func requireSchemaRef(label string, got modelcontract.SchemaRef, want modelcontract.SchemaRef) error {
	if got.Name != want.Name || got.Version != want.Version || got.URI != want.URI || got.Digest != want.Digest {
		return fmt.Errorf("%s schema mismatch: got %s@%s %s %s want %s@%s %s %s",
			label,
			got.Name,
			got.Version,
			got.URI,
			got.Digest,
			want.Name,
			want.Version,
			want.URI,
			want.Digest,
		)
	}
	return nil
}

func hasReliabilityEvidence(items []map[string]any) bool {
	for _, item := range items {
		if hasNestedMap(item, "metadata", "reliability") {
			return true
		}
	}
	return false
}

func hasPlacementReliability(services []map[string]any) bool {
	for _, service := range services {
		metadata, ok := service["metadata"].(map[string]any)
		if !ok {
			continue
		}
		placements, ok := metadata["placements"].([]any)
		if !ok {
			continue
		}
		for _, raw := range placements {
			placement, ok := raw.(map[string]any)
			if ok && hasNestedMap(placement, "reliability") {
				return true
			}
		}
	}
	return false
}

func hasEdgeIdentity(edges []map[string]any) bool {
	for _, edge := range edges {
		identity, ok := edge["identity"].(map[string]any)
		if !ok {
			continue
		}
		protocol, _ := identity["protocol"].(string)
		if strings.TrimSpace(protocol) == "" {
			continue
		}
		if operation, _ := identity["operation"].(string); strings.TrimSpace(operation) != "" {
			return true
		}
		if route, _ := identity["route"].(string); strings.TrimSpace(route) != "" {
			return true
		}
		if topic, _ := identity["topic"].(string); strings.TrimSpace(topic) != "" {
			return true
		}
	}
	return false
}

func endpointSemanticModes(endpoints []map[string]any) map[string]bool {
	out := map[string]bool{}
	for _, endpoint := range endpoints {
		metadata, ok := endpoint["metadata"].(map[string]any)
		if !ok {
			continue
		}
		semantics, ok := metadata["semantics"].(map[string]any)
		if !ok {
			continue
		}
		mode, _ := semantics["predicate_mode"].(string)
		if strings.TrimSpace(mode) != "" {
			out[mode] = true
		}
	}
	return out
}

func hasNestedMap(item map[string]any, path ...string) bool {
	current := any(item)
	for _, part := range path {
		asMap, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = asMap[part]
		if !ok {
			return false
		}
	}
	_, ok := current.(map[string]any)
	return ok
}

func requireExistingPath(root, path string) error {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
		return fmt.Errorf("required v1 release path %s: %w", path, err)
	}
	return nil
}

func requireContains(path, content string, needles []string) error {
	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			return fmt.Errorf("%s must contain %q", path, needle)
		}
	}
	return nil
}

func readReleaseText(root, path string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(raw), nil
}

func readReleaseJSON(root, path string, value any) error {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, value); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
