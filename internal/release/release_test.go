package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompatibilityManifestMatchesTrackedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", DefaultCompatibilityManifestPath)
	manifest, err := ReadCompatibilityManifest(path)
	if err != nil {
		t.Fatalf("read tracked compatibility manifest: %v", err)
	}

	if manifest.ProductName != ProductName {
		t.Fatalf("unexpected product name: %s", manifest.ProductName)
	}
}

func TestValidateChart(t *testing.T) {
	t.Parallel()

	if err := ValidateChart(filepath.Join("..", "..", "charts", "sheaft")); err != nil {
		t.Fatalf("validate chart: %v", err)
	}
}

func TestReleaseManifestValidationRejectsMissingDigestForPublishedImage(t *testing.T) {
	t.Parallel()

	manifest := ReleaseManifest{
		SchemaVersion: ReleaseManifestSchemaVersion,
		ProductName:   ProductName,
		AppVersion:    "1.0.0",
		GitCommit:     "deadbeef",
		BuildDate:     "2026-03-11T00:00:00Z",
		Binaries: []ReleaseBinary{
			{
				OS:   "linux",
				Arch: "amd64",
				Archive: AssetReference{
					Path:   "dist/example.tar.gz",
					SHA256: "abc",
				},
			},
		},
		ChecksumsFile: AssetReference{
			Path:   "dist/checksums.txt",
			SHA256: "abc",
		},
		OCIImages: []OCIImageMetadata{
			{
				Repository: "ghcr.io/example/sheaft",
				Tags:       []string{"v1.0.0"},
				Published:  true,
				References: []ImageReference{{Reference: "ghcr.io/example/sheaft:v1.0.0"}},
			},
		},
		Chart: ChartMetadata{
			Name:    "sheaft",
			Version: "1.0.0",
			Archive: AssetReference{
				Path:   "dist/sheaft-1.0.0.tgz",
				SHA256: "abc",
			},
		},
		CompatibilityManifest: AssetReference{
			Path:   "compatibility-manifest.json",
			SHA256: "abc",
		},
		DefaultConfigPacks: []DefaultConfigPackMetadata{
			{
				Name:    "default-config-pack",
				Version: "1.0.0",
				Archive: AssetReference{
					Path:   "dist/default-pack.tar.gz",
					SHA256: "abc",
				},
				Files: []string{"configs/analysis.example.yaml"},
			},
		},
	}

	if err := manifest.Validate(); err == nil {
		t.Fatal("expected validation error for published image without digest")
	}
}

func TestPackageDefaultConfigPackSupportsAbsoluteOutputPaths(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join("..", "..")
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "default-pack.tar.gz")
	metadataPath := filepath.Join(tempDir, "default-pack.json")

	if err := PackageDefaultConfigPack(repoRoot, filepath.Join(repoRoot, DefaultConfigPackSourceListPath), "0.0.0-test", archivePath, metadataPath); err != nil {
		t.Fatalf("package default config pack: %v", err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("stat archive: %v", err)
	}
	if _, err := os.Stat(metadataPath); err != nil {
		t.Fatalf("stat metadata: %v", err)
	}
}
