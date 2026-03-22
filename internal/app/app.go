package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/analyzer"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/discovery/otel"
	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/report"
	"github.com/MB3R-Lab/Sheaft/internal/service"
	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

const (
	ExitOK       = 0
	ExitError    = 1
	ExitGateDeny = 2
)

type Runner struct {
	stdout io.Writer
	stderr io.Writer
}

type executionResult struct {
	Model    model.ResilienceModel
	Config   config.AnalysisConfig
	Report   report.Report
	GateEval gate.Evaluation
}

func NewRunner(stdout, stderr io.Writer) Runner {
	return Runner{stdout: stdout, stderr: stderr}
}

func (r Runner) Run(args []string) int {
	if len(args) == 0 {
		r.printUsage()
		return ExitError
	}

	switch args[0] {
	case "discover":
		return r.runDiscover(args[1:])
	case "simulate":
		return r.runSimulate(args[1:])
	case "gate":
		return r.runGate(args[1:])
	case "run":
		return r.runPipeline(args[1:])
	case "serve", "watch":
		return r.runServe(args[1:])
	case "help", "--help", "-h":
		r.printUsage()
		return ExitOK
	default:
		r.printfErr("unknown command: %s\n\n", args[0])
		r.printUsage()
		return ExitError
	}
}

func (r Runner) runDiscover(args []string) int {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	input := fs.String("input", "", "Path to OTel trace file or directory")
	out := fs.String("out", "", "Path to output model JSON")
	if err := fs.Parse(args); err != nil {
		r.printfErr("discover flag parse error: %v\n", err)
		return ExitError
	}
	if strings.TrimSpace(*input) == "" || strings.TrimSpace(*out) == "" {
		r.printfErr("discover requires --input and --out\n")
		return ExitError
	}

	mdl, err := otel.Discover(*input)
	if err != nil {
		r.printfErr("discover error: %v\n", err)
		return ExitError
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		r.printfErr("create output dir: %v\n", err)
		return ExitError
	}
	if err := model.WriteToFile(*out, mdl); err != nil {
		r.printfErr("write model: %v\n", err)
		return ExitError
	}

	r.printf("warning: discover is experimental; production discovery is expected to run in Bering\n")
	r.printf("model written: %s\n", *out)
	return ExitOK
}

func (r Runner) runSimulate(args []string) int {
	fs := flag.NewFlagSet("simulate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	modelPath := fs.String("model", "", "Path to model or snapshot JSON")
	policyPath := fs.String("policy", "", "Path to policy YAML/JSON")
	analysisPath := fs.String("analysis", "", "Path to advanced analysis YAML/JSON")
	contractPolicyPath := fs.String("contract-policy", "", "Path to contract policy YAML/JSON")
	out := fs.String("out", "", "Path to output report JSON")
	journeysPath := fs.String("journeys", "", "Path to journey override JSON")
	seed := fs.Int64("seed", 42, "Random seed for deterministic simulation")
	if err := fs.Parse(args); err != nil {
		r.printfErr("simulate flag parse error: %v\n", err)
		return ExitError
	}
	if *modelPath == "" || *out == "" || (*policyPath == "" && *analysisPath == "") {
		r.printfErr("simulate requires --model, --out, and one of --policy or --analysis\n")
		return ExitError
	}

	result, err := executeAnalysis(*modelPath, *policyPath, *analysisPath, *contractPolicyPath, optionalInt64(*seed, isFlagSet(fs, "seed")), *journeysPath, nil)
	if err != nil {
		r.printfErr("simulate error: %v\n", err)
		return ExitError
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		r.printfErr("create output dir: %v\n", err)
		return ExitError
	}
	if err := report.WriteJSON(*out, result.Report); err != nil {
		r.printfErr("write report: %v\n", err)
		return ExitError
	}

	r.printf("report written: %s\n", *out)
	r.printf("decision: %s\n", result.GateEval.Decision)
	if result.Report.ContractPolicy != nil && result.Report.ContractPolicy.Status != config.ContractPolicyStatusCurrent {
		r.printf("contract policy: %s (%s)\n", result.Report.ContractPolicy.Status, result.Report.ContractPolicy.Action)
	}
	return decisionExitCode(result.GateEval.Decision)
}

func (r Runner) runGate(args []string) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reportPath := fs.String("report", "", "Path to report JSON")
	policyPath := fs.String("policy", "", "Path to policy YAML/JSON")
	analysisPath := fs.String("analysis", "", "Path to advanced analysis YAML/JSON")
	mode := fs.String("mode", "", "Override mode: warn|fail|report")
	if err := fs.Parse(args); err != nil {
		r.printfErr("gate flag parse error: %v\n", err)
		return ExitError
	}
	if *reportPath == "" || (*policyPath == "" && *analysisPath == "") {
		r.printfErr("gate requires --report and one of --policy or --analysis\n")
		return ExitError
	}

	rep, err := report.Load(*reportPath)
	if err != nil {
		r.printfErr("read report: %v\n", err)
		return ExitError
	}

	var eval gate.Evaluation
	if *analysisPath != "" {
		analysisCfg, err := config.LoadAnalysis(*analysisPath)
		if err != nil {
			r.printfErr("read analysis config: %v\n", err)
			return ExitError
		}
		if *mode != "" {
			analysisCfg.Gate.Mode = config.PolicyMode(*mode)
		}
		eval, err = evaluateReportProfiles(rep, analysisCfg)
		if err != nil {
			r.printfErr("evaluate gate: %v\n", err)
			return ExitError
		}
	} else {
		policy, err := config.LoadPolicy(*policyPath)
		if err != nil {
			r.printfErr("read policy: %v\n", err)
			return ExitError
		}
		eval, err = gate.Evaluate(rep.AvailabilityMap(), policy, *mode)
		if err != nil {
			r.printfErr("evaluate gate: %v\n", err)
			return ExitError
		}
	}

	r.printf("mode: %s\n", eval.Mode)
	r.printf("decision: %s\n", eval.Decision)
	if len(eval.FailedProfiles) > 0 {
		r.printf("failed profiles: %s\n", strings.Join(eval.FailedProfiles, ", "))
	}
	if len(eval.FailedEndpoints) > 0 {
		r.printf("failed endpoints: %s\n", strings.Join(eval.FailedEndpoints, ", "))
	}
	return decisionExitCode(eval.Decision)
}

func evaluateReportProfiles(rep report.Report, analysisCfg config.AnalysisConfig) (gate.Evaluation, error) {
	profiles := rep.NormalizedProfiles()
	outputs := make([]simulation.ProfileOutput, 0, len(profiles))
	for _, profile := range profiles {
		outputs = append(outputs, simulation.ProfileOutput{
			Name:                 profile.Name,
			WeightedAggregate:    profile.Simulation.WeightedAggregate,
			UnweightedAggregate:  profile.Simulation.UnweightedAggregate,
			EndpointAvailability: profile.Simulation.EndpointAvailability,
			Assertions:           profile.Simulation.Assertions,
		})
	}
	return gate.EvaluateProfiles(outputs, analysisCfg.Gate)
}

func (r Runner) runPipeline(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	modelPath := fs.String("model", "", "Path to model or snapshot JSON")
	policyPath := fs.String("policy", "", "Path to policy YAML/JSON")
	analysisPath := fs.String("analysis", "", "Path to advanced analysis YAML/JSON")
	contractPolicyPath := fs.String("contract-policy", "", "Path to contract policy YAML/JSON")
	journeysPath := fs.String("journeys", "", "Path to journey override JSON")
	outDir := fs.String("out-dir", "", "Output directory")
	seed := fs.Int64("seed", 42, "Random seed for deterministic simulation")
	if err := fs.Parse(args); err != nil {
		r.printfErr("run flag parse error: %v\n", err)
		return ExitError
	}
	if *modelPath == "" || *outDir == "" || (*policyPath == "" && *analysisPath == "") {
		r.printfErr("run requires --model, --out-dir, and one of --policy or --analysis\n")
		return ExitError
	}

	outputModelPath := filepath.Join(*outDir, "model.json")
	reportPath := filepath.Join(*outDir, "report.json")
	summaryPath := filepath.Join(*outDir, "summary.md")

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		r.printfErr("create out dir: %v\n", err)
		return ExitError
	}

	result, err := executeAnalysis(*modelPath, *policyPath, *analysisPath, *contractPolicyPath, optionalInt64(*seed, isFlagSet(fs, "seed")), *journeysPath, nil)
	if err != nil {
		r.printfErr("run error: %v\n", err)
		return ExitError
	}
	if err := model.WriteToFile(outputModelPath, result.Model); err != nil {
		r.printfErr("write model: %v\n", err)
		return ExitError
	}
	if err := report.WriteJSON(reportPath, result.Report); err != nil {
		r.printfErr("write report: %v\n", err)
		return ExitError
	}
	if err := report.WriteSummaryMarkdown(summaryPath, result.Report); err != nil {
		r.printfErr("write summary: %v\n", err)
		return ExitError
	}

	r.printf("model written: %s\n", outputModelPath)
	r.printf("report written: %s\n", reportPath)
	r.printf("summary written: %s\n", summaryPath)
	r.printf("decision: %s\n", result.GateEval.Decision)
	if result.Report.InputArtifact != nil {
		r.printf("contract: %s@%s\n", result.Report.InputArtifact.ContractName, result.Report.InputArtifact.ContractVersion)
	}
	if result.Report.ContractPolicy != nil && result.Report.ContractPolicy.Status != config.ContractPolicyStatusCurrent {
		r.printf("contract policy: %s (%s)\n", result.Report.ContractPolicy.Status, result.Report.ContractPolicy.Action)
	}
	r.printf("policy mode: %s\n", result.Config.Gate.Mode)
	if strings.TrimSpace(result.Config.Journeys) != "" {
		r.printf("journeys override: %s\n", result.Config.Journeys)
	}
	return decisionExitCode(result.GateEval.Decision)
}

func (r Runner) runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "Path to serve config YAML/JSON")
	artifactPath := fs.String("artifact", "", "Artifact file or directory to watch")
	artifactMode := fs.String("artifact-mode", "auto", "Artifact mode: auto|file|directory")
	policyPath := fs.String("policy", "", "Path to policy YAML/JSON")
	analysisPath := fs.String("analysis", "", "Path to advanced analysis YAML/JSON")
	contractPolicyPath := fs.String("contract-policy", "", "Path to contract policy YAML/JSON")
	listen := fs.String("listen", ":8080", "HTTP listen address")
	pollInterval := fs.String("poll-interval", "30s", "Polling interval")
	historyDir := fs.String("history-dir", "", "Optional on-disk history directory")
	if err := fs.Parse(args); err != nil {
		r.printfErr("serve flag parse error: %v\n", err)
		return ExitError
	}

	var serveCfg config.ServeConfig
	var analysisCfg config.AnalysisConfig
	var err error
	switch {
	case *configPath != "":
		serveCfg, err = config.LoadServeConfig(*configPath)
		if err != nil {
			r.printfErr("load serve config: %v\n", err)
			return ExitError
		}
		if serveCfg.AnalysisFile != "" {
			analysisCfg, err = config.LoadAnalysis(serveCfg.AnalysisFile)
		} else {
			analysisCfg = serveCfg.Analysis.Normalized()
			analysisCfg.Sources = config.BuildAnalysisParameterSources(serveCfg.Analysis, analysisCfg)
			err = analysisCfg.Validate()
		}
		if err != nil {
			r.printfErr("load analysis config: %v\n", err)
			return ExitError
		}
		if *contractPolicyPath != "" {
			analysisCfg.ContractPolicy, err = config.LoadContractPolicy(*contractPolicyPath)
			if err != nil {
				r.printfErr("load contract policy: %v\n", err)
				return ExitError
			}
		}
	default:
		if *artifactPath == "" || (*policyPath == "" && *analysisPath == "") {
			r.printfErr("serve requires --config or --artifact with one of --policy or --analysis\n")
			return ExitError
		}
		analysisCfg, err = loadExecutionConfig(*policyPath, *analysisPath, *contractPolicyPath, nil, "")
		if err != nil {
			r.printfErr("load analysis config: %v\n", err)
			return ExitError
		}
		serveCfg = config.ServeConfig{
			SchemaVersion: config.ServeSchemaVersion,
			Listen:        *listen,
			Artifact: config.ArtifactSource{
				Path: *artifactPath,
				Mode: *artifactMode,
			},
			PollInterval: *pollInterval,
			History: config.HistoryConfig{
				MaxItems: 10,
				DiskDir:  *historyDir,
			},
		}.Normalized()
		if err := serveCfg.Validate(); err != nil {
			r.printfErr("validate serve config: %v\n", err)
			return ExitError
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	svc, err := service.New(serveCfg, analysisCfg)
	if err != nil {
		r.printfErr("create service: %v\n", err)
		return ExitError
	}
	r.printf("listening on %s\n", serveCfg.Listen)
	if err := svc.Run(ctx); err != nil {
		r.printfErr("serve error: %v\n", err)
		return ExitError
	}
	return ExitOK
}

func executeAnalysis(modelPath, policyPath, analysisPath, contractPolicyPath string, seedOverride *int64, journeysPath string, previous *report.Report) (executionResult, error) {
	analysisCfg, err := loadExecutionConfig(policyPath, analysisPath, contractPolicyPath, seedOverride, journeysPath)
	if err != nil {
		return executionResult{}, err
	}
	result, err := analyzer.AnalyzeFile(modelPath, analysisCfg, previous)
	if err != nil {
		return executionResult{}, err
	}
	return executionResult{
		Model:    result.Artifact.Model,
		Config:   analysisCfg,
		Report:   result.Report,
		GateEval: result.Evaluation,
	}, nil
}

func loadExecutionConfig(policyPath, analysisPath, contractPolicyPath string, seedOverride *int64, journeysPath string) (config.AnalysisConfig, error) {
	switch {
	case policyPath != "" && analysisPath != "":
		return config.AnalysisConfig{}, errors.New("use either --policy or --analysis, not both")
	case analysisPath != "":
		cfg, err := config.LoadAnalysis(analysisPath)
		if err != nil {
			return config.AnalysisConfig{}, err
		}
		if contractPolicyPath != "" {
			cfg.ContractPolicy, err = config.LoadContractPolicy(contractPolicyPath)
			if err != nil {
				return config.AnalysisConfig{}, err
			}
		}
		if seedOverride != nil {
			cfg.Seed = *seedOverride
			cfg.Sources.Seed = config.ParameterSourceOverride
		}
		if journeysPath != "" {
			cfg.Journeys = journeysPath
			cfg.Sources.Journeys = config.ParameterSourceOverride
		}
		return cfg.Normalized(), nil
	case policyPath != "":
		policy, err := config.LoadPolicy(policyPath)
		if err != nil {
			return config.AnalysisConfig{}, err
		}
		cfg := policy.ToAnalysisConfig()
		if contractPolicyPath != "" {
			cfg.ContractPolicy, err = config.LoadContractPolicy(contractPolicyPath)
			if err != nil {
				return config.AnalysisConfig{}, err
			}
		}
		if seedOverride != nil {
			cfg.Seed = *seedOverride
			cfg.Sources.Seed = config.ParameterSourceOverride
		}
		if journeysPath != "" {
			cfg.Journeys = journeysPath
			cfg.Sources.Journeys = config.ParameterSourceOverride
		}
		return cfg.Normalized(), nil
	default:
		return config.AnalysisConfig{}, errors.New("missing policy or analysis config")
	}
}

func decisionExitCode(decision string) int {
	switch decision {
	case gate.StatusFail:
		return ExitGateDeny
	case gate.StatusPass, gate.StatusWarn, "report":
		return ExitOK
	default:
		return ExitError
	}
}

func (r Runner) printUsage() {
	fmt.Fprintln(r.stdout, "Sheaft CLI")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Usage:")
	fmt.Fprintln(r.stdout, "  sheaft discover --input <trace-file|dir> --out <model.json>    # experimental local discovery")
	fmt.Fprintln(r.stdout, "  sheaft simulate --model <artifact.json> --policy <policy.yaml> [--contract-policy <contract-policy.yaml>] --out <report.json> [--journeys <journeys.json>] --seed <int>")
	fmt.Fprintln(r.stdout, "  sheaft simulate --model <artifact.json> --analysis <analysis.yaml> [--contract-policy <contract-policy.yaml>] --out <report.json>")
	fmt.Fprintln(r.stdout, "  sheaft gate --report <report.json> --policy <policy.yaml> --mode warn|fail|report")
	fmt.Fprintln(r.stdout, "  sheaft gate --report <report.json> --analysis <analysis.yaml>")
	fmt.Fprintln(r.stdout, "  sheaft run --model <artifact.json> --policy <policy.yaml> [--contract-policy <contract-policy.yaml>] --out-dir <dir> [--journeys <journeys.json>] --seed <int>")
	fmt.Fprintln(r.stdout, "  sheaft run --model <artifact.json> --analysis <analysis.yaml> [--contract-policy <contract-policy.yaml>] --out-dir <dir>")
	fmt.Fprintln(r.stdout, "  sheaft serve --config <serve.yaml> [--contract-policy <contract-policy.yaml>]")
	fmt.Fprintln(r.stdout, "  sheaft serve --artifact <artifact.json|dir> --policy <policy.yaml> [--contract-policy <contract-policy.yaml>] [--listen :8080]")
}

func (r Runner) printf(format string, args ...any) {
	fmt.Fprintf(r.stdout, format, args...)
}

func (r Runner) printfErr(format string, args ...any) {
	fmt.Fprintf(r.stderr, format, args...)
}

var ErrUnsupportedCommand = errors.New("unsupported command")

func isFlagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

func optionalInt64(value int64, enabled bool) *int64 {
	if !enabled {
		return nil
	}
	out := value
	return &out
}
