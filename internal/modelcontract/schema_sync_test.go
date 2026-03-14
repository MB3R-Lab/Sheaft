package modelcontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAPISchemaStaysInSyncWithVendoredSchema(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	pkgDir := filepath.Dir(thisFile)
	for _, name := range []string{"model.schema.json", "snapshot.schema.json"} {
		vendoredPath := filepath.Join(pkgDir, "schema", name)
		apiPath := filepath.Join(pkgDir, "..", "..", "api", "schema", name)

		vendoredRaw, err := os.ReadFile(vendoredPath)
		if err != nil {
			t.Fatalf("read vendored schema %s: %v", name, err)
		}
		apiRaw, err := os.ReadFile(apiPath)
		if err != nil {
			t.Fatalf("read api schema %s: %v", name, err)
		}

		var vendored any
		if err := json.Unmarshal(vendoredRaw, &vendored); err != nil {
			t.Fatalf("decode vendored schema json %s: %v", name, err)
		}
		var api any
		if err := json.Unmarshal(apiRaw, &api); err != nil {
			t.Fatalf("decode api schema json %s: %v", name, err)
		}

		vendoredJSON, err := json.Marshal(vendored)
		if err != nil {
			t.Fatalf("marshal vendored schema %s: %v", name, err)
		}
		apiJSON, err := json.Marshal(api)
		if err != nil {
			t.Fatalf("marshal api schema %s: %v", name, err)
		}

		if string(vendoredJSON) != string(apiJSON) {
			t.Fatalf("schema mismatch: %s and %s must stay identical", vendoredPath, apiPath)
		}
	}
}
