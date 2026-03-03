package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func validateJSONAgainstSchemaRequired(t *testing.T, schemaPath, dataPath string) {
	t.Helper()

	schemaRaw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema %s: %v", schemaPath, err)
	}
	dataRaw, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("read data %s: %v", dataPath, err)
	}

	var schema any
	if err := json.Unmarshal(schemaRaw, &schema); err != nil {
		t.Fatalf("decode schema json: %v", err)
	}
	var data any
	if err := json.Unmarshal(dataRaw, &data); err != nil {
		t.Fatalf("decode data json: %v", err)
	}

	if err := validateNode(schema, data, "$"); err != nil {
		t.Fatalf("schema required validation failed: %v", err)
	}
}

func validateNode(schema any, data any, path string) error {
	sObj, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	dataType := strings.ToLower(asString(sObj["type"]))

	switch dataType {
	case "object":
		dObj, ok := data.(map[string]any)
		if !ok {
			return fmt.Errorf("%s expected object", path)
		}
		if reqList, ok := sObj["required"].([]any); ok {
			for _, r := range reqList {
				key := asString(r)
				if _, exists := dObj[key]; !exists {
					return fmt.Errorf("%s missing required field %q", path, key)
				}
			}
		}
		props, _ := sObj["properties"].(map[string]any)
		for key, propSchema := range props {
			if child, exists := dObj[key]; exists {
				if err := validateNode(propSchema, child, path+"."+key); err != nil {
					return err
				}
			}
		}
	case "array":
		itemsSchema, hasItems := sObj["items"]
		if !hasItems {
			return nil
		}
		arr, ok := data.([]any)
		if !ok {
			return fmt.Errorf("%s expected array", path)
		}
		for idx, item := range arr {
			if err := validateNode(itemsSchema, item, fmt.Sprintf("%s[%d]", path, idx)); err != nil {
				return err
			}
		}
	default:
		// Scalar validation intentionally omitted in MVP tests.
	}
	return nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
