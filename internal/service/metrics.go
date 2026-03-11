package service

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/MB3R-Lab/Sheaft/internal/report"
)

type Metrics struct {
	recomputesTotal                     *prometheus.CounterVec
	recomputeDurationSeconds            prometheus.Histogram
	currentModelAgeSeconds              prometheus.Gauge
	currentProfileAggregateAvailability *prometheus.GaugeVec
	currentEndpointAvailability         *prometheus.GaugeVec
	endpointsBelowThreshold             *prometheus.GaugeVec
	activeModelInfo                     *prometheus.GaugeVec
	activeTopologyVersion               *prometheus.GaugeVec
	previousGap                         *prometheus.GaugeVec
	previousGapAbsolute                 *prometheus.GaugeVec
	baselineGap                         *prometheus.GaugeVec
	baselineGapAbsolute                 *prometheus.GaugeVec
}

func newMetrics(registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		recomputesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "recomputes_total",
			Help: "Total number of posture recomputes by outcome.",
		}, []string{"outcome"}),
		recomputeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "recompute_duration_seconds",
			Help:    "Duration of posture recomputes.",
			Buckets: prometheus.DefBuckets,
		}),
		currentModelAgeSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "current_model_age_seconds",
			Help: "Age of the currently active model artifact in seconds.",
		}),
		currentProfileAggregateAvailability: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "current_profile_aggregate_availability",
			Help: "Current aggregate availability by profile and aggregate type.",
		}, []string{"profile", "aggregate"}),
		currentEndpointAvailability: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "current_endpoint_availability",
			Help: "Current endpoint availability by profile and endpoint.",
		}, []string{"profile", "endpoint"}),
		endpointsBelowThreshold: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "endpoints_below_threshold",
			Help: "Number of endpoints below threshold by profile.",
		}, []string{"profile"}),
		activeModelInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "active_model_info",
			Help: "Info metric for the active model artifact.",
		}, []string{"digest", "artifact_kind", "contract_name", "contract_version", "source_type"}),
		activeTopologyVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "active_topology_version",
			Help: "Info metric for the active topology version.",
		}, []string{"topology_version"}),
		previousGap: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "previous_gap",
			Help: "Signed gap versus the previous report.",
		}, []string{"profile", "endpoint", "kind"}),
		previousGapAbsolute: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "previous_gap_absolute",
			Help: "Absolute gap versus the previous report.",
		}, []string{"profile", "endpoint", "kind"}),
		baselineGap: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "baseline_gap",
			Help: "Signed gap versus named baselines.",
		}, []string{"baseline", "profile", "endpoint", "kind"}),
		baselineGapAbsolute: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "baseline_gap_absolute",
			Help: "Absolute gap versus named baselines.",
		}, []string{"baseline", "profile", "endpoint", "kind"}),
	}

	registry.MustRegister(
		m.recomputesTotal,
		m.recomputeDurationSeconds,
		m.currentModelAgeSeconds,
		m.currentProfileAggregateAvailability,
		m.currentEndpointAvailability,
		m.endpointsBelowThreshold,
		m.activeModelInfo,
		m.activeTopologyVersion,
		m.previousGap,
		m.previousGapAbsolute,
		m.baselineGap,
		m.baselineGapAbsolute,
	)
	return m
}

func (m *Metrics) recordOutcome(outcome string, duration time.Duration) {
	m.recomputesTotal.WithLabelValues(outcome).Inc()
	m.recomputeDurationSeconds.Observe(duration.Seconds())
}

func (m *Metrics) updateCurrent(rep report.Report, modelAge time.Duration) {
	m.currentModelAgeSeconds.Set(modelAge.Seconds())
	m.currentProfileAggregateAvailability.Reset()
	m.currentEndpointAvailability.Reset()
	m.endpointsBelowThreshold.Reset()
	m.activeModelInfo.Reset()
	m.activeTopologyVersion.Reset()
	m.previousGap.Reset()
	m.previousGapAbsolute.Reset()
	m.baselineGap.Reset()
	m.baselineGapAbsolute.Reset()

	if rep.InputArtifact != nil {
		m.activeModelInfo.WithLabelValues(
			rep.InputArtifact.Digest,
			rep.InputArtifact.Kind,
			rep.InputArtifact.ContractName,
			rep.InputArtifact.ContractVersion,
			rep.InputArtifact.SourceType,
		).Set(1)
		topologyVersion := rep.InputArtifact.TopologyVersion
		if topologyVersion == "" {
			topologyVersion = "unknown"
		}
		m.activeTopologyVersion.WithLabelValues(topologyVersion).Set(1)
	}

	profiles := rep.NormalizedProfiles()
	for _, profile := range profiles {
		m.currentProfileAggregateAvailability.WithLabelValues(profile.Name, "weighted").Set(profile.Simulation.WeightedAggregate)
		m.currentProfileAggregateAvailability.WithLabelValues(profile.Name, "unweighted").Set(profile.Simulation.UnweightedAggregate)
		m.endpointsBelowThreshold.WithLabelValues(profile.Name).Set(float64(profile.EndpointsBelowThreshold))
		for _, endpoint := range profile.EndpointResults {
			m.currentEndpointAvailability.WithLabelValues(profile.Name, endpoint.EndpointID).Set(endpoint.Availability)
		}
	}

	if rep.Diffs.Previous != nil {
		recordProfileDiffMetrics(m.previousGap, m.previousGapAbsolute, "", *rep.Diffs.Previous)
	}
	for _, baseline := range rep.Diffs.Baselines {
		recordProfileDiffMetrics(m.baselineGap, m.baselineGapAbsolute, baseline.Name, baseline)
	}
}

func recordProfileDiffMetrics(signed *prometheus.GaugeVec, absolute *prometheus.GaugeVec, baseline string, diff report.Diff) {
	if baseline == "" {
		signed.WithLabelValues("", "", "cross_profile_weighted").Set(diff.CrossProfileWeighted.Signed)
		absolute.WithLabelValues("", "", "cross_profile_weighted").Set(diff.CrossProfileWeighted.Absolute)
		signed.WithLabelValues("", "", "cross_profile_unweighted").Set(diff.CrossProfileUnweighted.Signed)
		absolute.WithLabelValues("", "", "cross_profile_unweighted").Set(diff.CrossProfileUnweighted.Absolute)
		for _, profile := range diff.Profiles {
			signed.WithLabelValues(profile.Profile, "", "weighted").Set(profile.WeightedAggregate.Signed)
			absolute.WithLabelValues(profile.Profile, "", "weighted").Set(profile.WeightedAggregate.Absolute)
			signed.WithLabelValues(profile.Profile, "", "unweighted").Set(profile.UnweightedAggregate.Signed)
			absolute.WithLabelValues(profile.Profile, "", "unweighted").Set(profile.UnweightedAggregate.Absolute)
			for _, endpoint := range profile.Endpoints {
				signed.WithLabelValues(profile.Profile, endpoint.EndpointID, "endpoint").Set(endpoint.Availability.Signed)
				absolute.WithLabelValues(profile.Profile, endpoint.EndpointID, "endpoint").Set(endpoint.Availability.Absolute)
			}
		}
		return
	}

	signed.WithLabelValues(baseline, "", "", "cross_profile_weighted").Set(diff.CrossProfileWeighted.Signed)
	absolute.WithLabelValues(baseline, "", "", "cross_profile_weighted").Set(diff.CrossProfileWeighted.Absolute)
	signed.WithLabelValues(baseline, "", "", "cross_profile_unweighted").Set(diff.CrossProfileUnweighted.Signed)
	absolute.WithLabelValues(baseline, "", "", "cross_profile_unweighted").Set(diff.CrossProfileUnweighted.Absolute)
	for _, profile := range diff.Profiles {
		signed.WithLabelValues(baseline, profile.Profile, "", "weighted").Set(profile.WeightedAggregate.Signed)
		absolute.WithLabelValues(baseline, profile.Profile, "", "weighted").Set(profile.WeightedAggregate.Absolute)
		signed.WithLabelValues(baseline, profile.Profile, "", "unweighted").Set(profile.UnweightedAggregate.Signed)
		absolute.WithLabelValues(baseline, profile.Profile, "", "unweighted").Set(profile.UnweightedAggregate.Absolute)
		for _, endpoint := range profile.Endpoints {
			signed.WithLabelValues(baseline, profile.Profile, endpoint.EndpointID, "endpoint").Set(endpoint.Availability.Signed)
			absolute.WithLabelValues(baseline, profile.Profile, endpoint.EndpointID, "endpoint").Set(endpoint.Availability.Absolute)
		}
	}
}
