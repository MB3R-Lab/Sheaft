package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/discovery/otel"
	"github.com/MB3R-Lab/Sheaft/internal/gate"
	"github.com/MB3R-Lab/Sheaft/internal/journeys"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/report"
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

func NewRunner(stdout, stderr io.Writer) Runner {
	return Runner{
		stdout: stdout,
		stderr: stderr,
	}
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
	modelPath := fs.String("model", "", "Path to model JSON")
	policyPath := fs.String("policy", "", "Path to policy YAML/JSON")
	out := fs.String("out", "", "Path to output report JSON")
	journeysPath := fs.String("journeys", "", "Path to journey override JSON")
	seed := fs.Int64("seed", 42, "Random seed for Monte Carlo")
	if err := fs.Parse(args); err != nil {
		r.printfErr("simulate flag parse error: %v\n", err)
		return ExitError
	}
	if *modelPath == "" || *policyPath == "" || *out == "" {
		r.printfErr("simulate requires --model, --policy, and --out\n")
		return ExitError
	}

	_, _, rep, eval, err := executeSimulation(*modelPath, *policyPath, *seed, *journeysPath)
	if err != nil {
		r.printfErr("simulate error: %v\n", err)
		return ExitError
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		r.printfErr("create output dir: %v\n", err)
		return ExitError
	}
	if err := report.WriteJSON(*out, rep); err != nil {
		r.printfErr("write report: %v\n", err)
		return ExitError
	}

	r.printf("report written: %s\n", *out)
	r.printf("decision: %s\n", eval.Decision)
	return decisionExitCode(eval.Decision)
}

func (r Runner) runGate(args []string) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reportPath := fs.String("report", "", "Path to report JSON")
	policyPath := fs.String("policy", "", "Path to policy YAML/JSON")
	mode := fs.String("mode", "", "Override mode: warn|fail|report")
	if err := fs.Parse(args); err != nil {
		r.printfErr("gate flag parse error: %v\n", err)
		return ExitError
	}
	if *reportPath == "" || *policyPath == "" {
		r.printfErr("gate requires --report and --policy\n")
		return ExitError
	}

	rep, err := report.Load(*reportPath)
	if err != nil {
		r.printfErr("read report: %v\n", err)
		return ExitError
	}

	policy, err := config.LoadPolicy(*policyPath)
	if err != nil {
		r.printfErr("read policy: %v\n", err)
		return ExitError
	}

	eval, err := gate.Evaluate(rep.AvailabilityMap(), policy, *mode)
	if err != nil {
		r.printfErr("evaluate gate: %v\n", err)
		return ExitError
	}

	r.printf("mode: %s\n", eval.Mode)
	r.printf("decision: %s\n", eval.Decision)
	if len(eval.FailedEndpoints) > 0 {
		r.printf("failed endpoints: %s\n", strings.Join(eval.FailedEndpoints, ", "))
	}

	return decisionExitCode(eval.Decision)
}

func (r Runner) runPipeline(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	modelPath := fs.String("model", "", "Path to Bering model JSON")
	policyPath := fs.String("policy", "", "Path to policy YAML/JSON")
	journeysPath := fs.String("journeys", "", "Path to journey override JSON")
	outDir := fs.String("out-dir", "", "Output directory")
	seed := fs.Int64("seed", 42, "Random seed for Monte Carlo")
	if err := fs.Parse(args); err != nil {
		r.printfErr("run flag parse error: %v\n", err)
		return ExitError
	}
	if *modelPath == "" || *policyPath == "" || *outDir == "" {
		r.printfErr("run requires --model, --policy, and --out-dir\n")
		return ExitError
	}

	outputModelPath := filepath.Join(*outDir, "model.json")
	reportPath := filepath.Join(*outDir, "report.json")
	summaryPath := filepath.Join(*outDir, "summary.md")

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		r.printfErr("create out dir: %v\n", err)
		return ExitError
	}

	mdl, policy, rep, eval, err := executeSimulation(*modelPath, *policyPath, *seed, *journeysPath)
	if err != nil {
		r.printfErr("run error: %v\n", err)
		return ExitError
	}
	if err := model.WriteToFile(outputModelPath, mdl); err != nil {
		r.printfErr("write model: %v\n", err)
		return ExitError
	}
	if err := report.WriteJSON(reportPath, rep); err != nil {
		r.printfErr("write report: %v\n", err)
		return ExitError
	}
	if err := report.WriteSummaryMarkdown(summaryPath, rep); err != nil {
		r.printfErr("write summary: %v\n", err)
		return ExitError
	}

	r.printf("model written: %s\n", outputModelPath)
	r.printf("report written: %s\n", reportPath)
	r.printf("summary written: %s\n", summaryPath)
	r.printf("decision: %s\n", eval.Decision)
	r.printf("contract: %s@%s\n", mdl.Metadata.Schema.Name, mdl.Metadata.Schema.Version)
	r.printf("policy mode: %s\n", policy.Mode)
	if strings.TrimSpace(*journeysPath) != "" {
		r.printf("journeys override: %s\n", *journeysPath)
	}
	return decisionExitCode(eval.Decision)
}

func executeSimulation(modelPath, policyPath string, seed int64, journeysPath string) (model.ResilienceModel, config.Policy, report.Report, gate.Evaluation, error) {
	mdl, err := model.LoadFromFile(modelPath)
	if err != nil {
		return model.ResilienceModel{}, config.Policy{}, report.Report{}, gate.Evaluation{}, err
	}
	policy, err := config.LoadPolicy(policyPath)
	if err != nil {
		return model.ResilienceModel{}, config.Policy{}, report.Report{}, gate.Evaluation{}, err
	}

	var journeyOverrides map[string][][]string
	if strings.TrimSpace(journeysPath) != "" {
		journeyOverrides, err = journeys.Load(journeysPath)
		if err != nil {
			return model.ResilienceModel{}, config.Policy{}, report.Report{}, gate.Evaluation{}, err
		}
		if err := journeys.ValidateAgainstModel(journeyOverrides, mdl); err != nil {
			return model.ResilienceModel{}, config.Policy{}, report.Report{}, gate.Evaluation{}, err
		}
	}

	params := simulation.Params{
		Trials:             policy.Trials,
		Seed:               seed,
		FailureProbability: policy.FailureProbability,
		JourneyOverrides:   journeyOverrides,
	}
	simOutput, err := simulation.Run(mdl, params)
	if err != nil {
		return model.ResilienceModel{}, config.Policy{}, report.Report{}, gate.Evaluation{}, err
	}
	eval, err := gate.Evaluate(simOutput.EndpointAvailability, policy, "")
	if err != nil {
		return model.ResilienceModel{}, config.Policy{}, report.Report{}, gate.Evaluation{}, err
	}
	rep := report.Compose(simOutput, eval, params, mdl.Metadata.Confidence)
	return mdl, policy, rep, eval, nil
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
	fmt.Fprintln(r.stdout, "  sheaft simulate --model <model.json> --policy <policy.yaml> --out <report.json> [--journeys <journeys.json>] --seed <int>")
	fmt.Fprintln(r.stdout, "  sheaft gate --report <report.json> --policy <policy.yaml> --mode warn|fail|report")
	fmt.Fprintln(r.stdout, "  sheaft run --model <model.json> --policy <policy.yaml> --out-dir <dir> [--journeys <journeys.json>] --seed <int>")
}

func (r Runner) printf(format string, args ...any) {
	fmt.Fprintf(r.stdout, format, args...)
}

func (r Runner) printfErr(format string, args ...any) {
	fmt.Fprintf(r.stderr, format, args...)
}

var ErrUnsupportedCommand = errors.New("unsupported command")
