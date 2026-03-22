package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/faults"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
)

const (
	endpointModePredicate = "predicate"
	endpointModeJourney   = "journey"

	advancedProvenanceArtifact            = "artifact"
	advancedProvenanceExternalContract    = "external_contract"
	advancedProvenanceUnavailable         = "unavailable"
	advancedProvenanceArtifactAndExternal = "artifact+external_contract"
)

type advancedContext struct {
	contractVersion string
	services        map[string]normalizedService
	serviceIDs      []string
	edges           map[string]normalizedEdge
	edgeIDs         []string
	endpoints       []endpointPlan
	endpointByID    map[string]endpointPlan
}

type normalizedService struct {
	ID                 string
	Name               string
	Labels             map[string]string
	SharedResourceRefs []string
	FailureEligible    bool
	Buckets            []replicaBucket
}

type replicaBucket struct {
	ID        string
	ServiceID string
	Replicas  int
	Labels    map[string]string
}

type normalizedEdge struct {
	ID         string
	From       string
	To         string
	Kind       model.EdgeKind
	Blocking   bool
	Labels     map[string]string
	Resilience *model.ResiliencePolicy
	Observed   *model.ObservedEdge
}

type endpointPlan struct {
	ID              string
	EntryService    string
	Method          string
	Path            string
	Mode            string
	Predicate       predicates.Definition
	Paths           []journeyPath
	DiagnosticPaths []journeyPath
}

type journeyPath struct {
	ID       string
	Services []string
	EdgeIDs  []string
}

type profileFaultState struct {
	hardBucketKill    map[string]struct{}
	hardEdgeKill      map[string]struct{}
	serviceErrorRates map[string]float64
	serviceLatencies  map[string]model.LatencySummary
	edgeErrorRates    map[string]float64
	edgeLatencies     map[string]model.LatencySummary
	faultMatches      []FaultMatch
	blastRadius       *BlastRadius
}

type sampledState struct {
	bucketAlive  map[string]bool
	serviceAlive map[string]bool
	edgeAlive    map[string]bool
}

type pathStatic struct {
	path                 journeyPath
	expectedSuccess      MetricFloat
	amplification        MetricFloat
	timeoutMismatchCount MetricInt
	edgeAmplification    map[string]MetricFloat
	edgeTimeoutMismatch  map[string]MetricInt
}

func RunArtifactProfiles(loaded artifact.Loaded, params AnalysisParams) (AnalysisOutput, error) {
	if loaded.Metadata.Contract.Version == "1.0.0" && params.FaultContract == nil {
		return RunProfiles(loaded.Model, params)
	}

	ctx, err := buildAdvancedContext(loaded, params.PredicateSet, params.JourneyOverrides)
	if err != nil {
		return AnalysisOutput{}, err
	}

	out := AnalysisOutput{
		Profiles: make([]ProfileOutput, 0, len(params.Profiles)),
	}
	for idx, profile := range params.Profiles {
		normalized, err := normalizeProfile(profile, params.Seed, idx)
		if err != nil {
			return AnalysisOutput{}, err
		}
		faultSpec, err := resolveFaultProfile(params.FaultContract, normalized.FaultProfile)
		if err != nil {
			return AnalysisOutput{}, fmt.Errorf("profile %q: %w", normalized.Name, err)
		}
		profileOut, err := runAdvancedProfile(ctx, normalized, params.DefaultWeights, faultSpec)
		if err != nil {
			return AnalysisOutput{}, fmt.Errorf("profile %q: %w", normalized.Name, err)
		}
		out.Profiles = append(out.Profiles, profileOut)
		out.CrossProfileWeighted += profileOut.WeightedAggregate
		out.CrossProfileUnweighted += profileOut.UnweightedAggregate
	}
	if len(out.Profiles) > 0 {
		out.CrossProfileWeighted /= float64(len(out.Profiles))
		out.CrossProfileUnweighted /= float64(len(out.Profiles))
	}
	return out, nil
}

func runAdvancedProfile(ctx advancedContext, profile ProfileParams, defaultWeights map[string]float64, faultSpec *faults.Profile) (ProfileOutput, error) {
	rng := rand.New(rand.NewSource(profile.Seed))
	faultState := buildProfileFaultState(ctx, faultSpec)
	staticByPath := make(map[string]pathStatic)
	for _, endpoint := range ctx.endpoints {
		for _, path := range endpoint.DiagnosticPaths {
			if _, ok := staticByPath[path.ID]; ok {
				continue
			}
			staticByPath[path.ID] = analyzePathStatic(ctx, path, faultState)
		}
	}

	endpointIDs := make([]string, 0, len(ctx.endpoints))
	endpointSuccess := make(map[string]int, len(ctx.endpoints))
	pathSuccess := make(map[string]int, len(staticByPath))
	for _, endpoint := range ctx.endpoints {
		endpointIDs = append(endpointIDs, endpoint.ID)
	}
	slices.Sort(endpointIDs)

	for trial := 0; trial < profile.Trials; trial++ {
		state, err := sampleAdvancedState(ctx, profile, rng, faultState)
		if err != nil {
			return ProfileOutput{}, err
		}
		for _, endpoint := range ctx.endpoints {
			diagnosticResults := make([]bool, 0, len(endpoint.DiagnosticPaths))
			for _, path := range endpoint.DiagnosticPaths {
				ok := executePath(ctx, state, faultState, path, rng)
				if ok {
					pathSuccess[path.ID]++
				}
				diagnosticResults = append(diagnosticResults, ok)
			}

			switch endpoint.Mode {
			case endpointModePredicate:
				if predicates.Evaluate(endpoint.Predicate, func(serviceID string) bool { return state.serviceAlive[serviceID] }) {
					endpointSuccess[endpoint.ID]++
				}
			case endpointModeJourney:
				success := false
				for _, ok := range diagnosticResults {
					if ok {
						success = true
						break
					}
				}
				if success {
					endpointSuccess[endpoint.ID]++
				}
			default:
				return ProfileOutput{}, fmt.Errorf("unsupported endpoint mode %q", endpoint.Mode)
			}
		}
	}

	availability := make(map[string]float64, len(endpointIDs))
	unweighted := 0.0
	for _, endpointID := range endpointIDs {
		value := float64(endpointSuccess[endpointID]) / float64(profile.Trials)
		availability[endpointID] = value
		unweighted += value
	}
	if len(endpointIDs) > 0 {
		unweighted /= float64(len(endpointIDs))
	}
	weights := mergeWeights(defaultWeights, profile.EndpointWeights, endpointIDs)
	weighted := aggregateWeightedAvailability(availability, weights, endpointIDs)
	advanced := buildAdvancedProfile(ctx, profile, availability, pathSuccess, staticByPath, faultState, profile.Trials)
	assertions := evaluateAssertions(weighted, advanced, faultSpec)

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
		Assertions:           assertions,
		Advanced:             advanced,
	}, nil
}

func buildAdvancedProfile(ctx advancedContext, profile ProfileParams, availability map[string]float64, pathSuccess map[string]int, staticByPath map[string]pathStatic, faultState profileFaultState, trials int) *AdvancedProfile {
	advanced := &AdvancedProfile{
		ActiveFaultProfile: profile.FaultProfile,
		FaultMatches:       slices.Clone(faultState.faultMatches),
		BlastRadius:        faultState.blastRadius,
		Endpoints:          make([]EndpointAdvanced, 0, len(ctx.endpoints)),
		Paths:              make([]PathAdvanced, 0, len(staticByPath)),
	}

	edgeMetrics := map[string]MetricFloat{}
	for _, endpoint := range ctx.endpoints {
		var endpointAmp MetricFloat
		endpointAmpSet := false
		endpointAmpUnavailable := false
		for _, path := range endpoint.DiagnosticPaths {
			static := staticByPath[path.ID]
			expectedSuccess := MetricFloat{
				Available:  true,
				Value:      float64(pathSuccess[path.ID]) / float64(trials),
				Provenance: static.expectedSuccess.Provenance,
			}
			advanced.Paths = append(advanced.Paths, PathAdvanced{
				PathID:                 path.ID,
				Services:               slices.Clone(path.Services),
				EdgeIDs:                slices.Clone(path.EdgeIDs),
				ExpectedSuccessRate:    expectedSuccess,
				MaxAmplificationFactor: static.amplification,
				TimeoutMismatchCount:   static.timeoutMismatchCount,
			})
			if !static.amplification.Available {
				endpointAmpUnavailable = true
			} else if !endpointAmpSet || static.amplification.Value > endpointAmp.Value {
				endpointAmp = static.amplification
				endpointAmpSet = true
			}
			for edgeID, metric := range static.edgeAmplification {
				current, ok := edgeMetrics[edgeID]
				if !ok || (metric.Available && (!current.Available || metric.Value > current.Value)) {
					edgeMetrics[edgeID] = metric
				} else if !metric.Available && !current.Available {
					edgeMetrics[edgeID] = metric
				}
			}
		}
		if endpointAmpUnavailable && !endpointAmpSet {
			endpointAmp = MetricFloat{
				Available:  false,
				Reason:     "retry metadata unavailable on at least one path",
				Provenance: advancedProvenanceUnavailable,
			}
		}
		advanced.Endpoints = append(advanced.Endpoints, EndpointAdvanced{
			EndpointID:             endpoint.ID,
			ExpectedSuccessRate:    MetricFloat{Available: true, Value: availability[endpoint.ID], Provenance: advancedProvenanceArtifact},
			MaxAmplificationFactor: endpointAmp,
		})
	}
	slices.SortFunc(advanced.Paths, func(a, b PathAdvanced) int {
		return strings.Compare(a.PathID, b.PathID)
	})
	slices.SortFunc(advanced.Endpoints, func(a, b EndpointAdvanced) int {
		return strings.Compare(a.EndpointID, b.EndpointID)
	})
	if len(edgeMetrics) > 0 {
		advanced.Edges = make([]EdgeAdvanced, 0, len(edgeMetrics))
		edgeIDs := make([]string, 0, len(edgeMetrics))
		for edgeID := range edgeMetrics {
			edgeIDs = append(edgeIDs, edgeID)
		}
		slices.Sort(edgeIDs)
		for _, edgeID := range edgeIDs {
			advanced.Edges = append(advanced.Edges, EdgeAdvanced{
				EdgeID:                 edgeID,
				MaxAmplificationFactor: edgeMetrics[edgeID],
			})
		}
	}
	return advanced
}

func evaluateAssertions(weightedAggregate float64, advanced *AdvancedProfile, faultSpec *faults.Profile) []AssertionResult {
	if faultSpec == nil || len(faultSpec.Assertions) == 0 {
		return nil
	}
	results := make([]AssertionResult, 0, len(faultSpec.Assertions))
	for _, assertion := range faultSpec.Assertions {
		actual, available, reason := resolveAssertionMetric(assertion, weightedAggregate, advanced)
		status := "pass"
		if !available {
			status = "unavailable"
		} else if !compareAssertion(actual, assertion.Op, assertion.Value) {
			status = "fail"
		}
		result := AssertionResult{
			Metric:    assertion.Metric,
			Target:    assertion.Target,
			Op:        assertion.Op,
			Expected:  assertion.Value,
			Status:    status,
			Available: available,
			Reason:    reason,
		}
		if available {
			result.ActualValue = actual
		}
		results = append(results, result)
	}
	return results
}

func resolveAssertionMetric(assertion faults.Assertion, weightedAggregate float64, advanced *AdvancedProfile) (float64, bool, string) {
	if advanced == nil {
		return 0, false, "advanced profile data unavailable"
	}
	switch assertion.Metric {
	case faults.MetricExpectedSuccessRate:
		switch assertion.Target.Type {
		case faults.TargetEndpoint:
			for _, endpoint := range advanced.Endpoints {
				if endpoint.EndpointID == assertion.Target.EndpointID {
					return endpoint.ExpectedSuccessRate.Value, endpoint.ExpectedSuccessRate.Available, endpoint.ExpectedSuccessRate.Reason
				}
			}
			return 0, false, "endpoint target not found"
		case faults.TargetPath:
			path := findPathMetric(advanced.Paths, assertion.Target.Services)
			if path == nil {
				return 0, false, "path target not found"
			}
			return path.ExpectedSuccessRate.Value, path.ExpectedSuccessRate.Available, path.ExpectedSuccessRate.Reason
		case faults.TargetProfile:
			return weightedAggregate, true, ""
		default:
			return 0, false, fmt.Sprintf("metric %s does not support target %s", assertion.Metric, assertion.Target.Type)
		}
	case faults.MetricMaxAmplificationFactor:
		switch assertion.Target.Type {
		case faults.TargetEndpoint:
			for _, endpoint := range advanced.Endpoints {
				if endpoint.EndpointID == assertion.Target.EndpointID {
					return endpoint.MaxAmplificationFactor.Value, endpoint.MaxAmplificationFactor.Available, endpoint.MaxAmplificationFactor.Reason
				}
			}
			return 0, false, "endpoint target not found"
		case faults.TargetPath:
			path := findPathMetric(advanced.Paths, assertion.Target.Services)
			if path == nil {
				return 0, false, "path target not found"
			}
			return path.MaxAmplificationFactor.Value, path.MaxAmplificationFactor.Available, path.MaxAmplificationFactor.Reason
		case faults.TargetEdge:
			for _, edge := range advanced.Edges {
				if edge.EdgeID == assertion.Target.EdgeID {
					return edge.MaxAmplificationFactor.Value, edge.MaxAmplificationFactor.Available, edge.MaxAmplificationFactor.Reason
				}
			}
			return 0, false, "edge target not found"
		case faults.TargetProfile:
			maxValue := 0.0
			found := false
			for _, path := range advanced.Paths {
				if !path.MaxAmplificationFactor.Available {
					return 0, false, path.MaxAmplificationFactor.Reason
				}
				if !found || path.MaxAmplificationFactor.Value > maxValue {
					maxValue = path.MaxAmplificationFactor.Value
					found = true
				}
			}
			if !found {
				return 0, false, "no path amplification metrics available"
			}
			return maxValue, true, ""
		default:
			return 0, false, fmt.Sprintf("metric %s does not support target %s", assertion.Metric, assertion.Target.Type)
		}
	case faults.MetricTimeoutMismatchCount:
		switch assertion.Target.Type {
		case faults.TargetPath:
			path := findPathMetric(advanced.Paths, assertion.Target.Services)
			if path == nil {
				return 0, false, "path target not found"
			}
			return float64(path.TimeoutMismatchCount.Value), path.TimeoutMismatchCount.Available, path.TimeoutMismatchCount.Reason
		case faults.TargetProfile:
			total := 0
			for _, path := range advanced.Paths {
				if !path.TimeoutMismatchCount.Available {
					return 0, false, path.TimeoutMismatchCount.Reason
				}
				total += path.TimeoutMismatchCount.Value
			}
			return float64(total), true, ""
		default:
			return 0, false, fmt.Sprintf("metric %s does not support target %s", assertion.Metric, assertion.Target.Type)
		}
	case faults.MetricBlastRadiusServiceCount:
		if assertion.Target.Type != faults.TargetProfile || advanced.BlastRadius == nil {
			return 0, false, "blast radius service count is available only for profile targets"
		}
		return float64(advanced.BlastRadius.ServiceCount.Value), advanced.BlastRadius.ServiceCount.Available, advanced.BlastRadius.ServiceCount.Reason
	case faults.MetricBlastRadiusEndpointCount:
		if assertion.Target.Type != faults.TargetProfile || advanced.BlastRadius == nil {
			return 0, false, "blast radius endpoint count is available only for profile targets"
		}
		return float64(advanced.BlastRadius.EndpointCount.Value), advanced.BlastRadius.EndpointCount.Available, advanced.BlastRadius.EndpointCount.Reason
	default:
		return 0, false, "unsupported assertion metric"
	}
}

func compareAssertion(actual float64, op string, expected float64) bool {
	switch op {
	case ">=":
		return actual >= expected
	case "<=":
		return actual <= expected
	case "==":
		return actual == expected
	default:
		return false
	}
}

func findPathMetric(paths []PathAdvanced, services []string) *PathAdvanced {
	target := strings.Join(services, "->")
	for idx := range paths {
		if paths[idx].PathID == target {
			return &paths[idx]
		}
	}
	return nil
}

func sampleAdvancedState(ctx advancedContext, profile ProfileParams, rng *rand.Rand, faultState profileFaultState) (sampledState, error) {
	state := sampledState{
		bucketAlive:  make(map[string]bool),
		serviceAlive: make(map[string]bool, len(ctx.serviceIDs)),
		edgeAlive:    make(map[string]bool, len(ctx.edgeIDs)),
	}

	switch profile.SamplingMode {
	case config.SamplingModeIndependentReplica:
		for _, serviceID := range ctx.serviceIDs {
			svc := ctx.services[serviceID]
			for _, bucket := range svc.Buckets {
				live := false
				effective := bucket.Replicas
				if len(svc.Buckets) == 1 && effective <= 0 {
					effective = 1
				}
				for replica := 0; replica < effective; replica++ {
					if rng.Float64() > profile.FailureProbability {
						live = true
						break
					}
				}
				state.bucketAlive[bucket.ID] = live
			}
		}
	case config.SamplingModeIndependentService:
		for _, serviceID := range ctx.serviceIDs {
			alive := rng.Float64() > profile.FailureProbability
			for _, bucket := range ctx.services[serviceID].Buckets {
				state.bucketAlive[bucket.ID] = alive && bucket.Replicas != 0
			}
		}
	case config.SamplingModeFixedKServiceSet:
		if profile.FixedKFailures > len(ctx.serviceIDs) {
			return sampledState{}, fmt.Errorf("fixed_k_failures %d exceeds service count %d", profile.FixedKFailures, len(ctx.serviceIDs))
		}
		failedServices := map[string]struct{}{}
		if profile.FixedKFailures > 0 {
			indices := rng.Perm(len(ctx.serviceIDs))
			for _, idx := range indices[:profile.FixedKFailures] {
				failedServices[ctx.serviceIDs[idx]] = struct{}{}
			}
		}
		for _, serviceID := range ctx.serviceIDs {
			_, failed := failedServices[serviceID]
			for _, bucket := range ctx.services[serviceID].Buckets {
				state.bucketAlive[bucket.ID] = !failed && bucket.Replicas != 0
			}
		}
	default:
		return sampledState{}, fmt.Errorf("unsupported sampling mode %q", profile.SamplingMode)
	}

	for bucketID := range faultState.hardBucketKill {
		state.bucketAlive[bucketID] = false
	}
	for _, serviceID := range ctx.serviceIDs {
		serviceAlive := false
		for _, bucket := range ctx.services[serviceID].Buckets {
			if state.bucketAlive[bucket.ID] {
				serviceAlive = true
				break
			}
		}
		state.serviceAlive[serviceID] = serviceAlive
	}
	for _, edgeID := range ctx.edgeIDs {
		_, dead := faultState.hardEdgeKill[edgeID]
		state.edgeAlive[edgeID] = !dead
	}
	return state, nil
}

func executePath(ctx advancedContext, state sampledState, faultState profileFaultState, path journeyPath, rng *rand.Rand) bool {
	return executeServiceAt(ctx, state, faultState, path, 0, rng)
}

func executeServiceAt(ctx advancedContext, state sampledState, faultState profileFaultState, path journeyPath, serviceIndex int, rng *rand.Rand) bool {
	serviceID := path.Services[serviceIndex]
	if !state.serviceAlive[serviceID] {
		return false
	}
	if errorRate := faultState.serviceErrorRates[serviceID]; errorRate > 0 && rng.Float64() < errorRate {
		return false
	}
	if serviceIndex == len(path.Services)-1 {
		return true
	}
	return executeEdgeAt(ctx, state, faultState, path, serviceIndex, rng)
}

func executeEdgeAt(ctx advancedContext, state sampledState, faultState profileFaultState, path journeyPath, edgeIndex int, rng *rand.Rand) bool {
	edgeID := path.EdgeIDs[edgeIndex]
	if !state.edgeAlive[edgeID] {
		return false
	}
	edge := ctx.edges[edgeID]
	attempts := effectiveAttempts(edge.Resilience)
	for attempt := 0; attempt < attempts; attempt++ {
		if errorRate := combinedEdgeErrorRate(edge, faultState.edgeErrorRates[edgeID]); errorRate > 0 && rng.Float64() < errorRate {
			continue
		}
		timeoutAvailable, timeoutMismatch, _ := timeoutMismatchForEdge(ctx, path, edgeIndex, faultState)
		if timeoutAvailable && timeoutMismatch {
			continue
		}
		if executeServiceAt(ctx, state, faultState, path, edgeIndex+1, rng) {
			return true
		}
	}
	return false
}

func analyzePathStatic(ctx advancedContext, path journeyPath, faultState profileFaultState) pathStatic {
	expectedSuccess := MetricFloat{
		Available:  true,
		Value:      staticPathSuccess(ctx, path, faultState, 0),
		Provenance: combineAdvancedProvenance(true, hasAnyExternalDegradation(path, faultState), true),
	}
	amplification := pathAmplification(ctx, path, faultState, 0)
	timeoutCount, edgeTimeouts, timeoutReason := timeoutMismatchForPath(ctx, path, faultState)
	timeoutMetric := MetricInt{
		Available:  timeoutReason == "",
		Value:      timeoutCount,
		Reason:     timeoutReason,
		Provenance: provenanceFromTimeout(ctx, path, faultState, timeoutReason == ""),
	}
	if timeoutReason != "" {
		timeoutMetric.Provenance = advancedProvenanceUnavailable
	}
	return pathStatic{
		path:                 path,
		expectedSuccess:      expectedSuccess,
		amplification:        amplification,
		timeoutMismatchCount: timeoutMetric,
		edgeAmplification:    collectEdgeAmplification(ctx, path, faultState),
		edgeTimeoutMismatch:  edgeTimeouts,
	}
}

func staticPathSuccess(ctx advancedContext, path journeyPath, faultState profileFaultState, serviceIndex int) float64 {
	serviceID := path.Services[serviceIndex]
	if isServiceHardDead(ctx, faultState, serviceID) {
		return 0
	}
	serviceSuccess := 1 - faultState.serviceErrorRates[serviceID]
	if serviceIndex == len(path.Services)-1 {
		return serviceSuccess
	}
	edgeID := path.EdgeIDs[serviceIndex]
	edge := ctx.edges[edgeID]
	if _, dead := faultState.hardEdgeKill[edgeID]; dead {
		return 0
	}
	attemptSuccess := (1 - combinedEdgeErrorRate(edge, faultState.edgeErrorRates[edgeID])) * staticPathSuccess(ctx, path, faultState, serviceIndex+1)
	timeoutAvailable, timeoutMismatch, _ := timeoutMismatchForEdge(ctx, path, serviceIndex, faultState)
	if timeoutAvailable && timeoutMismatch {
		attemptSuccess = 0
	}
	return serviceSuccess * retrySuccessProbability(attemptSuccess, effectiveAttempts(edge.Resilience))
}

func pathAmplification(ctx advancedContext, path journeyPath, faultState profileFaultState, serviceIndex int) MetricFloat {
	if serviceIndex == len(path.Services)-1 {
		return MetricFloat{Available: true, Value: 1, Provenance: advancedProvenanceArtifact}
	}
	edgeID := path.EdgeIDs[serviceIndex]
	edge := ctx.edges[edgeID]
	if edge.Resilience == nil || edge.Resilience.Retry == nil {
		return MetricFloat{
			Available:  false,
			Reason:     fmt.Sprintf("retry metadata unavailable for edge %s", edgeID),
			Provenance: advancedProvenanceUnavailable,
		}
	}
	downstream := pathAmplification(ctx, path, faultState, serviceIndex+1)
	if !downstream.Available {
		return downstream
	}
	attemptSuccess := (1 - combinedEdgeErrorRate(edge, faultState.edgeErrorRates[edgeID])) * staticPathSuccess(ctx, path, faultState, serviceIndex+1)
	expectedAttempts := retryExpectedAttempts(attemptSuccess, effectiveAttempts(edge.Resilience))
	return MetricFloat{
		Available:  true,
		Value:      expectedAttempts * downstream.Value,
		Provenance: combineAdvancedProvenance(true, hasEdgeExternalData(edgeID, path, faultState), true),
	}
}

func collectEdgeAmplification(ctx advancedContext, path journeyPath, faultState profileFaultState) map[string]MetricFloat {
	out := make(map[string]MetricFloat, len(path.EdgeIDs))
	for idx, edgeID := range path.EdgeIDs {
		out[edgeID] = pathAmplification(ctx, path, faultState, idx)
	}
	return out
}

func timeoutMismatchForPath(ctx advancedContext, path journeyPath, faultState profileFaultState) (int, map[string]MetricInt, string) {
	count := 0
	perEdge := make(map[string]MetricInt, len(path.EdgeIDs))
	for idx, edgeID := range path.EdgeIDs {
		available, mismatch, reason := timeoutMismatchForEdge(ctx, path, idx, faultState)
		perEdge[edgeID] = MetricInt{
			Available:  available,
			Value:      boolToInt(mismatch),
			Reason:     reason,
			Provenance: provenanceFromTimeout(ctx, path, faultState, available),
		}
		if !available {
			return 0, perEdge, reason
		}
		if mismatch {
			count++
		}
	}
	return count, perEdge, ""
}

func timeoutMismatchForEdge(ctx advancedContext, path journeyPath, edgeIndex int, faultState profileFaultState) (bool, bool, string) {
	edgeID := path.EdgeIDs[edgeIndex]
	edge := ctx.edges[edgeID]
	timeout := resolveTimeout(edge.Resilience)
	if timeout <= 0 {
		return false, false, fmt.Sprintf("timeout metadata unavailable for edge %s", edgeID)
	}
	latency, ok := downstreamLatency(ctx, path, edgeIndex, faultState)
	if !ok {
		return false, false, fmt.Sprintf("latency metadata unavailable for edge %s", edgeID)
	}
	return true, float64(timeout) < latency, ""
}

func downstreamLatency(ctx advancedContext, path journeyPath, edgeIndex int, faultState profileFaultState) (float64, bool) {
	total := 0.0
	for idx := edgeIndex; idx < len(path.EdgeIDs); idx++ {
		edgeID := path.EdgeIDs[idx]
		edge := ctx.edges[edgeID]
		latency, ok := representativeLatency(edge.Observed, faultState.edgeLatencies[edgeID])
		if !ok {
			return 0, false
		}
		total += latency
		serviceLatency, serviceOK := latencyFromSummary(faultState.serviceLatencies[path.Services[idx+1]])
		if serviceOK {
			total += serviceLatency
		}
	}
	return total, true
}

func resolveTimeout(policy *model.ResiliencePolicy) int {
	if policy == nil {
		return 0
	}
	if policy.PerTryTimeoutMS > 0 {
		return policy.PerTryTimeoutMS
	}
	if policy.RequestTimeoutMS > 0 {
		return policy.RequestTimeoutMS
	}
	return 0
}

func representativeLatency(observed *model.ObservedEdge, injected model.LatencySummary) (float64, bool) {
	base, baseOK := latencyFromObserved(observed)
	injectedValue, injectedOK := latencyFromSummary(injected)
	switch {
	case baseOK && injectedOK:
		return base + injectedValue, true
	case baseOK:
		return base, true
	case injectedOK:
		return injectedValue, true
	default:
		return 0, false
	}
}

func latencyFromObserved(observed *model.ObservedEdge) (float64, bool) {
	if observed == nil || observed.LatencyMS == nil {
		return 0, false
	}
	return latencyFromSummary(*observed.LatencyMS)
}

func latencyFromSummary(summary model.LatencySummary) (float64, bool) {
	switch {
	case summary.P99 > 0:
		return summary.P99, true
	case summary.P95 > 0:
		return summary.P95, true
	case summary.P90 > 0:
		return summary.P90, true
	case summary.P50 > 0:
		return summary.P50, true
	default:
		return 0, false
	}
}

func retrySuccessProbability(attemptSuccess float64, attempts int) float64 {
	if attemptSuccess <= 0 {
		return 0
	}
	if attemptSuccess >= 1 {
		return 1
	}
	return 1 - math.Pow(1-attemptSuccess, float64(attempts))
}

func retryExpectedAttempts(attemptSuccess float64, attempts int) float64 {
	if attempts <= 0 {
		return 0
	}
	if attemptSuccess <= 0 {
		return float64(attempts)
	}
	sum := 0.0
	failure := 1.0
	for attempt := 0; attempt < attempts; attempt++ {
		sum += failure
		failure *= 1 - attemptSuccess
	}
	return sum
}

func effectiveAttempts(policy *model.ResiliencePolicy) int {
	attempts := 1
	if policy != nil && policy.Retry != nil && policy.Retry.MaxAttempts > 1 {
		attempts = policy.Retry.MaxAttempts
	}
	if policy != nil && policy.CircuitBreaker != nil && policy.CircuitBreaker.Enabled != nil && *policy.CircuitBreaker.Enabled && policy.CircuitBreaker.MaxRequests > 0 && policy.CircuitBreaker.MaxRequests < attempts {
		attempts = policy.CircuitBreaker.MaxRequests
	}
	return attempts
}

func combinedEdgeErrorRate(edge normalizedEdge, injected float64) float64 {
	base := 0.0
	if edge.Observed != nil && edge.Observed.ErrorRate != nil {
		base = *edge.Observed.ErrorRate
	}
	return combineErrorRates(base, injected)
}

func combineErrorRates(values ...float64) float64 {
	success := 1.0
	for _, value := range values {
		if value <= 0 {
			continue
		}
		success *= 1 - value
	}
	return 1 - success
}

func buildProfileFaultState(ctx advancedContext, profile *faults.Profile) profileFaultState {
	state := profileFaultState{
		hardBucketKill:    map[string]struct{}{},
		hardEdgeKill:      map[string]struct{}{},
		serviceErrorRates: map[string]float64{},
		serviceLatencies:  map[string]model.LatencySummary{},
		edgeErrorRates:    map[string]float64{},
		edgeLatencies:     map[string]model.LatencySummary{},
	}
	if profile == nil {
		state.blastRadius = &BlastRadius{
			ServiceCount:  MetricInt{Available: true, Value: 0, Provenance: advancedProvenanceExternalContract},
			EndpointCount: MetricInt{Available: true, Value: 0, Provenance: advancedProvenanceExternalContract},
		}
		return state
	}

	impactedServices := map[string]struct{}{}
	for _, fault := range profile.Faults {
		match := FaultMatch{
			FaultType: fault.Type,
			Selector:  fault.Selector,
		}
		switch fault.Type {
		case faults.TypeCorrelatedFailureDomain:
			serviceIDs, bucketIDs, sharedRefs := selectServicesForFault(ctx, fault.Selector, fault.OnlyFailureEligible)
			match.MatchedServiceIDs = serviceIDs
			match.MatchedPlacementBucketIDs = bucketIDs
			match.MatchedSharedResources = sharedRefs
			for _, serviceID := range serviceIDs {
				impactedServices[serviceID] = struct{}{}
			}
			for _, bucketID := range bucketIDs {
				state.hardBucketKill[bucketID] = struct{}{}
			}
			if len(bucketIDs) == 0 {
				for _, serviceID := range serviceIDs {
					for _, bucket := range ctx.services[serviceID].Buckets {
						state.hardBucketKill[bucket.ID] = struct{}{}
					}
				}
			}
		case faults.TypeEdgeFailStop:
			for _, edgeID := range fault.Selector.EdgeIDs {
				if _, ok := ctx.edges[edgeID]; !ok {
					continue
				}
				state.hardEdgeKill[edgeID] = struct{}{}
				match.MatchedEdgeIDs = append(match.MatchedEdgeIDs, edgeID)
			}
		case faults.TypeEdgePartialDegradation:
			for _, edgeID := range fault.Selector.EdgeIDs {
				if _, ok := ctx.edges[edgeID]; !ok {
					continue
				}
				match.MatchedEdgeIDs = append(match.MatchedEdgeIDs, edgeID)
				if fault.ErrorRate != nil {
					state.edgeErrorRates[edgeID] = combineErrorRates(state.edgeErrorRates[edgeID], *fault.ErrorRate)
				}
				if fault.LatencyMS != nil {
					state.edgeLatencies[edgeID] = addLatencySummary(state.edgeLatencies[edgeID], *fault.LatencyMS)
				}
			}
		case faults.TypeServicePartialDegradation:
			serviceIDs, _, sharedRefs := selectServicesForFault(ctx, fault.Selector, false)
			match.MatchedServiceIDs = serviceIDs
			match.MatchedSharedResources = sharedRefs
			for _, serviceID := range serviceIDs {
				impactedServices[serviceID] = struct{}{}
				if fault.ErrorRate != nil {
					state.serviceErrorRates[serviceID] = combineErrorRates(state.serviceErrorRates[serviceID], *fault.ErrorRate)
				}
				if fault.LatencyMS != nil {
					state.serviceLatencies[serviceID] = addLatencySummary(state.serviceLatencies[serviceID], *fault.LatencyMS)
				}
			}
		}
		if len(match.MatchedServiceIDs) > 0 || len(match.MatchedEdgeIDs) > 0 || len(match.MatchedPlacementBucketIDs) > 0 {
			state.faultMatches = append(state.faultMatches, match)
		}
	}

	impactedEndpoints := map[string]struct{}{}
	for _, endpoint := range ctx.endpoints {
		if endpointImpacted(endpoint, impactedServices, state.hardEdgeKill) {
			impactedEndpoints[endpoint.ID] = struct{}{}
		}
	}
	state.blastRadius = &BlastRadius{
		ServiceCount: MetricInt{
			Available:  true,
			Value:      len(impactedServices),
			Provenance: advancedProvenanceExternalContract,
		},
		EndpointCount: MetricInt{
			Available:  true,
			Value:      len(impactedEndpoints),
			Provenance: advancedProvenanceExternalContract,
		},
		ServiceIDs:  sortedKeys(impactedServices),
		EndpointIDs: sortedKeys(impactedEndpoints),
	}
	for idx := range state.faultMatches {
		state.faultMatches[idx].MatchedEndpointIDs = slices.Clone(state.blastRadius.EndpointIDs)
	}
	return state
}

func endpointImpacted(endpoint endpointPlan, impactedServices map[string]struct{}, hardEdges map[string]struct{}) bool {
	if endpoint.Mode == endpointModePredicate {
		for _, serviceID := range predicateServices(endpoint.Predicate) {
			if _, ok := impactedServices[serviceID]; ok {
				return true
			}
		}
	}
	for _, path := range endpoint.DiagnosticPaths {
		for _, serviceID := range path.Services {
			if _, ok := impactedServices[serviceID]; ok {
				return true
			}
		}
		for _, edgeID := range path.EdgeIDs {
			if _, ok := hardEdges[edgeID]; ok {
				return true
			}
		}
	}
	return false
}

func predicateServices(def predicates.Definition) []string {
	out := make([]string, 0, len(def.Services))
	out = append(out, def.Services...)
	for _, child := range def.Children {
		out = append(out, predicateServices(child)...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func selectServicesForFault(ctx advancedContext, selector faults.Selector, onlyFailureEligible bool) ([]string, []string, []string) {
	serviceIDs := make([]string, 0)
	bucketIDs := make([]string, 0)
	sharedRefs := map[string]struct{}{}
	for _, serviceID := range ctx.serviceIDs {
		svc := ctx.services[serviceID]
		if onlyFailureEligible && !svc.FailureEligible {
			continue
		}
		if len(selector.ServiceIDs) > 0 && !slices.Contains(selector.ServiceIDs, serviceID) {
			continue
		}
		if !labelsMatch(svc.Labels, selector.ServiceLabels) {
			continue
		}
		if len(selector.SharedResourceRefs) > 0 && !sliceIntersects(svc.SharedResourceRefs, selector.SharedResourceRefs) {
			continue
		}
		matchedBucket := false
		if len(selector.PlacementLabels) > 0 {
			for _, bucket := range svc.Buckets {
				if labelsMatch(bucket.Labels, selector.PlacementLabels) {
					bucketIDs = append(bucketIDs, bucket.ID)
					matchedBucket = true
				}
			}
			if !matchedBucket {
				continue
			}
		}
		serviceIDs = append(serviceIDs, serviceID)
		for _, ref := range svc.SharedResourceRefs {
			if len(selector.SharedResourceRefs) == 0 || slices.Contains(selector.SharedResourceRefs, ref) {
				sharedRefs[ref] = struct{}{}
			}
		}
	}
	slices.Sort(serviceIDs)
	slices.Sort(bucketIDs)
	return serviceIDs, bucketIDs, sortedKeys(sharedRefs)
}

func buildAdvancedContext(loaded artifact.Loaded, predicateSet map[string]predicates.Definition, journeyOverrides map[string][][]string) (advancedContext, error) {
	if err := loaded.Model.Validate(); err != nil {
		return advancedContext{}, fmt.Errorf("invalid model: %w", err)
	}
	serviceRecords := map[string]artifact.SnapshotDiscoveryService{}
	edgeRecords := map[string]artifact.SnapshotDiscoveryEdge{}
	endpointRecords := map[string]artifact.SnapshotDiscoveryEndpoint{}
	if loaded.Snapshot != nil {
		for _, service := range loaded.Snapshot.Discovery.Services {
			serviceRecords[service.ID] = service
		}
		for _, edge := range loaded.Snapshot.Discovery.Edges {
			edgeRecords[edge.ID] = edge
		}
		for _, endpoint := range loaded.Snapshot.Discovery.Endpoints {
			endpointRecords[endpoint.ID] = endpoint
		}
	}

	ctx := advancedContext{
		contractVersion: loaded.Metadata.Contract.Version,
		services:        make(map[string]normalizedService, len(loaded.Model.Services)),
		edges:           make(map[string]normalizedEdge, len(loaded.Model.Edges)),
		endpointByID:    map[string]endpointPlan{},
	}

	for _, svc := range loaded.Model.Services {
		record := serviceRecords[svc.ID]
		labels := map[string]string{}
		sharedRefs := []string{}
		failureEligible := false
		if svc.Metadata != nil {
			labels = mergeStringMap(labels, svc.Metadata.Labels)
			sharedRefs = append(sharedRefs, svc.Metadata.SharedResourceRefs...)
			failureEligible = svc.Metadata.FailureEligible != nil && *svc.Metadata.FailureEligible
		}
		if len(record.Metadata.Labels) > 0 {
			labels = mergeStringMap(labels, record.Metadata.Labels)
		}
		if len(sharedRefs) == 0 && len(record.Metadata.SharedResourceRefs) > 0 {
			sharedRefs = append(sharedRefs, record.Metadata.SharedResourceRefs...)
		}
		if !failureEligible && record.Metadata.FailureEligible != nil && *record.Metadata.FailureEligible {
			failureEligible = true
		}
		buckets := normalizeBuckets(svc, record)
		ctx.services[svc.ID] = normalizedService{
			ID:                 svc.ID,
			Name:               svc.Name,
			Labels:             labels,
			SharedResourceRefs: sortedUnique(sharedRefs),
			FailureEligible:    failureEligible,
			Buckets:            buckets,
		}
		ctx.serviceIDs = append(ctx.serviceIDs, svc.ID)
	}
	slices.Sort(ctx.serviceIDs)

	edgeLookup := map[string]map[string]string{}
	for _, edge := range loaded.Model.Edges {
		edgeID := edge.ID
		if strings.TrimSpace(edgeID) == "" {
			edgeID = canonicalEdgeID(edge.From, edge.To, edge.Kind, edge.Blocking)
		}
		record := findEdgeRecord(edgeRecords, edgeID, edge.From, edge.To, edge.Kind, edge.Blocking)
		labels := map[string]string{}
		if edge.Metadata != nil {
			labels = mergeStringMap(labels, edge.Metadata.Labels)
		}
		labels = mergeStringMap(labels, record.Metadata.Labels)
		resilience := edge.Resilience
		if resilience == nil {
			resilience = record.Resilience
		}
		observed := edge.Observed
		if observed == nil {
			observed = record.Observed
		}
		ctx.edges[edgeID] = normalizedEdge{
			ID:         edgeID,
			From:       edge.From,
			To:         edge.To,
			Kind:       edge.Kind,
			Blocking:   edge.Blocking,
			Labels:     labels,
			Resilience: resilience,
			Observed:   observed,
		}
		if _, ok := edgeLookup[edge.From]; !ok {
			edgeLookup[edge.From] = map[string]string{}
		}
		if existing, ok := edgeLookup[edge.From][edge.To]; !ok || strings.Compare(edgeID, existing) < 0 {
			edgeLookup[edge.From][edge.To] = edgeID
		}
		ctx.edgeIDs = append(ctx.edgeIDs, edgeID)
	}
	slices.Sort(ctx.edgeIDs)

	serviceSet := make(map[string]struct{}, len(ctx.services))
	for serviceID := range ctx.services {
		serviceSet[serviceID] = struct{}{}
	}
	mergedPredicates := make(map[string]predicates.Definition, len(loaded.Model.Predicates)+len(predicateSet))
	for key, value := range loaded.Model.Predicates {
		mergedPredicates[key] = value
	}
	for key, value := range predicateSet {
		mergedPredicates[key] = value
	}

	adj := map[string][]string{}
	for _, edgeID := range ctx.edgeIDs {
		edge := ctx.edges[edgeID]
		if !edge.Blocking || edge.Kind == model.EdgeKindAsync {
			continue
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
	}
	for serviceID := range adj {
		slices.Sort(adj[serviceID])
	}

	for _, ep := range loaded.Model.Endpoints {
		record := endpointRecords[ep.ID]
		paths := journeyOverrides[ep.ID]
		if len(paths) == 0 {
			paths = discoverJourneys(ep.EntryService, adj)
		}
		journeyPaths, err := mapJourneyPaths(paths, edgeLookup)
		if err != nil {
			return advancedContext{}, fmt.Errorf("endpoint %s: %w", ep.ID, err)
		}

		plan := endpointPlan{
			ID:              ep.ID,
			EntryService:    ep.EntryService,
			Method:          firstNonEmptyString(ep.Method, record.Method),
			Path:            firstNonEmptyString(ep.Path, record.Path),
			DiagnosticPaths: journeyPaths,
		}
		switch {
		case ep.SuccessPredicate != nil:
			if err := validatePredicateServices(*ep.SuccessPredicate, serviceSet); err != nil {
				return advancedContext{}, fmt.Errorf("endpoint %s: %w", ep.ID, err)
			}
			plan.Mode = endpointModePredicate
			plan.Predicate = *ep.SuccessPredicate
		case hasPredicate(mergedPredicates, ep.SuccessPredicateRef):
			def := mergedPredicates[ep.SuccessPredicateRef]
			if err := validatePredicateServices(def, serviceSet); err != nil {
				return advancedContext{}, fmt.Errorf("endpoint %s: %w", ep.ID, err)
			}
			plan.Mode = endpointModePredicate
			plan.Predicate = def
		default:
			plan.Mode = endpointModeJourney
			plan.Paths = journeyPaths
		}
		ctx.endpoints = append(ctx.endpoints, plan)
		ctx.endpointByID[plan.ID] = plan
	}
	slices.SortFunc(ctx.endpoints, func(a, b endpointPlan) int {
		return strings.Compare(a.ID, b.ID)
	})
	return ctx, nil
}

func mapJourneyPaths(paths [][]string, edgeLookup map[string]map[string]string) ([]journeyPath, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out := make([]journeyPath, 0, len(paths))
	for _, services := range cloneAndNormalizeJourneys(paths) {
		edgeIDs := make([]string, 0, max(0, len(services)-1))
		for idx := 0; idx < len(services)-1; idx++ {
			from := services[idx]
			to := services[idx+1]
			targets, ok := edgeLookup[from]
			if !ok {
				return nil, fmt.Errorf("missing blocking sync edge %s -> %s", from, to)
			}
			edgeID, ok := targets[to]
			if !ok {
				return nil, fmt.Errorf("missing blocking sync edge %s -> %s", from, to)
			}
			edgeIDs = append(edgeIDs, edgeID)
		}
		out = append(out, journeyPath{
			ID:       strings.Join(services, "->"),
			Services: slices.Clone(services),
			EdgeIDs:  edgeIDs,
		})
	}
	return out, nil
}

func normalizeBuckets(svc model.Service, record artifact.SnapshotDiscoveryService) []replicaBucket {
	placements := []model.Placement{}
	switch {
	case svc.Metadata != nil && len(svc.Metadata.Placements) > 0:
		placements = append(placements, svc.Metadata.Placements...)
	case len(record.Metadata.Placements) > 0:
		placements = append(placements, record.Metadata.Placements...)
	}
	if len(placements) == 0 {
		return []replicaBucket{
			{
				ID:        svc.ID + "#0",
				ServiceID: svc.ID,
				Replicas:  svc.Replicas,
			},
		}
	}
	out := make([]replicaBucket, 0, len(placements))
	for idx, placement := range placements {
		out = append(out, replicaBucket{
			ID:        fmt.Sprintf("%s#%d", svc.ID, idx),
			ServiceID: svc.ID,
			Replicas:  placement.Replicas,
			Labels:    mergeStringMap(nil, placement.Labels),
		})
	}
	return out
}

func findEdgeRecord(records map[string]artifact.SnapshotDiscoveryEdge, edgeID, from, to string, kind model.EdgeKind, blocking bool) artifact.SnapshotDiscoveryEdge {
	if record, ok := records[edgeID]; ok {
		return record
	}
	fallback := canonicalEdgeID(from, to, kind, blocking)
	if record, ok := records[fallback]; ok {
		return record
	}
	return artifact.SnapshotDiscoveryEdge{}
}

func canonicalEdgeID(from, to string, kind model.EdgeKind, blocking bool) string {
	return fmt.Sprintf("%s|%s|%s|%t", from, to, kind, blocking)
}

func resolveFaultProfile(contract *faults.Contract, name string) (*faults.Profile, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	if contract == nil {
		return nil, fmt.Errorf("fault_profile %q requires analysis.fault_contract", name)
	}
	profile, ok := contract.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("fault_profile %q not found in fault contract", name)
	}
	return &profile, nil
}

func isServiceHardDead(ctx advancedContext, faultState profileFaultState, serviceID string) bool {
	svc := ctx.services[serviceID]
	for _, bucket := range svc.Buckets {
		if _, dead := faultState.hardBucketKill[bucket.ID]; !dead && bucket.Replicas != 0 {
			return false
		}
	}
	return true
}

func addLatencySummary(dst, src model.LatencySummary) model.LatencySummary {
	dst.P50 += src.P50
	dst.P90 += src.P90
	dst.P95 += src.P95
	dst.P99 += src.P99
	return dst
}

func hasAnyExternalDegradation(path journeyPath, faultState profileFaultState) bool {
	for _, serviceID := range path.Services {
		if faultState.serviceErrorRates[serviceID] > 0 {
			return true
		}
		if _, ok := latencyFromSummary(faultState.serviceLatencies[serviceID]); ok {
			return true
		}
	}
	for _, edgeID := range path.EdgeIDs {
		if hasEdgeExternalData(edgeID, path, faultState) {
			return true
		}
	}
	return false
}

func hasEdgeExternalData(edgeID string, _ journeyPath, faultState profileFaultState) bool {
	if faultState.edgeErrorRates[edgeID] > 0 {
		return true
	}
	_, ok := latencyFromSummary(faultState.edgeLatencies[edgeID])
	return ok
}

func provenanceFromTimeout(ctx advancedContext, path journeyPath, faultState profileFaultState, available bool) string {
	if !available {
		return advancedProvenanceUnavailable
	}
	artifactUsed := false
	externalUsed := hasAnyExternalDegradation(path, faultState)
	for _, edgeID := range path.EdgeIDs {
		edge := ctx.edges[edgeID]
		if resolveTimeout(edge.Resilience) > 0 {
			artifactUsed = true
		}
		if latency, ok := latencyFromObserved(edge.Observed); ok && latency > 0 {
			artifactUsed = true
		}
	}
	return combineAdvancedProvenance(artifactUsed, externalUsed, true)
}

func combineAdvancedProvenance(artifactUsed, externalUsed, available bool) string {
	if !available {
		return advancedProvenanceUnavailable
	}
	switch {
	case artifactUsed && externalUsed:
		return advancedProvenanceArtifactAndExternal
	case artifactUsed:
		return advancedProvenanceArtifact
	case externalUsed:
		return advancedProvenanceExternalContract
	default:
		return advancedProvenanceArtifact
	}
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func mergeStringMap(dst map[string]string, src map[string]string) map[string]string {
	if dst == nil {
		dst = map[string]string{}
	}
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := slices.Clone(values)
	slices.Sort(out)
	return slices.Compact(out)
}

func labelsMatch(actual map[string]string, selector map[string]string) bool {
	for key, expected := range selector {
		if actual[key] != expected {
			return false
		}
	}
	return true
}

func sliceIntersects(left []string, right []string) bool {
	for _, candidate := range left {
		if slices.Contains(right, candidate) {
			return true
		}
	}
	return false
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
