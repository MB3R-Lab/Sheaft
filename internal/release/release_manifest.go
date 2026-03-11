package release

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type AssetReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type ReleaseBinary struct {
	OS      string          `json:"os"`
	Arch    string          `json:"arch"`
	Archive AssetReference  `json:"archive"`
	SBOM    *AssetReference `json:"sbom,omitempty"`
}

type ImageReference struct {
	Reference string `json:"reference"`
}

type OCIImageMetadata struct {
	SchemaVersion string           `json:"schema_version"`
	Repository    string           `json:"repository"`
	Tags          []string         `json:"tags"`
	Platforms     []string         `json:"platforms"`
	Published     bool             `json:"published"`
	Digest        string           `json:"digest,omitempty"`
	References    []ImageReference `json:"references"`
}

type ChartMetadata struct {
	SchemaVersion string         `json:"schema_version"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Published     bool           `json:"published"`
	Archive       AssetReference `json:"archive"`
	OCIReference  string         `json:"oci_reference,omitempty"`
	Digest        string         `json:"digest,omitempty"`
}

type DefaultConfigPackMetadata struct {
	SchemaVersion string         `json:"schema_version"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Archive       AssetReference `json:"archive"`
	Files         []string       `json:"files"`
}

type ReleaseManifest struct {
	Schema                string                      `json:"$schema"`
	SchemaVersion         string                      `json:"schema_version"`
	ProductName           string                      `json:"product_name"`
	AppVersion            string                      `json:"app_version"`
	GitCommit             string                      `json:"git_commit"`
	BuildDate             string                      `json:"build_date"`
	Binaries              []ReleaseBinary             `json:"binaries"`
	ChecksumsFile         AssetReference              `json:"checksums_file"`
	OCIImages             []OCIImageMetadata          `json:"oci_images"`
	Chart                 ChartMetadata               `json:"chart"`
	CompatibilityManifest AssetReference              `json:"compatibility_manifest"`
	DefaultConfigPacks    []DefaultConfigPackMetadata `json:"default_config_packs"`
}

type goreleaserArtifact struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
	Type   string `json:"type"`
}

type ReleaseManifestOptions struct {
	RepositoryRoot            string
	DistDir                   string
	OutputPath                string
	AppVersion                string
	GitCommit                 string
	BuildDate                 string
	CompatibilityManifestPath string
	DefaultConfigPackMetadata string
	ImageMetadataPath         string
	ChartMetadataPath         string
}

type ChartDefinition struct {
	APIVersion string `yaml:"apiVersion"`
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

func WriteImageMetadata(path, repository string, tags, platforms []string, published bool, digest string) error {
	metadata := OCIImageMetadata{
		SchemaVersion: "1.0.0",
		Repository:    repository,
		Tags:          slices.Clone(tags),
		Platforms:     slices.Clone(platforms),
		Published:     published,
		Digest:        strings.TrimSpace(digest),
		References:    make([]ImageReference, 0, len(tags)),
	}
	for _, tag := range tags {
		metadata.References = append(metadata.References, ImageReference{
			Reference: fmt.Sprintf("%s:%s", repository, tag),
		})
	}
	return writeJSON(path, metadata)
}

func WriteChartMetadata(repositoryRoot, path, name, version, archivePath, ociReference, digest string, published bool) error {
	archive, err := assetReference(repositoryRoot, archivePath)
	if err != nil {
		return err
	}
	metadata := ChartMetadata{
		SchemaVersion: "1.0.0",
		Name:          name,
		Version:       version,
		Published:     published,
		Archive:       archive,
		OCIReference:  strings.TrimSpace(ociReference),
		Digest:        strings.TrimSpace(digest),
	}
	return writeJSON(path, metadata)
}

func WriteReleaseManifest(opts ReleaseManifestOptions) error {
	manifest, err := GenerateReleaseManifest(opts)
	if err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	return writeJSON(opts.OutputPath, manifest)
}

func GenerateReleaseManifest(opts ReleaseManifestOptions) (ReleaseManifest, error) {
	root := opts.RepositoryRoot
	if root == "" {
		root = "."
	}
	distDir := opts.DistDir
	if distDir == "" {
		distDir = filepath.Join(root, "dist")
	}

	compatibilityPath := opts.CompatibilityManifestPath
	if compatibilityPath == "" {
		compatibilityPath = filepath.Join(root, DefaultCompatibilityManifestPath)
	}
	if _, err := ReadCompatibilityManifest(compatibilityPath); err != nil {
		return ReleaseManifest{}, fmt.Errorf("read compatibility manifest: %w", err)
	}
	compatibilityAsset, err := assetReference(root, compatibilityPath)
	if err != nil {
		return ReleaseManifest{}, err
	}

	appVersion, err := resolveAppVersion(root, opts.AppVersion)
	if err != nil {
		return ReleaseManifest{}, err
	}
	gitCommit, err := resolveGitCommit(root, opts.GitCommit)
	if err != nil {
		return ReleaseManifest{}, err
	}
	buildDate, err := resolveBuildDate(root, opts.BuildDate)
	if err != nil {
		return ReleaseManifest{}, err
	}

	binaries, checksums, err := collectBinaryAssets(root, distDir)
	if err != nil {
		return ReleaseManifest{}, err
	}

	imageMetadata, err := maybeReadJSON[OCIImageMetadata](opts.ImageMetadataPath)
	if err != nil {
		return ReleaseManifest{}, err
	}
	chartMetadata, err := maybeReadJSON[ChartMetadata](opts.ChartMetadataPath)
	if err != nil {
		return ReleaseManifest{}, err
	}
	defaultPackMetadata, err := maybeReadJSON[DefaultConfigPackMetadata](opts.DefaultConfigPackMetadata)
	if err != nil {
		return ReleaseManifest{}, err
	}

	manifest := ReleaseManifest{
		Schema:                DefaultReleaseSchemaPath,
		SchemaVersion:         ReleaseManifestSchemaVersion,
		ProductName:           ProductName,
		AppVersion:            appVersion,
		GitCommit:             gitCommit,
		BuildDate:             buildDate,
		Binaries:              binaries,
		ChecksumsFile:         checksums,
		CompatibilityManifest: compatibilityAsset,
	}
	if imageMetadata != nil {
		manifest.OCIImages = []OCIImageMetadata{*imageMetadata}
	}
	if chartMetadata != nil {
		manifest.Chart = *chartMetadata
	}
	if defaultPackMetadata != nil {
		manifest.DefaultConfigPacks = []DefaultConfigPackMetadata{*defaultPackMetadata}
	}
	return manifest, nil
}

func (m ReleaseManifest) Validate() error {
	if m.SchemaVersion != ReleaseManifestSchemaVersion {
		return fmt.Errorf("release manifest schema_version must be %q", ReleaseManifestSchemaVersion)
	}
	if m.ProductName != ProductName {
		return fmt.Errorf("release manifest product_name must be %q", ProductName)
	}
	if strings.TrimSpace(m.AppVersion) == "" {
		return fmt.Errorf("release manifest app_version cannot be empty")
	}
	if strings.TrimSpace(m.GitCommit) == "" {
		return fmt.Errorf("release manifest git_commit cannot be empty")
	}
	if _, err := time.Parse(time.RFC3339, m.BuildDate); err != nil {
		return fmt.Errorf("release manifest build_date must be RFC3339: %w", err)
	}
	if len(m.Binaries) == 0 {
		return fmt.Errorf("release manifest must contain binaries")
	}
	for _, binary := range m.Binaries {
		if strings.TrimSpace(binary.OS) == "" || strings.TrimSpace(binary.Arch) == "" {
			return fmt.Errorf("release manifest binary os/arch cannot be empty")
		}
		if err := validateAssetReference(binary.Archive, "binary archive"); err != nil {
			return err
		}
		if binary.SBOM != nil {
			if err := validateAssetReference(*binary.SBOM, "binary sbom"); err != nil {
				return err
			}
		}
	}
	if err := validateAssetReference(m.ChecksumsFile, "checksums file"); err != nil {
		return err
	}
	for _, image := range m.OCIImages {
		if strings.TrimSpace(image.Repository) == "" {
			return fmt.Errorf("release manifest image repository cannot be empty")
		}
		if len(image.Tags) == 0 {
			return fmt.Errorf("release manifest image tags cannot be empty")
		}
		if len(image.References) == 0 {
			return fmt.Errorf("release manifest image references cannot be empty")
		}
		if image.Published && strings.TrimSpace(image.Digest) == "" {
			return fmt.Errorf("release manifest published image must declare digest")
		}
	}
	if strings.TrimSpace(m.Chart.Name) == "" || strings.TrimSpace(m.Chart.Version) == "" {
		return fmt.Errorf("release manifest chart name/version cannot be empty")
	}
	if err := validateAssetReference(m.Chart.Archive, "chart archive"); err != nil {
		return err
	}
	if m.Chart.Published && strings.TrimSpace(m.Chart.OCIReference) == "" {
		return fmt.Errorf("release manifest published chart must declare oci_reference")
	}
	if m.Chart.Published && strings.TrimSpace(m.Chart.Digest) == "" {
		return fmt.Errorf("release manifest published chart must declare digest")
	}
	if err := validateAssetReference(m.CompatibilityManifest, "compatibility manifest"); err != nil {
		return err
	}
	if len(m.DefaultConfigPacks) == 0 {
		return fmt.Errorf("release manifest must declare at least one default config pack")
	}
	for _, pack := range m.DefaultConfigPacks {
		if strings.TrimSpace(pack.Name) == "" || strings.TrimSpace(pack.Version) == "" {
			return fmt.Errorf("release manifest default config pack name/version cannot be empty")
		}
		if err := validateAssetReference(pack.Archive, "default config pack"); err != nil {
			return err
		}
		if len(pack.Files) == 0 {
			return fmt.Errorf("release manifest default config pack files cannot be empty")
		}
	}
	return nil
}

func validateAssetReference(asset AssetReference, label string) error {
	if strings.TrimSpace(asset.Path) == "" {
		return fmt.Errorf("%s path cannot be empty", label)
	}
	if strings.TrimSpace(asset.SHA256) == "" {
		return fmt.Errorf("%s sha256 cannot be empty", label)
	}
	return nil
}

func ValidateChart(chartDir string) error {
	if chartDir == "" {
		chartDir = filepath.Join(".", "charts", "sheaft")
	}
	raw, err := os.ReadFile(filepath.Join(chartDir, "Chart.yaml"))
	if err != nil {
		return fmt.Errorf("read chart definition: %w", err)
	}
	var chart ChartDefinition
	if err := yaml.Unmarshal(raw, &chart); err != nil {
		return fmt.Errorf("decode chart definition: %w", err)
	}
	if chart.APIVersion != "v2" {
		return fmt.Errorf("chart apiVersion must be v2")
	}
	if chart.Name != "sheaft" {
		return fmt.Errorf("chart name must be sheaft")
	}
	if chart.Type != "application" {
		return fmt.Errorf("chart type must be application")
	}
	if strings.TrimSpace(chart.Version) == "" {
		return fmt.Errorf("chart version cannot be empty")
	}
	if strings.TrimSpace(chart.AppVersion) == "" {
		return fmt.Errorf("chart appVersion cannot be empty")
	}

	valuesPath := filepath.Join(chartDir, "values.yaml")
	var values map[string]any
	if err := readYAML(valuesPath, &values); err != nil {
		return err
	}
	requiredPaths := [][]string{
		{"mode"},
		{"image", "repository"},
		{"image", "tag"},
		{"image", "digest"},
		{"paths", "model"},
		{"paths", "artifact"},
		{"paths", "policy"},
		{"paths", "analysis"},
		{"paths", "outputDir"},
		{"paths", "historyDir"},
		{"resources"},
		{"env"},
		{"config"},
		{"service", "enabled"},
		{"metrics", "enabled"},
		{"securityContext"},
	}
	for _, path := range requiredPaths {
		if !yamlPathExists(values, path) {
			return fmt.Errorf("chart values.yaml is missing %s", strings.Join(path, "."))
		}
	}

	requiredTemplates := []string{
		"deployment.yaml",
		"job.yaml",
		"service.yaml",
		"configmap.yaml",
		"_helpers.tpl",
	}
	for _, name := range requiredTemplates {
		if _, err := os.Stat(filepath.Join(chartDir, "templates", name)); err != nil {
			return fmt.Errorf("chart template missing %s: %w", name, err)
		}
	}
	return nil
}

func yamlPathExists(root map[string]any, path []string) bool {
	var current any = root
	for _, segment := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = next[segment]
		if !ok {
			return false
		}
	}
	return true
}

func readYAML(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func maybeReadJSON[T any](path string) (*T, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	var value T
	if err := readJSON(path, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

func collectBinaryAssets(root, distDir string) ([]ReleaseBinary, AssetReference, error) {
	artifacts, err := loadArtifacts(distDir)
	if err != nil {
		return nil, AssetReference{}, err
	}

	var checksumPath string
	for _, artifact := range artifacts {
		if strings.HasSuffix(artifact.Name, "checksums.txt") {
			checksumPath = resolveArtifactPath(distDir, artifact.Path)
			break
		}
	}
	if checksumPath == "" {
		return nil, AssetReference{}, fmt.Errorf("dist does not contain checksums.txt")
	}
	checksumAsset, err := assetReference(root, checksumPath)
	if err != nil {
		return nil, AssetReference{}, err
	}

	binariesByTarget := map[string]ReleaseBinary{}
	archiveNameToTarget := map[string]string{}
	for _, artifact := range artifacts {
		fullPath := resolveArtifactPath(distDir, artifact.Path)
		switch {
		case artifact.Goos != "" && artifact.Goarch != "" && isArchive(artifact.Name):
			key := artifact.Goos + "/" + artifact.Goarch
			archiveRef, err := assetReference(root, fullPath)
			if err != nil {
				return nil, AssetReference{}, err
			}
			binariesByTarget[key] = ReleaseBinary{
				OS:      artifact.Goos,
				Arch:    artifact.Goarch,
				Archive: archiveRef,
			}
			archiveNameToTarget[artifact.Name] = key
		case artifact.Goos != "" && artifact.Goarch != "" && strings.Contains(artifact.Name, ".sbom."):
			key := artifact.Goos + "/" + artifact.Goarch
			binary := binariesByTarget[key]
			sbomRef, err := assetReference(root, fullPath)
			if err != nil {
				return nil, AssetReference{}, err
			}
			binary.OS = artifact.Goos
			binary.Arch = artifact.Goarch
			binary.SBOM = &sbomRef
			binariesByTarget[key] = binary
		case strings.HasSuffix(artifact.Name, ".sbom.json"):
			archiveName := strings.TrimSuffix(artifact.Name, ".sbom.json")
			key, ok := archiveNameToTarget[archiveName]
			if !ok {
				continue
			}
			binary := binariesByTarget[key]
			sbomRef, err := assetReference(root, fullPath)
			if err != nil {
				return nil, AssetReference{}, err
			}
			binary.SBOM = &sbomRef
			binariesByTarget[key] = binary
		}
	}

	if len(binariesByTarget) == 0 {
		return nil, AssetReference{}, fmt.Errorf("dist does not contain any binary archives")
	}

	keys := make([]string, 0, len(binariesByTarget))
	for key := range binariesByTarget {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make([]ReleaseBinary, 0, len(keys))
	for _, key := range keys {
		binary := binariesByTarget[key]
		if binary.Archive.Path == "" {
			continue
		}
		out = append(out, binary)
	}
	return out, checksumAsset, nil
}

func loadArtifacts(distDir string) ([]goreleaserArtifact, error) {
	data, err := os.ReadFile(filepath.Join(distDir, "artifacts.json"))
	if err != nil {
		return nil, fmt.Errorf("read dist/artifacts.json: %w", err)
	}
	var artifacts []goreleaserArtifact
	if err := json.Unmarshal(data, &artifacts); err == nil {
		return artifacts, nil
	}
	var wrapped struct {
		Artifacts []goreleaserArtifact `json:"artifacts"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("decode dist/artifacts.json: %w", err)
	}
	return wrapped.Artifacts, nil
}

func resolveArtifactPath(distDir, artifactPath string) string {
	if filepath.IsAbs(artifactPath) {
		return artifactPath
	}
	if _, err := os.Stat(artifactPath); err == nil {
		return artifactPath
	}
	return filepath.Join(distDir, artifactPath)
}

func isArchive(name string) bool {
	return strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")
}

func assetReference(root, path string) (AssetReference, error) {
	sha, err := fileSHA256(path)
	if err != nil {
		return AssetReference{}, err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return AssetReference{}, fmt.Errorf("relative path for %s: %w", path, err)
	}
	return AssetReference{
		Path:   filepath.ToSlash(relative),
		SHA256: sha,
	}, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func resolveAppVersion(root, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimPrefix(strings.TrimSpace(explicit), "v"), nil
	}
	for _, envName := range []string{"SHEAFT_VERSION", "RELEASE_VERSION", "GORELEASER_CURRENT_TAG"} {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			return strings.TrimPrefix(value, "v"), nil
		}
	}
	if tag, err := gitOutput(root, "describe", "--tags", "--exact-match"); err == nil && strings.TrimSpace(tag) != "" {
		return strings.TrimPrefix(strings.TrimSpace(tag), "v"), nil
	}
	return "0.0.0-dev", nil
}

func resolveGitCommit(root, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	for _, envName := range []string{"GIT_COMMIT", "GITHUB_SHA", "CI_COMMIT_SHA"} {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			return value, nil
		}
	}
	return gitOutput(root, "rev-parse", "HEAD")
}

func resolveBuildDate(root, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return normalizeRFC3339(strings.TrimSpace(explicit))
	}
	if sourceDateEpoch := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); sourceDateEpoch != "" {
		if seconds, err := time.ParseDuration(sourceDateEpoch + "s"); err == nil {
			return time.Unix(0, 0).UTC().Add(seconds).Format(time.RFC3339), nil
		}
	}
	for _, envName := range []string{"BUILD_DATE", "GORELEASER_CURRENT_DATE"} {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			return normalizeRFC3339(value)
		}
	}
	value, err := gitOutput(root, "log", "-1", "--format=%cI")
	if err != nil {
		return "", err
	}
	return normalizeRFC3339(strings.TrimSpace(value))
}

func normalizeRFC3339(value string) (string, error) {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC().Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("invalid RFC3339 date %q", value)
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func PackageDefaultConfigPack(repositoryRoot, sourceListPath, version, archivePath, metadataPath string) error {
	if repositoryRoot == "" {
		repositoryRoot = "."
	}
	if sourceListPath == "" {
		sourceListPath = filepath.Join(repositoryRoot, DefaultConfigPackSourceListPath)
	}
	if version == "" {
		resolved, err := resolveAppVersion(repositoryRoot, "")
		if err != nil {
			return err
		}
		version = resolved
	}
	files, err := readPackSources(sourceListPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return fmt.Errorf("create pack dir: %w", err)
	}
	if err := writeDefaultPackArchive(repositoryRoot, version, files, archivePath); err != nil {
		return err
	}
	archiveRef, err := assetReference(repositoryRoot, archivePath)
	if err != nil {
		return err
	}
	metadata := DefaultConfigPackMetadata{
		SchemaVersion: "1.0.0",
		Name:          "default-config-pack",
		Version:       version,
		Archive:       archiveRef,
		Files:         files,
	}
	if metadataPath != "" {
		if err := writeJSON(metadataPath, metadata); err != nil {
			return err
		}
	}
	return nil
}

func writeDefaultPackArchive(repositoryRoot, version string, files []string, archivePath string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create %s: %w", archivePath, err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	prefix := fmt.Sprintf("sheaft-default-config-pack_%s", version)
	manifest := map[string]any{
		"product_name": ProductName,
		"pack_name":    "default-config-pack",
		"version":      version,
		"files":        files,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal default pack manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')
	if err := addTarFile(tarWriter, filepath.ToSlash(filepath.Join(prefix, "manifest.json")), manifestData, 0o644); err != nil {
		return err
	}

	for _, relPath := range files {
		fullPath := filepath.Join(repositoryRoot, filepath.FromSlash(relPath))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("read pack source %s: %w", relPath, err)
		}
		if err := addTarFile(tarWriter, filepath.ToSlash(filepath.Join(prefix, filepath.FromSlash(relPath))), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func addTarFile(writer *tar.Writer, name string, data []byte, mode int64) error {
	header := &tar.Header{
		Name:     name,
		Mode:     mode,
		Size:     int64(len(data)),
		Typeflag: tar.TypeReg,
	}
	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("write tar header %s: %w", name, err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("write tar file %s: %w", name, err)
	}
	return nil
}

func readPackSources(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, filepath.ToSlash(line))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("default config pack sources are empty")
	}
	return out, nil
}
