package analyzer

import (
	"fmt"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/journeys"
	"github.com/MB3R-Lab/Sheaft/internal/predicates"
	"github.com/MB3R-Lab/Sheaft/internal/report"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

type Result struct {
	Artifact        artifact.Loaded
	ContractPolicy  config.ContractPolicyDecision
	Simulation      simulation.AnalysisOutput
	Evaluation      gate.Evaluation
	Report          report.Report
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

	profiles := make([]simulation.ProfileParams, 0, len(cfg.Profiles))
	for _, profile := range cfg.Profiles {
		profiles = append(profiles, simulation.ProfileParams{
			Name:               profile.Name,
			Trials:             profile.Trials,
			SamplingMode:       profile.SamplingMode,
			FailureProbability: profile.FailureProbability,
			FixedKFailures:     profile.FixedKFailures,
			EndpointWeights:    profile.EndpointWeights,
		})
	}

	simOut, err := simulation.RunProfiles(loaded.Model, simulation.AnalysisParams{
		Seed:             cfg.Seed,
		JourneyOverrides: journeyOverrides,
		PredicateSet:     mergePredicates(loaded.Predicates, overlayPredicates),
		DefaultWeights:   mergeWeights(loaded.EndpointWeights, overlayWeights, cfg.EndpointWeights),
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

	baselines, err := loadBaselines(cfg.Baselines)
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

func loadBaselines(refs []config.BaselineRef) (map[string]report.Report, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	out := make(map[string]report.Report, len(refs))
	for _, ref := range refs {
		rep, err := report.Load(ref.Path)
		if err != nil {
			return nil, fmt.Errorf("load baseline %q: %w", ref.Name, err)
		}
		out[ref.Name] = rep
	}
	return out, nil
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
