package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

const (
	ProductName                         = "Sheaft"
	UpstreamProducer                    = "Bering"
	CompatibilityManifestSchemaVersion  = "1.0.0"
	ReleaseManifestSchemaVersion        = "1.0.0"
	DefaultCompatibilityManifestPath    = "compatibility-manifest.json"
	DefaultReleaseManifestPath          = "release-manifest.json"
	DefaultCompatibilitySchemaPath      = "api/schema/compatibility-manifest.schema.json"
	DefaultReleaseSchemaPath            = "api/schema/release-manifest.schema.json"
	DefaultConfigPackSourceListPath     = "release/packs/default-config-pack.files.txt"
	DefaultConfigPackMetadataOutputPath = "dist/default-config-pack.json"
)

type CompatibilityContract struct {
	ArtifactType  string `json:"artifact_type"`
	SchemaName    string `json:"schema_name"`
	SchemaVersion string `json:"schema_version"`
	SchemaURI     string `json:"schema_uri"`
	SchemaDigest  string `json:"schema_digest"`
}

type CompatibilityManifest struct {
	Schema                          string                  `json:"$schema"`
	SchemaVersion                   string                  `json:"schema_version"`
	ProductName                     string                  `json:"product_name"`
	DownstreamConsumerOf            string                  `json:"downstream_consumer_of"`
	StrictContractValidation        bool                    `json:"strict_contract_validation"`
	SupportedUpstreamArtifactTypes  []string                `json:"supported_upstream_artifact_types"`
	SupportedUpstreamSchemaNames    []string                `json:"supported_upstream_schema_names"`
	SupportedUpstreamSchemaVersions []string                `json:"supported_upstream_schema_versions"`
	SupportedContracts              []CompatibilityContract `json:"supported_contracts"`
	TestedBeringAppVersions         []string                `json:"tested_bering_app_versions"`
}

func GenerateCompatibilityManifest() CompatibilityManifest {
	contracts := modelcontract.Supported()
	artifactTypes := make([]string, 0, len(contracts))
	schemaNames := make([]string, 0, len(contracts))
	schemaVersions := make([]string, 0, len(contracts))
	supportedContracts := make([]CompatibilityContract, 0, len(contracts))

	seenArtifactTypes := map[string]struct{}{}
	seenSchemaNames := map[string]struct{}{}
	seenSchemaVersions := map[string]struct{}{}

	for _, contract := range contracts {
		artifactType := string(contract.Kind)
		if _, ok := seenArtifactTypes[artifactType]; !ok {
			seenArtifactTypes[artifactType] = struct{}{}
			artifactTypes = append(artifactTypes, artifactType)
		}
		if _, ok := seenSchemaNames[contract.Name]; !ok {
			seenSchemaNames[contract.Name] = struct{}{}
			schemaNames = append(schemaNames, contract.Name)
		}
		if _, ok := seenSchemaVersions[contract.Version]; !ok {
			seenSchemaVersions[contract.Version] = struct{}{}
			schemaVersions = append(schemaVersions, contract.Version)
		}
		supportedContracts = append(supportedContracts, CompatibilityContract{
			ArtifactType:  artifactType,
			SchemaName:    contract.Name,
			SchemaVersion: contract.Version,
			SchemaURI:     contract.URI,
			SchemaDigest:  contract.Digest,
		})
	}

	slices.Sort(artifactTypes)
	slices.Sort(schemaNames)
	slices.Sort(schemaVersions)
	slices.SortFunc(supportedContracts, func(a, b CompatibilityContract) int {
		left := a.ArtifactType + "\x00" + a.SchemaName + "\x00" + a.SchemaVersion
		right := b.ArtifactType + "\x00" + b.SchemaName + "\x00" + b.SchemaVersion
		return strings.Compare(left, right)
	})

	return CompatibilityManifest{
		Schema:                          DefaultCompatibilitySchemaPath,
		SchemaVersion:                   CompatibilityManifestSchemaVersion,
		ProductName:                     ProductName,
		DownstreamConsumerOf:            UpstreamProducer,
		StrictContractValidation:        true,
		SupportedUpstreamArtifactTypes:  artifactTypes,
		SupportedUpstreamSchemaNames:    schemaNames,
		SupportedUpstreamSchemaVersions: schemaVersions,
		SupportedContracts:              supportedContracts,
		TestedBeringAppVersions:         []string{},
	}
}

func (m CompatibilityManifest) Validate() error {
	if m.SchemaVersion != CompatibilityManifestSchemaVersion {
		return fmt.Errorf("compatibility manifest schema_version must be %q", CompatibilityManifestSchemaVersion)
	}
	if strings.TrimSpace(m.ProductName) != ProductName {
		return fmt.Errorf("compatibility manifest product_name must be %q", ProductName)
	}
	if strings.TrimSpace(m.DownstreamConsumerOf) != UpstreamProducer {
		return fmt.Errorf("compatibility manifest downstream_consumer_of must be %q", UpstreamProducer)
	}
	if !m.StrictContractValidation {
		return fmt.Errorf("compatibility manifest strict_contract_validation must stay true")
	}
	if len(m.SupportedContracts) == 0 {
		return fmt.Errorf("compatibility manifest must declare at least one supported contract")
	}
	for _, contract := range m.SupportedContracts {
		if strings.TrimSpace(contract.ArtifactType) == "" {
			return fmt.Errorf("compatibility manifest contract artifact_type cannot be empty")
		}
		if strings.TrimSpace(contract.SchemaName) == "" {
			return fmt.Errorf("compatibility manifest contract schema_name cannot be empty")
		}
		if strings.TrimSpace(contract.SchemaVersion) == "" {
			return fmt.Errorf("compatibility manifest contract schema_version cannot be empty")
		}
		if strings.TrimSpace(contract.SchemaURI) == "" {
			return fmt.Errorf("compatibility manifest contract schema_uri cannot be empty")
		}
		if strings.TrimSpace(contract.SchemaDigest) == "" {
			return fmt.Errorf("compatibility manifest contract schema_digest cannot be empty")
		}
	}

	expected := GenerateCompatibilityManifest()
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return fmt.Errorf("marshal expected compatibility manifest: %w", err)
	}
	actualJSON, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal actual compatibility manifest: %w", err)
	}
	if string(actualJSON) != string(expectedJSON) {
		return fmt.Errorf("compatibility manifest drifts from internal/modelcontract strict pins")
	}
	return nil
}

func ReadCompatibilityManifest(path string) (CompatibilityManifest, error) {
	var manifest CompatibilityManifest
	if err := readJSON(path, &manifest); err != nil {
		return CompatibilityManifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return CompatibilityManifest{}, err
	}
	return manifest, nil
}

func WriteCompatibilityManifest(path string) error {
	manifest := GenerateCompatibilityManifest()
	if err := manifest.Validate(); err != nil {
		return err
	}
	return writeJSON(path, manifest)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
