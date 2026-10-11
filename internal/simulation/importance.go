package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

type ImportanceParams struct {
	Spec        config.Importance
	BaseProfile ProfileParams
}

type ImportanceOutput struct {
	Name                string               `json:"name"`
	Method              string               `json:"method"`
	BaseProfile         string               `json:"base_profile"`
	SamplingMode        string               `json:"sampling_mode"`
	Trials              int                  `json:"trials"`
	Seed                int64                `json:"seed"`
	ConfidenceLevel     float64              `json:"confidence_level"`
	EvaluatedComponents int                  `json:"evaluated_components"`
	TopK                int                  `json:"top_k,omitempty"`
	Endpoints           []EndpointImportance `json:"endpoints"`
}

type EndpointImportance struct {
	EndpointID           string                `json:"endpoint_id"`
	BaselineAvailability float64               `json:"baseline_availability"`
	Components           []ComponentImportance `json:"components"`
}

type ComponentImportance struct {
	Kind                 string           `json:"kind"`
	ID                   string           `json:"id"`
	StateSemantics       string           `json:"state_semantics"`
	Rank                 int              `json:"rank"`
	LiveProbability      float64          `json:"live_probability"`
	AvailabilityIfUp     float64          `json:"availability_if_up"`
	AvailabilityIfDown   float64          `json:"availability_if_down"`
	BirnbaumImportance   float64          `json:"birnbaum_importance"`
	Confidence           BinomialInterval `json:"confidence"`
	ForcedDownImpact     float64          `json:"forced_down_impact"`
	PerfectComponentGain float64          `json:"perfect_component_gain"`
}

type importanceCount struct{ up, down, pivotal int }

func runImportance(ctx advancedContext, params AnalysisParams) ([]ImportanceOutput, error) {
	outputs := make([]ImportanceOutput, 0, len(params.Importance))
	for _, request := range params.Importance {
		output, err := runBirnbaum(ctx, params.Seed, request)
		if err != nil {
			return nil, fmt.Errorf("importance %q: %w", request.Spec.Name, err)
		}
		outputs = append(outputs, output)
	}
	return outputs, nil
}

func runBirnbaum(ctx advancedContext, seed int64, request ImportanceParams) (ImportanceOutput, error) {
	spec := request.Spec
	profile, err := normalizeProfile(request.BaseProfile, seed, 0)
	if err != nil {
		return ImportanceOutput{}, err
	}
	// Validate here as well as at the config boundary: the simulation API is used directly.
	validation := config.AnalysisConfig{
		SchemaVersion: config.AnalysisSchemaVersionV130,
		Profiles: []config.Profile{{Name: profile.Name, Trials: profile.Trials, SamplingMode: profile.SamplingMode,
			FailureProbability: profile.FailureProbability, FaultProfile: profile.FaultProfile}},
		Importance: []config.Importance{spec},
	}.Normalized()
	if err := validation.Validate(); err != nil {
		return ImportanceOutput{}, err
	}
	spec = validation.Importance[0]
	if err := validateAdvancedReliability(ctx, profile); err != nil {
		return ImportanceOutput{}, err
	}
	components, err := selectImportanceComponents(ctx, profile, spec.Components)
	if err != nil {
		return ImportanceOutput{}, err
	}
	if len(components) == 0 {
		return ImportanceOutput{}, fmt.Errorf("component selection is empty")
	}
	endpoints := make([]endpointPlan, 0, len(spec.Targets))
	for _, target := range spec.Targets {
		endpoint, exists := ctx.endpointByID[target.EndpointID]
		if !exists {
			return ImportanceOutput{}, fmt.Errorf("target endpoint %q is not present in the artifact", target.EndpointID)
		}
		endpoints = append(endpoints, endpoint)
	}
	slices.SortFunc(endpoints, func(a, b endpointPlan) int { return strings.Compare(a.ID, b.ID) })
	paths := map[string]journeyPath{}
	pathKeys := make([]map[string]string, len(endpoints))
	for i, endpoint := range endpoints {
		pathKeys[i] = map[string]string{}
		if endpoint.Mode != endpointModePredicate {
			for _, path := range endpoint.DiagnosticPaths {
				// Display IDs contain only services. Endpoint modes can select different
				// edges along the same service chain, so execution identity includes both.
				key := fmt.Sprintf("%q/%q", path.Services, path.EdgeIDs)
				pathKeys[i][path.ID] = key
				paths[key] = path
			}
		}
	}
	allUp := sampledState{serviceAlive: map[string]bool{}, edgeAlive: map[string]bool{}}
	for _, id := range ctx.serviceIDs {
		allUp.serviceAlive[id] = true
	}
	for _, id := range ctx.edgeIDs {
		allUp.edgeAlive[id] = true
	}
	faultState := buildProfileFaultState(ctx, nil)
	counts := make([][]importanceCount, len(endpoints))
	baseline := make([]int, len(endpoints))
	for i := range counts {
		counts[i] = make([]importanceCount, len(components))
	}
	importanceSeed := derivedSeed(seed, "importance:"+spec.Name+":"+profile.Name, 0)
	for trial := 0; trial < profile.Trials; trial++ {
		stateRNG := rand.New(rand.NewSource(derivedEventSeed(importanceSeed, trial, "state")))
		state, err := sampleAdvancedState(ctx, profile, stateRNG, faultState)
		if err != nil {
			return ImportanceOutput{}, err
		}
		// Fix all non-connectivity execution randomness before toggling components.
		// A hard-down service/edge makes its path fail regardless of retries. In an
		// all-up world, executePath samples the existing brownout/timeout semantics.
		viable := make(map[string]bool, len(paths))
		for id, path := range paths {
			pathRNG := rand.New(rand.NewSource(derivedEventSeed(importanceSeed, trial, "path:"+id)))
			viable[id] = executePath(ctx, allUp, faultState, path, pathRNG)
		}
		for i, endpoint := range endpoints {
			if importanceEndpointSuccess(endpoint, state, viable, pathKeys[i]) {
				baseline[i]++
			}
		}
		for componentIndex, component := range components {
			states := state.serviceAlive
			if component.Kind == config.ImportanceKindEdge {
				states = state.edgeAlive
			}
			original := states[component.ID]
			for endpointIndex, endpoint := range endpoints {
				states[component.ID] = true
				up := importanceEndpointSuccess(endpoint, state, viable, pathKeys[endpointIndex])
				states[component.ID] = false
				down := importanceEndpointSuccess(endpoint, state, viable, pathKeys[endpointIndex])
				if down && !up {
					return ImportanceOutput{}, fmt.Errorf("non-monotonic component %q for endpoint %q", component.ID, endpoint.ID)
				}
				count := &counts[endpointIndex][componentIndex]
				count.up += boolToInt(up)
				count.down += boolToInt(down)
				count.pivotal += boolToInt(up && !down)
			}
			states[component.ID] = original
		}
	}
	output := ImportanceOutput{
		Name: spec.Name, Method: spec.Method, BaseProfile: profile.Name, SamplingMode: profile.SamplingMode,
		Trials: profile.Trials, Seed: importanceSeed, ConfidenceLevel: spec.ConfidenceLevel,
		EvaluatedComponents: len(components), TopK: spec.TopK,
	}
	for endpointIndex, endpoint := range endpoints {
		result := EndpointImportance{EndpointID: endpoint.ID, BaselineAvailability: float64(baseline[endpointIndex]) / float64(profile.Trials)}
		for componentIndex, component := range components {
			count := counts[endpointIndex][componentIndex]
			component.AvailabilityIfUp = float64(count.up) / float64(profile.Trials)
			component.AvailabilityIfDown = float64(count.down) / float64(profile.Trials)
			component.Confidence = wilsonInterval(count.pivotal, profile.Trials, spec.ConfidenceLevel)
			component.BirnbaumImportance = component.Confidence.Estimate
			component.ForcedDownImpact = component.LiveProbability * component.BirnbaumImportance
			component.PerfectComponentGain = (1 - component.LiveProbability) * component.BirnbaumImportance
			result.Components = append(result.Components, component)
		}
		slices.SortFunc(result.Components, func(a, b ComponentImportance) int {
			if a.BirnbaumImportance != b.BirnbaumImportance {
				if a.BirnbaumImportance > b.BirnbaumImportance {
					return -1
				}
				return 1
			}
			if a.Kind != b.Kind {
				return strings.Compare(a.Kind, b.Kind)
			}
			return strings.Compare(a.ID, b.ID)
		})
		for i := range result.Components {
			result.Components[i].Rank = i + 1
			if i > 0 && result.Components[i].BirnbaumImportance == result.Components[i-1].BirnbaumImportance {
				result.Components[i].Rank = result.Components[i-1].Rank
			}
		}
		if spec.TopK > 0 && spec.TopK < len(result.Components) {
			result.Components = result.Components[:spec.TopK]
		}
		output.Endpoints = append(output.Endpoints, result)
	}
	return output, nil
}

func importanceEndpointSuccess(endpoint endpointPlan, state sampledState, viable map[string]bool, pathKeys map[string]string) bool {
	if endpoint.Mode == endpointModePredicate {
		return predicates.Evaluate(endpoint.Predicate, func(id string) bool { return state.serviceAlive[id] })
	}
	pathSuccess := func(path journeyPath) bool {
		if !viable[pathKeys[path.ID]] {
			return false
		}
		for _, id := range path.Services {
			if !state.serviceAlive[id] {
				return false
			}
		}
		for _, id := range path.EdgeIDs {
			if !state.edgeAlive[id] {
				return false
			}
		}
		return true
	}
	if endpoint.Mode == endpointModeEdgeAware {
		if len(endpoint.TargetPaths) == 0 {
			return false
		}
		for _, paths := range endpoint.TargetPaths {
			if !slices.ContainsFunc(paths, pathSuccess) {
				return false
			}
		}
		return true
	}
	return slices.ContainsFunc(endpoint.DiagnosticPaths, pathSuccess)
}

func selectImportanceComponents(ctx advancedContext, profile ProfileParams, selection config.ImportanceComponents) ([]ComponentImportance, error) {
	out := []ComponentImportance{}
	for _, kind := range selection.Kinds {
		ids, selected := ctx.serviceIDs, selection.ServiceIDs
		if kind == config.ImportanceKindEdge {
			ids, selected = ctx.edgeIDs, selection.EdgeIDs
		}
		for _, id := range selected {
			if !slices.Contains(ids, id) {
				return nil, fmt.Errorf("unknown %s component %q", kind, id)
			}
		}
		if len(selected) > 0 {
			ids = selected
		}
		for _, id := range ids {
			probability := edgeLiveProbability(profile, id)
			semantics := "edge_connectivity"
			if kind == config.ImportanceKindService {
				semantics = "service_liveness"
				service := ctx.services[id]
				probability = 1
				if service.SamplingEligible {
					probability = serviceLiveProbability(profile, id)
					if profile.SamplingMode == config.SamplingModeIndependentReplica {
						replicas := 0
						for _, bucket := range service.Buckets {
							replicas += bucket.Replicas
						}
						probability = 1 - math.Pow(1-probability, float64(replicas))
					}
				}
			}
			out = append(out, ComponentImportance{Kind: kind, ID: id, StateSemantics: semantics, LiveProbability: probability})
		}
	}
	return out, nil
}
