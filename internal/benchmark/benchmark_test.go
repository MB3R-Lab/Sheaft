package benchmark

import (
	"testing"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/report"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

func TestBuildQualityReportPassesFixedExpectations(t *testing.T) {
	t.Parallel()

	rep := report.Report{
		Summary: report.Summary{
			Confidence:                       0.95,
			CrossProfileWeightedAvailability: 0.75,
			RiskScore:                        0.1,
		},
		PolicyEvaluation: report.PolicyEvaluation{Decision: "fail"},
		InputArtifact: &report.InputArtifact{
			ContractName:    "io.mb3r.bering.snapshot",
			ContractVersion: "1.1.0",
		},
		Profiles: []report.ProfileSummary{
			{
				Name: "steady",
				Simulation: simulation.ProfileOutput{
					Advanced: &simulation.AdvancedProfile{
						Paths: []simulation.PathAdvanced{
							{
								ExpectedSuccessRate:    simulation.MetricFloat{Available: true},
								MaxAmplificationFactor: simulation.MetricFloat{Available: true},
								TimeoutMismatchCount:   simulation.MetricInt{Available: true},
							},
						},
					},
				},
			},
			{Name: "fault"},
		},
		Diffs: report.Diffs{
			Baselines: []report.Diff{{Name: "baseline"}},
		},
	}

	quality := BuildQualityReport(Manifest{
		Name: "fixed",
		Expected: Expected{
			Decision:                      "fail",
			ContractName:                  "io.mb3r.bering.snapshot",
			ContractVersion:               "1.1.0",
			Profiles:                      2,
			MinConfidence:                 0.9,
			MaxUnavailableAdvancedMetrics: 0,
			RequireBaselineDiff:           true,
		},
		QualityGates: QualityGates{
			RequireRepeatableStableReport:       true,
			MinCrossProfileWeightedAvailability: 0.7,
		},
	}, rep, "sha256:a", "sha256:a", time.Unix(0, 0), Inputs{}, Outputs{})

	if quality.Status != "pass" {
		t.Fatalf("expected quality report to pass, got %+v", quality)
	}
	if quality.Metrics.UnavailableAdvancedMetrics != 0 {
		t.Fatalf("expected no unavailable advanced metrics, got %+v", quality.Metrics)
	}
}

func TestCountUnavailableAdvancedMetrics(t *testing.T) {
	t.Parallel()

	rep := report.Report{
		Profiles: []report.ProfileSummary{
			{
				Simulation: simulation.ProfileOutput{
					Advanced: &simulation.AdvancedProfile{
						Endpoints: []simulation.EndpointAdvanced{
							{
								ExpectedSuccessRate:    simulation.MetricFloat{Available: false},
								MaxAmplificationFactor: simulation.MetricFloat{Available: true},
							},
						},
						Edges: []simulation.EdgeAdvanced{
							{MaxAmplificationFactor: simulation.MetricFloat{Available: false}},
						},
					},
				},
				EndpointResults: []gate.EndpointResult{{EndpointID: "gateway:GET /health"}},
			},
		},
	}

	if got := CountUnavailableAdvancedMetrics(rep); got != 2 {
		t.Fatalf("expected 2 unavailable advanced metrics, got %d", got)
	}
}
