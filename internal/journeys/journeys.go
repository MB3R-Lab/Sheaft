package journeys

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/model"
)

const SchemaVersion = "1.0"

type File struct {
	SchemaVersion string                `json:"schema_version"`
	Journeys      map[string][][]string `json:"journeys"`
}

func Load(path string) (map[string][][]string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".json" {
		return nil, fmt.Errorf("unsupported journeys extension %q (expected .json)", ext)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read journeys file: %w", err)
	}

	var payload File
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode journeys json: %w", err)
	}
	if err := payload.validateBasic(); err != nil {
		return nil, err
	}

	return payload.cloneJourneys(), nil
}

func ValidateAgainstModel(overrides map[string][][]string, mdl model.ResilienceModel) error {
	if len(overrides) == 0 {
		return nil
	}

	serviceSet := make(map[string]struct{}, len(mdl.Services))
	for _, svc := range mdl.Services {
		serviceSet[svc.ID] = struct{}{}
	}

	entryByEndpoint := make(map[string]string, len(mdl.Endpoints))
	for _, ep := range mdl.Endpoints {
		entryByEndpoint[ep.ID] = ep.EntryService
	}

	adj := make(map[string]map[string]struct{}, len(mdl.Services))
	for _, edge := range mdl.Edges {
		if !edge.Blocking || edge.Kind == model.EdgeKindAsync {
			continue
		}
		if _, ok := adj[edge.From]; !ok {
			adj[edge.From] = map[string]struct{}{}
		}
		adj[edge.From][edge.To] = struct{}{}
	}

	for endpointID, paths := range overrides {
		entryService, ok := entryByEndpoint[endpointID]
		if !ok {
			return fmt.Errorf("journeys endpoint not found in model: %s", endpointID)
		}
		if len(paths) == 0 {
			return fmt.Errorf("journeys[%s] must define at least one path", endpointID)
		}

		for pathIdx, path := range paths {
			if len(path) == 0 {
				return fmt.Errorf("journeys[%s][%d] path cannot be empty", endpointID, pathIdx)
			}
			if path[0] != entryService {
				return fmt.Errorf(
					"journeys[%s][%d] must start with endpoint entry service %q",
					endpointID,
					pathIdx,
					entryService,
				)
			}

			for nodeIdx, serviceID := range path {
				if strings.TrimSpace(serviceID) == "" {
					return fmt.Errorf("journeys[%s][%d][%d] service id cannot be empty", endpointID, pathIdx, nodeIdx)
				}
				if _, exists := serviceSet[serviceID]; !exists {
					return fmt.Errorf("journeys[%s][%d][%d] service not in model: %s", endpointID, pathIdx, nodeIdx, serviceID)
				}
				if nodeIdx == 0 {
					continue
				}

				from := path[nodeIdx-1]
				to := serviceID
				neighbors, ok := adj[from]
				if !ok {
					return fmt.Errorf("journeys[%s][%d] invalid hop %s -> %s: source has no blocking sync edges", endpointID, pathIdx, from, to)
				}
				if _, ok := neighbors[to]; !ok {
					return fmt.Errorf("journeys[%s][%d] invalid hop %s -> %s: edge not in blocking sync graph", endpointID, pathIdx, from, to)
				}
			}
		}
	}

	return nil
}

func (f File) validateBasic() error {
	if strings.TrimSpace(f.SchemaVersion) == "" {
		return fmt.Errorf("journeys.schema_version cannot be empty")
	}
	if f.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported journeys.schema_version: got %q want %q", f.SchemaVersion, SchemaVersion)
	}
	if len(f.Journeys) == 0 {
		return fmt.Errorf("journeys must contain at least one endpoint")
	}

	for endpointID, paths := range f.Journeys {
		if strings.TrimSpace(endpointID) == "" {
			return fmt.Errorf("journeys endpoint key cannot be empty")
		}
		if len(paths) == 0 {
			return fmt.Errorf("journeys[%s] must contain at least one path", endpointID)
		}
		for pathIdx, path := range paths {
			if len(path) == 0 {
				return fmt.Errorf("journeys[%s][%d] path cannot be empty", endpointID, pathIdx)
			}
		}
	}
	return nil
}

func (f File) cloneJourneys() map[string][][]string {
	keys := make([]string, 0, len(f.Journeys))
	for endpointID := range f.Journeys {
		keys = append(keys, endpointID)
	}
	slices.Sort(keys)

	out := make(map[string][][]string, len(keys))
	for _, endpointID := range keys {
		paths := f.Journeys[endpointID]
		clonedPaths := make([][]string, 0, len(paths))
		for _, path := range paths {
			clonedPaths = append(clonedPaths, slices.Clone(path))
		}
		out[endpointID] = clonedPaths
	}
	return out
}
