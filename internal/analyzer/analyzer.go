package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/faults"
	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/journeys"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
	"github.com/MB3R-Lab/Sheaft/internal/report"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

type Result struct {
	Artifact       artifact.Loaded
	ContractPolicy config.ContractPolicyDecision
	Simulation     simulation.AnalysisOutput
	Evaluation     gate.Evaluation
	Report         report.Report
}

func AnalyzeFile(path string, cfg config.AnalysisConfig, previous *report.Report) (Result, error) {
	loaded, err := artifact.Load(path)
	if err != nil {
		return Result{}, err
	}
	return AnalyzeLoaded(loaded, cfg, previous)
}

func AnalyzeLoaded(loaded artifact.Loaded, cfg config.AnalysisConfig, previous *report.Report) (Result, error) {
	cfg = cfg.Normalized()
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	contractDecision, err := cfg.ContractPolicy.Evaluate(loaded.Metadata.Contract)
	if err != nil {
		return Result{}, fmt.Errorf("contract policy: %w", err)
	}

	started := time.Now()

	overlayPredicates := map[string]predicates.Definition{}
	overlayWeights := map[string]float64{}
	var faultContract *faults.Contract
	if cfg.PredicateContract != "" {
		contract, err := predicates.Load(cfg.PredicateContract)
		if err != nil {
			return Result{}, fmt.Errorf("load predicate contract: %w", err)
		}
		overlayPredicates = contract.Predicates
		overlayWeights = contract.EndpointWeights
		if len(overlayPredicates) > 0 {
			loaded.PredicateSource = artifact.ProvenanceExternal
		}
		if len(overlayWeights) > 0 {
			loaded.WeightsSource = artifact.ProvenanceExternal
		}
	}
	if cfg.FaultContract != "" {
		contract, err := faults.Load(cfg.FaultContract)
		if err != nil {
			return Result{}, fmt.Errorf("load fault contract: %w", err)
		}
		faultContract = &contract
	}

	journeyOverrides := map[string][][]string{}
	if cfg.Journeys != "" {
		var err error
		journeyOverrides, err = journeys.Load(cfg.Journeys)
		if err != nil {
			return Result{}, fmt.Errorf("load journeys: %w", err)
		}
		if err := journeys.ValidateAgainstModel(journeyOverrides, loaded.Model); err != nil {
			return Result{}, fmt.Errorf("validate journeys: %w", err)
		}
	}

	artifactReliability := reliabilityFromArtifact(loaded.Model)
	profiles := make([]simulation.ProfileParams, 0, len(cfg.Profiles))
	for _, profile := range cfg.Profiles {
		reliability, usedArtifactReliability := mergeArtifactReliability(artifactReliability, profile.Reliability)
		if usedArtifactReliability && cfg.Sources.Profiles != nil {
			profileSources := cfg.Sources.Profiles[profile.Name]
			if profileSources.Reliability == config.ParameterSourceDefault {
				profileSources.Reliability = config.ParameterSourceExternal
				cfg.Sources.Profiles[profile.Name] = profileSources
			}
		}
		profiles = append(profiles, simulation.ProfileParams{
			Name:               profile.Name,
			Trials:             profile.Trials,
			SamplingMode:       profile.SamplingMode,
			FailureProbability: profile.FailureProbability,
			Reliability:        reliability,
			FixedKFailures:     profile.FixedKFailures,
			FaultProfile:       profile.FaultProfile,
			EndpointWeights:    profile.EndpointWeights,
		})
	}

	simOut, err := simulation.RunArtifactProfiles(loaded, simulation.AnalysisParams{
		Seed:             cfg.Seed,
		JourneyOverrides: journeyOverrides,
		PredicateSet:     mergePredicates(loaded.Predicates, overlayPredicates),
		DefaultWeights:   mergeWeights(loaded.EndpointWeights, overlayWeights, cfg.EndpointWeights),
		FaultContract:    faultContract,
		Profiles:         profiles,
	})
	if err != nil {
		return Result{}, err
	}

	eval, err := gate.EvaluateProfiles(simOut.Profiles, cfg.Gate)
	if err != nil {
		return Result{}, err
	}

	rep := report.ComposeAnalysis(loaded, simOut, eval, cfg, contractDecision, loaded.Model.Metadata.Confidence, time.Now(), time.Since(started))
	rep.SetPreviousDiff(previous)

	baselines, err := loadBaselines(cfg.Baselines, cfg)
	if err != nil {
		return Result{}, err
	}
	rep.SetBaselineDiffs(baselines)

	return Result{
		Artifact:       loaded,
		ContractPolicy: contractDecision,
		Simulation:     simOut,
		Evaluation:     eval,
		Report:         rep,
	}, nil
}

func loadBaselines(refs []config.BaselineRef, cfg config.AnalysisConfig) (map[string]report.Report, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	out := make(map[string]report.Report, len(refs))
	for _, ref := range refs {
		kind, err := baselineKind(ref.Path)
		if err != nil {
			return nil, fmt.Errorf("load baseline %q: %w", ref.Name, err)
		}
		switch kind {
		case "report":
			rep, err := report.Load(ref.Path)
			if err != nil {
				return nil, fmt.Errorf("load baseline %q: %w", ref.Name, err)
			}
			out[ref.Name] = rep
		case "artifact":
			loaded, err := artifact.Load(ref.Path)
			if err != nil {
				return nil, fmt.Errorf("load baseline artifact %q: %w", ref.Name, err)
			}
			baselineCfg := cfg
			baselineCfg.Baselines = nil
			baselineCfg.ContractPolicy = config.ContractPolicy{}
			result, err := AnalyzeLoaded(loaded, baselineCfg, nil)
			if err != nil {
				return nil, fmt.Errorf("analyze baseline artifact %q: %w", ref.Name, err)
			}
			out[ref.Name] = result.Report
		default:
			return nil, fmt.Errorf("unsupported baseline kind %q", kind)
		}
	}
	return out, nil
}

func reliabilityFromArtifact(mdl model.ResilienceModel) config.ReliabilityConfig {
	out := config.ReliabilityConfig{
		Services: map[string]float64{},
		Edges:    map[string]float64{},
	}
	for _, service := range mdl.Services {
		if service.Metadata == nil || service.Metadata.Reliability == nil || service.Metadata.Reliability.LiveProbability == nil {
			continue
		}
		serviceID := strings.TrimSpace(service.ID)
		if serviceID == "" {
			continue
		}
		out.Services[serviceID] = *service.Metadata.Reliability.LiveProbability
	}
	for _, edge := range mdl.Edges {
		if edge.Metadata == nil || edge.Metadata.Reliability == nil || edge.Metadata.Reliability.LiveProbability == nil {
			continue
		}
		edgeID := strings.TrimSpace(edge.ID)
		if edgeID == "" {
			continue
		}
		out.Edges[edgeID] = *edge.Metadata.Reliability.LiveProbability
	}
	return out
}

func mergeArtifactReliability(artifactDefaults, profile config.ReliabilityConfig) (config.ReliabilityConfig, bool) {
	out := config.ReliabilityConfig{
		NodeLiveProbability: profile.NodeLiveProbability,
		EdgeLiveProbability: profile.EdgeLiveProbability,
		Services:            cloneFloatMap(profile.Services),
		Edges:               cloneFloatMap(profile.Edges),
	}
	if out.Services == nil {
		out.Services = map[string]float64{}
	}
	if out.Edges == nil {
		out.Edges = map[string]float64{}
	}
	usedArtifact := false
	if out.NodeLiveProbability == nil {
		for serviceID, probability := range artifactDefaults.Services {
			if _, ok := out.Services[serviceID]; ok {
				continue
			}
			out.Services[serviceID] = probability
			usedArtifact = true
		}
	}
	if out.EdgeLiveProbability == nil {
		for edgeID, probability := range artifactDefaults.Edges {
			if _, ok := out.Edges[edgeID]; ok {
				continue
			}
			out.Edges[edgeID] = probability
			usedArtifact = true
		}
	}
	return out, usedArtifact
}

func baselineKind(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read baseline file: %w", err)
	}
	var probe struct {
		Simulation       json.RawMessage `json:"simulation"`
		PolicyEvaluation json.RawMessage `json:"policy_evaluation"`
		Metadata         json.RawMessage `json:"metadata"`
		Model            json.RawMessage `json:"model"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return "", fmt.Errorf("decode baseline file: %w", err)
	}
	if len(probe.Simulation) > 0 && len(probe.PolicyEvaluation) > 0 {
		return "report", nil
	}
	if len(probe.Metadata) > 0 || len(probe.Model) > 0 {
		return "artifact", nil
	}
	return "", fmt.Errorf("baseline file is neither a Sheaft report nor a supported artifact")
}

func cloneFloatMap(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func mergePredicates(base map[string]predicates.Definition, overrides map[string]predicates.Definition) map[string]predicates.Definition {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	out := make(map[string]predicates.Definition, len(base)+len(overrides))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overrides {
		out[key] = value
	}
	return out
}

func mergeWeights(weights ...map[string]float64) map[string]float64 {
	size := 0
	for _, weightSet := range weights {
		size += len(weightSet)
	}
	out := make(map[string]float64, size)
	for _, weightSet := range weights {
		for key, value := range weightSet {
			out[key] = value
		}
	}
	return out
}
