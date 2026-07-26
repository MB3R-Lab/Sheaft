package gate

import (
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

func TestEvaluate_ModeWarn(t *testing.T) {
	t.Parallel()

	policy := config.Policy{
		Mode:            config.ModeWarn,
		DefaultAction:   config.ModeWarn,
		GlobalThreshold: 0.95,
		Trials:          1000,
		EndpointThresholds: map[string]float64{
			"frontend:GET /health": 0.99,
		},
	}.Normalized()
	availability := map[string]float64{
		"frontend:GET /health":   0.98,
		"frontend:GET /checkout": 0.96,
	}

	eval, err := Evaluate(availability, policy, "")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if eval.Decision != "warn" {
		t.Fatalf("expected warn decision, got %s", eval.Decision)
	}
	if len(eval.FailedEndpoints) != 1 {
		t.Fatalf("expected 1 failed endpoint, got %d", len(eval.FailedEndpoints))
	}
	if len(eval.Reasons) != 1 || eval.Reasons[0].ID != "endpoint_below_threshold" {
		t.Fatalf("expected endpoint why reason, got %+v", eval.Reasons)
	}
	if eval.EndpointResults[0].ThresholdDelta == 0 && eval.EndpointResults[0].EndpointID == "frontend:GET /health" {
		t.Fatalf("expected threshold delta to be recorded, got %+v", eval.EndpointResults[0])
	}
}

func TestEvaluate_ModeFail(t *testing.T) {
	t.Parallel()

	policy := config.Policy{
		Mode:            config.ModeFail,
		DefaultAction:   config.ModeFail,
		GlobalThreshold: 0.99,
		Trials:          1000,
	}.Normalized()
	availability := map[string]float64{
		"frontend:GET /checkout": 0.90,
	}

	eval, err := Evaluate(availability, policy, "")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if eval.Decision != "fail" {
		t.Fatalf("expected fail decision, got %s", eval.Decision)
	}
}

func TestEvaluateProfiles_AssertionFailuresAffectGate(t *testing.T) {
	t.Parallel()

	eval, err := EvaluateProfiles([]simulation.ProfileOutput{
		{
			Name:                 "brownout",
			WeightedAggregate:    0.99,
			EndpointAvailability: map[string]float64{"gateway:POST /checkout": 0.99},
			Assertions: []simulation.AssertionResult{
				{Metric: "timeout_mismatch_count", Status: "fail", Available: true, ActualValue: 2, Expected: 0, Op: "=="},
			},
		},
	}, config.GateConfig{
		Mode:            config.ModeFail,
		DefaultAction:   config.ModeFail,
		EvaluationRule:  config.GateEvaluationAllProfiles,
		GlobalThreshold: 0.95,
	})
	if err != nil {
		t.Fatalf("EvaluateProfiles returned error: %v", err)
	}
	if eval.Decision != "fail" {
		t.Fatalf("expected assertion failure to fail gate, got %+v", eval)
	}
	if len(eval.FailedAssertions) != 1 {
		t.Fatalf("expected failed assertion to be surfaced, got %+v", eval.FailedAssertions)
	}
	if len(eval.Reasons) == 0 {
		t.Fatalf("expected why reasons for assertion failure, got %+v", eval)
	}
	foundAssertionReason := false
	for _, reason := range eval.Reasons {
		if reason.ID == "assertion_failed" {
			foundAssertionReason = true
		}
	}
	if !foundAssertionReason {
		t.Fatalf("expected assertion_failed why reason, got %+v", eval.Reasons)
	}
}

func TestEvaluateAnalysis_CertifiedBoundaryRules(t *testing.T) {
	t.Parallel()

	profiles := []simulation.ProfileOutput{{Name: "steady", EndpointAvailability: map[string]float64{"checkout": 1}, WeightedAggregate: 1}}
	sweep := certifiedSweep(0.1, simulation.BinomialInterval{Estimate: 0.995, LowerBound: 0.991, UpperBound: 0.998})
	minimum := 0.1
	eval, err := EvaluateAnalysis(profiles, []simulation.SweepOutput{sweep}, nil, config.GateConfig{
		Mode: config.ModeFail, DefaultAction: config.ModeFail, GlobalThreshold: 0.9,
		BoundaryRules: []config.BoundaryRule{{Sweep: "checkout-boundary", EndpointID: "checkout", MinimumCertifiedTolerance: &minimum}},
	})
	if err != nil {
		t.Fatalf("EvaluateAnalysis failed: %v", err)
	}
	if eval.Decision != StatusPass || len(eval.BoundaryResults) != 1 || eval.BoundaryResults[0].Status != StatusPass {
		t.Fatalf("expected certified boundary pass, got %+v", eval)
	}
}

func TestEvaluateAnalysis_DistinguishesViolationAndIndeterminate(t *testing.T) {
	t.Parallel()

	profiles := []simulation.ProfileOutput{{Name: "steady", EndpointAvailability: map[string]float64{"checkout": 1}, WeightedAggregate: 1}}
	minimum := 0.1
	baseGate := config.GateConfig{
		Mode: config.ModeFail, DefaultAction: config.ModeFail, GlobalThreshold: 0.9,
		BoundaryRules: []config.BoundaryRule{{Sweep: "checkout-boundary", EndpointID: "checkout", MinimumCertifiedTolerance: &minimum}},
	}
	violating := certifiedSweep(0, simulation.BinomialInterval{Estimate: 0.97, LowerBound: 0.96, UpperBound: 0.98})
	eval, err := EvaluateAnalysis(profiles, []simulation.SweepOutput{violating}, nil, baseGate)
	if err != nil {
		t.Fatalf("EvaluateAnalysis failed: %v", err)
	}
	if eval.Decision != StatusFail || eval.Reasons[len(eval.Reasons)-1].ID != "boundary_below_minimum" {
		t.Fatalf("expected definite boundary violation, got %+v", eval)
	}

	indeterminate := certifiedSweep(0, simulation.BinomialInterval{Estimate: 0.99, LowerBound: 0.98, UpperBound: 0.995})
	eval, err = EvaluateAnalysis(profiles, []simulation.SweepOutput{indeterminate}, nil, baseGate)
	if err != nil {
		t.Fatalf("EvaluateAnalysis failed: %v", err)
	}
	if eval.Decision != StatusFail || eval.Reasons[len(eval.Reasons)-1].ID != "boundary_indeterminate" {
		t.Fatalf("expected fail-closed indeterminate boundary, got %+v", eval)
	}
	baseGate.BoundaryRules[0].IndeterminateAction = config.ModeWarn
	eval, err = EvaluateAnalysis(profiles, []simulation.SweepOutput{indeterminate}, nil, baseGate)
	if err != nil {
		t.Fatalf("EvaluateAnalysis failed: %v", err)
	}
	if eval.Decision != StatusWarn || eval.BoundaryResults[0].Status != StatusWarn {
		t.Fatalf("expected explicit warn action for indeterminate boundary, got %+v", eval)
	}
}

func TestEvaluateAnalysis_BaselineRegression(t *testing.T) {
	t.Parallel()

	profiles := []simulation.ProfileOutput{{Name: "steady", EndpointAvailability: map[string]float64{"checkout": 1}, WeightedAggregate: 1}}
	sweep := certifiedSweep(0.1, simulation.BinomialInterval{Estimate: 0.995, LowerBound: 0.991, UpperBound: 0.998})
	maxRegression := 0.05
	referenceTolerance := 0.2
	eval, err := EvaluateAnalysis(profiles, []simulation.SweepOutput{sweep}, []BoundaryReference{
		{Baseline: "last-release", Sweep: "checkout-boundary", EndpointID: "checkout", Compatible: true, CertifiedTolerance: &referenceTolerance},
	}, config.GateConfig{
		Mode: config.ModeFail, DefaultAction: config.ModeFail, GlobalThreshold: 0.9,
		BoundaryRules: []config.BoundaryRule{{Sweep: "checkout-boundary", EndpointID: "checkout", Baseline: "last-release", MaxRegression: &maxRegression}},
	})
	if err != nil {
		t.Fatalf("EvaluateAnalysis failed: %v", err)
	}
	if eval.Decision != StatusFail || eval.BoundaryResults[0].Regression == nil || *eval.BoundaryResults[0].Regression > -0.09 || eval.Reasons[len(eval.Reasons)-1].ID != "boundary_regressed" {
		t.Fatalf("expected boundary regression failure, got %+v", eval)
	}
}

func certifiedSweep(certifiedAxis float64, requiredInterval simulation.BinomialInterval) simulation.SweepOutput {
	certified := &simulation.BoundaryPoint{AxisValue: certifiedAxis, Availability: 1, ConfidenceLowerBound: 0.999, ConfidenceUpperBound: 1}
	return simulation.SweepOutput{
		Name: "checkout-boundary", AxisType: config.SweepAxisIndependentReplicaFailureProbability, Fingerprint: "sha256:test",
		Points: []simulation.SweepPoint{
			{AxisValue: 0, EndpointAvailability: map[string]float64{"checkout": 1}, EndpointConfidence: map[string]simulation.BinomialInterval{"checkout": {Estimate: 1, LowerBound: 0.999, UpperBound: 1}}},
			{AxisValue: 0.1, EndpointAvailability: map[string]float64{"checkout": requiredInterval.Estimate}, EndpointConfidence: map[string]simulation.BinomialInterval{"checkout": requiredInterval}},
		},
		Boundaries: []simulation.SweepBoundary{{EndpointID: "checkout", SLO: 0.99, ObservedMonotonic: true, CertifiedTolerance: certified}},
	}
}
