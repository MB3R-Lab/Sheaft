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

func TestRequiredEdgesNeedVersionedContract(t *testing.T) {
	d := Definition{Type: TypeEdgeAware, EntryService: "frontend", MandatoryTargets: []string{"catalog"}, EdgeModes: []string{"sync"}, RequiredEdges: []string{"rpc-id"}}
	c := Contract{SchemaVersion: SchemaVersion, Predicates: Set{"operation": d}}
	if err := c.Validate(); err == nil {
		t.Fatal("version 1.0 must not accept required_edges")
	}
	c.SchemaVersion = SchemaVersionRequiredEdges
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	d.RequiredEdges = []string{"rpc-id", "rpc-id"}
	if err := d.Validate(); err == nil {
		t.Fatal("duplicate required edges accepted")
	}
	d.Type = TypeAllOf
	if err := d.Validate(); err == nil {
		t.Fatal("required edges silently ignored by a service-only predicate")
	}
}
