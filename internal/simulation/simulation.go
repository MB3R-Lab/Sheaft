package simulation

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"

	"github.com/MB3R-Lab/Sheaft/internal/model"
)

type Params struct {
	Trials             int
	Seed               int64
	FailureProbability float64
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

	requiredByEndpoint := make(map[string][]string, len(mdl.Endpoints))
	endpointIDs := make([]string, 0, len(mdl.Endpoints))
	for _, ep := range mdl.Endpoints {
		req := requiredServices(ep.EntryService, adj)
		requiredByEndpoint[ep.ID] = req
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
			required := requiredByEndpoint[endpointID]
			ok := true
			for _, serviceID := range required {
				if !alive[serviceID] {
					ok = false
					break
				}
			}
			if ok {
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

func requiredServices(entry string, adjacency map[string][]string) []string {
	visited := map[string]struct{}{}
	stack := []string{entry}

	for len(stack) > 0 {
		idx := len(stack) - 1
		current := stack[idx]
		stack = stack[:idx]
		if _, ok := visited[current]; ok {
			continue
		}
		visited[current] = struct{}{}
		for _, next := range adjacency[current] {
			if _, ok := visited[next]; !ok {
				stack = append(stack, next)
			}
		}
	}

	out := make([]string, 0, len(visited))
	for serviceID := range visited {
		out = append(out, serviceID)
	}
	slices.Sort(out)
	return out
}
