package config

type ParameterSource string

const (
	ParameterSourceDefault  ParameterSource = "default"
	ParameterSourcePolicy   ParameterSource = "policy"
	ParameterSourceOverride ParameterSource = "override"
	ParameterSourceExternal ParameterSource = "external"
)

type ParameterSources struct {
	ConfigSource       ParameterSource                 `json:"-" yaml:"-"`
	Seed               ParameterSource                 `json:"-" yaml:"-"`
	Trials             ParameterSource                 `json:"-" yaml:"-"`
	SamplingMode       ParameterSource                 `json:"-" yaml:"-"`
	FailureProbability ParameterSource                 `json:"-" yaml:"-"`
	FixedKFailures     ParameterSource                 `json:"-" yaml:"-"`
	EndpointWeights    ParameterSource                 `json:"-" yaml:"-"`
	Journeys           ParameterSource                 `json:"-" yaml:"-"`
	PredicateContract  ParameterSource                 `json:"-" yaml:"-"`
	Baselines          ParameterSource                 `json:"-" yaml:"-"`
	Profiles           map[string]ProfileParameterSources `json:"-" yaml:"-"`
}

type ProfileParameterSources struct {
	Trials             ParameterSource `json:"-" yaml:"-"`
	SamplingMode       ParameterSource `json:"-" yaml:"-"`
	FailureProbability ParameterSource `json:"-" yaml:"-"`
	FixedKFailures     ParameterSource `json:"-" yaml:"-"`
	EndpointWeights    ParameterSource `json:"-" yaml:"-"`
}

func BuildAnalysisParameterSources(raw, normalized AnalysisConfig) ParameterSources {
	sources := ParameterSources{
		ConfigSource:       ParameterSourceOverride,
		Seed:               pickSource(raw.Seed != 0, ParameterSourceOverride, ParameterSourceDefault),
		Trials:             pickSource(raw.Trials > 0, ParameterSourceOverride, ParameterSourceDefault),
		SamplingMode:       pickSource(raw.SamplingMode != "", ParameterSourceOverride, ParameterSourceDefault),
		FailureProbability: pickSource(raw.FailureProbability != 0, ParameterSourceOverride, ParameterSourceDefault),
		FixedKFailures:     pickSource(raw.FixedKFailures != 0, ParameterSourceOverride, ParameterSourceDefault),
		EndpointWeights:    pickSource(len(raw.EndpointWeights) > 0, ParameterSourceOverride, ParameterSourceDefault),
		Journeys:           pickSource(raw.Journeys != "", ParameterSourceOverride, ParameterSourceDefault),
		PredicateContract:  pickSource(raw.PredicateContract != "", ParameterSourceExternal, ParameterSourceDefault),
		Baselines:          pickSource(len(raw.Baselines) > 0, ParameterSourceExternal, ParameterSourceDefault),
		Profiles:           make(map[string]ProfileParameterSources, len(normalized.Profiles)),
	}

	rawProfiles := make(map[string]Profile, len(raw.Profiles))
	for _, profile := range raw.Profiles {
		rawProfiles[profile.Name] = profile
	}

	for _, profile := range normalized.Profiles {
		rawProfile, ok := rawProfiles[profile.Name]
		if !ok {
			rawProfile = Profile{}
		}
		sources.Profiles[profile.Name] = ProfileParameterSources{
			Trials:             inheritedSource(rawProfile.Trials > 0, sources.Trials),
			SamplingMode:       inheritedSource(rawProfile.SamplingMode != "", sources.SamplingMode),
			FailureProbability: inheritedSource(rawProfile.FailureProbability != 0, sources.FailureProbability),
			FixedKFailures:     inheritedSource(rawProfile.FixedKFailures != 0, sources.FixedKFailures),
			EndpointWeights:    inheritedSource(len(rawProfile.EndpointWeights) > 0, sources.EndpointWeights),
		}
	}

	return sources
}

func BuildPolicyParameterSources(normalized AnalysisConfig) ParameterSources {
	sources := ParameterSources{
		ConfigSource:       ParameterSourcePolicy,
		Seed:               ParameterSourcePolicy,
		Trials:             ParameterSourcePolicy,
		SamplingMode:       ParameterSourcePolicy,
		FailureProbability: ParameterSourcePolicy,
		FixedKFailures:     ParameterSourceDefault,
		EndpointWeights:    ParameterSourceDefault,
		Journeys:           ParameterSourceDefault,
		PredicateContract:  ParameterSourceDefault,
		Baselines:          ParameterSourceDefault,
		Profiles:           make(map[string]ProfileParameterSources, len(normalized.Profiles)),
	}

	for _, profile := range normalized.Profiles {
		sources.Profiles[profile.Name] = ProfileParameterSources{
			Trials:             ParameterSourcePolicy,
			SamplingMode:       ParameterSourcePolicy,
			FailureProbability: ParameterSourcePolicy,
			FixedKFailures:     ParameterSourceDefault,
			EndpointWeights:    ParameterSourceDefault,
		}
	}

	return sources
}

func pickSource(active bool, activeSource, inactiveSource ParameterSource) ParameterSource {
	if active {
		return activeSource
	}
	return inactiveSource
}

func inheritedSource(active bool, fallback ParameterSource) ParameterSource {
	if active {
		return ParameterSourceOverride
	}
	return fallback
}
