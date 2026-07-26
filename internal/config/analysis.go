package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	AnalysisSchemaVersion     = "1.0"
	AnalysisSchemaVersionV110 = "1.1"
	AnalysisSchemaVersionV120 = "1.2"
	ServeSchemaVersion        = "1.0"
)

const (
	SamplingModeIndependentReplica = "independent_replica"
	SamplingModeIndependentService = "independent_service"
	SamplingModeFixedKServiceSet   = "fixed_k_service_set"
	SamplingModeFixedKReplicaSlots = "fixed_k_replica_slots"
)

const (
	SweepAxisIndependentReplicaFailureProbability = "independent_replica_failure_probability"
	SweepAxisFailedReplicaSlots                   = "failed_replica_slots"
)

type GateEvaluationRule string

const (
	GateEvaluationAllProfiles GateEvaluationRule = "all_profiles"
	GateEvaluationAnyProfile  GateEvaluationRule = "any_profile"
)

type AnalysisConfig struct {
	SchemaVersion      string             `json:"schema_version" yaml:"schema_version"`
	Seed               int64              `json:"seed" yaml:"seed"`
	Trials             int                `json:"trials,omitempty" yaml:"trials,omitempty"`
	SamplingMode       string             `json:"sampling_mode,omitempty" yaml:"sampling_mode,omitempty"`
	FailureProbability float64            `json:"failure_probability,omitempty" yaml:"failure_probability,omitempty"`
	Reliability        ReliabilityConfig  `json:"reliability,omitempty" yaml:"reliability,omitempty"`
	FixedKFailures     int                `json:"fixed_k_failures,omitempty" yaml:"fixed_k_failures,omitempty"`
	Journeys           string             `json:"journeys,omitempty" yaml:"journeys,omitempty"`
	PredicateContract  string             `json:"predicate_contract,omitempty" yaml:"predicate_contract,omitempty"`
	FaultContract      string             `json:"fault_contract,omitempty" yaml:"fault_contract,omitempty"`
	EndpointWeights    map[string]float64 `json:"endpoint_weights,omitempty" yaml:"endpoint_weights,omitempty"`
	Profiles           []Profile          `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Sweeps             []Sweep            `json:"sweeps,omitempty" yaml:"sweeps,omitempty"`
	Baselines          []BaselineRef      `json:"baselines,omitempty" yaml:"baselines,omitempty"`
	ContractPolicy     ContractPolicy     `json:"contract_policy,omitempty" yaml:"contract_policy,omitempty"`
	Gate               GateConfig         `json:"gate" yaml:"gate"`
	Sources            ParameterSources   `json:"-" yaml:"-"`
}

type Sweep struct {
	Name            string        `json:"name" yaml:"name"`
	Profile         string        `json:"profile" yaml:"profile"`
	ConfidenceLevel float64       `json:"confidence_level,omitempty" yaml:"confidence_level,omitempty"`
	Axis            SweepAxis     `json:"axis" yaml:"axis"`
	Targets         []SweepTarget `json:"targets" yaml:"targets"`
}

type SweepAxis struct {
	Type   string    `json:"type" yaml:"type"`
	Values []float64 `json:"values" yaml:"values"`
}

type SweepTarget struct {
	EndpointID string  `json:"endpoint_id" yaml:"endpoint_id"`
	SLO        float64 `json:"slo" yaml:"slo"`
}

type Profile struct {
	Name               string             `json:"name" yaml:"name"`
	Trials             int                `json:"trials,omitempty" yaml:"trials,omitempty"`
	SamplingMode       string             `json:"sampling_mode,omitempty" yaml:"sampling_mode,omitempty"`
	FailureProbability float64            `json:"failure_probability,omitempty" yaml:"failure_probability,omitempty"`
	Reliability        ReliabilityConfig  `json:"reliability,omitempty" yaml:"reliability,omitempty"`
	FixedKFailures     int                `json:"fixed_k_failures,omitempty" yaml:"fixed_k_failures,omitempty"`
	FaultProfile       string             `json:"fault_profile,omitempty" yaml:"fault_profile,omitempty"`
	EndpointWeights    map[string]float64 `json:"endpoint_weights,omitempty" yaml:"endpoint_weights,omitempty"`
}

type ReliabilityConfig struct {
	NodeLiveProbability *float64           `json:"node_live_probability,omitempty" yaml:"node_live_probability,omitempty"`
	EdgeLiveProbability *float64           `json:"edge_live_probability,omitempty" yaml:"edge_live_probability,omitempty"`
	Services            map[string]float64 `json:"services,omitempty" yaml:"services,omitempty"`
	Edges               map[string]float64 `json:"edges,omitempty" yaml:"edges,omitempty"`
}

type BaselineRef struct {
	Name string `json:"name" yaml:"name"`
	Path string `json:"path" yaml:"path"`
}

type GateConfig struct {
	Mode                           PolicyMode                    `json:"mode" yaml:"mode"`
	DefaultAction                  PolicyMode                    `json:"default_action" yaml:"default_action"`
	EvaluationRule                 GateEvaluationRule            `json:"evaluation_rule,omitempty" yaml:"evaluation_rule,omitempty"`
	GlobalThreshold                float64                       `json:"global_threshold,omitempty" yaml:"global_threshold,omitempty"`
	AggregateThreshold             *float64                      `json:"aggregate_threshold,omitempty" yaml:"aggregate_threshold,omitempty"`
	CrossProfileAggregateThreshold *float64                      `json:"cross_profile_aggregate_threshold,omitempty" yaml:"cross_profile_aggregate_threshold,omitempty"`
	EndpointThresholds             map[string]float64            `json:"endpoint_thresholds,omitempty" yaml:"endpoint_thresholds,omitempty"`
	ProfileAggregateThresholds     map[string]float64            `json:"profile_aggregate_thresholds,omitempty" yaml:"profile_aggregate_thresholds,omitempty"`
	ProfileEndpointThresholds      map[string]map[string]float64 `json:"profile_endpoint_thresholds,omitempty" yaml:"profile_endpoint_thresholds,omitempty"`
	BoundaryRules                  []BoundaryRule                `json:"boundary_rules,omitempty" yaml:"boundary_rules,omitempty"`
}

type BoundaryRule struct {
	Sweep                     string     `json:"sweep" yaml:"sweep"`
	EndpointID                string     `json:"endpoint_id" yaml:"endpoint_id"`
	MinimumCertifiedTolerance *float64   `json:"minimum_certified_tolerance,omitempty" yaml:"minimum_certified_tolerance,omitempty"`
	Baseline                  string     `json:"baseline,omitempty" yaml:"baseline,omitempty"`
	MaxRegression             *float64   `json:"max_regression,omitempty" yaml:"max_regression,omitempty"`
	IndeterminateAction       PolicyMode `json:"indeterminate_action,omitempty" yaml:"indeterminate_action,omitempty"`
}

type RuntimeConfig struct {
	Model      string `json:"model" yaml:"model"`
	Journeys   string `json:"journeys" yaml:"journeys"`
	OutputDir  string `json:"output_dir" yaml:"output_dir"`
	Seed       int64  `json:"seed" yaml:"seed"`
	Simulation struct {
		Trials             int     `json:"trials" yaml:"trials"`
		FailureProbability float64 `json:"failure_probability" yaml:"failure_probability"`
	} `json:"simulation" yaml:"simulation"`
	Policy struct {
		File string `json:"file" yaml:"file"`
	} `json:"policy" yaml:"policy"`
}

type ServeConfig struct {
	SchemaVersion string         `json:"schema_version" yaml:"schema_version"`
	Listen        string         `json:"listen" yaml:"listen"`
	Artifact      ArtifactSource `json:"artifact" yaml:"artifact"`
	AnalysisFile  string         `json:"analysis_file,omitempty" yaml:"analysis_file,omitempty"`
	Analysis      AnalysisConfig `json:"analysis,omitempty" yaml:"analysis,omitempty"`
	PollInterval  string         `json:"poll_interval,omitempty" yaml:"poll_interval,omitempty"`
	WatchFS       *bool          `json:"watch_fs,omitempty" yaml:"watch_fs,omitempty"`
	WatchPolling  *bool          `json:"watch_polling,omitempty" yaml:"watch_polling,omitempty"`
	History       HistoryConfig  `json:"history,omitempty" yaml:"history,omitempty"`
}

type ArtifactSource struct {
	Path     string   `json:"path" yaml:"path"`
	Mode     string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Patterns []string `json:"patterns,omitempty" yaml:"patterns,omitempty"`
}

type HistoryConfig struct {
	MaxItems int    `json:"max_items,omitempty" yaml:"max_items,omitempty"`
	DiskDir  string `json:"disk_dir,omitempty" yaml:"disk_dir,omitempty"`
}

func (p Policy) ToAnalysisConfig() AnalysisConfig {
	cfg := AnalysisConfig{
		SchemaVersion:      AnalysisSchemaVersion,
		Seed:               42,
		Trials:             p.Trials,
		SamplingMode:       SamplingModeIndependentReplica,
		FailureProbability: p.FailureProbability,
		Profiles: []Profile{
			{
				Name:               "default",
				Trials:             p.Trials,
				SamplingMode:       SamplingModeIndependentReplica,
				FailureProbability: p.FailureProbability,
			},
		},
		Gate: GateConfig{
			Mode:               p.Mode,
			DefaultAction:      p.DefaultAction,
			EvaluationRule:     GateEvaluationAllProfiles,
			GlobalThreshold:    p.GlobalThreshold,
			EndpointThresholds: cloneFloatMap(p.EndpointThresholds),
		},
	}
	cfg.Sources = BuildPolicyParameterSources(cfg)
	return cfg
}

func (c AnalysisConfig) Normalized() AnalysisConfig {
	out := c
	if out.SchemaVersion == "" {
		out.SchemaVersion = AnalysisSchemaVersion
	}
	if out.Seed == 0 {
		out.Seed = 42
	}
	if out.Trials <= 0 {
		out.Trials = 10000
	}
	if out.SamplingMode == "" {
		out.SamplingMode = SamplingModeIndependentReplica
	}
	if out.FailureProbability == 0 {
		out.FailureProbability = 0.05
	}
	if out.EndpointWeights == nil {
		out.EndpointWeights = map[string]float64{}
	}
	out.ContractPolicy = out.ContractPolicy.Normalized()
	if out.Gate.Mode == "" {
		out.Gate.Mode = ModeWarn
	}
	if out.Gate.DefaultAction == "" {
		out.Gate.DefaultAction = out.Gate.Mode
	}
	if out.Gate.EvaluationRule == "" {
		out.Gate.EvaluationRule = GateEvaluationAllProfiles
	}
	if out.Gate.GlobalThreshold == 0 {
		out.Gate.GlobalThreshold = 0.99
	}
	if out.Gate.EndpointThresholds == nil {
		out.Gate.EndpointThresholds = map[string]float64{}
	}
	if out.Gate.ProfileAggregateThresholds == nil {
		out.Gate.ProfileAggregateThresholds = map[string]float64{}
	}
	if out.Gate.ProfileEndpointThresholds == nil {
		out.Gate.ProfileEndpointThresholds = map[string]map[string]float64{}
	}
	if len(out.Profiles) == 0 {
		out.Profiles = []Profile{
			{
				Name:               "default",
				Trials:             out.Trials,
				SamplingMode:       out.SamplingMode,
				FailureProbability: out.FailureProbability,
				Reliability:        normalizeProfileReliability(ReliabilityConfig{}, out.Reliability),
				FixedKFailures:     out.FixedKFailures,
				EndpointWeights:    cloneFloatMap(out.EndpointWeights),
			},
		}
	}
	for i := range out.Profiles {
		if out.Profiles[i].Trials <= 0 {
			out.Profiles[i].Trials = out.Trials
		}
		if out.Profiles[i].SamplingMode == "" {
			out.Profiles[i].SamplingMode = out.SamplingMode
		}
		if out.Profiles[i].FailureProbability == 0 {
			out.Profiles[i].FailureProbability = out.FailureProbability
		}
		out.Profiles[i].Reliability = normalizeProfileReliability(out.Reliability, out.Profiles[i].Reliability)
		if out.Profiles[i].FixedKFailures == 0 {
			out.Profiles[i].FixedKFailures = out.FixedKFailures
		}
		if out.Profiles[i].EndpointWeights == nil {
			out.Profiles[i].EndpointWeights = map[string]float64{}
		}
	}
	for i := range out.Sweeps {
		if out.Sweeps[i].ConfidenceLevel == 0 {
			out.Sweeps[i].ConfidenceLevel = 0.95
		}
	}
	return out
}

func (c AnalysisConfig) Validate() error {
	if c.SchemaVersion != AnalysisSchemaVersion && c.SchemaVersion != AnalysisSchemaVersionV110 && c.SchemaVersion != AnalysisSchemaVersionV120 {
		return fmt.Errorf("unsupported analysis schema_version: got %q want one of %q, %q, %q", c.SchemaVersion, AnalysisSchemaVersion, AnalysisSchemaVersionV110, AnalysisSchemaVersionV120)
	}
	if len(c.Profiles) == 0 {
		return errors.New("analysis requires at least one profile")
	}
	if c.SchemaVersion == AnalysisSchemaVersion && strings.TrimSpace(c.FaultContract) != "" {
		return fmt.Errorf("analysis schema_version %q does not support fault_contract; use %q", AnalysisSchemaVersion, AnalysisSchemaVersionV110)
	}
	profileNames := make(map[string]struct{}, len(c.Profiles))
	for _, baseline := range c.Baselines {
		if strings.TrimSpace(baseline.Name) == "" {
			return errors.New("baseline name cannot be empty")
		}
		if strings.TrimSpace(baseline.Path) == "" {
			return fmt.Errorf("baseline %q path cannot be empty", baseline.Name)
		}
	}
	for name, weight := range c.EndpointWeights {
		if strings.TrimSpace(name) == "" {
			return errors.New("endpoint_weights key cannot be empty")
		}
		if weight < 0 {
			return fmt.Errorf("endpoint_weights[%s] must be >= 0", name)
		}
	}
	if err := validateReliability("analysis.reliability", c.Reliability); err != nil {
		return err
	}
	if err := c.ContractPolicy.Validate(); err != nil {
		return err
	}
	for _, profile := range c.Profiles {
		if strings.TrimSpace(profile.Name) == "" {
			return errors.New("profile name cannot be empty")
		}
		if c.SchemaVersion == AnalysisSchemaVersion && strings.TrimSpace(profile.FaultProfile) != "" {
			return fmt.Errorf("profile %q uses fault_profile but analysis schema_version %q does not support it; use %q", profile.Name, AnalysisSchemaVersion, AnalysisSchemaVersionV110)
		}
		if _, exists := profileNames[profile.Name]; exists {
			return fmt.Errorf("duplicate profile name: %s", profile.Name)
		}
		profileNames[profile.Name] = struct{}{}
		if profile.Trials <= 0 {
			return fmt.Errorf("profile %q trials must be > 0", profile.Name)
		}
		if !isValidSamplingMode(profile.SamplingMode) {
			return fmt.Errorf("profile %q has unsupported sampling_mode %q", profile.Name, profile.SamplingMode)
		}
		switch profile.SamplingMode {
		case SamplingModeIndependentReplica, SamplingModeIndependentService, SamplingModeFixedKReplicaSlots:
			if profile.FailureProbability < 0 || profile.FailureProbability > 1 {
				return fmt.Errorf("profile %q failure_probability must be in range [0,1]", profile.Name)
			}
			if profile.SamplingMode == SamplingModeFixedKReplicaSlots && profile.FixedKFailures < 0 {
				return fmt.Errorf("profile %q fixed_k_failures must be >= 0", profile.Name)
			}
		case SamplingModeFixedKServiceSet:
			if profile.FixedKFailures < 0 {
				return fmt.Errorf("profile %q fixed_k_failures must be >= 0", profile.Name)
			}
		}
		for endpoint, weight := range profile.EndpointWeights {
			if strings.TrimSpace(endpoint) == "" {
				return fmt.Errorf("profile %q has empty endpoint_weights key", profile.Name)
			}
			if weight < 0 {
				return fmt.Errorf("profile %q endpoint_weights[%s] must be >= 0", profile.Name, endpoint)
			}
		}
		if err := validateReliability(fmt.Sprintf("profile %q reliability", profile.Name), profile.Reliability); err != nil {
			return err
		}
	}
	if len(c.Sweeps) > 0 && c.SchemaVersion != AnalysisSchemaVersionV120 {
		return fmt.Errorf("analysis schema_version %q does not support sweeps; use %q", c.SchemaVersion, AnalysisSchemaVersionV120)
	}
	sweepNames := make(map[string]struct{}, len(c.Sweeps))
	for _, sweep := range c.Sweeps {
		if err := validateSweep(sweep, profileNames, sweepNames); err != nil {
			return err
		}
		sweepNames[sweep.Name] = struct{}{}
	}
	baselineNames := make(map[string]struct{}, len(c.Baselines))
	for _, baseline := range c.Baselines {
		baselineNames[baseline.Name] = struct{}{}
	}
	for idx, rule := range c.Gate.BoundaryRules {
		if err := validateBoundaryRule(idx, rule, c.Sweeps, baselineNames); err != nil {
			return err
		}
	}
	if !isValidPolicyMode(c.Gate.Mode) {
		return fmt.Errorf("unsupported gate mode: %q", c.Gate.Mode)
	}
	if !isValidPolicyMode(c.Gate.DefaultAction) {
		return fmt.Errorf("unsupported gate default_action: %q", c.Gate.DefaultAction)
	}
	if c.Gate.EvaluationRule != GateEvaluationAllProfiles && c.Gate.EvaluationRule != GateEvaluationAnyProfile {
		return fmt.Errorf("unsupported gate evaluation_rule: %q", c.Gate.EvaluationRule)
	}
	if c.Gate.GlobalThreshold < 0 || c.Gate.GlobalThreshold > 1 {
		return errors.New("gate.global_threshold must be in range [0,1]")
	}
	if c.Gate.AggregateThreshold != nil && (*c.Gate.AggregateThreshold < 0 || *c.Gate.AggregateThreshold > 1) {
		return errors.New("gate.aggregate_threshold must be in range [0,1]")
	}
	if c.Gate.CrossProfileAggregateThreshold != nil && (*c.Gate.CrossProfileAggregateThreshold < 0 || *c.Gate.CrossProfileAggregateThreshold > 1) {
		return errors.New("gate.cross_profile_aggregate_threshold must be in range [0,1]")
	}
	for endpoint, threshold := range c.Gate.EndpointThresholds {
		if strings.TrimSpace(endpoint) == "" {
			return errors.New("gate.endpoint_thresholds key cannot be empty")
		}
		if threshold < 0 || threshold > 1 {
			return fmt.Errorf("gate.endpoint_thresholds[%s] must be in range [0,1]", endpoint)
		}
	}
	for profile, threshold := range c.Gate.ProfileAggregateThresholds {
		if strings.TrimSpace(profile) == "" {
			return errors.New("gate.profile_aggregate_thresholds key cannot be empty")
		}
		if threshold < 0 || threshold > 1 {
			return fmt.Errorf("gate.profile_aggregate_thresholds[%s] must be in range [0,1]", profile)
		}
	}
	for profile, thresholds := range c.Gate.ProfileEndpointThresholds {
		if strings.TrimSpace(profile) == "" {
			return errors.New("gate.profile_endpoint_thresholds key cannot be empty")
		}
		for endpoint, threshold := range thresholds {
			if strings.TrimSpace(endpoint) == "" {
				return fmt.Errorf("gate.profile_endpoint_thresholds[%s] key cannot be empty", profile)
			}
			if threshold < 0 || threshold > 1 {
				return fmt.Errorf("gate.profile_endpoint_thresholds[%s][%s] must be in range [0,1]", profile, endpoint)
			}
		}
	}
	return nil
}

func validateSweep(sweep Sweep, profileNames, sweepNames map[string]struct{}) error {
	if strings.TrimSpace(sweep.Name) == "" {
		return errors.New("sweep name cannot be empty")
	}
	if _, exists := sweepNames[sweep.Name]; exists {
		return fmt.Errorf("duplicate sweep name: %s", sweep.Name)
	}
	if _, exists := profileNames[sweep.Profile]; !exists {
		return fmt.Errorf("sweep %q references unknown profile %q", sweep.Name, sweep.Profile)
	}
	if sweep.ConfidenceLevel <= 0 || sweep.ConfidenceLevel >= 1 {
		return fmt.Errorf("sweep %q confidence_level must be in range (0,1)", sweep.Name)
	}
	if len(sweep.Axis.Values) == 0 {
		return fmt.Errorf("sweep %q axis.values must contain at least one value", sweep.Name)
	}
	for idx, value := range sweep.Axis.Values {
		if idx > 0 && value <= sweep.Axis.Values[idx-1] {
			return fmt.Errorf("sweep %q axis.values must be strictly increasing", sweep.Name)
		}
		switch sweep.Axis.Type {
		case SweepAxisIndependentReplicaFailureProbability:
			if value < 0 || value > 1 {
				return fmt.Errorf("sweep %q axis.values[%d] must be in range [0,1]", sweep.Name, idx)
			}
		case SweepAxisFailedReplicaSlots:
			if value < 0 || value != float64(int(value)) {
				return fmt.Errorf("sweep %q axis.values[%d] must be a non-negative integer", sweep.Name, idx)
			}
		default:
			return fmt.Errorf("sweep %q has unsupported axis type %q", sweep.Name, sweep.Axis.Type)
		}
	}
	if len(sweep.Targets) == 0 {
		return fmt.Errorf("sweep %q requires at least one target", sweep.Name)
	}
	targets := make(map[string]struct{}, len(sweep.Targets))
	for _, target := range sweep.Targets {
		if strings.TrimSpace(target.EndpointID) == "" {
			return fmt.Errorf("sweep %q target endpoint_id cannot be empty", sweep.Name)
		}
		if _, exists := targets[target.EndpointID]; exists {
			return fmt.Errorf("sweep %q has duplicate target %q", sweep.Name, target.EndpointID)
		}
		targets[target.EndpointID] = struct{}{}
		if target.SLO < 0 || target.SLO > 1 {
			return fmt.Errorf("sweep %q target %q slo must be in range [0,1]", sweep.Name, target.EndpointID)
		}
	}
	return nil
}

func validateBoundaryRule(index int, rule BoundaryRule, sweeps []Sweep, baselines map[string]struct{}) error {
	if strings.TrimSpace(rule.Sweep) == "" || strings.TrimSpace(rule.EndpointID) == "" {
		return fmt.Errorf("gate.boundary_rules[%d] requires sweep and endpoint_id", index)
	}
	var matched *Sweep
	for idx := range sweeps {
		if sweeps[idx].Name == rule.Sweep {
			matched = &sweeps[idx]
			break
		}
	}
	if matched == nil {
		return fmt.Errorf("gate.boundary_rules[%d] references unknown sweep %q", index, rule.Sweep)
	}
	targetFound := false
	for _, target := range matched.Targets {
		if target.EndpointID == rule.EndpointID {
			targetFound = true
			break
		}
	}
	if !targetFound {
		return fmt.Errorf("gate.boundary_rules[%d] references endpoint %q outside sweep %q targets", index, rule.EndpointID, rule.Sweep)
	}
	if rule.MinimumCertifiedTolerance == nil && rule.MaxRegression == nil {
		return fmt.Errorf("gate.boundary_rules[%d] requires minimum_certified_tolerance or max_regression", index)
	}
	if rule.MinimumCertifiedTolerance != nil {
		found := false
		for _, value := range matched.Axis.Values {
			if value == *rule.MinimumCertifiedTolerance {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("gate.boundary_rules[%d] minimum_certified_tolerance must equal an evaluated axis value", index)
		}
	}
	if rule.MaxRegression != nil {
		if *rule.MaxRegression < 0 {
			return fmt.Errorf("gate.boundary_rules[%d] max_regression must be >= 0", index)
		}
		if strings.TrimSpace(rule.Baseline) == "" {
			return fmt.Errorf("gate.boundary_rules[%d] max_regression requires baseline", index)
		}
		if _, exists := baselines[rule.Baseline]; !exists {
			return fmt.Errorf("gate.boundary_rules[%d] references unknown baseline %q", index, rule.Baseline)
		}
	}
	if rule.IndeterminateAction != "" && !isValidPolicyMode(rule.IndeterminateAction) {
		return fmt.Errorf("gate.boundary_rules[%d] has unsupported indeterminate_action %q", index, rule.IndeterminateAction)
	}
	return nil
}

func LoadAnalysis(path string) (AnalysisConfig, error) {
	var cfg AnalysisConfig
	if err := loadStructuredFile(path, &cfg); err != nil {
		return AnalysisConfig{}, err
	}
	if len(cfg.Profiles) == 0 && cfg.SchemaVersion == "" {
		policy, err := LoadPolicy(path)
		if err != nil {
			return AnalysisConfig{}, err
		}
		return policy.ToAnalysisConfig().Normalized(), nil
	}
	raw := cfg
	cfg = cfg.Normalized()
	cfg.Sources = BuildAnalysisParameterSources(raw, cfg)
	if err := cfg.Validate(); err != nil {
		return AnalysisConfig{}, fmt.Errorf("validate analysis config: %w", err)
	}
	return cfg, nil
}

func LoadRuntimeConfig(path string) (RuntimeConfig, error) {
	var cfg RuntimeConfig
	if err := loadStructuredFile(path, &cfg); err != nil {
		return RuntimeConfig{}, err
	}
	return cfg, nil
}

func LoadServeConfig(path string) (ServeConfig, error) {
	var cfg ServeConfig
	if err := loadStructuredFile(path, &cfg); err != nil {
		return ServeConfig{}, err
	}
	cfg = cfg.Normalized()
	cfg.ResolveRelativePaths(filepath.Dir(path))
	if err := cfg.Validate(); err != nil {
		return ServeConfig{}, fmt.Errorf("validate serve config: %w", err)
	}
	return cfg, nil
}

func (c ServeConfig) Normalized() ServeConfig {
	out := c
	if out.SchemaVersion == "" {
		out.SchemaVersion = ServeSchemaVersion
	}
	if strings.TrimSpace(out.Listen) == "" {
		out.Listen = ":8080"
	}
	if strings.TrimSpace(out.Artifact.Mode) == "" {
		out.Artifact.Mode = "auto"
	}
	if len(out.Artifact.Patterns) == 0 {
		out.Artifact.Patterns = []string{"*.json"}
	}
	if strings.TrimSpace(out.PollInterval) == "" {
		out.PollInterval = "30s"
	}
	if out.History.MaxItems <= 0 {
		out.History.MaxItems = 10
	}
	return out
}

func (c ServeConfig) Validate() error {
	if c.SchemaVersion != ServeSchemaVersion {
		return fmt.Errorf("unsupported serve schema_version: got %q want %q", c.SchemaVersion, ServeSchemaVersion)
	}
	if strings.TrimSpace(c.Artifact.Path) == "" {
		return errors.New("artifact.path cannot be empty")
	}
	switch c.Artifact.Mode {
	case "auto", "file", "directory":
	default:
		return fmt.Errorf("unsupported artifact.mode: %q", c.Artifact.Mode)
	}
	if _, err := c.PollDuration(); err != nil {
		return err
	}
	if c.History.MaxItems <= 0 {
		return errors.New("history.max_items must be > 0")
	}
	if strings.TrimSpace(c.AnalysisFile) == "" && len(c.Analysis.Profiles) == 0 && c.Analysis.SchemaVersion == "" {
		return errors.New("serve config requires analysis_file or inline analysis")
	}
	if strings.TrimSpace(c.AnalysisFile) == "" {
		analysis := c.Analysis.Normalized()
		analysis.Sources = BuildAnalysisParameterSources(c.Analysis, analysis)
		if err := analysis.Validate(); err != nil {
			return fmt.Errorf("inline analysis: %w", err)
		}
	}
	return nil
}

func (c *ServeConfig) ResolveRelativePaths(baseDir string) {
	c.Artifact.Path = resolveRelative(baseDir, c.Artifact.Path)
	c.AnalysisFile = resolveRelative(baseDir, c.AnalysisFile)
	c.History.DiskDir = resolveRelative(baseDir, c.History.DiskDir)
	c.Analysis.ResolveRelativePaths(baseDir)
}

func (c *AnalysisConfig) ResolveRelativePaths(baseDir string) {
	c.Journeys = resolveRelative(baseDir, c.Journeys)
	c.PredicateContract = resolveRelative(baseDir, c.PredicateContract)
	c.FaultContract = resolveRelative(baseDir, c.FaultContract)
	for i := range c.Baselines {
		c.Baselines[i].Path = resolveRelative(baseDir, c.Baselines[i].Path)
	}
}

func (c ServeConfig) PollDuration() (time.Duration, error) {
	d, err := time.ParseDuration(c.PollInterval)
	if err != nil {
		return 0, fmt.Errorf("invalid poll_interval %q: %w", c.PollInterval, err)
	}
	if d <= 0 {
		return 0, errors.New("poll_interval must be > 0")
	}
	return d, nil
}

func resolveRelative(baseDir, target string) string {
	if strings.TrimSpace(target) == "" {
		return ""
	}
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Clean(filepath.Join(baseDir, target))
}

func isValidSamplingMode(mode string) bool {
	switch mode {
	case SamplingModeIndependentReplica, SamplingModeIndependentService, SamplingModeFixedKServiceSet, SamplingModeFixedKReplicaSlots:
		return true
	default:
		return false
	}
}

func cloneFloatMap(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return map[string]float64{}
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make(map[string]float64, len(in))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}

func normalizeProfileReliability(parent, profile ReliabilityConfig) ReliabilityConfig {
	out := ReliabilityConfig{
		NodeLiveProbability: profile.NodeLiveProbability,
		EdgeLiveProbability: profile.EdgeLiveProbability,
		Services:            mergeFloatMaps(parent.Services, profile.Services),
		Edges:               mergeFloatMaps(parent.Edges, profile.Edges),
	}
	if out.NodeLiveProbability == nil {
		out.NodeLiveProbability = parent.NodeLiveProbability
	}
	if out.EdgeLiveProbability == nil {
		out.EdgeLiveProbability = parent.EdgeLiveProbability
	}
	if out.Services == nil {
		out.Services = map[string]float64{}
	}
	if out.Edges == nil {
		out.Edges = map[string]float64{}
	}
	return out
}

func validateReliability(label string, value ReliabilityConfig) error {
	for _, item := range []struct {
		name  string
		value *float64
	}{
		{"node_live_probability", value.NodeLiveProbability},
		{"edge_live_probability", value.EdgeLiveProbability},
	} {
		if item.value != nil && (*item.value < 0 || *item.value > 1) {
			return fmt.Errorf("%s.%s must be in range [0,1]", label, item.name)
		}
	}
	for service, probability := range value.Services {
		if strings.TrimSpace(service) == "" {
			return fmt.Errorf("%s.services key cannot be empty", label)
		}
		if probability < 0 || probability > 1 {
			return fmt.Errorf("%s.services[%s] must be in range [0,1]", label, service)
		}
	}
	for edge, probability := range value.Edges {
		if strings.TrimSpace(edge) == "" {
			return fmt.Errorf("%s.edges key cannot be empty", label)
		}
		if probability < 0 || probability > 1 {
			return fmt.Errorf("%s.edges[%s] must be in range [0,1]", label, edge)
		}
	}
	return nil
}

func mergeFloatMaps(parent, override map[string]float64) map[string]float64 {
	if len(parent) == 0 && len(override) == 0 {
		return map[string]float64{}
	}
	out := cloneFloatMap(parent)
	for key, value := range override {
		out[key] = value
	}
	return out
}
