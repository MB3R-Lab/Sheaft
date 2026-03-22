package modelcontract

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

const (
	BeringModelName        = "io.mb3r.bering.model"
	BeringModelV100Name    = BeringModelName
	BeringModelV100Version = "1.0.0"
	BeringModelV100URI     = "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json"
	BeringModelV100Digest  = "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7"

	BeringModelV110Name    = BeringModelName
	BeringModelV110Version = "1.1.0"
	BeringModelV110URI     = "https://mb3r-lab.github.io/Bering/schema/model/v1.1.0/model.schema.json"
	BeringModelV110Digest  = "sha256:bc9a60736c9e6bda9599243fd68f293b88f42ade65321d8267369a5c3214779a"

	BeringSnapshotName        = "io.mb3r.bering.snapshot"
	BeringSnapshotV100Name    = BeringSnapshotName
	BeringSnapshotV100Version = "1.0.0"
	BeringSnapshotV100URI     = "https://mb3r-lab.github.io/Bering/schema/snapshot/v1.0.0/snapshot.schema.json"
	BeringSnapshotV100Digest  = "sha256:87e4e887ed4a37b72f6136e268b73552eccb92941c4de2c6f3a514dd066ea972"

	BeringSnapshotV110Name    = BeringSnapshotName
	BeringSnapshotV110Version = "1.1.0"
	BeringSnapshotV110URI     = "https://mb3r-lab.github.io/Bering/schema/snapshot/v1.1.0/snapshot.schema.json"
	BeringSnapshotV110Digest  = "sha256:53b127608b2aaa4fabb352b998cd6b2c5ed558764729a09abea56f4f9b40fa01"

	ExpectedSchemaName    = BeringModelV100Name
	ExpectedSchemaVersion = BeringModelV100Version
	ExpectedSchemaURI     = BeringModelV100URI
	ExpectedSchemaDigest  = BeringModelV100Digest
)

//go:embed schema/model.v1.0.0.schema.json
var VendoredSchema string

//go:embed schema/model.v1.1.0.schema.json
var VendoredModelV110Schema string

//go:embed schema/snapshot.v1.0.0.schema.json
var VendoredSnapshotSchema string

//go:embed schema/snapshot.v1.1.0.schema.json
var VendoredSnapshotV110Schema string

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
		Name:    BeringModelV100Name,
		Version: BeringModelV100Version,
		URI:     BeringModelV100URI,
		Digest:  BeringModelV100Digest,
		Kind:    KindModel,
	},
	{
		Name:    BeringModelV110Name,
		Version: BeringModelV110Version,
		URI:     BeringModelV110URI,
		Digest:  BeringModelV110Digest,
		Kind:    KindModel,
	},
	{
		Name:    BeringSnapshotV100Name,
		Version: BeringSnapshotV100Version,
		URI:     BeringSnapshotV100URI,
		Digest:  BeringSnapshotV100Digest,
		Kind:    KindSnapshot,
	},
	{
		Name:    BeringSnapshotV110Name,
		Version: BeringSnapshotV110Version,
		URI:     BeringSnapshotV110URI,
		Digest:  BeringSnapshotV110Digest,
		Kind:    KindSnapshot,
	},
}

func Supported() []SupportedContract {
	return slices.Clone(supportedContracts)
}

func ExpectedRef() SchemaRef {
	return SchemaRef{
		Name:    BeringModelV100Name,
		Version: BeringModelV100Version,
		URI:     BeringModelV100URI,
		Digest:  BeringModelV100Digest,
	}
}

func ExpectedSnapshotRef() SchemaRef {
	return SchemaRef{
		Name:    BeringSnapshotV100Name,
		Version: BeringSnapshotV100Version,
		URI:     BeringSnapshotV100URI,
		Digest:  BeringSnapshotV100Digest,
	}
}

func ExpectedModelV110Ref() SchemaRef {
	return SchemaRef{
		Name:    BeringModelV110Name,
		Version: BeringModelV110Version,
		URI:     BeringModelV110URI,
		Digest:  BeringModelV110Digest,
	}
}

func ExpectedSnapshotV110Ref() SchemaRef {
	return SchemaRef{
		Name:    BeringSnapshotV110Name,
		Version: BeringSnapshotV110Version,
		URI:     BeringSnapshotV110URI,
		Digest:  BeringSnapshotV110Digest,
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
