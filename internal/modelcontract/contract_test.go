package modelcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVendoredSchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredSchema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != ExpectedSchemaDigest {
		t.Fatalf("vendored schema digest mismatch: got=%s want=%s", got, ExpectedSchemaDigest)
	}
}

func TestVendoredModelV110SchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredModelV110Schema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != BeringModelV110Digest {
		t.Fatalf("vendored model v1.1.0 schema digest mismatch: got=%s want=%s", got, BeringModelV110Digest)
	}
}

func TestVendoredSnapshotSchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredSnapshotSchema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != BeringSnapshotV100Digest {
		t.Fatalf("vendored snapshot schema digest mismatch: got=%s want=%s", got, BeringSnapshotV100Digest)
	}
}

func TestVendoredSnapshotV110SchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredSnapshotV110Schema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != BeringSnapshotV110Digest {
		t.Fatalf("vendored snapshot v1.1.0 schema digest mismatch: got=%s want=%s", got, BeringSnapshotV110Digest)
	}
}

func TestValidateStrict(t *testing.T) {
	t.Parallel()

	if err := ValidateStrict(ExpectedRef()); err != nil {
		t.Fatalf("expected strict validation to pass, got error: %v", err)
	}
}

func TestValidateStrictSnapshot(t *testing.T) {
	t.Parallel()

	if err := ValidateStrict(ExpectedSnapshotRef()); err != nil {
		t.Fatalf("expected strict snapshot validation to pass, got error: %v", err)
	}
}

func TestValidateStrictModelV110(t *testing.T) {
	t.Parallel()

	if err := ValidateStrict(ExpectedModelV110Ref()); err != nil {
		t.Fatalf("expected strict v1.1.0 model validation to pass, got error: %v", err)
	}
}

func TestValidateStrictSnapshotV110(t *testing.T) {
	t.Parallel()

	if err := ValidateStrict(ExpectedSnapshotV110Ref()); err != nil {
		t.Fatalf("expected strict v1.1.0 snapshot validation to pass, got error: %v", err)
	}
}

func TestValidateStrictRejectsURIMismatch(t *testing.T) {
	t.Parallel()

	err := ValidateStrict(SchemaRef{
		Name:    BeringModelV110Name,
		Version: BeringModelV110Version,
		URI:     "https://example.invalid/model.schema.json",
		Digest:  BeringModelV110Digest,
	})
	if err == nil {
		t.Fatal("expected uri mismatch to fail strict validation")
	}
}

func TestValidateStrictRejectsDigestMismatch(t *testing.T) {
	t.Parallel()

	err := ValidateStrict(SchemaRef{
		Name:    BeringSnapshotV110Name,
		Version: BeringSnapshotV110Version,
		URI:     BeringSnapshotV110URI,
		Digest:  "sha256:deadbeef",
	})
	if err == nil {
		t.Fatal("expected digest mismatch to fail strict validation")
	}
}
