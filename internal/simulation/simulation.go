package simulation

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/model"
)

type Params struct {
	Trials             int
	Seed               int64
	FailureProbability float64
	JourneyOverrides   map[string][][]string
}

type Output struct {
	EndpointAvailability map[string]float64
	OverallAvailability  float64
}

func Run(mdl model.ResilienceModel, params Params) (Output, error) {
	if err := mdl.Validate(); err != nil {
		return Output{}, fmt.Errorf("invalid model: %w", err)
	}
	if params.Trials <= 0 {
		return Output{}, errors.New("trials must be > 0")
	}
	if params.FailureProbability < 0 || params.FailureProbability > 1 {
		return Output{}, errors.New("failure_probability must be in range [0,1]")
	}

	serviceReplicas := make(map[string]int, len(mdl.Services))
	for _, svc := range mdl.Services {
		replicas := svc.Replicas
		if replicas <= 0 {
			replicas = 1
		}
		serviceReplicas[svc.ID] = replicas
	}

	adj := make(map[string][]string)
	for _, edge := range mdl.Edges {
		// Non-blocking or async edges are intentionally not part of immediate HTTP success path.
		if !edge.Blocking || edge.Kind == model.EdgeKindAsync {
			continue
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
	}

	for serviceID := range serviceReplicas {
		if _, ok := adj[serviceID]; !ok {
			adj[serviceID] = []string{}
		}
		slices.Sort(adj[serviceID])
	}

	journeysByEndpoint := make(map[string][][]string, len(mdl.Endpoints))
	endpointIDs := make([]string, 0, len(mdl.Endpoints))
	endpointSet := make(map[string]struct{}, len(mdl.Endpoints))
	for _, ep := range mdl.Endpoints {
		endpointSet[ep.ID] = struct{}{}
	}
	for endpointID := range params.JourneyOverrides {
		if _, ok := endpointSet[endpointID]; !ok {
			return Output{}, fmt.Errorf("journey override endpoint not found in model: %s", endpointID)
		}
	}
	for _, ep := range mdl.Endpoints {
		if override, ok := params.JourneyOverrides[ep.ID]; ok {
			if err := validateJourneyPaths(override); err != nil {
				return Output{}, fmt.Errorf("invalid journey override for endpoint %s: %w", ep.ID, err)
			}
			journeysByEndpoint[ep.ID] = cloneAndNormalizeJourneys(override)
		} else {
			journeysByEndpoint[ep.ID] = discoverJourneys(ep.EntryService, adj)
		}
		endpointIDs = append(endpointIDs, ep.ID)
	}
	slices.Sort(endpointIDs)

	rng := rand.New(rand.NewSource(params.Seed))
	successCount := make(map[string]int, len(endpointIDs))

	for trial := 0; trial < params.Trials; trial++ {
		alive := make(map[string]bool, len(serviceReplicas))
		for serviceID, replicas := range serviceReplicas {
			live := false
			for i := 0; i < replicas; i++ {
				if rng.Float64() > params.FailureProbability {
					live = true
					break
				}
			}
			alive[serviceID] = live
		}

		for _, endpointID := range endpointIDs {
			journeys := journeysByEndpoint[endpointID]
			endpointOK := false
			for _, journey := range journeys {
				journeyOK := true
				for _, serviceID := range journey {
					if !alive[serviceID] {
						journeyOK = false
						break
					}
				}
				if journeyOK {
					endpointOK = true
					break
				}
			}
			if endpointOK {
				successCount[endpointID]++
			}
		}
	}

	availability := make(map[string]float64, len(endpointIDs))
	overall := 0.0
	for _, endpointID := range endpointIDs {
		avail := float64(successCount[endpointID]) / float64(params.Trials)
		availability[endpointID] = avail
		overall += avail
	}
	if len(endpointIDs) > 0 {
		overall /= float64(len(endpointIDs))
	}

	return Output{
		EndpointAvailability: availability,
		OverallAvailability:  overall,
	}, nil
}

func discoverJourneys(entry string, adjacency map[string][]string) [][]string {
	visited := make(map[string]bool)
	path := make([]string, 0, 8)
	paths := make([][]string, 0, 8)

	var dfs func(current string)
	dfs = func(current string) {
		visited[current] = true
		path = append(path, current)

		nexts := make([]string, 0, len(adjacency[current]))
		for _, next := range adjacency[current] {
			if !visited[next] {
				nexts = append(nexts, next)
			}
		}
		slices.Sort(nexts)

		if len(nexts) == 0 {
			paths = append(paths, slices.Clone(path))
		} else {
			for _, next := range nexts {
				dfs(next)
			}
		}

		path = path[:len(path)-1]
		visited[current] = false
	}

	dfs(entry)

	uniq := make(map[string][]string, len(paths))
	keys := make([]string, 0, len(paths))
	for _, p := range paths {
		key := strings.Join(p, "->")
		if _, ok := uniq[key]; ok {
			continue
		}
		uniq[key] = p
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make([][]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, uniq[key])
	}
	return out
}

func cloneAndNormalizeJourneys(paths [][]string) [][]string {
	uniq := make(map[string][]string, len(paths))
	keys := make([]string, 0, len(paths))
	for _, path := range paths {
		key := strings.Join(path, "->")
		if _, ok := uniq[key]; ok {
			continue
		}
		uniq[key] = slices.Clone(path)
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make([][]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, uniq[key])
	}
	return out
}

func validateJourneyPaths(paths [][]string) error {
	if len(paths) == 0 {
		return errors.New("no paths defined")
	}
	for pathIdx, path := range paths {
		if len(path) == 0 {
			return fmt.Errorf("path %d is empty", pathIdx)
		}
		for nodeIdx, serviceID := range path {
			if strings.TrimSpace(serviceID) == "" {
				return fmt.Errorf("path %d has empty service id at index %d", pathIdx, nodeIdx)
			}
		}
	}
	return nil
}
