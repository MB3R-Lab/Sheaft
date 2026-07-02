package modelcontract

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

const (
	BeringModelName        = "io.mb3r.bering.model"
	BeringModelV130Name    = BeringModelName
	BeringModelV130Version = "1.3.0"
	BeringModelV130URI     = "https://mb3r-lab.github.io/Bering/schema/model/v1.3.0/model.schema.json"
	BeringModelV130Digest  = "sha256:2aa8a3550a25dc626ba6d2f5833569efca2f382b9e5c9c3405be93695d7d48ae"

	BeringSnapshotName        = "io.mb3r.bering.snapshot"
	BeringSnapshotV130Name    = BeringSnapshotName
	BeringSnapshotV130Version = "1.3.0"
	BeringSnapshotV130URI     = "https://mb3r-lab.github.io/Bering/schema/snapshot/v1.3.0/snapshot.schema.json"
	BeringSnapshotV130Digest  = "sha256:cb778e5b0866d9ce5cfe7f23b8d98a339603593a0247cccd9cddaf05c7ae4bb1"

	ExpectedSchemaName    = BeringModelV130Name
	ExpectedSchemaVersion = BeringModelV130Version
	ExpectedSchemaURI     = BeringModelV130URI
	ExpectedSchemaDigest  = BeringModelV130Digest
)

//go:embed schema/model.v1.3.0.schema.json
var VendoredSchema string

//go:embed schema/model.v1.3.0.schema.json
var VendoredModelV130Schema string

//go:embed schema/snapshot.v1.3.0.schema.json
var VendoredSnapshotSchema string

//go:embed schema/snapshot.v1.3.0.schema.json
var VendoredSnapshotV130Schema string

type ArtifactKind string

const (
	KindModel    ArtifactKind = "model"
	KindSnapshot ArtifactKind = "snapshot"
)

type SchemaRef struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	URI     string `json:"uri" yaml:"uri"`
	Digest  string `json:"digest" yaml:"digest"`
}

type SupportedContract struct {
	Name    string
	Version string
	URI     string
	Digest  string
	Kind    ArtifactKind
}

var supportedContracts = []SupportedContract{
	{
		Name:    BeringModelV130Name,
		Version: BeringModelV130Version,
		URI:     BeringModelV130URI,
		Digest:  BeringModelV130Digest,
		Kind:    KindModel,
	},
	{
		Name:    BeringSnapshotV130Name,
		Version: BeringSnapshotV130Version,
		URI:     BeringSnapshotV130URI,
		Digest:  BeringSnapshotV130Digest,
		Kind:    KindSnapshot,
	},
}

func Supported() []SupportedContract {
	return slices.Clone(supportedContracts)
}

func ExpectedRef() SchemaRef {
	return SchemaRef{
		Name:    BeringModelV130Name,
		Version: BeringModelV130Version,
		URI:     BeringModelV130URI,
		Digest:  BeringModelV130Digest,
	}
}

func ExpectedSnapshotRef() SchemaRef {
	return SchemaRef{
		Name:    BeringSnapshotV130Name,
		Version: BeringSnapshotV130Version,
		URI:     BeringSnapshotV130URI,
		Digest:  BeringSnapshotV130Digest,
	}
}

func ExpectedModelV130Ref() SchemaRef {
	return SchemaRef{
		Name:    BeringModelV130Name,
		Version: BeringModelV130Version,
		URI:     BeringModelV130URI,
		Digest:  BeringModelV130Digest,
	}
}

func ExpectedSnapshotV130Ref() SchemaRef {
	return SchemaRef{
		Name:    BeringSnapshotV130Name,
		Version: BeringSnapshotV130Version,
		URI:     BeringSnapshotV130URI,
		Digest:  BeringSnapshotV130Digest,
	}
}

func ValidateStrict(schema SchemaRef) error {
	_, err := Resolve(schema)
	return err
}

func Resolve(schema SchemaRef) (SupportedContract, error) {
	if strings.TrimSpace(schema.Name) == "" {
		return SupportedContract{}, fmt.Errorf("metadata.schema.name cannot be empty")
	}
	if strings.TrimSpace(schema.Version) == "" {
		return SupportedContract{}, fmt.Errorf("metadata.schema.version cannot be empty")
	}
	if strings.TrimSpace(schema.URI) == "" {
		return SupportedContract{}, fmt.Errorf("metadata.schema.uri cannot be empty")
	}
	if strings.TrimSpace(schema.Digest) == "" {
		return SupportedContract{}, fmt.Errorf("metadata.schema.digest cannot be empty")
	}

	for _, contract := range supportedContracts {
		if schema.Name != contract.Name || schema.Version != contract.Version {
			continue
		}
		if schema.URI != contract.URI {
			return SupportedContract{}, fmt.Errorf(
				"unsupported %s@%s: uri mismatch: got %q want %q",
				schema.Name,
				schema.Version,
				schema.URI,
				contract.URI,
			)
		}
		if schema.Digest != contract.Digest {
			return SupportedContract{}, fmt.Errorf(
				"unsupported %s@%s: digest mismatch: got %q want %q",
				schema.Name,
				schema.Version,
				schema.Digest,
				contract.Digest,
			)
		}
		return contract, nil
	}

	supported := make([]string, 0, len(supportedContracts))
	for _, contract := range supportedContracts {
		supported = append(supported, fmt.Sprintf("%s@%s", contract.Name, contract.Version))
	}
	slices.Sort(supported)
	return SupportedContract{}, fmt.Errorf(
		"unsupported contract %s@%s; supported contracts: %s",
		schema.Name,
		schema.Version,
		strings.Join(supported, ", "),
	)
}
