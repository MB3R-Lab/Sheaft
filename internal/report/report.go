package report

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

type SimulationInfo struct {
	Trials             int     `json:"trials"`
	Seed               int64   `json:"seed"`
	FailureProbability float64 `json:"failure_probability"`
}

type Summary struct {
	OverallAvailability              float64 `json:"overall_availability"`
	WeightedOverallAvailability      float64 `json:"weighted_overall_availability,omitempty"`
	CrossProfileAvailability         float64 `json:"cross_profile_availability,omitempty"`
	CrossProfileWeightedAvailability float64 `json:"cross_profile_weighted_availability,omitempty"`
	RiskScore                        float64 `json:"risk_score"`
	Confidence                       float64 `json:"confidence"`
}

type PolicyEvaluation struct {
	Mode             string   `json:"mode"`
	Decision         string   `json:"decision"`
	FailedEndpoints  []string `json:"failed_endpoints"`
	FailedAssertions []string `json:"failed_assertions,omitempty"`
	FailedProfiles   []string `json:"failed_profiles,omitempty"`
	EvaluationRule   string   `json:"evaluation_rule,omitempty"`
}

type InputArtifact struct {
	Path            string `json:"path"`
	Digest          string `json:"digest"`
	ArtifactID      string `json:"artifact_id,omitempty"`
	Kind            string `json:"kind"`
	ContractName    string `json:"contract_name"`
	ContractVersion string `json:"contract_version"`
	SourceType      string `json:"source_type"`
	SourceRef       string `json:"source_ref"`
	ProducedAt      string `json:"produced_at,omitempty"`
	TopologyVersion string `json:"topology_version,omitempty"`
}

type Provenance struct {
	PredicateSource string `json:"predicate_source"`
	WeightsSource   string `json:"weights_source"`
}

type ContractPolicy struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Message string `json:"message,omitempty"`
}

type IntParameter struct {
	Value  int64  `json:"value"`
	Source string `json:"source"`
}

type FloatParameter struct {
	Value  float64 `json:"value"`
	Source string  `json:"source"`
}

type StringParameter struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

type ParameterStatus struct {
	Active   bool     `json:"active"`
	Source   string   `json:"source"`
	Path     string   `json:"path,omitempty"`
	Names    []string `json:"names,omitempty"`
	Fallback string   `json:"fallback,omitempty"`
}

type ProfileParameters struct {
	Name               string          `json:"name"`
	Trials             IntParameter    `json:"trials"`
	Seed               IntParameter    `json:"seed"`
	SamplingMode       StringParameter `json:"sampling_mode"`
	FailureProbability FloatParameter  `json:"failure_probability"`
	FixedKFailures     IntParameter    `json:"fixed_k_failures"`
	FaultProfile       StringParameter `json:"fault_profile"`
	EndpointWeights    ParameterStatus `json:"endpoint_weights"`
}

type CalibrationParameters struct {
	PredicateOverlay  ParameterStatus `json:"predicate_overlay"`
	FaultContract     ParameterStatus `json:"fault_contract"`
	JourneyOverrides  ParameterStatus `json:"journey_overrides"`
	Baselines         ParameterStatus `json:"baselines"`
	HistoricalSignals ParameterStatus `json:"historical_signals"`
}

type Parameters struct {
	ConfigSource string                `json:"config_source"`
	Profiles     []ProfileParameters   `json:"profiles"`
	Calibration  CalibrationParameters `json:"calibration"`
}

type ProfileSummary struct {
	Name                    string                   `json:"name"`
	Simulation              simulation.ProfileOutput `json:"simulation"`
	EndpointResults         []gate.EndpointResult    `json:"endpoint_results"`
	Decision                string                   `json:"decision"`
	EndpointsBelowThreshold int                      `json:"endpoints_below_threshold"`
	Aggregate               *gate.AggregateResult    `json:"aggregate,omitempty"`
}

type Delta struct {
	Current   float64 `json:"current"`
	Reference float64 `json:"reference"`
	Signed    float64 `json:"signed"`
	Absolute  float64 `json:"absolute"`
}

type StatusDelta struct {
	Current   string `json:"current"`
	Reference string `json:"reference"`
	Changed   bool   `json:"changed"`
}

type EndpointDiff struct {
	Profile      string      `json:"profile"`
	EndpointID   string      `json:"endpoint_id"`
	Availability Delta       `json:"availability"`
	Status       StatusDelta `json:"status"`
}

type ProfileDiff struct {
	Profile             string         `json:"profile"`
	WeightedAggregate   Delta          `json:"weighted_aggregate"`
	UnweightedAggregate Delta          `json:"unweighted_aggregate"`
	Decision            StatusDelta    `json:"decision"`
	Endpoints           []EndpointDiff `json:"endpoints"`
	AdvancedMetrics     []MetricDiff   `json:"advanced_metrics,omitempty"`
}

type MetricDiff struct {
	Metric     string `json:"metric"`
	TargetType string `json:"target_type"`
	Target     string `json:"target"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	Delta      *Delta `json:"delta,omitempty"`
}

type Diff struct {
	Name                     string        `json:"name,omitempty"`
	CurrentDigest            string        `json:"current_digest,omitempty"`
	ReferenceDigest          string        `json:"reference_digest,omitempty"`
	CurrentTopologyVersion   string        `json:"current_topology_version,omitempty"`
	ReferenceTopologyVersion string        `json:"reference_topology_version,omitempty"`
	CrossProfileWeighted     Delta         `json:"cross_profile_weighted"`
	CrossProfileUnweighted   Delta         `json:"cross_profile_unweighted"`
	Profiles                 []ProfileDiff `json:"profiles"`
}

type Diffs struct {
	Previous  *Diff  `json:"previous,omitempty"`
	Baselines []Diff `json:"baselines,omitempty"`
}

type Report struct {
	Simulation          SimulationInfo        `json:"simulation"`
	EndpointResults     []gate.EndpointResult `json:"endpoint_results"`
	Summary             Summary               `json:"summary"`
	PolicyEvaluation    PolicyEvaluation      `json:"policy_evaluation"`
	InputArtifact       *InputArtifact        `json:"input_artifact,omitempty"`
	ContractPolicy      *ContractPolicy       `json:"contract_policy,omitempty"`
	Provenance          *Provenance           `json:"provenance,omitempty"`
	Parameters          *Parameters           `json:"parameters,omitempty"`
	Profiles            []ProfileSummary      `json:"profiles,omitempty"`
	Diffs               Diffs                 `json:"diffs,omitempty"`
	GeneratedAt         string                `json:"generated_at,omitempty"`
	RecomputeDurationMS int64                 `json:"recompute_duration_ms,omitempty"`
}

func Compose(simOut simulation.Output, eval gate.Evaluation, params simulation.Params, confidence float64) Report {
	return Report{
		Simulation: SimulationInfo{
			Trials:             params.Trials,
			Seed:               params.Seed,
			FailureProbability: params.FailureProbability,
		},
		EndpointResults: eval.EndpointResults,
		Summary: Summary{
			OverallAvailability:         simOut.OverallAvailability,
			WeightedOverallAvailability: simOut.OverallAvailability,
			RiskScore:                   1 - simOut.OverallAvailability,
			Confidence:                  confidence,
		},
		PolicyEvaluation: PolicyEvaluation{
			Mode:            string(eval.Mode),
			Decision:        eval.Decision,
			FailedEndpoints: eval.FailedEndpoints,
		},
	}
}

func ComposeAnalysis(meta artifact.Loaded, simOut simulation.AnalysisOutput, eval gate.Evaluation, cfg config.AnalysisConfig, contractDecision config.ContractPolicyDecision, confidence float64, generatedAt time.Time, duration time.Duration) Report {
	report := Report{
		Simulation: SimulationInfo{},
		Summary: Summary{
			Confidence:                       confidence,
			CrossProfileAvailability:         simOut.CrossProfileUnweighted,
			CrossProfileWeightedAvailability: simOut.CrossProfileWeighted,
			OverallAvailability:              simOut.CrossProfileUnweighted,
			WeightedOverallAvailability:      simOut.CrossProfileWeighted,
			RiskScore:                        1 - simOut.CrossProfileWeighted,
		},
		PolicyEvaluation: PolicyEvaluation{
			Mode:             string(eval.Mode),
			Decision:         eval.Decision,
			FailedEndpoints:  slices.Clone(eval.FailedEndpoints),
			FailedAssertions: slices.Clone(eval.FailedAssertions),
			FailedProfiles:   slices.Clone(eval.FailedProfiles),
			EvaluationRule:   string(eval.EvaluationRule),
		},
		InputArtifact: &InputArtifact{
			Path:            meta.Metadata.Path,
			Digest:          meta.Metadata.Digest,
			ArtifactID:      meta.Metadata.ArtifactID,
			Kind:            string(meta.Metadata.Kind),
			ContractName:    meta.Metadata.Contract.Name,
			ContractVersion: meta.Metadata.Contract.Version,
			SourceType:      meta.Metadata.SourceType,
			SourceRef:       meta.Metadata.SourceRef,
			ProducedAt:      meta.Metadata.ProducedAt,
			TopologyVersion: meta.Metadata.TopologyVersion,
		},
		ContractPolicy: &ContractPolicy{
			Status:  contractDecision.Status,
			Action:  contractDecision.Action,
			Message: contractDecision.Message,
		},
		Provenance: &Provenance{
			PredicateSource: meta.PredicateSource,
			WeightsSource:   meta.WeightsSource,
		},
		Parameters:          buildParameters(cfg, meta),
		Profiles:            make([]ProfileSummary, 0, len(simOut.Profiles)),
		GeneratedAt:         generatedAt.UTC().Format(time.RFC3339Nano),
		RecomputeDurationMS: duration.Milliseconds(),
	}

	if len(simOut.Profiles) > 0 {
		first := simOut.Profiles[0]
		report.Simulation = SimulationInfo{
			Trials:             first.Trials,
			Seed:               first.Seed,
			FailureProbability: first.FailureProbability,
		}
		report.Summary.OverallAvailability = first.UnweightedAggregate
		report.Summary.WeightedOverallAvailability = first.WeightedAggregate
		report.Summary.RiskScore = 1 - first.WeightedAggregate
	}

	for _, profile := range simOut.Profiles {
		profileEval := findProfileEvaluation(eval.ProfileEvaluations, profile.Name)
		if report.EndpointResults == nil {
			report.EndpointResults = slices.Clone(profileEval.EndpointResults)
		}
		report.Profiles = append(report.Profiles, ProfileSummary{
			Name:                    profile.Name,
			Simulation:              profile,
			EndpointResults:         slices.Clone(profileEval.EndpointResults),
			Decision:                profileEval.Decision,
			EndpointsBelowThreshold: profileEval.EndpointsBelowThreshold,
			Aggregate:               profileEval.Aggregate,
		})
	}
	return report
}

func buildParameters(cfg config.AnalysisConfig, meta artifact.Loaded) *Parameters {
	profiles := make([]ProfileParameters, 0, len(cfg.Profiles))
	for _, profile := range cfg.Profiles {
		profileSources := cfg.Sources.Profiles[profile.Name]
		profiles = append(profiles, ProfileParameters{
			Name: profile.Name,
			Trials: IntParameter{
				Value:  int64(profile.Trials),
				Source: string(profileSources.Trials),
			},
			Seed: IntParameter{
				Value:  derivedProfileSeed(cfg, profile),
				Source: string(cfg.Sources.Seed),
			},
			SamplingMode: StringParameter{
				Value:  profile.SamplingMode,
				Source: string(profileSources.SamplingMode),
			},
			FailureProbability: FloatParameter{
				Value:  profile.FailureProbability,
				Source: string(profileSources.FailureProbability),
			},
			FixedKFailures: IntParameter{
				Value:  int64(profile.FixedKFailures),
				Source: string(profileSources.FixedKFailures),
			},
			FaultProfile: StringParameter{
				Value:  profile.FaultProfile,
				Source: string(profileSources.FaultProfile),
			},
			EndpointWeights: parameterStatusForWeights(profileSources.EndpointWeights, meta.WeightsSource),
		})
	}

	return &Parameters{
		ConfigSource: string(cfg.Sources.ConfigSource),
		Profiles:     profiles,
		Calibration: CalibrationParameters{
			PredicateOverlay: parameterStatus(
				cfg.PredicateContract != "",
				config.ParameterSourceExternal,
				cfg.PredicateContract,
				nil,
				"using artifact predicates or legacy path resolution without external predicate overlay",
			),
			FaultContract: parameterStatus(
				cfg.FaultContract != "",
				cfg.Sources.FaultContract,
				cfg.FaultContract,
				nil,
				"no fault contract configured; advanced fault profiles are inactive",
			),
			JourneyOverrides: parameterStatus(
				cfg.Journeys != "",
				cfg.Sources.Journeys,
				cfg.Journeys,
				nil,
				"using richer predicates or discovered journeys without manual journey override",
			),
			Baselines: parameterStatus(
				len(cfg.Baselines) > 0,
				config.ParameterSourceExternal,
				"",
				baselineNames(cfg.Baselines),
				"no baseline reports configured",
			),
			HistoricalSignals: parameterStatus(
				false,
				config.ParameterSourceDefault,
				"",
				nil,
				"historical calibration inputs are not implemented; static configuration is used",
			),
		},
	}
}

func parameterStatus(active bool, source config.ParameterSource, path string, names []string, fallback string) ParameterStatus {
	status := ParameterStatus{
		Active: active,
		Source: string(source),
	}
	if active {
		status.Path = path
		status.Names = slices.Clone(names)
		return status
	}
	status.Fallback = fallback
	return status
}

func parameterStatusForWeights(source config.ParameterSource, artifactSource string) ParameterStatus {
	if source != config.ParameterSourceDefault {
		return ParameterStatus{
			Active: true,
			Source: string(source),
		}
	}
	if artifactSource != artifact.ProvenanceDefault {
		return ParameterStatus{
			Active: true,
			Source: string(config.ParameterSourceExternal),
		}
	}
	return ParameterStatus{
		Active:   false,
		Source:   string(config.ParameterSourceDefault),
		Fallback: "no endpoint weights configured; weighted aggregate falls back to arithmetic mean",
	}
}

func baselineNames(baselines []config.BaselineRef) []string {
	names := make([]string, 0, len(baselines))
	for _, baseline := range baselines {
		names = append(names, baseline.Name)
	}
	slices.Sort(names)
	return names
}

func derivedProfileSeed(cfg config.AnalysisConfig, profile config.Profile) int64 {
	for idx, candidate := range cfg.Profiles {
		if candidate.Name == profile.Name {
			if cfg.Seed == 0 {
				return 0
			}
			h := fnv.New64a()
			_, _ = h.Write([]byte(fmt.Sprintf("%d:%s:%d", cfg.Seed, profile.Name, idx)))
			return int64(h.Sum64())
		}
	}
	return cfg.Seed
}

func (r Report) AvailabilityMap() map[string]float64 {
	out := make(map[string]float64, len(r.EndpointResults))
	for _, endpoint := range r.EndpointResults {
		out[endpoint.EndpointID] = endpoint.Availability
	}
	return out
}

func (r Report) NormalizedProfiles() []ProfileSummary {
	if len(r.Profiles) > 0 {
		return slices.Clone(r.Profiles)
	}
	return []ProfileSummary{
		{
			Name: "default",
			Simulation: simulation.ProfileOutput{
				Name:                 "default",
				Trials:               r.Simulation.Trials,
				Seed:                 r.Simulation.Seed,
				FailureProbability:   r.Simulation.FailureProbability,
				SamplingMode:         "",
				EndpointAvailability: r.AvailabilityMap(),
				WeightedAggregate:    firstNonZero(r.Summary.WeightedOverallAvailability, r.Summary.OverallAvailability),
				UnweightedAggregate:  r.Summary.OverallAvailability,
			},
			EndpointResults: slices.Clone(r.EndpointResults),
			Decision:        r.PolicyEvaluation.Decision,
		},
	}
}

func Compare(current Report, reference Report, name string) Diff {
	currentProfiles := current.NormalizedProfiles()
	referenceProfiles := reference.NormalizedProfiles()
	refByName := make(map[string]ProfileSummary, len(referenceProfiles))
	for _, profile := range referenceProfiles {
		refByName[profile.Name] = profile
	}

	diff := Diff{
		Name:                     name,
		CurrentDigest:            inputDigest(current.InputArtifact),
		ReferenceDigest:          inputDigest(reference.InputArtifact),
		CurrentTopologyVersion:   inputTopology(current.InputArtifact),
		ReferenceTopologyVersion: inputTopology(reference.InputArtifact),
		CrossProfileWeighted:     delta(current.Summary.CrossProfileWeightedAvailabilityOrFallback(), reference.Summary.CrossProfileWeightedAvailabilityOrFallback()),
		CrossProfileUnweighted:   delta(current.Summary.CrossProfileAvailabilityOrFallback(), reference.Summary.CrossProfileAvailabilityOrFallback()),
		Profiles:                 make([]ProfileDiff, 0, len(currentProfiles)),
	}

	for _, profile := range currentProfiles {
		refProfile, ok := refByName[profile.Name]
		if !ok {
			if len(referenceProfiles) == 1 {
				refProfile = referenceProfiles[0]
			} else {
				refProfile = ProfileSummary{Name: profile.Name}
			}
		}
		currentEndpoints := endpointMap(profile.EndpointResults)
		refEndpoints := endpointMap(refProfile.EndpointResults)
		endpointIDs := make([]string, 0, len(currentEndpoints))
		for endpointID := range currentEndpoints {
			endpointIDs = append(endpointIDs, endpointID)
		}
		slices.Sort(endpointIDs)
		endpoints := make([]EndpointDiff, 0, len(endpointIDs))
		for _, endpointID := range endpointIDs {
			currentResult := currentEndpoints[endpointID]
			refResult := refEndpoints[endpointID]
			endpoints = append(endpoints, EndpointDiff{
				Profile:      profile.Name,
				EndpointID:   endpointID,
				Availability: delta(currentResult.Availability, refResult.Availability),
				Status: StatusDelta{
					Current:   currentResult.Status,
					Reference: refResult.Status,
					Changed:   currentResult.Status != refResult.Status,
				},
			})
		}
		diff.Profiles = append(diff.Profiles, ProfileDiff{
			Profile:             profile.Name,
			WeightedAggregate:   delta(profile.Simulation.WeightedAggregate, refProfile.Simulation.WeightedAggregate),
			UnweightedAggregate: delta(profile.Simulation.UnweightedAggregate, refProfile.Simulation.UnweightedAggregate),
			Decision: StatusDelta{
				Current:   profile.Decision,
				Reference: refProfile.Decision,
				Changed:   profile.Decision != refProfile.Decision,
			},
			Endpoints:       endpoints,
			AdvancedMetrics: compareAdvancedMetrics(profile.Simulation, refProfile.Simulation),
		})
	}
	return diff
}

func (r *Report) SetPreviousDiff(previous *Report) {
	if previous == nil {
		return
	}
	diff := Compare(*r, *previous, "previous")
	r.Diffs.Previous = &diff
}

func (r *Report) SetBaselineDiffs(baselines map[string]Report) {
	if len(baselines) == 0 {
		return
	}
	names := make([]string, 0, len(baselines))
	for name := range baselines {
		names = append(names, name)
	}
	slices.Sort(names)
	r.Diffs.Baselines = make([]Diff, 0, len(names))
	for _, name := range names {
		r.Diffs.Baselines = append(r.Diffs.Baselines, Compare(*r, baselines[name], name))
	}
}

func Load(path string) (Report, error) {
	var rep Report
	raw, err := os.ReadFile(path)
	if err != nil {
		return rep, fmt.Errorf("read report file: %w", err)
	}
	if err := json.Unmarshal(raw, &rep); err != nil {
		return rep, fmt.Errorf("decode report json: %w", err)
	}
	return rep, nil
}

func WriteJSON(path string, payload any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write json file: %w", err)
	}
	return nil
}

func WriteSummaryMarkdown(path string, rep Report) error {
	var b strings.Builder
	b.WriteString("# Sheaft Report Summary\n\n")
	b.WriteString(fmt.Sprintf("- Decision: **%s**\n", rep.PolicyEvaluation.Decision))
	b.WriteString(fmt.Sprintf("- Mode: `%s`\n", rep.PolicyEvaluation.Mode))
	b.WriteString(fmt.Sprintf("- Overall availability: `%.4f`\n", rep.Summary.OverallAvailability))
	if rep.Summary.WeightedOverallAvailability > 0 {
		b.WriteString(fmt.Sprintf("- Weighted overall availability: `%.4f`\n", rep.Summary.WeightedOverallAvailability))
	}
	if rep.Summary.CrossProfileAvailability > 0 {
		b.WriteString(fmt.Sprintf("- Cross-profile availability: `%.4f`\n", rep.Summary.CrossProfileAvailability))
	}
	if rep.Summary.CrossProfileWeightedAvailability > 0 {
		b.WriteString(fmt.Sprintf("- Cross-profile weighted availability: `%.4f`\n", rep.Summary.CrossProfileWeightedAvailability))
	}
	b.WriteString(fmt.Sprintf("- Risk score: `%.4f`\n", rep.Summary.RiskScore))
	b.WriteString(fmt.Sprintf("- Confidence: `%.2f`\n\n", rep.Summary.Confidence))

	if len(rep.Profiles) > 0 {
		b.WriteString("## Profiles\n\n")
		for _, profile := range rep.Profiles {
			b.WriteString(fmt.Sprintf(
				"- `%s`: decision=`%s`, weighted=`%.4f`, unweighted=`%.4f`, below-threshold=`%d`\n",
				profile.Name,
				profile.Decision,
				profile.Simulation.WeightedAggregate,
				profile.Simulation.UnweightedAggregate,
				profile.EndpointsBelowThreshold,
			))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Endpoint results\n\n")
	for _, endpoint := range rep.EndpointResults {
		if endpoint.Profile != "" {
			b.WriteString(fmt.Sprintf(
				"- `%s` / `%s`: availability=`%.4f`, threshold=`%.4f`, status=`%s`\n",
				endpoint.Profile,
				endpoint.EndpointID,
				endpoint.Availability,
				endpoint.Threshold,
				endpoint.Status,
			))
			continue
		}
		b.WriteString(fmt.Sprintf(
			"- `%s`: availability=`%.4f`, threshold=`%.4f`, status=`%s`\n",
			endpoint.EndpointID,
			endpoint.Availability,
			endpoint.Threshold,
			endpoint.Status,
		))
	}

	if rep.Diffs.Previous != nil || len(rep.Diffs.Baselines) > 0 {
		b.WriteString("\n## Diffs\n\n")
		if rep.Diffs.Previous != nil {
			b.WriteString(fmt.Sprintf(
				"- Previous: weighted delta=`%.4f`, unweighted delta=`%.4f`\n",
				rep.Diffs.Previous.CrossProfileWeighted.Signed,
				rep.Diffs.Previous.CrossProfileUnweighted.Signed,
			))
		}
		for _, baseline := range rep.Diffs.Baselines {
			b.WriteString(fmt.Sprintf(
				"- Baseline `%s`: weighted delta=`%.4f`, unweighted delta=`%.4f`\n",
				baseline.Name,
				baseline.CrossProfileWeighted.Signed,
				baseline.CrossProfileUnweighted.Signed,
			))
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create summary dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write summary markdown: %w", err)
	}
	return nil
}

func (s Summary) CrossProfileWeightedAvailabilityOrFallback() float64 {
	return firstNonZero(s.CrossProfileWeightedAvailability, s.WeightedOverallAvailability, s.OverallAvailability)
}

func (s Summary) CrossProfileAvailabilityOrFallback() float64 {
	return firstNonZero(s.CrossProfileAvailability, s.OverallAvailability)
}

func findProfileEvaluation(evals []gate.ProfileEvaluation, name string) gate.ProfileEvaluation {
	for _, eval := range evals {
		if eval.Profile == name {
			return eval
		}
	}
	return gate.ProfileEvaluation{Profile: name}
}

func endpointMap(results []gate.EndpointResult) map[string]gate.EndpointResult {
	out := make(map[string]gate.EndpointResult, len(results))
	for _, result := range results {
		out[result.EndpointID] = result
	}
	return out
}

func delta(current, reference float64) Delta {
	signed := current - reference
	absolute := signed
	if absolute < 0 {
		absolute = -absolute
	}
	return Delta{
		Current:   current,
		Reference: reference,
		Signed:    signed,
		Absolute:  absolute,
	}
}

func inputDigest(meta *InputArtifact) string {
	if meta == nil {
		return ""
	}
	return meta.Digest
}

func inputTopology(meta *InputArtifact) string {
	if meta == nil {
		return ""
	}
	return meta.TopologyVersion
}

func firstNonZero(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func compareAdvancedMetrics(current simulation.ProfileOutput, reference simulation.ProfileOutput) []MetricDiff {
	currentMetrics := collectAdvancedMetrics(current.Advanced)
	referenceMetrics := collectAdvancedMetrics(reference.Advanced)
	if len(currentMetrics) == 0 && len(referenceMetrics) == 0 {
		return nil
	}
	keys := make([]string, 0, len(currentMetrics)+len(referenceMetrics))
	seen := map[string]struct{}{}
	for key := range currentMetrics {
		keys = append(keys, key)
		seen[key] = struct{}{}
	}
	for key := range referenceMetrics {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make([]MetricDiff, 0, len(keys))
	for _, key := range keys {
		currentMetric, currentOK := currentMetrics[key]
		referenceMetric, referenceOK := referenceMetrics[key]
		diff := MetricDiff{
			Metric:     metricNameFromKey(key, 0),
			TargetType: metricNameFromKey(key, 1),
			Target:     metricNameFromKey(key, 2),
		}
		switch {
		case !currentOK || !referenceOK:
			diff.Status = "non_comparable"
			diff.Reason = "metric missing on one side"
		case !currentMetric.available:
			diff.Status = "non_comparable"
			diff.Reason = currentMetric.reason
		case !referenceMetric.available:
			diff.Status = "non_comparable"
			diff.Reason = referenceMetric.reason
		default:
			deltaValue := delta(currentMetric.value, referenceMetric.value)
			diff.Status = "comparable"
			diff.Delta = &deltaValue
		}
		out = append(out, diff)
	}
	return out
}

type numericMetric struct {
	value     float64
	available bool
	reason    string
}

func collectAdvancedMetrics(advanced *simulation.AdvancedProfile) map[string]numericMetric {
	if advanced == nil {
		return nil
	}
	out := map[string]numericMetric{}
	if advanced.BlastRadius != nil {
		out[metricKey("blast_radius_service_count", "profile", "profile")] = numericMetric{
			value:     float64(advanced.BlastRadius.ServiceCount.Value),
			available: advanced.BlastRadius.ServiceCount.Available,
			reason:    advanced.BlastRadius.ServiceCount.Reason,
		}
		out[metricKey("blast_radius_endpoint_count", "profile", "profile")] = numericMetric{
			value:     float64(advanced.BlastRadius.EndpointCount.Value),
			available: advanced.BlastRadius.EndpointCount.Available,
			reason:    advanced.BlastRadius.EndpointCount.Reason,
		}
	}
	for _, path := range advanced.Paths {
		out[metricKey("expected_success_rate", "path", path.PathID)] = numericMetric{
			value:     path.ExpectedSuccessRate.Value,
			available: path.ExpectedSuccessRate.Available,
			reason:    path.ExpectedSuccessRate.Reason,
		}
		out[metricKey("max_amplification_factor", "path", path.PathID)] = numericMetric{
			value:     path.MaxAmplificationFactor.Value,
			available: path.MaxAmplificationFactor.Available,
			reason:    path.MaxAmplificationFactor.Reason,
		}
		out[metricKey("timeout_mismatch_count", "path", path.PathID)] = numericMetric{
			value:     float64(path.TimeoutMismatchCount.Value),
			available: path.TimeoutMismatchCount.Available,
			reason:    path.TimeoutMismatchCount.Reason,
		}
	}
	for _, edge := range advanced.Edges {
		out[metricKey("max_amplification_factor", "edge", edge.EdgeID)] = numericMetric{
			value:     edge.MaxAmplificationFactor.Value,
			available: edge.MaxAmplificationFactor.Available,
			reason:    edge.MaxAmplificationFactor.Reason,
		}
	}
	return out
}

func metricKey(metric, targetType, target string) string {
	return metric + "\x00" + targetType + "\x00" + target
}

func metricNameFromKey(key string, idx int) string {
	parts := strings.Split(key, "\x00")
	if idx < 0 || idx >= len(parts) {
		return ""
	}
	return parts[idx]
}
