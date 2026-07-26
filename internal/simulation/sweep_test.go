package simulation

import (
	"math"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/config"
)

func TestWilsonIntervalKnownProportion(t *testing.T) {
	t.Parallel()

	interval := wilsonInterval(95, 100, 0.95)
	if math.Abs(interval.Estimate-0.95) > 1e-12 {
		t.Fatalf("unexpected estimate: %+v", interval)
	}
	if interval.LowerBound < 0.88 || interval.LowerBound > 0.89 || interval.UpperBound < 0.97 || interval.UpperBound > 0.99 {
		t.Fatalf("unexpected Wilson interval: %+v", interval)
	}
}

func TestSweepFingerprintIgnoresArtifactReliabilityButTracksDefinition(t *testing.T) {
	t.Parallel()

	liveA := 0.9
	liveB := 0.7
	base := SweepParams{
		Name:            "checkout",
		BaseProfile:     ProfileParams{Name: "steady", Trials: 1000, Reliability: config.ReliabilityConfig{NodeLiveProbability: &liveA}},
		ConfidenceLevel: 0.95,
		Axis:            config.SweepAxis{Type: config.SweepAxisIndependentReplicaFailureProbability, Values: []float64{0, 0.1}},
		Targets:         []config.SweepTarget{{EndpointID: "checkout", SLO: 0.99}},
	}
	changedReliability := base
	changedReliability.BaseProfile.Reliability.NodeLiveProbability = &liveB
	if got, want := sweepFingerprint(base, 42), sweepFingerprint(changedReliability, 42); got != want {
		t.Fatalf("artifact reliability must not make paired sweep definitions incompatible: got=%s want=%s", got, want)
	}
	changedAxis := base
	changedAxis.Axis.Values = []float64{0, 0.2}
	if sweepFingerprint(base, 42) == sweepFingerprint(changedAxis, 42) {
		t.Fatal("axis changes must change the sweep fingerprint")
	}
}
