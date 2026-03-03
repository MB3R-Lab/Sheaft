package otel

import (
	"path/filepath"
	"testing"
)

func TestDiscover_FromFixture(t *testing.T) {
	t.Parallel()

	input := filepath.Join("..", "..", "..", "test", "fixtures", "traces.fixture.json")
	mdl, err := Discover(input)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if got, want := len(mdl.Services), 4; got != want {
		t.Fatalf("services count mismatch: got=%d want=%d", got, want)
	}
	if got, want := len(mdl.Edges), 3; got != want {
		t.Fatalf("edges count mismatch: got=%d want=%d", got, want)
	}
	if got, want := len(mdl.Endpoints), 4; got != want {
		t.Fatalf("endpoints count mismatch: got=%d want=%d", got, want)
	}
}

func TestDiscover_EmptyTraces(t *testing.T) {
	t.Parallel()

	input := filepath.Join("..", "..", "..", "test", "fixtures", "traces.empty.json")
	if _, err := Discover(input); err == nil {
		t.Fatal("expected error for empty traces, got nil")
	}
}
