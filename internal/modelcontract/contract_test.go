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

func TestVendoredModelV130SchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredModelV130Schema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != BeringModelV130Digest {
		t.Fatalf("vendored model v1.3.0 schema digest mismatch: got=%s want=%s", got, BeringModelV130Digest)
	}
}

func TestVendoredSnapshotSchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredSnapshotSchema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != BeringSnapshotV130Digest {
		t.Fatalf("vendored snapshot schema digest mismatch: got=%s want=%s", got, BeringSnapshotV130Digest)
	}
}

func TestVendoredSnapshotV130SchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredSnapshotV130Schema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != BeringSnapshotV130Digest {
		t.Fatalf("vendored snapshot v1.3.0 schema digest mismatch: got=%s want=%s", got, BeringSnapshotV130Digest)
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

func TestValidateStrictModelV130(t *testing.T) {
	t.Parallel()

	if err := ValidateStrict(ExpectedModelV130Ref()); err != nil {
		t.Fatalf("expected strict v1.3.0 model validation to pass, got error: %v", err)
	}
}

func TestValidateStrictSnapshotV130(t *testing.T) {
	t.Parallel()

	if err := ValidateStrict(ExpectedSnapshotV130Ref()); err != nil {
		t.Fatalf("expected strict v1.3.0 snapshot validation to pass, got error: %v", err)
	}
}

func TestValidateStrictRejectsURIMismatch(t *testing.T) {
	t.Parallel()

	err := ValidateStrict(SchemaRef{
		Name:    BeringModelV130Name,
		Version: BeringModelV130Version,
		URI:     "https://example.invalid/model.schema.json",
		Digest:  BeringModelV130Digest,
	})
	if err == nil {
		t.Fatal("expected uri mismatch to fail strict validation")
	}
}

func TestValidateStrictRejectsDigestMismatch(t *testing.T) {
	t.Parallel()

	err := ValidateStrict(SchemaRef{
		Name:    BeringSnapshotV130Name,
		Version: BeringSnapshotV130Version,
		URI:     BeringSnapshotV130URI,
		Digest:  "sha256:deadbeef",
	})
	if err == nil {
		t.Fatal("expected digest mismatch to fail strict validation")
	}
}
