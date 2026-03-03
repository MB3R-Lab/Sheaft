package gate

import (
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/config"
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
