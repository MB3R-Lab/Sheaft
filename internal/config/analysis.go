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
	ServeSchemaVersion        = "1.0"
)

const (
	SamplingModeIndependentReplica = "independent_replica"
	SamplingModeIndependentService = "independent_service"
	SamplingModeFixedKServiceSet   = "fixed_k_service_set"
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
	FixedKFailures     int                `json:"fixed_k_failures,omitempty" yaml:"fixed_k_failures,omitempty"`
	Journeys           string             `json:"journeys,omitempty" yaml:"journeys,omitempty"`
	PredicateContract  string             `json:"predicate_contract,omitempty" yaml:"predicate_contract,omitempty"`
	FaultContract      string             `json:"fault_contract,omitempty" yaml:"fault_contract,omitempty"`
	EndpointWeights    map[string]float64 `json:"endpoint_weights,omitempty" yaml:"endpoint_weights,omitempty"`
	Profiles           []Profile          `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Baselines          []BaselineRef      `json:"baselines,omitempty" yaml:"baselines,omitempty"`
	ContractPolicy     ContractPolicy     `json:"contract_policy,omitempty" yaml:"contract_policy,omitempty"`
	Gate               GateConfig         `json:"gate" yaml:"gate"`
	Sources            ParameterSources   `json:"-" yaml:"-"`
}

type Profile struct {
	Name               string             `json:"name" yaml:"name"`
	Trials             int                `json:"trials,omitempty" yaml:"trials,omitempty"`
	SamplingMode       string             `json:"sampling_mode,omitempty" yaml:"sampling_mode,omitempty"`
	FailureProbability float64            `json:"failure_probability,omitempty" yaml:"failure_probability,omitempty"`
	FixedKFailures     int                `json:"fixed_k_failures,omitempty" yaml:"fixed_k_failures,omitempty"`
	FaultProfile       string             `json:"fault_profile,omitempty" yaml:"fault_profile,omitempty"`
	EndpointWeights    map[string]float64 `json:"endpoint_weights,omitempty" yaml:"endpoint_weights,omitempty"`
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
		if out.Profiles[i].FixedKFailures == 0 {
			out.Profiles[i].FixedKFailures = out.FixedKFailures
		}
		if out.Profiles[i].EndpointWeights == nil {
			out.Profiles[i].EndpointWeights = map[string]float64{}
		}
	}
	return out
}

func (c AnalysisConfig) Validate() error {
	if c.SchemaVersion != AnalysisSchemaVersion && c.SchemaVersion != AnalysisSchemaVersionV110 {
		return fmt.Errorf("unsupported analysis schema_version: got %q want one of %q, %q", c.SchemaVersion, AnalysisSchemaVersion, AnalysisSchemaVersionV110)
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
		case SamplingModeIndependentReplica, SamplingModeIndependentService:
			if profile.FailureProbability < 0 || profile.FailureProbability > 1 {
				return fmt.Errorf("profile %q failure_probability must be in range [0,1]", profile.Name)
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
	case SamplingModeIndependentReplica, SamplingModeIndependentService, SamplingModeFixedKServiceSet:
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
