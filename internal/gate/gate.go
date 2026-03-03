package gate

import (
	"fmt"
	"slices"

	"github.com/MB3R-Lab/Sheaft/internal/config"
)

const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

type EndpointResult struct {
	EndpointID   string  `json:"endpoint_id"`
	Availability float64 `json:"availability"`
	Threshold    float64 `json:"threshold"`
	Status       string  `json:"status"`
}

type Evaluation struct {
	Mode            config.PolicyMode `json:"mode"`
	Decision        string            `json:"decision"`
	FailedEndpoints []string          `json:"failed_endpoints"`
	EndpointResults []EndpointResult  `json:"endpoint_results"`
}

func Evaluate(availability map[string]float64, policy config.Policy, modeOverride string) (Evaluation, error) {
	mode := policy.Mode
	if modeOverride != "" {
		mode = config.PolicyMode(modeOverride)
	}
	if mode != config.ModeWarn && mode != config.ModeFail && mode != config.ModeReport {
		return Evaluation{}, fmt.Errorf("unsupported mode: %q", mode)
	}

	endpointIDs := make([]string, 0, len(availability))
	for endpointID := range availability {
		endpointIDs = append(endpointIDs, endpointID)
	}
	slices.Sort(endpointIDs)

	failed := make([]string, 0)
	results := make([]EndpointResult, 0, len(endpointIDs))
	for _, endpointID := range endpointIDs {
		threshold := policy.GlobalThreshold
		if specific, ok := policy.EndpointThresholds[endpointID]; ok {
			threshold = specific
		}
		avail := availability[endpointID]
		status := StatusPass
		if avail < threshold {
			switch mode {
			case config.ModeFail:
				status = StatusFail
			case config.ModeWarn, config.ModeReport:
				status = StatusWarn
			}
			failed = append(failed, endpointID)
		}
		results = append(results, EndpointResult{
			EndpointID:   endpointID,
			Availability: avail,
			Threshold:    threshold,
			Status:       status,
		})
	}

	decision := StatusPass
	switch mode {
	case config.ModeReport:
		decision = "report"
	case config.ModeFail:
		if len(failed) > 0 {
			decision = StatusFail
		}
	case config.ModeWarn:
		if len(failed) > 0 {
			decision = StatusWarn
		}
	}

	return Evaluation{
		Mode:            mode,
		Decision:        decision,
		FailedEndpoints: failed,
		EndpointResults: results,
	}, nil
}
