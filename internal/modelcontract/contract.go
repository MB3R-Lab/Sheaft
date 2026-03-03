package modelcontract

import (
	_ "embed"
	"fmt"
	"strings"
)

const (
	ExpectedSchemaName    = "io.mb3r.bering.model"
	ExpectedSchemaVersion = "1.0.0"
	ExpectedSchemaURI     = "https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json"
	ExpectedSchemaDigest  = "sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7"
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
