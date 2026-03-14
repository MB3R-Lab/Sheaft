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

func TestVendoredSnapshotSchemaDigestMatchesPinned(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(VendoredSnapshotSchema))
	got := "sha256:" + hex.EncodeToString(sum[:])
	if got != BeringSnapshotV100Digest {
		t.Fatalf("vendored snapshot schema digest mismatch: got=%s want=%s", got, BeringSnapshotV100Digest)
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
