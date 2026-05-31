package benchmark

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/analyzer"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/report"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

const (
	DefaultManifestPath = "benchmarks/fixed-slice/manifest.json"
	DefaultOutputDir    = ".tmp/benchmark-slice"
)

type Manifest struct {
	SchemaVersion string       `json:"schema_version"`
	Name          string       `json:"name"`
	Description   string       `json:"description,omitempty"`
	Artifact      string       `json:"artifact"`
	Analysis      string       `json:"analysis"`
	Expected      Expected     `json:"expected"`
	QualityGates  QualityGates `json:"quality_gates"`
}

type Expected struct {
	Decision                      string  `json:"decision"`
	ContractName                  string  `json:"contract_name"`
	ContractVersion               string  `json:"contract_version"`
	Profiles                      int     `json:"profiles"`
	MinConfidence                 float64 `json:"min_confidence"`
	MaxUnavailableAdvancedMetrics int     `json:"max_unavailable_advanced_metrics"`
	RequireBaselineDiff           bool    `json:"require_baseline_diff"`
}

type QualityGates struct {
	RequireRepeatableStableReport       bool    `json:"require_repeatable_stable_report"`
	MinCrossProfileWeightedAvailability float64 `json:"min_cross_profile_weighted_availability"`
}

type Outputs struct {
	Model         string `json:"model"`
	Report        string `json:"report"`
	Summary       string `json:"summary"`
	QualityReport string `json:"quality_report"`
}

type QualityReport struct {
	SchemaVersion string         `json:"schema_version"`
	BenchmarkName string         `json:"benchmark_name"`
	Status        string         `json:"status"`
	GeneratedAt   string         `json:"generated_at"`
	Inputs        Inputs         `json:"inputs"`
	Outputs       Outputs        `json:"outputs"`
	Metrics       QualityMetrics `json:"metrics"`
	Checks        []QualityCheck `json:"checks"`
}

type Inputs struct {
	Artifact string `json:"artifact"`
	Analysis string `json:"analysis"`
}

type QualityMetrics struct {
	Decision                         string  `json:"decision"`
	ProfileCount                     int     `json:"profile_count"`
	Confidence                       float64 `json:"confidence"`
	CrossProfileWeightedAvailability float64 `json:"cross_profile_weighted_availability"`
	RiskScore                        float64 `json:"risk_score"`
	UnavailableAdvancedMetrics       int     `json:"unavailable_advanced_metrics"`
	BaselineDiffCount                int     `json:"baseline_diff_count"`
	StableReportSHA256               string  `json:"stable_report_sha256"`
	RepeatStableReportSHA256         string  `json:"repeat_stable_report_sha256"`
}

type QualityCheck struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
}

type RunOptions struct {
	RepositoryRoot string
	ManifestPath   string
	OutputDir      string
	GeneratedAt    time.Time
}

func Run(opts RunOptions) (QualityReport, error) {
	root := opts.RepositoryRoot
	if root == "" {
		root = "."
	}
	manifestPath := opts.ManifestPath
	if manifestPath == "" {
		manifestPath = DefaultManifestPath
	}
	outDir := opts.OutputDir
	if outDir == "" {
		outDir = DefaultOutputDir
	}
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	manifest, err := LoadManifest(filepath.Join(root, manifestPath))
	if err != nil {
		return QualityReport{}, err
	}
	if err := os.MkdirAll(filepath.Join(root, outDir), 0o755); err != nil {
		return QualityReport{}, fmt.Errorf("create benchmark output dir: %w", err)
	}

	cfg, err := config.LoadAnalysis(filepath.Join(root, manifest.Analysis))
	if err != nil {
		return QualityReport{}, fmt.Errorf("load benchmark analysis: %w", err)
	}
	cfg.ResolveRelativePaths(root)

	first, err := analyzer.AnalyzeFile(filepath.Join(root, manifest.Artifact), cfg, nil)
	if err != nil {
		return QualityReport{}, fmt.Errorf("run benchmark analysis: %w", err)
	}
	second, err := analyzer.AnalyzeFile(filepath.Join(root, manifest.Artifact), cfg, nil)
	if err != nil {
		return QualityReport{}, fmt.Errorf("rerun benchmark analysis: %w", err)
	}

	modelPath := filepath.Join(root, outDir, "model.json")
	reportPath := filepath.Join(root, outDir, "report.json")
	summaryPath := filepath.Join(root, outDir, "summary.md")
	qualityPath := filepath.Join(root, outDir, "quality-report.json")

	if err := model.WriteToFile(modelPath, first.Artifact.Model); err != nil {
		return QualityReport{}, fmt.Errorf("write benchmark model: %w", err)
	}
	if err := report.WriteJSON(reportPath, first.Report); err != nil {
		return QualityReport{}, fmt.Errorf("write benchmark report: %w", err)
	}
	if err := report.WriteSummaryMarkdown(summaryPath, first.Report); err != nil {
		return QualityReport{}, fmt.Errorf("write benchmark summary: %w", err)
	}

	stableHash, err := StableReportSHA256(first.Report)
	if err != nil {
		return QualityReport{}, err
	}
	repeatHash, err := StableReportSHA256(second.Report)
	if err != nil {
		return QualityReport{}, err
	}
	quality := BuildQualityReport(manifest, first.Report, stableHash, repeatHash, generatedAt, Inputs{
		Artifact: manifest.Artifact,
		Analysis: manifest.Analysis,
	}, Outputs{
		Model:         filepath.ToSlash(filepath.Join(outDir, "model.json")),
		Report:        filepath.ToSlash(filepath.Join(outDir, "report.json")),
		Summary:       filepath.ToSlash(filepath.Join(outDir, "summary.md")),
		QualityReport: filepath.ToSlash(filepath.Join(outDir, "quality-report.json")),
	})
	if err := report.WriteJSON(qualityPath, quality); err != nil {
		return QualityReport{}, fmt.Errorf("write benchmark quality report: %w", err)
	}
	return quality, nil
}

func LoadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read benchmark manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode benchmark manifest: %w", err)
	}
	if manifest.SchemaVersion != "1.0" {
		return Manifest{}, fmt.Errorf("benchmark manifest schema_version must be %q", "1.0")
	}
	if manifest.Name == "" {
		return Manifest{}, fmt.Errorf("benchmark manifest name cannot be empty")
	}
	if manifest.Artifact == "" || manifest.Analysis == "" {
		return Manifest{}, fmt.Errorf("benchmark manifest requires artifact and analysis")
	}
	return manifest, nil
}

func BuildQualityReport(manifest Manifest, rep report.Report, stableHash, repeatHash string, generatedAt time.Time, inputs Inputs, outputs Outputs) QualityReport {
	metrics := QualityMetrics{
		Decision:                         rep.PolicyEvaluation.Decision,
		ProfileCount:                     len(rep.Profiles),
		Confidence:                       rep.Summary.Confidence,
		CrossProfileWeightedAvailability: rep.Summary.CrossProfileWeightedAvailabilityOrFallback(),
		RiskScore:                        rep.Summary.RiskScore,
		UnavailableAdvancedMetrics:       CountUnavailableAdvancedMetrics(rep),
		BaselineDiffCount:                len(rep.Diffs.Baselines),
		StableReportSHA256:               stableHash,
		RepeatStableReportSHA256:         repeatHash,
	}
	checks := []QualityCheck{
		checkEqual("expected_decision", manifest.Expected.Decision, metrics.Decision, "gate decision matches the fixed slice expectation"),
		checkEqual("expected_profile_count", fmt.Sprintf("%d", manifest.Expected.Profiles), fmt.Sprintf("%d", metrics.ProfileCount), "profile count matches the fixed slice expectation"),
		checkAtLeast("min_confidence", manifest.Expected.MinConfidence, metrics.Confidence, "artifact confidence is high enough for release evidence"),
		checkAtMostInt("max_unavailable_advanced_metrics", manifest.Expected.MaxUnavailableAdvancedMetrics, metrics.UnavailableAdvancedMetrics, "advanced diagnostics are available on the fixed slice"),
		checkAtLeast("min_cross_profile_weighted_availability", manifest.QualityGates.MinCrossProfileWeightedAvailability, metrics.CrossProfileWeightedAvailability, "cross-profile weighted availability stays above the benchmark floor"),
	}
	if manifest.Expected.ContractName != "" || manifest.Expected.ContractVersion != "" {
		actual := ""
		if rep.InputArtifact != nil {
			actual = rep.InputArtifact.ContractName + "@" + rep.InputArtifact.ContractVersion
		}
		checks = append(checks, checkEqual("expected_contract", manifest.Expected.ContractName+"@"+manifest.Expected.ContractVersion, actual, "input contract matches the benchmark fixture"))
	}
	if manifest.Expected.RequireBaselineDiff {
		checks = append(checks, checkAtLeastInt("require_baseline_diff", 1, metrics.BaselineDiffCount, "baseline diff is present"))
	}
	if manifest.QualityGates.RequireRepeatableStableReport {
		checks = append(checks, checkEqual("repeatable_stable_report", stableHash, repeatHash, "stable report hash is repeatable across two runs"))
	}

	status := "pass"
	for _, check := range checks {
		if check.Status != "pass" {
			status = "fail"
			break
		}
	}
	return QualityReport{
		SchemaVersion: "1.0",
		BenchmarkName: manifest.Name,
		Status:        status,
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339Nano),
		Inputs:        inputs,
		Outputs:       outputs,
		Metrics:       metrics,
		Checks:        checks,
	}
}

func StableReportSHA256(rep report.Report) (string, error) {
	rep.GeneratedAt = ""
	rep.RecomputeDurationMS = 0
	raw, err := json.Marshal(rep)
	if err != nil {
		return "", fmt.Errorf("marshal stable report: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func CountUnavailableAdvancedMetrics(rep report.Report) int {
	total := 0
	for _, profile := range rep.Profiles {
		total += countUnavailableAdvanced(profile.Simulation.Advanced)
	}
	return total
}

func countUnavailableAdvanced(advanced *simulation.AdvancedProfile) int {
	if advanced == nil {
		return 0
	}
	total := 0
	if advanced.BlastRadius != nil {
		if !advanced.BlastRadius.ServiceCount.Available {
			total++
		}
		if !advanced.BlastRadius.EndpointCount.Available {
			total++
		}
	}
	for _, endpoint := range advanced.Endpoints {
		if !endpoint.ExpectedSuccessRate.Available {
			total++
		}
		if !endpoint.MaxAmplificationFactor.Available {
			total++
		}
	}
	for _, path := range advanced.Paths {
		if !path.ExpectedSuccessRate.Available {
			total++
		}
		if !path.MaxAmplificationFactor.Available {
			total++
		}
		if !path.TimeoutMismatchCount.Available {
			total++
		}
	}
	for _, edge := range advanced.Edges {
		if !edge.MaxAmplificationFactor.Available {
			total++
		}
	}
	return total
}

func checkEqual(id, expected, actual, message string) QualityCheck {
	status := "pass"
	if expected != actual {
		status = "fail"
	}
	return QualityCheck{ID: id, Status: status, Expected: expected, Actual: actual, Message: message}
}

func checkAtLeast(id string, expected, actual float64, message string) QualityCheck {
	status := "pass"
	if actual < expected {
		status = "fail"
	}
	return QualityCheck{ID: id, Status: status, Expected: fmt.Sprintf("%.6f", expected), Actual: fmt.Sprintf("%.6f", actual), Message: message}
}

func checkAtLeastInt(id string, expected, actual int, message string) QualityCheck {
	status := "pass"
	if actual < expected {
		status = "fail"
	}
	return QualityCheck{ID: id, Status: status, Expected: fmt.Sprintf("%d", expected), Actual: fmt.Sprintf("%d", actual), Message: message}
}

func checkAtMostInt(id string, expected, actual int, message string) QualityCheck {
	status := "pass"
	if actual > expected {
		status = "fail"
	}
	return QualityCheck{ID: id, Status: status, Expected: fmt.Sprintf("%d", expected), Actual: fmt.Sprintf("%d", actual), Message: message}
}
