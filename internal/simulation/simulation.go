package simulation

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/faults"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

type Params struct {
	Trials             int
	Seed               int64
	FailureProbability float64
	JourneyOverrides   map[string][][]string
}

type ProfileParams struct {
	Name               string
	Trials             int
	Seed               int64
	SamplingMode       string
	FailureProbability float64
	FixedKFailures     int
	FaultProfile       string
	EndpointWeights    map[string]float64
}

type AnalysisParams struct {
	Seed             int64
	JourneyOverrides map[string][][]string
	PredicateSet     map[string]predicates.Definition
	DefaultWeights   map[string]float64
	FaultContract    *faults.Contract
	Profiles         []ProfileParams
}

type Output struct {
	EndpointAvailability map[string]float64
	OverallAvailability  float64
}

type ProfileOutput struct {
	Name                 string             `json:"name"`
	Trials               int                `json:"trials"`
	Seed                 int64              `json:"seed"`
	SamplingMode         string             `json:"sampling_mode"`
	FailureProbability   float64            `json:"failure_probability,omitempty"`
	FixedKFailures       int                `json:"fixed_k_failures,omitempty"`
	FaultProfile         string             `json:"fault_profile,omitempty"`
	EndpointAvailability map[string]float64 `json:"endpoint_availability"`
	EndpointWeights      map[string]float64 `json:"endpoint_weights,omitempty"`
	WeightedAggregate    float64            `json:"weighted_aggregate"`
	UnweightedAggregate  float64            `json:"unweighted_aggregate"`
	Assertions           []AssertionResult  `json:"assertions,omitempty"`
	Advanced             *AdvancedProfile   `json:"advanced,omitempty"`
}

type AnalysisOutput struct {
	Profiles               []ProfileOutput `json:"profiles"`
	CrossProfileWeighted   float64         `json:"cross_profile_weighted_aggregate"`
	CrossProfileUnweighted float64         `json:"cross_profile_unweighted_aggregate"`
}

type MetricFloat struct {
	Available  bool    `json:"available"`
	Value      float64 `json:"value,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	Provenance string  `json:"provenance,omitempty"`
}

type MetricInt struct {
	Available  bool   `json:"available"`
	Value      int    `json:"value,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Provenance string `json:"provenance,omitempty"`
}

type FaultMatch struct {
	FaultType                 string          `json:"fault_type"`
	Selector                  faults.Selector `json:"selector"`
	MatchedServiceIDs         []string        `json:"matched_service_ids,omitempty"`
	MatchedPlacementBucketIDs []string        `json:"matched_placement_bucket_ids,omitempty"`
	MatchedEdgeIDs            []string        `json:"matched_edge_ids,omitempty"`
	MatchedEndpointIDs        []string        `json:"matched_endpoint_ids,omitempty"`
	MatchedSharedResources    []string        `json:"matched_shared_resources,omitempty"`
}

type BlastRadius struct {
	ServiceCount  MetricInt `json:"service_count"`
	EndpointCount MetricInt `json:"endpoint_count"`
	ServiceIDs    []string  `json:"service_ids,omitempty"`
	EndpointIDs   []string  `json:"endpoint_ids,omitempty"`
}

type EndpointAdvanced struct {
	EndpointID             string      `json:"endpoint_id"`
	ExpectedSuccessRate    MetricFloat `json:"expected_success_rate"`
	MaxAmplificationFactor MetricFloat `json:"max_amplification_factor"`
}

type PathAdvanced struct {
	PathID                 string      `json:"path_id"`
	Services               []string    `json:"services"`
	EdgeIDs                []string    `json:"edge_ids"`
	ExpectedSuccessRate    MetricFloat `json:"expected_success_rate"`
	MaxAmplificationFactor MetricFloat `json:"max_amplification_factor"`
	TimeoutMismatchCount   MetricInt   `json:"timeout_mismatch_count"`
}

type EdgeAdvanced struct {
	EdgeID                 string      `json:"edge_id"`
	MaxAmplificationFactor MetricFloat `json:"max_amplification_factor"`
}

type AssertionResult struct {
	Metric      string                 `json:"metric"`
	Target      faults.AssertionTarget `json:"target"`
	Op          string                 `json:"op"`
	Expected    float64                `json:"expected"`
	Status      string                 `json:"status"`
	Available   bool                   `json:"available"`
	ActualValue float64                `json:"actual_value,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
}

type AdvancedProfile struct {
	ActiveFaultProfile string             `json:"active_fault_profile,omitempty"`
	FaultMatches       []FaultMatch       `json:"fault_matches,omitempty"`
	BlastRadius        *BlastRadius       `json:"blast_radius,omitempty"`
	Endpoints          []EndpointAdvanced `json:"endpoints,omitempty"`
	Paths              []PathAdvanced     `json:"paths,omitempty"`
	Edges              []EdgeAdvanced     `json:"edges,omitempty"`
}

func Run(mdl model.ResilienceModel, params Params) (Output, error) {
	analysisOut, err := RunProfiles(mdl, AnalysisParams{
		Seed:             params.Seed,
		JourneyOverrides: params.JourneyOverrides,
		Profiles: []ProfileParams{
			{
				Name:               "default",
				Trials:             params.Trials,
				Seed:               params.Seed,
				SamplingMode:       config.SamplingModeIndependentReplica,
				FailureProbability: params.FailureProbability,
			},
		},
	})
	if err != nil {
		return Output{}, err
	}
	profile := analysisOut.Profiles[0]
	return Output{
		EndpointAvailability: profile.EndpointAvailability,
		OverallAvailability:  profile.UnweightedAggregate,
	}, nil
}

func RunProfiles(mdl model.ResilienceModel, params AnalysisParams) (AnalysisOutput, error) {
	if err := mdl.Validate(); err != nil {
		return AnalysisOutput{}, fmt.Errorf("invalid model: %w", err)
	}
	if len(params.Profiles) == 0 {
		return AnalysisOutput{}, errors.New("at least one profile is required")
	}
	resolved, endpointIDs, serviceIDs, err := resolveEndpointPredicates(mdl, params.PredicateSet, params.JourneyOverrides)
	if err != nil {
		return AnalysisOutput{}, err
	}
	serviceReplicas := make(map[string]int, len(mdl.Services))
	for _, svc := range mdl.Services {
		replicas := svc.Replicas
		if replicas <= 0 {
			replicas = 1
		}
		serviceReplicas[svc.ID] = replicas
	}

	out := AnalysisOutput{
		Profiles: make([]ProfileOutput, 0, len(params.Profiles)),
	}
	for idx, profile := range params.Profiles {
		normalized, err := normalizeProfile(profile, params.Seed, idx)
		if err != nil {
			return AnalysisOutput{}, err
		}
		profileOutput, err := runProfile(normalized, endpointIDs, serviceIDs, serviceReplicas, resolved, params.DefaultWeights)
		if err != nil {
			return AnalysisOutput{}, fmt.Errorf("profile %q: %w", normalized.Name, err)
		}
		out.Profiles = append(out.Profiles, profileOutput)
		out.CrossProfileWeighted += profileOutput.WeightedAggregate
		out.CrossProfileUnweighted += profileOutput.UnweightedAggregate
	}
	if len(out.Profiles) > 0 {
		out.CrossProfileWeighted /= float64(len(out.Profiles))
		out.CrossProfileUnweighted /= float64(len(out.Profiles))
	}
	return out, nil
}

func runProfile(profile ProfileParams, endpointIDs []string, serviceIDs []string, serviceReplicas map[string]int, resolved map[string]predicates.Definition, defaultWeights map[string]float64) (ProfileOutput, error) {
	rng := rand.New(rand.NewSource(profile.Seed))
	successCount := make(map[string]int, len(endpointIDs))
	for trial := 0; trial < profile.Trials; trial++ {
		alive, err := sampleAlive(profile, rng, serviceIDs, serviceReplicas)
		if err != nil {
			return ProfileOutput{}, err
		}
		for _, endpointID := range endpointIDs {
			if predicates.Evaluate(resolved[endpointID], func(serviceID string) bool { return alive[serviceID] }) {
				successCount[endpointID]++
			}
		}
	}

	availability := make(map[string]float64, len(endpointIDs))
	unweighted := 0.0
	for _, endpointID := range endpointIDs {
		avail := float64(successCount[endpointID]) / float64(profile.Trials)
		availability[endpointID] = avail
		unweighted += avail
	}
	if len(endpointIDs) > 0 {
		unweighted /= float64(len(endpointIDs))
	}

	weights := mergeWeights(defaultWeights, profile.EndpointWeights, endpointIDs)
	weighted := aggregateWeightedAvailability(availability, weights, endpointIDs)

	return ProfileOutput{
		Name:                 profile.Name,
		Trials:               profile.Trials,
		Seed:                 profile.Seed,
		SamplingMode:         profile.SamplingMode,
		FailureProbability:   profile.FailureProbability,
		FixedKFailures:       profile.FixedKFailures,
		FaultProfile:         profile.FaultProfile,
		EndpointAvailability: availability,
		EndpointWeights:      weights,
		WeightedAggregate:    weighted,
		UnweightedAggregate:  unweighted,
	}, nil
}

func sampleAlive(profile ProfileParams, rng *rand.Rand, serviceIDs []string, serviceReplicas map[string]int) (map[string]bool, error) {
	alive := make(map[string]bool, len(serviceIDs))
	switch profile.SamplingMode {
	case config.SamplingModeIndependentReplica:
		for _, serviceID := range serviceIDs {
			live := false
			for i := 0; i < serviceReplicas[serviceID]; i++ {
				if rng.Float64() > profile.FailureProbability {
					live = true
					break
				}
			}
			alive[serviceID] = live
		}
	case config.SamplingModeIndependentService:
		for _, serviceID := range serviceIDs {
			alive[serviceID] = rng.Float64() > profile.FailureProbability
		}
	case config.SamplingModeFixedKServiceSet:
		if profile.FixedKFailures > len(serviceIDs) {
			return nil, fmt.Errorf("fixed_k_failures %d exceeds service count %d", profile.FixedKFailures, len(serviceIDs))
		}
		for _, serviceID := range serviceIDs {
			alive[serviceID] = true
		}
		if profile.FixedKFailures == 0 {
			return alive, nil
		}
		indices := rng.Perm(len(serviceIDs))
		for _, idx := range indices[:profile.FixedKFailures] {
			alive[serviceIDs[idx]] = false
		}
	default:
		return nil, fmt.Errorf("unsupported sampling mode %q", profile.SamplingMode)
	}
	return alive, nil
}

func resolveEndpointPredicates(mdl model.ResilienceModel, predicateSet map[string]predicates.Definition, journeyOverrides map[string][][]string) (map[string]predicates.Definition, []string, []string, error) {
	serviceSet := make(map[string]struct{}, len(mdl.Services))
	serviceIDs := make([]string, 0, len(mdl.Services))
	for _, svc := range mdl.Services {
		serviceSet[svc.ID] = struct{}{}
		serviceIDs = append(serviceIDs, svc.ID)
	}
	slices.Sort(serviceIDs)

	adj := make(map[string][]string)
	for _, edge := range mdl.Edges {
		if !edge.Blocking || edge.Kind == model.EdgeKindAsync {
			continue
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
	}
	for _, serviceID := range serviceIDs {
		slices.Sort(adj[serviceID])
	}

	endpointIDs := make([]string, 0, len(mdl.Endpoints))
	resolved := make(map[string]predicates.Definition, len(mdl.Endpoints))
	endpointSet := make(map[string]struct{}, len(mdl.Endpoints))
	for _, ep := range mdl.Endpoints {
		endpointSet[ep.ID] = struct{}{}
	}
	for endpointID := range journeyOverrides {
		if _, ok := endpointSet[endpointID]; !ok {
			return nil, nil, nil, fmt.Errorf("journey override endpoint not found in model: %s", endpointID)
		}
	}

	mergedPredicates := make(map[string]predicates.Definition, len(mdl.Predicates)+len(predicateSet))
	for key, def := range mdl.Predicates {
		mergedPredicates[key] = def
	}
	for key, def := range predicateSet {
		mergedPredicates[key] = def
	}

	for _, ep := range mdl.Endpoints {
		var def predicates.Definition
		switch {
		case ep.SuccessPredicate != nil:
			def = *ep.SuccessPredicate
		case hasPredicate(mergedPredicates, ep.SuccessPredicateRef):
			def = mergedPredicates[ep.SuccessPredicateRef]
		default:
			paths := journeyOverrides[ep.ID]
			if len(paths) == 0 {
				paths = discoverJourneys(ep.EntryService, adj)
			} else if err := validateJourneyPaths(paths); err != nil {
				return nil, nil, nil, fmt.Errorf("invalid journey override for endpoint %s: %w", ep.ID, err)
			}
			def = journeysToPredicate(paths)
		}
		if err := validatePredicateServices(def, serviceSet); err != nil {
			return nil, nil, nil, fmt.Errorf("endpoint %s: %w", ep.ID, err)
		}
		resolved[ep.ID] = def
		endpointIDs = append(endpointIDs, ep.ID)
	}
	slices.Sort(endpointIDs)
	return resolved, endpointIDs, serviceIDs, nil
}

func discoverJourneys(entry string, adjacency map[string][]string) [][]string {
	visited := make(map[string]bool)
	path := make([]string, 0, 8)
	paths := make([][]string, 0, 8)

	var dfs func(string)
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
	return cloneAndNormalizeJourneys(paths)
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

func journeysToPredicate(paths [][]string) predicates.Definition {
	children := make([]predicates.Definition, 0, len(paths))
	for _, path := range cloneAndNormalizeJourneys(paths) {
		children = append(children, predicates.Definition{
			Type:     predicates.TypeAllOf,
			Services: slices.Clone(path),
		})
	}
	return predicates.Definition{
		Type:     predicates.TypeAnyOf,
		Children: children,
	}
}

func normalizeProfile(profile ProfileParams, seed int64, index int) (ProfileParams, error) {
	out := profile
	if strings.TrimSpace(out.Name) == "" {
		out.Name = fmt.Sprintf("profile-%d", index+1)
	}
	if out.Trials <= 0 {
		return ProfileParams{}, errors.New("trials must be > 0")
	}
	if out.SamplingMode == "" {
		out.SamplingMode = config.SamplingModeIndependentReplica
	}
	if out.Seed == 0 {
		out.Seed = derivedSeed(seed, out.Name, index)
	}
	switch out.SamplingMode {
	case config.SamplingModeIndependentReplica, config.SamplingModeIndependentService:
		if out.FailureProbability < 0 || out.FailureProbability > 1 {
			return ProfileParams{}, errors.New("failure_probability must be in range [0,1]")
		}
	case config.SamplingModeFixedKServiceSet:
		if out.FixedKFailures < 0 {
			return ProfileParams{}, errors.New("fixed_k_failures must be >= 0")
		}
	default:
		return ProfileParams{}, fmt.Errorf("unsupported sampling mode %q", out.SamplingMode)
	}
	if out.EndpointWeights == nil {
		out.EndpointWeights = map[string]float64{}
	}
	return out, nil
}

func validatePredicateServices(def predicates.Definition, serviceSet map[string]struct{}) error {
	for _, service := range def.Services {
		if _, ok := serviceSet[service]; !ok {
			return fmt.Errorf("predicate references unknown service %q", service)
		}
	}
	for idx, child := range def.Children {
		if err := validatePredicateServices(child, serviceSet); err != nil {
			return fmt.Errorf("child %d: %w", idx, err)
		}
	}
	return nil
}

func hasPredicate(set map[string]predicates.Definition, name string) bool {
	_, ok := set[name]
	return ok
}

func mergeWeights(defaults map[string]float64, overrides map[string]float64, endpointIDs []string) map[string]float64 {
	weights := make(map[string]float64, len(endpointIDs))
	for _, endpointID := range endpointIDs {
		if weight, ok := defaults[endpointID]; ok {
			weights[endpointID] = weight
		}
		if weight, ok := overrides[endpointID]; ok {
			weights[endpointID] = weight
		}
	}
	return weights
}

func aggregateWeightedAvailability(availability map[string]float64, weights map[string]float64, endpointIDs []string) float64 {
	totalWeight := 0.0
	sum := 0.0
	for _, endpointID := range endpointIDs {
		if weight, ok := weights[endpointID]; ok && weight > 0 {
			sum += availability[endpointID] * weight
			totalWeight += weight
		}
	}
	if totalWeight == 0 {
		if len(endpointIDs) == 0 {
			return 0
		}
		sum = 0
		for _, endpointID := range endpointIDs {
			sum += availability[endpointID]
		}
		return sum / float64(len(endpointIDs))
	}
	return sum / totalWeight
}

func derivedSeed(base int64, profileName string, index int) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s:%d", base, profileName, index)))
	return int64(h.Sum64())
}
