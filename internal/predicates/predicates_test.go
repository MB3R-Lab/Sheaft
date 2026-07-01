package predicates

import (
	"strings"
	"testing"
)

func TestDefinitionValidate_EdgeAware(t *testing.T) {
	t.Parallel()

	def := Definition{
		Type:             TypeEdgeAware,
		EntryService:     "frontend",
		MandatoryTargets: []string{"checkout"},
		EdgeModes:        []string{EdgeModeSync, EdgeModeAsync},
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("expected edge-aware predicate to validate, got %v", err)
	}
}

func TestDefinitionValidate_EdgeAwareRejectsInvalidMode(t *testing.T) {
	t.Parallel()

	def := Definition{
		Type:             TypeEdgeAware,
		EntryService:     "frontend",
		MandatoryTargets: []string{"checkout"},
		EdgeModes:        []string{"eventual"},
	}
	err := def.Validate()
	if err == nil {
		t.Fatal("expected invalid edge mode to fail validation")
	}
	if !strings.Contains(err.Error(), "edge_modes[0]") {
		t.Fatalf("unexpected error: %v", err)
	}
}
