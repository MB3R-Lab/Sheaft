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
	Profile        string  `json:"profile,omitempty"`
	EndpointID     string  `json:"endpoint_id"`
	Availability   float64 `json:"availability"`
	Threshold      float64 `json:"threshold"`
	ThresholdDelta float64 `json:"threshold_delta"`
	Status         string  `json:"status"`
}

type AggregateResult struct {
	Availability   float64 `json:"availability"`
	Threshold      float64 `json:"threshold"`
	ThresholdDelta float64 `json:"threshold_delta"`
	Status         string  `json:"status"`
}

type DecisionReason struct {
	ID           string   `json:"id"`
	Scope        string   `json:"scope"`
	Profile      string   `json:"profile,omitempty"`
	Sweep        string   `json:"sweep,omitempty"`
	Baseline     string   `json:"baseline,omitempty"`
	EndpointID   string   `json:"endpoint_id,omitempty"`
	Metric       string   `json:"metric,omitempty"`
	Status       string   `json:"status"`
	Availability *float64 `json:"availability,omitempty"`
	Threshold    *float64 `json:"threshold,omitempty"`
	Delta        *float64 `json:"delta,omitempty"`
	Message      string   `json:"message"`
}

type BoundaryReference struct {
	Baseline           string
	Sweep              string
	EndpointID         string
	Fingerprint        string
	Compatible         bool
	Reason             string
	CertifiedTolerance *float64
}

type BoundaryResult struct {
	Sweep                     string   `json:"sweep"`
	EndpointID                string   `json:"endpoint_id"`
	Status                    string   `json:"status"`
	Reason                    string   `json:"reason,omitempty"`
	CertifiedTolerance        *float64 `json:"certified_tolerance,omitempty"`
	MinimumCertifiedTolerance *float64 `json:"minimum_certified_tolerance,omitempty"`
	Baseline                  string   `json:"baseline,omitempty"`
	ReferenceTolerance        *float64 `json:"reference_tolerance,omitempty"`
	Regression                *float64 `json:"regression,omitempty"`
	MaxRegression             *float64 `json:"max_regression,omitempty"`
}

type ProfileEvaluation struct {
	Profile                 string           `json:"profile"`
	Decision                string           `json:"decision"`
	FailedEndpoints         []string         `json:"failed_endpoints"`
	FailedAssertions        []string         `json:"failed_assertions,omitempty"`
	EndpointsBelowThreshold int              `json:"endpoints_below_threshold"`
	EndpointResults         []EndpointResult `json:"endpoint_results"`
	Aggregate               *AggregateResult `json:"aggregate,omitempty"`
	Reasons                 []DecisionReason `json:"reasons,omitempty"`
}

type Evaluation struct {
	Mode                  config.PolicyMode         `json:"mode"`
	Decision              string                    `json:"decision"`
	EvaluationRule        config.GateEvaluationRule `json:"evaluation_rule,omitempty"`
	FailedEndpoints       []string                  `json:"failed_endpoints"`
	FailedAssertions      []string                  `json:"failed_assertions,omitempty"`
	FailedProfiles        []string                  `json:"failed_profiles,omitempty"`
	FailedBoundaries      []string                  `json:"failed_boundaries,omitempty"`
	EndpointResults       []EndpointResult          `json:"endpoint_results"`
	ProfileEvaluations    []ProfileEvaluation       `json:"profile_evaluations,omitempty"`
	CrossProfileAggregate *AggregateResult          `json:"cross_profile_aggregate,omitempty"`
	BoundaryResults       []BoundaryResult          `json:"boundary_results,omitempty"`
	Reasons               []DecisionReason          `json:"reasons,omitempty"`
}

func EvaluateAnalysis(outputs []simulation.ProfileOutput, sweeps []simulation.SweepOutput, references []BoundaryReference, gateCfg config.GateConfig) (Evaluation, error) {
	eval, err := EvaluateProfiles(outputs, gateCfg)
	if err != nil || len(gateCfg.BoundaryRules) == 0 {
		return eval, err
	}

	for _, rule := range gateCfg.BoundaryRules {
		result, reason := evaluateBoundaryRule(rule, sweeps, references, gateCfg)
		eval.BoundaryResults = append(eval.BoundaryResults, result)
		if result.Status == StatusPass {
			continue
		}
		eval.FailedBoundaries = append(eval.FailedBoundaries, rule.Sweep+":"+rule.EndpointID)
		eval.Reasons = append(eval.Reasons, reason)
	}
	if len(eval.FailedBoundaries) > 0 {
		eval.Reasons = removePassReasons(eval.Reasons)
	}
	slices.Sort(eval.FailedBoundaries)
	eval.Decision = mergeBoundaryDecision(eval.Decision, eval.BoundaryResults, gateCfg.Mode)
	return eval, nil
}

func evaluateBoundaryRule(rule config.BoundaryRule, sweeps []simulation.SweepOutput, references []BoundaryReference, gateCfg config.GateConfig) (BoundaryResult, DecisionReason) {
	result := BoundaryResult{
		Sweep: rule.Sweep, EndpointID: rule.EndpointID, Status: StatusPass,
		MinimumCertifiedTolerance: cloneFloatPointer(rule.MinimumCertifiedTolerance),
		Baseline:                  rule.Baseline, MaxRegression: cloneFloatPointer(rule.MaxRegression),
	}
	sweep, boundary := findSweepBoundary(sweeps, rule.Sweep, rule.EndpointID)
	if sweep == nil || boundary == nil {
		return indeterminateBoundary(result, rule, gateCfg, "required sweep boundary is unavailable")
	}
	if boundary.CertifiedTolerance != nil {
		result.CertifiedTolerance = ptrFloat64(boundary.CertifiedTolerance.AxisValue)
	}
	if !boundary.ObservedMonotonic {
		return indeterminateBoundary(result, rule, gateCfg, "observed sweep is non-monotonic")
	}

	if rule.MinimumCertifiedTolerance != nil {
		if result.CertifiedTolerance != nil && *result.CertifiedTolerance >= *rule.MinimumCertifiedTolerance {
			// The entire evaluated prefix through the required point is certified.
		} else {
			point, ok := findSweepPoint(*sweep, *rule.MinimumCertifiedTolerance)
			if !ok {
				return indeterminateBoundary(result, rule, gateCfg, "required minimum is not an evaluated sweep point")
			}
			interval := point.EndpointConfidence[rule.EndpointID]
			if interval.UpperBound < boundary.SLO {
				result.Status = classify(gateCfg.Mode, true)
				result.Reason = "certified tolerance is below the required minimum"
				return result, boundaryReason("boundary_below_minimum", result, result.Reason)
			}
			return indeterminateBoundary(result, rule, gateCfg, "available trials do not certify the required minimum")
		}
	}

	if rule.MaxRegression != nil {
		reference := findBoundaryReference(references, rule.Baseline, rule.Sweep, rule.EndpointID)
		if reference == nil || !reference.Compatible || reference.CertifiedTolerance == nil || result.CertifiedTolerance == nil {
			reason := "compatible certified baseline boundary is unavailable"
			if reference != nil && reference.Reason != "" {
				reason = reference.Reason
			}
			return indeterminateBoundary(result, rule, gateCfg, reason)
		}
		result.ReferenceTolerance = cloneFloatPointer(reference.CertifiedTolerance)
		regression := *result.CertifiedTolerance - *reference.CertifiedTolerance
		result.Regression = ptrFloat64(regression)
		if regression < -*rule.MaxRegression {
			result.Status = classify(gateCfg.Mode, true)
			result.Reason = "certified tolerance regression exceeds the configured budget"
			return result, boundaryReason("boundary_regressed", result, result.Reason)
		}
	}
	return result, DecisionReason{}
}

func indeterminateBoundary(result BoundaryResult, rule config.BoundaryRule, gateCfg config.GateConfig, message string) (BoundaryResult, DecisionReason) {
	action := rule.IndeterminateAction
	if action == "" {
		action = gateCfg.DefaultAction
	}
	if action == "" {
		action = gateCfg.Mode
	}
	result.Status = classify(action, true)
	result.Reason = message
	return result, boundaryReason("boundary_indeterminate", result, message)
}

func boundaryReason(id string, result BoundaryResult, message string) DecisionReason {
	return DecisionReason{
		ID: id, Scope: "boundary", Sweep: result.Sweep, Baseline: result.Baseline,
		EndpointID: result.EndpointID, Status: result.Status, Delta: cloneFloatPointer(result.Regression),
		Threshold: cloneFloatPointer(result.MinimumCertifiedTolerance), Message: message,
	}
}

func findSweepBoundary(sweeps []simulation.SweepOutput, sweepName, endpointID string) (*simulation.SweepOutput, *simulation.SweepBoundary) {
	for sweepIndex := range sweeps {
		if sweeps[sweepIndex].Name != sweepName {
			continue
		}
		for boundaryIndex := range sweeps[sweepIndex].Boundaries {
			if sweeps[sweepIndex].Boundaries[boundaryIndex].EndpointID == endpointID {
				return &sweeps[sweepIndex], &sweeps[sweepIndex].Boundaries[boundaryIndex]
			}
		}
		return &sweeps[sweepIndex], nil
	}
	return nil, nil
}

func findSweepPoint(sweep simulation.SweepOutput, axisValue float64) (simulation.SweepPoint, bool) {
	for _, point := range sweep.Points {
		if point.AxisValue == axisValue {
			return point, true
		}
	}
	return simulation.SweepPoint{}, false
}

func findBoundaryReference(references []BoundaryReference, baseline, sweep, endpointID string) *BoundaryReference {
	for idx := range references {
		if references[idx].Baseline == baseline && references[idx].Sweep == sweep && references[idx].EndpointID == endpointID {
			return &references[idx]
		}
	}
	return nil
}

func mergeBoundaryDecision(current string, results []BoundaryResult, mode config.PolicyMode) string {
	if mode == config.ModeReport {
		return "report"
	}
	decision := current
	for _, result := range results {
		if result.Status == StatusFail {
			return StatusFail
		}
		if result.Status == StatusWarn && decision == StatusPass {
			decision = StatusWarn
		}
	}
	return decision
}

func removePassReasons(reasons []DecisionReason) []DecisionReason {
	out := reasons[:0]
	for _, reason := range reasons {
		if reason.ID != "gate_pass" {
			out = append(out, reason)
		}
	}
	return out
}

func cloneFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	return ptrFloat64(*value)
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
	reasons := make([]DecisionReason, 0)
	for _, endpointID := range endpointIDs {
		threshold := policy.GlobalThreshold
		if specific, ok := policy.EndpointThresholds[endpointID]; ok {
			threshold = specific
		}
		value := availability[endpointID]
		delta := value - threshold
		status := classify(mode, value < threshold)
		if status != StatusPass {
			failed = append(failed, endpointID)
			reasons = append(reasons, endpointBelowThresholdReason("", endpointID, value, threshold, delta, status))
		}
		results = append(results, EndpointResult{
			EndpointID:     endpointID,
			Availability:   value,
			Threshold:      threshold,
			ThresholdDelta: delta,
			Status:         status,
		})
	}
	decision := aggregateDecision(mode, len(failed) > 0)
	if len(reasons) == 0 {
		reasons = append(reasons, passReason(decision))
	}
	return Evaluation{
		Mode:            mode,
		Decision:        decision,
		FailedEndpoints: failed,
		EndpointResults: results,
		Reasons:         reasons,
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
	unionFailedAssertions := map[string]struct{}{}

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
		for _, assertion := range profileEval.FailedAssertions {
			unionFailedAssertions[assertion] = struct{}{}
		}
		eval.Reasons = append(eval.Reasons, profileEval.Reasons...)
		if len(eval.EndpointResults) == 0 {
			eval.EndpointResults = slices.Clone(profileEval.EndpointResults)
			eval.FailedEndpoints = slices.Clone(profileEval.FailedEndpoints)
			eval.FailedAssertions = slices.Clone(profileEval.FailedAssertions)
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
			Availability:   crossProfile,
			Threshold:      *gateCfg.CrossProfileAggregateThreshold,
			ThresholdDelta: crossProfile - *gateCfg.CrossProfileAggregateThreshold,
			Status:         classify(gateCfg.Mode, aggregateFailed),
		}
		if aggregateFailed {
			aggregateFailedProfiles = len(outputs)
			passingProfiles = 0
			eval.Reasons = append(eval.Reasons, aggregateBelowThresholdReason("", "cross_profile_aggregate", crossProfile, *gateCfg.CrossProfileAggregateThreshold, eval.CrossProfileAggregate.ThresholdDelta, eval.CrossProfileAggregate.Status))
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
	if len(unionFailedAssertions) > 0 {
		failedAssertions := make([]string, 0, len(unionFailedAssertions))
		for assertion := range unionFailedAssertions {
			failedAssertions = append(failedAssertions, assertion)
		}
		slices.Sort(failedAssertions)
		eval.FailedAssertions = failedAssertions
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
	if len(eval.Reasons) == 0 {
		eval.Reasons = append(eval.Reasons, passReason(eval.Decision))
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
	failedAssertions := make([]string, 0)
	reasons := make([]DecisionReason, 0)
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
		delta := availability - threshold
		status := classify(gateCfg.Mode, availability < threshold)
		if status != StatusPass {
			failed = append(failed, endpointID)
			reasons = append(reasons, endpointBelowThresholdReason(output.Name, endpointID, availability, threshold, delta, status))
		}
		results = append(results, EndpointResult{
			Profile:        output.Name,
			EndpointID:     endpointID,
			Availability:   availability,
			Threshold:      threshold,
			ThresholdDelta: delta,
			Status:         status,
		})
	}

	var aggregate *AggregateResult
	aggregateFailed := false
	if threshold, ok := gateCfg.ProfileAggregateThresholds[output.Name]; ok {
		aggregateFailed = output.WeightedAggregate < threshold
		aggregate = &AggregateResult{
			Availability:   output.WeightedAggregate,
			Threshold:      threshold,
			ThresholdDelta: output.WeightedAggregate - threshold,
			Status:         classify(gateCfg.Mode, aggregateFailed),
		}
	} else if gateCfg.AggregateThreshold != nil {
		aggregateFailed = output.WeightedAggregate < *gateCfg.AggregateThreshold
		aggregate = &AggregateResult{
			Availability:   output.WeightedAggregate,
			Threshold:      *gateCfg.AggregateThreshold,
			ThresholdDelta: output.WeightedAggregate - *gateCfg.AggregateThreshold,
			Status:         classify(gateCfg.Mode, aggregateFailed),
		}
	}
	if aggregateFailed && aggregate != nil {
		reasons = append(reasons, aggregateBelowThresholdReason(output.Name, "profile_weighted_aggregate", aggregate.Availability, aggregate.Threshold, aggregate.ThresholdDelta, aggregate.Status))
	}
	for _, assertion := range output.Assertions {
		if assertion.Status == StatusPass {
			continue
		}
		formatted := formatAssertionFailure(assertion)
		failedAssertions = append(failedAssertions, formatted)
		reasons = append(reasons, assertionReason(output.Name, assertion, classify(gateCfg.Mode, true), formatted))
	}

	return ProfileEvaluation{
		Profile:                 output.Name,
		Decision:                aggregateDecision(gateCfg.Mode, len(failed) > 0 || aggregateFailed || len(failedAssertions) > 0),
		FailedEndpoints:         failed,
		FailedAssertions:        failedAssertions,
		EndpointsBelowThreshold: len(failed),
		EndpointResults:         results,
		Aggregate:               aggregate,
		Reasons:                 reasons,
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

func formatAssertionFailure(result simulation.AssertionResult) string {
	if result.Available {
		return fmt.Sprintf("%s %s %s %.4f (actual=%.4f)", result.Metric, result.Target.Type, result.Op, result.Expected, result.ActualValue)
	}
	return fmt.Sprintf("%s %s unavailable: %s", result.Metric, result.Target.Type, result.Reason)
}

func endpointBelowThresholdReason(profile, endpointID string, availability, threshold, delta float64, status string) DecisionReason {
	return DecisionReason{
		ID:           "endpoint_below_threshold",
		Scope:        "endpoint",
		Profile:      profile,
		EndpointID:   endpointID,
		Status:       status,
		Availability: ptrFloat64(availability),
		Threshold:    ptrFloat64(threshold),
		Delta:        ptrFloat64(delta),
		Message:      fmt.Sprintf("endpoint %q availability %.4f is below threshold %.4f", endpointID, availability, threshold),
	}
}

func aggregateBelowThresholdReason(profile, scope string, availability, threshold, delta float64, status string) DecisionReason {
	return DecisionReason{
		ID:           "aggregate_below_threshold",
		Scope:        scope,
		Profile:      profile,
		Status:       status,
		Availability: ptrFloat64(availability),
		Threshold:    ptrFloat64(threshold),
		Delta:        ptrFloat64(delta),
		Message:      fmt.Sprintf("%s availability %.4f is below threshold %.4f", scope, availability, threshold),
	}
}

func assertionReason(profile string, result simulation.AssertionResult, status string, formatted string) DecisionReason {
	id := "assertion_failed"
	if !result.Available {
		id = "assertion_unavailable"
	}
	return DecisionReason{
		ID:      id,
		Scope:   "assertion",
		Profile: profile,
		Metric:  result.Metric,
		Status:  status,
		Message: formatted,
	}
}

func passReason(decision string) DecisionReason {
	return DecisionReason{
		ID:      "gate_pass",
		Scope:   "gate",
		Status:  decision,
		Message: "all evaluated endpoints, assertions, and aggregates satisfy the configured gate",
	}
}

func ptrFloat64(value float64) *float64 {
	out := value
	return &out
}
