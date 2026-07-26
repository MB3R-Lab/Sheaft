package simulation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"github.com/MB3R-Lab/Sheaft/internal/config"
)

const (
	SweepBoundaryCrossed              = "crossed"
	SweepBoundaryNotCrossed           = "not_crossed"
	SweepBoundaryBelowSLOAtStart      = "below_slo_at_start"
	SweepBoundaryNonMonotonic         = "non_monotonic"
	SweepCertificationCertified       = "certified"
	SweepCertificationThroughEnd      = "certified_through_end"
	SweepCertificationIndeterminate   = "indeterminate"
	SweepCertificationBelowSLOAtStart = "below_slo_at_start"
)

func runSweeps(ctx advancedContext, params AnalysisParams) ([]SweepOutput, error) {
	if len(params.Sweeps) == 0 {
		return nil, nil
	}

	endpointSet := make(map[string]struct{}, len(ctx.endpoints))
	for _, endpoint := range ctx.endpoints {
		endpointSet[endpoint.ID] = struct{}{}
	}
	totalSlots := totalAdvancedReplicaSlots(ctx)
	outputs := make([]SweepOutput, 0, len(params.Sweeps))

	for sweepIndex, sweep := range params.Sweeps {
		if sweep.ConfidenceLevel == 0 {
			sweep.ConfidenceLevel = 0.95
		}
		for _, target := range sweep.Targets {
			if _, exists := endpointSet[target.EndpointID]; !exists {
				return nil, fmt.Errorf("sweep %q target endpoint %q is not present in the artifact", sweep.Name, target.EndpointID)
			}
		}

		sweepSeed := derivedSeed(params.Seed, "sweep:"+sweep.Name, sweepIndex)
		output := SweepOutput{
			Name:              sweep.Name,
			BaseProfile:       sweep.BaseProfile.Name,
			AxisType:          sweep.Axis.Type,
			Trials:            sweep.BaseProfile.Trials,
			Seed:              sweepSeed,
			ConfidenceLevel:   sweep.ConfidenceLevel,
			Fingerprint:       sweepFingerprint(sweep, sweepSeed),
			TotalReplicaSlots: totalSlots,
			Points:            make([]SweepPoint, 0, len(sweep.Axis.Values)),
		}

		for pointIndex, value := range sweep.Axis.Values {
			pointProfile, point := sweepPointProfile(sweep, value, totalSlots, sweepSeed, pointIndex)
			normalized, err := normalizeProfile(pointProfile, sweepSeed, pointIndex)
			if err != nil {
				return nil, fmt.Errorf("sweep %q axis value %v: %w", sweep.Name, value, err)
			}
			faultSpec, err := resolveFaultProfile(params.FaultContract, normalized.FaultProfile)
			if err != nil {
				return nil, fmt.Errorf("sweep %q axis value %v: %w", sweep.Name, value, err)
			}
			profileOutput, err := runAdvancedProfile(ctx, normalized, params.DefaultWeights, faultSpec)
			if err != nil {
				return nil, fmt.Errorf("sweep %q axis value %v: %w", sweep.Name, value, err)
			}
			point.EndpointAvailability = make(map[string]float64, len(sweep.Targets))
			point.EndpointConfidence = make(map[string]BinomialInterval, len(sweep.Targets))
			for _, target := range sweep.Targets {
				availability := profileOutput.EndpointAvailability[target.EndpointID]
				point.EndpointAvailability[target.EndpointID] = availability
				point.EndpointConfidence[target.EndpointID] = wilsonInterval(
					profileOutput.EndpointSuccesses[target.EndpointID],
					profileOutput.Trials,
					sweep.ConfidenceLevel,
				)
			}
			output.Points = append(output.Points, point)
		}

		output.Boundaries = buildSweepBoundaries(output.Points, sweep.Targets, sweep.ConfidenceLevel)
		outputs = append(outputs, output)
	}
	return outputs, nil
}

func sweepPointProfile(sweep SweepParams, value float64, totalSlots int, seed int64, pointIndex int) (ProfileParams, SweepPoint) {
	profile := sweep.BaseProfile
	profile.Name = fmt.Sprintf("sweep:%s:%d", sweep.Name, pointIndex)
	profile.Seed = seed
	profile.CoupledTrials = true
	point := SweepPoint{AxisValue: value}

	switch sweep.Axis.Type {
	case config.SweepAxisIndependentReplicaFailureProbability:
		liveProbability := 1 - value
		profile.SamplingMode = config.SamplingModeIndependentReplica
		profile.FailureProbability = value
		profile.FixedKFailures = 0
		profile.Reliability.NodeLiveProbability = &liveProbability
		profile.Reliability.Services = map[string]float64{}
		point.FailureProbability = float64Pointer(value)
	case config.SweepAxisFailedReplicaSlots:
		failedSlots := int(value)
		fraction := 0.0
		if totalSlots > 0 {
			fraction = float64(failedSlots) / float64(totalSlots)
		}
		profile.SamplingMode = config.SamplingModeFixedKReplicaSlots
		profile.FailureProbability = 0
		profile.FixedKFailures = failedSlots
		point.FailedReplicaSlots = intPointer(failedSlots)
		point.FailedReplicaFraction = float64Pointer(fraction)
	}
	return profile, point
}

func buildSweepBoundaries(points []SweepPoint, targets []config.SweepTarget, confidenceLevel float64) []SweepBoundary {
	boundaries := make([]SweepBoundary, 0, len(targets))
	for _, target := range targets {
		boundary := SweepBoundary{
			EndpointID:        target.EndpointID,
			SLO:               target.SLO,
			ObservedMonotonic: true,
			ConfidenceLevel:   confidenceLevel,
		}
		previousAvailability := 1.0
		firstViolationIndex := -1
		for idx, point := range points {
			availability := point.EndpointAvailability[target.EndpointID]
			if idx > 0 && availability > previousAvailability {
				boundary.ObservedMonotonic = false
			}
			previousAvailability = availability
			if firstViolationIndex < 0 && availability < target.SLO {
				firstViolationIndex = idx
				boundary.FirstViolatingSLO = sweepBoundaryPoint(point, target.EndpointID)
				continue
			}
			if firstViolationIndex < 0 {
				boundary.LastMeetingSLO = sweepBoundaryPoint(point, target.EndpointID)
			}
		}
		certifySweepBoundary(&boundary, points, target)

		switch {
		case !boundary.ObservedMonotonic:
			boundary.Status = SweepBoundaryNonMonotonic
		case firstViolationIndex < 0:
			boundary.Status = SweepBoundaryNotCrossed
		case firstViolationIndex == 0:
			boundary.Status = SweepBoundaryBelowSLOAtStart
		default:
			boundary.Status = SweepBoundaryCrossed
			boundary.CrossingBracket = &CrossingBracket{
				LowerAxisValue: points[firstViolationIndex-1].AxisValue,
				UpperAxisValue: points[firstViolationIndex].AxisValue,
			}
		}
		boundaries = append(boundaries, boundary)
	}
	slices.SortFunc(boundaries, func(a, b SweepBoundary) int {
		if a.EndpointID < b.EndpointID {
			return -1
		}
		if a.EndpointID > b.EndpointID {
			return 1
		}
		return 0
	})
	return boundaries
}

func certifySweepBoundary(boundary *SweepBoundary, points []SweepPoint, target config.SweepTarget) {
	foundCertified := false
	for _, point := range points {
		interval := point.EndpointConfidence[target.EndpointID]
		if interval.LowerBound >= target.SLO {
			boundary.CertifiedTolerance = sweepBoundaryPoint(point, target.EndpointID)
			foundCertified = true
			continue
		}
		if interval.UpperBound < target.SLO {
			if foundCertified {
				boundary.CertificationStatus = SweepCertificationCertified
			} else {
				boundary.CertificationStatus = SweepCertificationBelowSLOAtStart
			}
			return
		}
		boundary.CertificationStatus = SweepCertificationIndeterminate
		return
	}
	if foundCertified {
		boundary.CertificationStatus = SweepCertificationThroughEnd
		return
	}
	boundary.CertificationStatus = SweepCertificationIndeterminate
}

func sweepBoundaryPoint(point SweepPoint, endpointID string) *BoundaryPoint {
	interval := point.EndpointConfidence[endpointID]
	return &BoundaryPoint{
		AxisValue:            point.AxisValue,
		Availability:         point.EndpointAvailability[endpointID],
		ConfidenceLowerBound: interval.LowerBound,
		ConfidenceUpperBound: interval.UpperBound,
	}
}

func wilsonInterval(successes, trials int, confidenceLevel float64) BinomialInterval {
	interval := BinomialInterval{Successes: successes, Trials: trials, ConfidenceLevel: confidenceLevel}
	if trials <= 0 {
		return interval
	}
	estimate := float64(successes) / float64(trials)
	z := math.Sqrt2 * math.Erfinv(confidenceLevel)
	z2 := z * z
	n := float64(trials)
	denominator := 1 + z2/n
	center := (estimate + z2/(2*n)) / denominator
	margin := z * math.Sqrt(estimate*(1-estimate)/n+z2/(4*n*n)) / denominator
	interval.Estimate = estimate
	interval.LowerBound = math.Max(0, center-margin)
	interval.UpperBound = math.Min(1, center+margin)
	return interval
}

func sweepFingerprint(sweep SweepParams, seed int64) string {
	payload, _ := json.Marshal(struct {
		Name            string               `json:"name"`
		BaseProfile     string               `json:"base_profile"`
		Trials          int                  `json:"trials"`
		FaultProfile    string               `json:"fault_profile,omitempty"`
		ConfidenceLevel float64              `json:"confidence_level"`
		Axis            config.SweepAxis     `json:"axis"`
		Targets         []config.SweepTarget `json:"targets"`
		Seed            int64                `json:"seed"`
	}{
		Name: sweep.Name, BaseProfile: sweep.BaseProfile.Name, Trials: sweep.BaseProfile.Trials,
		FaultProfile: sweep.BaseProfile.FaultProfile, ConfidenceLevel: sweep.ConfidenceLevel,
		Axis: sweep.Axis, Targets: sweep.Targets, Seed: seed,
	})
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func float64Pointer(value float64) *float64 { return &value }

func intPointer(value int) *int { return &value }
