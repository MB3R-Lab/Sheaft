package oracle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunWritesPassingOracleArtifacts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	rep, err := Run(RunOptions{
		RepositoryRoot: root,
		OutputDir:      "oracle-out",
		GeneratedAt:    time.Unix(0, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if rep.Status != "pass" {
		t.Fatalf("expected oracle suite to pass, got %+v", rep)
	}
	if len(rep.Cases) < 8 {
		t.Fatalf("expected boundary and oracle cases, got %d", len(rep.Cases))
	}
	for _, path := range []string{rep.Outputs.Report, rep.Outputs.Summary} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
}
