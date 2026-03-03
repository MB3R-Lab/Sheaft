package modelcontract

import (
	_ "embed"
	"fmt"
	"strings"
)

const (
	ExpectedSchemaName    = "io.mb3r.bering.model"
	ExpectedSchemaVersion = "1.0.0"
	ExpectedSchemaURI     = "https://schemas.mb3r.dev/bering/model/v1.0.0/model.schema.json"
	ExpectedSchemaDigest  = "sha256:7dc733936a9d3f94ab92f46a30d4c8d0f5c05d60670c4247786c59a3fe7630f7"
)

//go:embed schema/model.schema.json
var VendoredSchema string

type SchemaRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URI     string `json:"uri"`
	Digest  string `json:"digest"`
}

func ExpectedRef() SchemaRef {
	return SchemaRef{
		Name:    ExpectedSchemaName,
		Version: ExpectedSchemaVersion,
		URI:     ExpectedSchemaURI,
		Digest:  ExpectedSchemaDigest,
	}
}

func ValidateStrict(schema SchemaRef) error {
	if strings.TrimSpace(schema.Name) == "" {
		return fmt.Errorf("metadata.schema.name cannot be empty")
	}
	if strings.TrimSpace(schema.Version) == "" {
		return fmt.Errorf("metadata.schema.version cannot be empty")
	}
	if strings.TrimSpace(schema.URI) == "" {
		return fmt.Errorf("metadata.schema.uri cannot be empty")
	}
	if strings.TrimSpace(schema.Digest) == "" {
		return fmt.Errorf("metadata.schema.digest cannot be empty")
	}

	if schema.Name != ExpectedSchemaName {
		return fmt.Errorf("schema name mismatch: got %q want %q", schema.Name, ExpectedSchemaName)
	}
	if schema.Version != ExpectedSchemaVersion {
		return fmt.Errorf("schema version mismatch: got %q want %q", schema.Version, ExpectedSchemaVersion)
	}
	if schema.URI != ExpectedSchemaURI {
		return fmt.Errorf("schema uri mismatch: got %q want %q", schema.URI, ExpectedSchemaURI)
	}
	if schema.Digest != ExpectedSchemaDigest {
		return fmt.Errorf("schema digest mismatch: got %q want %q", schema.Digest, ExpectedSchemaDigest)
	}
	return nil
}
