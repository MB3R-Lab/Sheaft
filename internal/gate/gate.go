package gate

import (
	"fmt"
	"slices"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

type EndpointResult struct {
	Profile      string  `json:"profile,omitempty"`
	EndpointID   string  `json:"endpoint_id"`
	Availability float64 `json:"availability"`
	Threshold    float64 `json:"threshold"`
	Status       string  `json:"status"`
}

type AggregateResult struct {
	Availability float64 `json:"availability"`
	Threshold    float64 `json:"threshold"`
	Status       string  `json:"status"`
}

type ProfileEvaluation struct {
	Profile                 string           `json:"profile"`
	Decision                string           `json:"decision"`
	FailedEndpoints         []string         `json:"failed_endpoints"`
	EndpointsBelowThreshold int              `json:"endpoints_below_threshold"`
	EndpointResults         []EndpointResult `json:"endpoint_results"`
	Aggregate               *AggregateResult `json:"aggregate,omitempty"`
}

type Evaluation struct {
	Mode                  config.PolicyMode         `json:"mode"`
	Decision              string                    `json:"decision"`
	EvaluationRule        config.GateEvaluationRule `json:"evaluation_rule,omitempty"`
	FailedEndpoints       []string                  `json:"failed_endpoints"`
	FailedProfiles        []string                  `json:"failed_profiles,omitempty"`
	EndpointResults       []EndpointResult          `json:"endpoint_results"`
	ProfileEvaluations    []ProfileEvaluation       `json:"profile_evaluations,omitempty"`
	CrossProfileAggregate *AggregateResult          `json:"cross_profile_aggregate,omitempty"`
}

func Evaluate(availability map[string]float64, policy config.Policy, modeOverride string) (Evaluation, error) {
	mode := policy.Mode
	if modeOverride != "" {
		mode = config.PolicyMode(modeOverride)
	}
	if !isValidMode(mode) {
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
		status := classify(mode, availability[endpointID] < threshold)
		if status != StatusPass {
			failed = append(failed, endpointID)
		}
		results = append(results, EndpointResult{
			EndpointID:   endpointID,
			Availability: availability[endpointID],
			Threshold:    threshold,
			Status:       status,
		})
	}
	return Evaluation{
		Mode:            mode,
		Decision:        aggregateDecision(mode, len(failed) > 0),
		FailedEndpoints: failed,
		EndpointResults: results,
	}, nil
}

func EvaluateProfiles(outputs []simulation.ProfileOutput, gateCfg config.GateConfig) (Evaluation, error) {
	if !isValidMode(gateCfg.Mode) {
		return Evaluation{}, fmt.Errorf("unsupported gate mode: %q", gateCfg.Mode)
	}
	if gateCfg.EvaluationRule == "" {
		gateCfg.EvaluationRule = config.GateEvaluationAllProfiles
	}

	eval := Evaluation{
		Mode:               gateCfg.Mode,
		EvaluationRule:     gateCfg.EvaluationRule,
		ProfileEvaluations: make([]ProfileEvaluation, 0, len(outputs)),
	}
	aggregateFailedProfiles := 0
	passingProfiles := 0
	unionFailedEndpoints := map[string]struct{}{}

	for _, output := range outputs {
		profileEval := evaluateProfile(output, gateCfg)
		if profileEval.Decision != StatusPass {
			eval.FailedProfiles = append(eval.FailedProfiles, output.Name)
			aggregateFailedProfiles++
		} else {
			passingProfiles++
		}
		eval.ProfileEvaluations = append(eval.ProfileEvaluations, profileEval)
		for _, endpoint := range profileEval.FailedEndpoints {
			unionFailedEndpoints[endpoint] = struct{}{}
		}
		if len(eval.EndpointResults) == 0 {
			eval.EndpointResults = slices.Clone(profileEval.EndpointResults)
			eval.FailedEndpoints = slices.Clone(profileEval.FailedEndpoints)
		}
	}

	if len(outputs) > 0 && gateCfg.CrossProfileAggregateThreshold != nil {
		crossProfile := 0.0
		for _, output := range outputs {
			crossProfile += output.WeightedAggregate
		}
		crossProfile /= float64(len(outputs))
		aggregateFailed := crossProfile < *gateCfg.CrossProfileAggregateThreshold
		eval.CrossProfileAggregate = &AggregateResult{
			Availability: crossProfile,
			Threshold:    *gateCfg.CrossProfileAggregateThreshold,
			Status:       classify(gateCfg.Mode, aggregateFailed),
		}
		if aggregateFailed {
			aggregateFailedProfiles = len(outputs)
			passingProfiles = 0
		}
	}

	failedEndpoints := make([]string, 0, len(unionFailedEndpoints))
	for endpoint := range unionFailedEndpoints {
		failedEndpoints = append(failedEndpoints, endpoint)
	}
	slices.Sort(failedEndpoints)
	if len(failedEndpoints) > 0 {
		eval.FailedEndpoints = failedEndpoints
	}

	switch gateCfg.Mode {
	case config.ModeReport:
		eval.Decision = "report"
	case config.ModeFail:
		switch gateCfg.EvaluationRule {
		case config.GateEvaluationAnyProfile:
			if passingProfiles == 0 {
				eval.Decision = StatusFail
			} else {
				eval.Decision = StatusPass
			}
		default:
			if aggregateFailedProfiles > 0 {
				eval.Decision = StatusFail
			} else {
				eval.Decision = StatusPass
			}
		}
	case config.ModeWarn:
		switch gateCfg.EvaluationRule {
		case config.GateEvaluationAnyProfile:
			if passingProfiles == 0 {
				eval.Decision = StatusWarn
			} else {
				eval.Decision = StatusPass
			}
		default:
			if aggregateFailedProfiles > 0 {
				eval.Decision = StatusWarn
			} else {
				eval.Decision = StatusPass
			}
		}
	}
	return eval, nil
}

func evaluateProfile(output simulation.ProfileOutput, gateCfg config.GateConfig) ProfileEvaluation {
	endpointIDs := make([]string, 0, len(output.EndpointAvailability))
	for endpointID := range output.EndpointAvailability {
		endpointIDs = append(endpointIDs, endpointID)
	}
	slices.Sort(endpointIDs)

	results := make([]EndpointResult, 0, len(endpointIDs))
	failed := make([]string, 0)
	for _, endpointID := range endpointIDs {
		threshold := gateCfg.GlobalThreshold
		if specific, ok := gateCfg.EndpointThresholds[endpointID]; ok {
			threshold = specific
		}
		if profileThresholds, ok := gateCfg.ProfileEndpointThresholds[output.Name]; ok {
			if specific, ok := profileThresholds[endpointID]; ok {
				threshold = specific
			}
		}
		availability := output.EndpointAvailability[endpointID]
		status := classify(gateCfg.Mode, availability < threshold)
		if status != StatusPass {
			failed = append(failed, endpointID)
		}
		results = append(results, EndpointResult{
			Profile:      output.Name,
			EndpointID:   endpointID,
			Availability: availability,
			Threshold:    threshold,
			Status:       status,
		})
	}

	var aggregate *AggregateResult
	aggregateFailed := false
	if threshold, ok := gateCfg.ProfileAggregateThresholds[output.Name]; ok {
		aggregateFailed = output.WeightedAggregate < threshold
		aggregate = &AggregateResult{
			Availability: output.WeightedAggregate,
			Threshold:    threshold,
			Status:       classify(gateCfg.Mode, aggregateFailed),
		}
	} else if gateCfg.AggregateThreshold != nil {
		aggregateFailed = output.WeightedAggregate < *gateCfg.AggregateThreshold
		aggregate = &AggregateResult{
			Availability: output.WeightedAggregate,
			Threshold:    *gateCfg.AggregateThreshold,
			Status:       classify(gateCfg.Mode, aggregateFailed),
		}
	}

	return ProfileEvaluation{
		Profile:                 output.Name,
		Decision:                aggregateDecision(gateCfg.Mode, len(failed) > 0 || aggregateFailed),
		FailedEndpoints:         failed,
		EndpointsBelowThreshold: len(failed),
		EndpointResults:         results,
		Aggregate:               aggregate,
	}
}

func classify(mode config.PolicyMode, failed bool) string {
	if !failed {
		return StatusPass
	}
	switch mode {
	case config.ModeFail:
		return StatusFail
	case config.ModeWarn, config.ModeReport:
		return StatusWarn
	default:
		return StatusFail
	}
}

func aggregateDecision(mode config.PolicyMode, failed bool) string {
	switch mode {
	case config.ModeReport:
		return "report"
	case config.ModeWarn:
		if failed {
			return StatusWarn
		}
		return StatusPass
	case config.ModeFail:
		if failed {
			return StatusFail
		}
		return StatusPass
	default:
		return StatusFail
	}
}

func isValidMode(mode config.PolicyMode) bool {
	return mode == config.ModeWarn || mode == config.ModeFail || mode == config.ModeReport
}
