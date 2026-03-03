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
	vendoredPath := filepath.Join(pkgDir, "schema", "model.schema.json")
	apiPath := filepath.Join(pkgDir, "..", "..", "api", "schema", "model.schema.json")

	vendoredRaw, err := os.ReadFile(vendoredPath)
	if err != nil {
		t.Fatalf("read vendored schema: %v", err)
	}
	apiRaw, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("read api schema: %v", err)
	}

	var vendored any
	if err := json.Unmarshal(vendoredRaw, &vendored); err != nil {
		t.Fatalf("decode vendored schema json: %v", err)
	}
	var api any
	if err := json.Unmarshal(apiRaw, &api); err != nil {
		t.Fatalf("decode api schema json: %v", err)
	}

	vendoredJSON, err := json.Marshal(vendored)
	if err != nil {
		t.Fatalf("marshal vendored schema: %v", err)
	}
	apiJSON, err := json.Marshal(api)
	if err != nil {
		t.Fatalf("marshal api schema: %v", err)
	}

	if string(vendoredJSON) != string(apiJSON) {
		t.Fatalf("schema mismatch: %s and %s must stay identical", vendoredPath, apiPath)
	}
}
