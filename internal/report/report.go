package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

type SimulationInfo struct {
	Trials             int     `json:"trials"`
	Seed               int64   `json:"seed"`
	FailureProbability float64 `json:"failure_probability"`
}

type Summary struct {
	OverallAvailability float64 `json:"overall_availability"`
	RiskScore           float64 `json:"risk_score"`
	Confidence          float64 `json:"confidence"`
}

type PolicyEvaluation struct {
	Mode            string   `json:"mode"`
	Decision        string   `json:"decision"`
	FailedEndpoints []string `json:"failed_endpoints"`
}

type Report struct {
	Simulation       SimulationInfo        `json:"simulation"`
	EndpointResults  []gate.EndpointResult `json:"endpoint_results"`
	Summary          Summary               `json:"summary"`
	PolicyEvaluation PolicyEvaluation      `json:"policy_evaluation"`
}

func Compose(simOut simulation.Output, eval gate.Evaluation, params simulation.Params, confidence float64) Report {
	return Report{
		Simulation: SimulationInfo{
			Trials:             params.Trials,
			Seed:               params.Seed,
			FailureProbability: params.FailureProbability,
		},
		EndpointResults: eval.EndpointResults,
		Summary: Summary{
			OverallAvailability: simOut.OverallAvailability,
			RiskScore:           1 - simOut.OverallAvailability,
			Confidence:          confidence,
		},
		PolicyEvaluation: PolicyEvaluation{
			Mode:            string(eval.Mode),
			Decision:        eval.Decision,
			FailedEndpoints: eval.FailedEndpoints,
		},
	}
}

func (r Report) AvailabilityMap() map[string]float64 {
	out := make(map[string]float64, len(r.EndpointResults))
	for _, endpoint := range r.EndpointResults {
		out[endpoint.EndpointID] = endpoint.Availability
	}
	return out
}

func Load(path string) (Report, error) {
	var rep Report
	raw, err := os.ReadFile(path)
	if err != nil {
		return rep, fmt.Errorf("read report file: %w", err)
	}
	if err := json.Unmarshal(raw, &rep); err != nil {
		return rep, fmt.Errorf("decode report json: %w", err)
	}
	return rep, nil
}

func WriteJSON(path string, payload any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write json file: %w", err)
	}
	return nil
}

func WriteSummaryMarkdown(path string, rep Report) error {
	var b strings.Builder
	b.WriteString("# Sheaft Report Summary\n\n")
	b.WriteString(fmt.Sprintf("- Decision: **%s**\n", rep.PolicyEvaluation.Decision))
	b.WriteString(fmt.Sprintf("- Mode: `%s`\n", rep.PolicyEvaluation.Mode))
	b.WriteString(fmt.Sprintf("- Overall availability: `%.4f`\n", rep.Summary.OverallAvailability))
	b.WriteString(fmt.Sprintf("- Risk score: `%.4f`\n", rep.Summary.RiskScore))
	b.WriteString(fmt.Sprintf("- Confidence: `%.2f`\n\n", rep.Summary.Confidence))
	b.WriteString("## Endpoint results\n\n")
	for _, endpoint := range rep.EndpointResults {
		b.WriteString(fmt.Sprintf(
			"- `%s`: availability=`%.4f`, threshold=`%.4f`, status=`%s`\n",
			endpoint.EndpointID,
			endpoint.Availability,
			endpoint.Threshold,
			endpoint.Status,
		))
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create summary dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write summary markdown: %w", err)
	}
	return nil
}
