package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MB3R-Lab/Sheaft/internal/release"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "compatibility-manifest":
		runCompatibilityManifest(os.Args[2:])
	case "validate-compatibility-manifest":
		runValidateCompatibilityManifest(os.Args[2:])
	case "package-default-config-pack":
		runPackageDefaultConfigPack(os.Args[2:])
	case "image-metadata":
		runImageMetadata(os.Args[2:])
	case "chart-metadata":
		runChartMetadata(os.Args[2:])
	case "release-manifest":
		runReleaseManifest(os.Args[2:])
	case "validate-release-manifest":
		runValidateReleaseManifest(os.Args[2:])
	case "validate-chart":
		runValidateChart(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  releasectl compatibility-manifest [--out compatibility-manifest.json]")
	fmt.Fprintln(os.Stderr, "  releasectl validate-compatibility-manifest [--manifest compatibility-manifest.json]")
	fmt.Fprintln(os.Stderr, "  releasectl package-default-config-pack [--source-list release/packs/default-config-pack.files.txt] [--version 0.0.0-dev] [--out dist/sheaft-default-config-pack_0.0.0-dev.tar.gz] [--metadata-out dist/default-config-pack.json]")
	fmt.Fprintln(os.Stderr, "  releasectl image-metadata --repository <repo> --tag <tag> [--tag <tag>] [--platform linux/amd64] [--published] [--digest sha256:...] [--out dist/image-metadata.json]")
	fmt.Fprintln(os.Stderr, "  releasectl chart-metadata --name sheaft --version 1.0.0 --archive dist/charts/sheaft-1.0.0.tgz [--published] [--oci-reference oci://...] [--digest sha256:...] [--out dist/chart-metadata.json]")
	fmt.Fprintln(os.Stderr, "  releasectl release-manifest [--dist dist] [--out release-manifest.json]")
	fmt.Fprintln(os.Stderr, "  releasectl validate-release-manifest [--manifest release-manifest.json]")
	fmt.Fprintln(os.Stderr, "  releasectl validate-chart [--chart-dir charts/sheaft]")
}

func runCompatibilityManifest(args []string) {
	fs := flag.NewFlagSet("compatibility-manifest", flag.ExitOnError)
	out := fs.String("out", release.DefaultCompatibilityManifestPath, "Path to compatibility-manifest.json")
	_ = fs.Parse(args)

	if err := release.WriteCompatibilityManifest(*out); err != nil {
		fail(err)
	}
}

func runValidateCompatibilityManifest(args []string) {
	fs := flag.NewFlagSet("validate-compatibility-manifest", flag.ExitOnError)
	path := fs.String("manifest", release.DefaultCompatibilityManifestPath, "Path to compatibility-manifest.json")
	_ = fs.Parse(args)

	if _, err := release.ReadCompatibilityManifest(*path); err != nil {
		fail(err)
	}
}

func runPackageDefaultConfigPack(args []string) {
	fs := flag.NewFlagSet("package-default-config-pack", flag.ExitOnError)
	sourceList := fs.String("source-list", release.DefaultConfigPackSourceListPath, "Path to default config pack source list")
	version := fs.String("version", "", "Release app version")
	out := fs.String("out", filepath.Join("dist", "sheaft-default-config-pack_0.0.0-dev.tar.gz"), "Path to output archive")
	metadataOut := fs.String("metadata-out", release.DefaultConfigPackMetadataOutputPath, "Path to output metadata json")
	_ = fs.Parse(args)

	if err := release.PackageDefaultConfigPack(".", *sourceList, *version, *out, *metadataOut); err != nil {
		fail(err)
	}
}

type multiValue []string

func (m *multiValue) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiValue) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func runImageMetadata(args []string) {
	fs := flag.NewFlagSet("image-metadata", flag.ExitOnError)
	repository := fs.String("repository", "", "OCI image repository")
	digest := fs.String("digest", "", "OCI image digest")
	out := fs.String("out", filepath.Join("dist", "image-metadata.json"), "Path to output metadata json")
	published := fs.Bool("published", false, "Whether the image was pushed to a registry")
	var tags multiValue
	var platforms multiValue
	fs.Var(&tags, "tag", "OCI image tag")
	fs.Var(&platforms, "platform", "OCI image platform")
	_ = fs.Parse(args)

	if *repository == "" {
		fail(fmt.Errorf("image repository is required"))
	}
	if len(tags) == 0 {
		fail(fmt.Errorf("at least one image tag is required"))
	}
	if len(platforms) == 0 {
		fail(fmt.Errorf("at least one image platform is required"))
	}
	if err := release.WriteImageMetadata(*out, *repository, tags, platforms, *published, *digest); err != nil {
		fail(err)
	}
}

func runChartMetadata(args []string) {
	fs := flag.NewFlagSet("chart-metadata", flag.ExitOnError)
	name := fs.String("name", "sheaft", "Chart name")
	version := fs.String("version", "", "Chart version")
	archive := fs.String("archive", "", "Path to chart archive")
	ociReference := fs.String("oci-reference", "", "OCI chart reference")
	digest := fs.String("digest", "", "OCI chart digest")
	out := fs.String("out", filepath.Join("dist", "chart-metadata.json"), "Path to output metadata json")
	published := fs.Bool("published", false, "Whether the chart was pushed to an OCI registry")
	_ = fs.Parse(args)

	if *version == "" {
		fail(fmt.Errorf("chart version is required"))
	}
	if *archive == "" {
		fail(fmt.Errorf("chart archive path is required"))
	}
	if err := release.WriteChartMetadata(".", *out, *name, *version, *archive, *ociReference, *digest, *published); err != nil {
		fail(err)
	}
}

func runReleaseManifest(args []string) {
	fs := flag.NewFlagSet("release-manifest", flag.ExitOnError)
	dist := fs.String("dist", "dist", "Path to dist directory")
	out := fs.String("out", release.DefaultReleaseManifestPath, "Path to release-manifest.json")
	appVersion := fs.String("app-version", "", "App version override")
	gitCommit := fs.String("git-commit", "", "Git commit override")
	buildDate := fs.String("build-date", "", "Build date override")
	compatibilityPath := fs.String("compatibility-manifest", release.DefaultCompatibilityManifestPath, "Path to compatibility-manifest.json")
	defaultPackMetadata := fs.String("default-pack-metadata", release.DefaultConfigPackMetadataOutputPath, "Path to default config pack metadata")
	imageMetadata := fs.String("image-metadata", filepath.Join("dist", "image-metadata.json"), "Path to image metadata json")
	chartMetadata := fs.String("chart-metadata", filepath.Join("dist", "chart-metadata.json"), "Path to chart metadata json")
	_ = fs.Parse(args)

	err := release.WriteReleaseManifest(release.ReleaseManifestOptions{
		RepositoryRoot:            ".",
		DistDir:                   *dist,
		OutputPath:                *out,
		AppVersion:                *appVersion,
		GitCommit:                 *gitCommit,
		BuildDate:                 *buildDate,
		CompatibilityManifestPath: *compatibilityPath,
		DefaultConfigPackMetadata: *defaultPackMetadata,
		ImageMetadataPath:         *imageMetadata,
		ChartMetadataPath:         *chartMetadata,
	})
	if err != nil {
		fail(err)
	}
}

func runValidateReleaseManifest(args []string) {
	fs := flag.NewFlagSet("validate-release-manifest", flag.ExitOnError)
	path := fs.String("manifest", release.DefaultReleaseManifestPath, "Path to release-manifest.json")
	_ = fs.Parse(args)

	var manifest release.ReleaseManifest
	if err := readJSON(*path, &manifest); err != nil {
		fail(err)
	}
	if err := manifest.Validate(); err != nil {
		fail(err)
	}
}

func runValidateChart(args []string) {
	fs := flag.NewFlagSet("validate-chart", flag.ExitOnError)
	chartDir := fs.String("chart-dir", filepath.Join("charts", "sheaft"), "Path to chart directory")
	_ = fs.Parse(args)

	if err := release.ValidateChart(*chartDir); err != nil {
		fail(err)
	}
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
