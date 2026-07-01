package oracle

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/faults"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

const (
	DefaultOutputDir = ".tmp/oracle-suite"
	SuiteName        = "sheaft-stochastic-connectivity-oracles"
)

type RunOptions struct {
	RepositoryRoot string
	OutputDir      string
	GeneratedAt    time.Time
}

type Outputs struct {
	Report  string `json:"report"`
	Summary string `json:"summary"`
}

type Report struct {
	SchemaVersion string       `json:"schema_version"`
	SuiteName     string       `json:"suite_name"`
	Status        string       `json:"status"`
	GeneratedAt   string       `json:"generated_at"`
	Outputs       Outputs      `json:"outputs"`
	Cases         []CaseResult `json:"cases"`
}

type CaseResult struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Trials      int     `json:"trials"`
	Checks      []Check `json:"checks"`
	Error       string  `json:"error,omitempty"`
}

type Check struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	Expected  float64 `json:"expected"`
	Actual    float64 `json:"actual"`
	Tolerance float64 `json:"tolerance"`
	Message   string  `json:"message"`
}

type oracleCase struct {
	id          string
	description string
	run         func() (CaseResult, error)
}

func Run(opts RunOptions) (Report, error) {
	root := opts.RepositoryRoot
	if root == "" {
		root = "."
	}
	outDir := opts.OutputDir
	if outDir == "" {
		outDir = DefaultOutputDir
	}
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	report := Execute(generatedAt)
	report.Outputs = Outputs{
		Report:  filepath.ToSlash(filepath.Join(outDir, "oracle-report.json")),
		Summary: filepath.ToSlash(filepath.Join(outDir, "summary.md")),
	}
	if err := os.MkdirAll(filepath.Join(root, outDir), 0o755); err != nil {
		return Report{}, fmt.Errorf("create oracle output dir: %w", err)
	}
	if err := writeJSON(filepath.Join(root, report.Outputs.Report), report); err != nil {
		return Report{}, fmt.Errorf("write oracle report: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, report.Outputs.Summary), []byte(SummaryMarkdown(report)), 0o644); err != nil {
		return Report{}, fmt.Errorf("write oracle summary: %w", err)
	}
	return report, nil
}

func Execute(generatedAt time.Time) Report {
	cases := oracleCases()
	results := make([]CaseResult, 0, len(cases))
	status := "pass"
	for _, item := range cases {
		result, err := item.run()
		if err != nil {
			result = CaseResult{
				ID:          item.id,
				Description: item.description,
				Status:      "fail",
				Error:       err.Error(),
			}
		}
		if result.Status != "pass" {
			status = "fail"
		}
		results = append(results, result)
	}
	return Report{
		SchemaVersion: "1.0",
		SuiteName:     SuiteName,
		Status:        status,
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339Nano),
		Cases:         results,
	}
}

func SummaryMarkdown(rep Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Oracle Suite\n\n")
	fmt.Fprintf(&b, "- Suite: `%s`\n", rep.SuiteName)
	fmt.Fprintf(&b, "- Status: `%s`\n", rep.Status)
	fmt.Fprintf(&b, "- Cases: `%d`\n\n", len(rep.Cases))
	for _, item := range rep.Cases {
		fmt.Fprintf(&b, "## %s\n\n", item.ID)
		fmt.Fprintf(&b, "- Status: `%s`\n", item.Status)
		if item.Trials > 0 {
			fmt.Fprintf(&b, "- Trials: `%d`\n", item.Trials)
		}
		if item.Error != "" {
			fmt.Fprintf(&b, "- Error: `%s`\n", item.Error)
		}
		for _, check := range item.Checks {
			fmt.Fprintf(&b, "- `%s`: %s expected=`%.6f` actual=`%.6f` tolerance=`%.6f`\n", check.ID, check.Status, check.Expected, check.Actual, check.Tolerance)
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func oracleCases() []oracleCase {
	return []oracleCase{
		{
			id:          "sync_chain_product",
			description: "Synchronous chain follows the product of node and edge reliability.",
			run: func() (CaseResult, error) {
				theta := 0.90
				rho := 0.80
				trials := 100000
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "checkout", Name: "checkout", Replicas: 1},
						{ID: "payment", Name: "payment", Replicas: 1},
					},
					[]model.Edge{
						syncEdge("frontend", "checkout"),
						syncEdge("checkout", "payment"),
					},
					[]model.Endpoint{{ID: "frontend:GET /payment", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /payment"}},
				)
				return endpointOracle(itemID("sync_chain_product"), "Synchronous chain follows the product of node and edge reliability.", loaded, profileParams("chain", trials, config.ReliabilityConfig{
					NodeLiveProbability: &theta,
					EdgeLiveProbability: &rho,
				}), map[string]float64{
					"frontend:GET /payment": math.Pow(theta, 3) * math.Pow(rho, 2),
				})
			},
		},
		{
			id:          "fanout_any_path",
			description: "Fan-out fallback succeeds when the entry service and at least one branch survive.",
			run: func() (CaseResult, error) {
				trials := 100000
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "search-a", Name: "search-a", Replicas: 1},
						{ID: "search-b", Name: "search-b", Replicas: 1},
					},
					[]model.Edge{
						syncEdge("frontend", "search-a"),
						syncEdge("frontend", "search-b"),
					},
					[]model.Endpoint{{ID: "frontend:GET /search", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /search"}},
				)
				return endpointOracle(itemID("fanout_any_path"), "Fan-out fallback succeeds when the entry service and at least one branch survive.", loaded, profileParams("fanout", trials, config.ReliabilityConfig{
					Services: map[string]float64{
						"frontend": 0.80,
						"search-a": 0.50,
						"search-b": 0.50,
					},
				}), map[string]float64{
					"frontend:GET /search": 0.80 * (1 - math.Pow(1-0.50, 2)),
				})
			},
		},
		{
			id:          "mandatory_single_point",
			description: "A mandatory shared dependency remains the bottleneck even when upstream branches are replicated.",
			run: func() (CaseResult, error) {
				trials := 100000
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "checkout-a", Name: "checkout-a", Replicas: 1},
						{ID: "checkout-b", Name: "checkout-b", Replicas: 1},
						{ID: "db", Name: "db", Replicas: 1},
					},
					[]model.Edge{
						syncEdge("frontend", "checkout-a"),
						syncEdge("frontend", "checkout-b"),
						syncEdge("checkout-a", "db"),
						syncEdge("checkout-b", "db"),
					},
					[]model.Endpoint{edgeAwareEndpoint("frontend:POST /checkout", "frontend", "db", model.EndpointPredicateModeEventual, []string{string(model.EdgeKindSync)})},
				)
				return endpointOracle(itemID("mandatory_single_point"), "A mandatory shared dependency remains the bottleneck even when upstream branches are replicated.", loaded, profileParams("mandatory", trials, config.ReliabilityConfig{
					Services: map[string]float64{
						"frontend":   1,
						"checkout-a": 1,
						"checkout-b": 1,
						"db":         0.40,
					},
				}), map[string]float64{
					"frontend:POST /checkout": 0.40,
				})
			},
		},
		{
			id:          "replicated_target_edge_bottleneck",
			description: "Replicated target availability saturates at the edge reliability bottleneck.",
			run: func() (CaseResult, error) {
				trials := 100000
				rho := 0.82
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "target", Name: "target", Replicas: 12},
					},
					[]model.Edge{syncEdge("frontend", "target")},
					[]model.Endpoint{{ID: "frontend:GET /target", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /target"}},
				)
				return endpointOracle(itemID("replicated_target_edge_bottleneck"), "Replicated target availability saturates at the edge reliability bottleneck.", loaded, profileParams("replicated", trials, config.ReliabilityConfig{
					Services: map[string]float64{
						"frontend": 1,
						"target":   0.80,
					},
					Edges: map[string]float64{
						edgeID("frontend", "target", model.EdgeKindSync, true): rho,
					},
				}), map[string]float64{
					"frontend:GET /target": (1 - math.Pow(1-0.80, 12)) * rho,
				})
			},
		},
		{
			id:          "node_edge_sensitivity_grid",
			description: "A compact grid distinguishes node and edge reliability sensitivity.",
			run:         runSensitivityGrid,
		},
		{
			id:          "async_immediate_eventual_contrast",
			description: "Immediate predicates ignore async edges; eventual predicates include async edges only when declared.",
			run: func() (CaseResult, error) {
				trials := 100000
				rhoAsync := 0.30
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "checkout", Name: "checkout", Replicas: 1},
						{ID: "payment", Name: "payment", Replicas: 1},
					},
					[]model.Edge{
						syncEdge("frontend", "checkout"),
						asyncEdge("checkout", "payment"),
					},
					[]model.Endpoint{
						edgeAwareEndpoint("frontend:GET /checkout", "frontend", "checkout", model.EndpointPredicateModeImmediate, []string{string(model.EdgeKindSync), string(model.EdgeKindAsync)}),
						edgeAwareEndpoint("frontend:POST /checkout", "frontend", "payment", model.EndpointPredicateModeEventual, []string{string(model.EdgeKindSync), string(model.EdgeKindAsync)}),
					},
				)
				return endpointOracle(itemID("async_immediate_eventual_contrast"), "Immediate predicates ignore async edges; eventual predicates include async edges only when declared.", loaded, profileParams("async", trials, config.ReliabilityConfig{
					Services: map[string]float64{
						"frontend": 1,
						"checkout": 1,
						"payment":  1,
					},
					Edges: map[string]float64{
						edgeID("frontend", "checkout", model.EdgeKindSync, true):  1,
						edgeID("checkout", "payment", model.EdgeKindAsync, false): rhoAsync,
					},
				}), map[string]float64{
					"frontend:GET /checkout":  1,
					"frontend:POST /checkout": rhoAsync,
				})
			},
		},
		{
			id:          "correlated_failure_boundary",
			description: "A targeted correlated failure domain kills the matching mandatory service.",
			run: func() (CaseResult, error) {
				trials := 1000
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "checkout", Name: "checkout", Replicas: 1},
					},
					[]model.Edge{syncEdge("frontend", "checkout")},
					[]model.Endpoint{{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"}},
				)
				contract := faults.Contract{
					SchemaVersion: faults.SchemaVersion,
					Profiles: map[string]faults.Profile{
						"checkout-down": {
							Faults: []faults.Fault{{Type: faults.TypeCorrelatedFailureDomain, Selector: faults.Selector{ServiceIDs: []string{"checkout"}}}},
						},
					},
				}
				return endpointOracleWithFault(itemID("correlated_failure_boundary"), "A targeted correlated failure domain kills the matching mandatory service.", loaded, profileParamsWithFault("checkout-down", trials, "checkout-down", config.ReliabilityConfig{
					Services: map[string]float64{"frontend": 1, "checkout": 1},
				}), &contract, map[string]float64{
					"frontend:GET /checkout": 0,
				})
			},
		},
		{
			id:          "timeout_retry_boundary",
			description: "Timeout mismatch on a live edge fails path execution despite live services and edge.",
			run: func() (CaseResult, error) {
				trials := 1000
				edge := syncEdge("frontend", "checkout")
				edge.Resilience = &model.ResiliencePolicy{PerTryTimeoutMS: 100}
				edge.Observed = &model.ObservedEdge{LatencyMS: &model.LatencySummary{P90: 50}}
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "checkout", Name: "checkout", Replicas: 1},
					},
					[]model.Edge{edge},
					[]model.Endpoint{{ID: "frontend:GET /checkout", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /checkout"}},
				)
				contract := faults.Contract{
					SchemaVersion: faults.SchemaVersion,
					Profiles: map[string]faults.Profile{
						"latency": {
							Faults: []faults.Fault{{
								Type:      faults.TypeEdgePartialDegradation,
								LatencyMS: &model.LatencySummary{P90: 100},
								Selector:  faults.Selector{EdgeIDs: []string{edge.ID}},
							}},
						},
					},
				}
				return endpointOracleWithFault(itemID("timeout_retry_boundary"), "Timeout mismatch on a live edge fails path execution despite live services and edge.", loaded, profileParamsWithFault("latency", trials, "latency", config.ReliabilityConfig{
					Services: map[string]float64{"frontend": 1, "checkout": 1},
					Edges:    map[string]float64{edge.ID: 1},
				}), &contract, map[string]float64{
					"frontend:GET /checkout": 0,
				})
			},
		},
		{
			id:          "trace_incompleteness_boundary",
			description: "Eventual completion is not guessed when traces do not provide a path to the mandatory target.",
			run: func() (CaseResult, error) {
				trials := 1000
				loaded := loadedFixture(
					[]model.Service{
						{ID: "frontend", Name: "frontend", Replicas: 1},
						{ID: "payment", Name: "payment", Replicas: 1},
					},
					nil,
					[]model.Endpoint{edgeAwareEndpoint("frontend:POST /checkout", "frontend", "payment", model.EndpointPredicateModeEventual, []string{string(model.EdgeKindSync), string(model.EdgeKindAsync)})},
				)
				return endpointOracle(itemID("trace_incompleteness_boundary"), "Eventual completion is not guessed when traces do not provide a path to the mandatory target.", loaded, profileParams("incomplete", trials, config.ReliabilityConfig{
					Services: map[string]float64{"frontend": 1, "payment": 1},
				}), map[string]float64{
					"frontend:POST /checkout": 0,
				})
			},
		},
	}
}

func runSensitivityGrid() (CaseResult, error) {
	trials := 60000
	loaded := loadedFixture(
		[]model.Service{
			{ID: "frontend", Name: "frontend", Replicas: 1},
			{ID: "target", Name: "target", Replicas: 1},
		},
		[]model.Edge{syncEdge("frontend", "target")},
		[]model.Endpoint{{ID: "frontend:GET /target", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /target"}},
	)
	result := CaseResult{
		ID:          itemID("node_edge_sensitivity_grid"),
		Description: "A compact grid distinguishes node and edge reliability sensitivity.",
		Status:      "pass",
		Trials:      trials,
	}
	for _, theta := range []float64{0.70, 0.90} {
		for _, rho := range []float64{0.60, 0.95} {
			out, err := simulation.RunArtifactProfiles(loaded, simulation.AnalysisParams{
				Seed: int64(theta*1000 + rho*1000),
				Profiles: []simulation.ProfileParams{
					profileParams("grid", trials, config.ReliabilityConfig{
						NodeLiveProbability: &theta,
						EdgeLiveProbability: &rho,
					}),
				},
			})
			if err != nil {
				return result, err
			}
			expected := theta * theta * rho
			actual := out.Profiles[0].EndpointAvailability["frontend:GET /target"]
			check := numericCheck(fmt.Sprintf("theta_%.2f_rho_%.2f", theta, rho), expected, actual, trials)
			result.Checks = append(result.Checks, check)
			if check.Status != "pass" {
				result.Status = "fail"
			}
		}
	}
	return result, nil
}

func endpointOracle(id, description string, loaded artifact.Loaded, profile simulation.ProfileParams, expectations map[string]float64) (CaseResult, error) {
	return endpointOracleWithFault(id, description, loaded, profile, nil, expectations)
}

func endpointOracleWithFault(id, description string, loaded artifact.Loaded, profile simulation.ProfileParams, contract *faults.Contract, expectations map[string]float64) (CaseResult, error) {
	out, err := simulation.RunArtifactProfiles(loaded, simulation.AnalysisParams{
		Seed:          profile.Seed,
		FaultContract: contract,
		Profiles:      []simulation.ProfileParams{profile},
	})
	result := CaseResult{
		ID:          id,
		Description: description,
		Status:      "pass",
		Trials:      profile.Trials,
	}
	if err != nil {
		return result, err
	}
	for endpointID, expected := range expectations {
		actual := out.Profiles[0].EndpointAvailability[endpointID]
		check := numericCheck(endpointID, expected, actual, profile.Trials)
		result.Checks = append(result.Checks, check)
		if check.Status != "pass" {
			result.Status = "fail"
		}
	}
	return result, nil
}

func numericCheck(id string, expected, actual float64, trials int) Check {
	tolerance := monteCarloTolerance(expected, trials)
	status := "pass"
	if math.Abs(actual-expected) > tolerance {
		status = "fail"
	}
	return Check{
		ID:        id,
		Status:    status,
		Expected:  expected,
		Actual:    actual,
		Tolerance: tolerance,
		Message:   "actual endpoint availability is within deterministic Monte Carlo confidence tolerance",
	}
}

func monteCarloTolerance(expected float64, trials int) float64 {
	if expected <= 0 || expected >= 1 {
		return 0.001
	}
	stdErr := math.Sqrt(expected * (1 - expected) / float64(trials))
	return math.Max(0.012, 4*stdErr)
}

func profileParams(name string, trials int, reliability config.ReliabilityConfig) simulation.ProfileParams {
	return profileParamsWithFault(name, trials, "", reliability)
}

func profileParamsWithFault(name string, trials int, faultProfile string, reliability config.ReliabilityConfig) simulation.ProfileParams {
	return simulation.ProfileParams{
		Name:         name,
		Trials:       trials,
		Seed:         int64(1000 + len(name)*17 + trials),
		SamplingMode: config.SamplingModeIndependentReplica,
		Reliability:  reliability,
		FaultProfile: faultProfile,
	}
}

func loadedFixture(services []model.Service, edges []model.Edge, endpoints []model.Endpoint) artifact.Loaded {
	return artifact.Loaded{
		Metadata: artifact.Metadata{
			Kind: modelcontract.KindModel,
			Contract: modelcontract.SupportedContract{
				Name:    modelcontract.BeringModelV110Name,
				Version: modelcontract.BeringModelV110Version,
				URI:     modelcontract.BeringModelV110URI,
				Digest:  modelcontract.BeringModelV110Digest,
				Kind:    modelcontract.KindModel,
			},
		},
		Model: model.ResilienceModel{
			Services:  services,
			Edges:     edges,
			Endpoints: endpoints,
			Metadata: model.Metadata{
				SourceType:   "oracle",
				SourceRef:    "synthetic://oracle-suite",
				DiscoveredAt: "2026-07-01T00:00:00Z",
				Confidence:   1,
				Schema: model.Schema{
					Name:    modelcontract.BeringModelV110Name,
					Version: modelcontract.BeringModelV110Version,
					URI:     modelcontract.BeringModelV110URI,
					Digest:  modelcontract.BeringModelV110Digest,
				},
			},
		},
	}
}

func edgeAwareEndpoint(id, entry, target, predicateMode string, dependencyModes []string) model.Endpoint {
	return model.Endpoint{
		ID:                  id,
		EntryService:        entry,
		SuccessPredicateRef: id,
		Metadata: &model.EndpointMetadata{
			Semantics: &model.EndpointSemantics{
				PredicateMode:    predicateMode,
				MandatoryTargets: []string{target},
				DependencyModes:  dependencyModes,
			},
		},
	}
}

func syncEdge(from, to string) model.Edge {
	return model.Edge{ID: edgeID(from, to, model.EdgeKindSync, true), From: from, To: to, Kind: model.EdgeKindSync, Blocking: true}
}

func asyncEdge(from, to string) model.Edge {
	return model.Edge{ID: edgeID(from, to, model.EdgeKindAsync, false), From: from, To: to, Kind: model.EdgeKindAsync, Blocking: false}
}

func edgeID(from, to string, kind model.EdgeKind, blocking bool) string {
	return fmt.Sprintf("%s|%s|%s|%t", from, to, kind, blocking)
}

func itemID(id string) string {
	return id
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
