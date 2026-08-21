package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/tofchaliss/themis/benchmarks/internal/benchmark"
	"github.com/tofchaliss/themis/benchmarks/internal/evaluator"
	"github.com/tofchaliss/themis/benchmarks/internal/gate"
	"github.com/tofchaliss/themis/benchmarks/internal/report"
	"github.com/tofchaliss/themis/benchmarks/internal/runtime"
	"github.com/tofchaliss/themis/benchmarks/internal/validator"
)

const defaultEndpoint = "http://localhost:11434"

var (
	flagRoot     string
	flagDate     string
	flagEndpoint string
	flagTimeout  time.Duration
)

// date returns the --date flag, defaulting to today.
func date() string {
	if flagDate != "" {
		return flagDate
	}
	return time.Now().Format("2006-01-02")
}

// endpoint returns the --endpoint flag, falling back to $OLLAMA_HOST and
// then the local default.
func endpoint() string {
	if flagEndpoint != "" {
		return flagEndpoint
	}
	if env := os.Getenv("OLLAMA_HOST"); env != "" {
		return env
	}
	return defaultEndpoint
}

var rootCmd = &cobra.Command{
	Use:          "themis-bench",
	Short:        "Themis Benchmark Suite",
	SilenceUsage: true,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if flagDate == "" {
			return nil
		}
		if _, err := time.Parse("2006-01-02", flagDate); err != nil {
			return fmt.Errorf("invalid --date %q: expected YYYY-MM-DD", flagDate)
		}
		return nil
	},
}

var runCmd = &cobra.Command{
	Use:   "run MODEL",
	Short: "Run all benchmark definitions against a model",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		ctx, stop := signal.NotifyContext(
			cmd.Context(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer stop()

		registry, err := runtime.LoadRegistry(flagRoot)
		if err != nil {
			return err
		}

		rt, modelName, options, err := registry.Resolve(args[0])
		if err != nil {
			return err
		}

		// CLI flags override the registry.
		switch r := rt.(type) {
		case *runtime.Ollama:
			if flagEndpoint != "" || os.Getenv("OLLAMA_HOST") != "" {
				r.Endpoint = endpoint()
			}
			r.Client.Timeout = flagTimeout
		case *runtime.OpenAI:
			if flagEndpoint != "" {
				r.Endpoint = flagEndpoint
			}
			r.Client.Timeout = flagTimeout
		}

		fmt.Printf(
			"Model %s via %s (temperature=%g seed=%d)\n",
			args[0],
			rt.Name(),
			options.Temperature,
			options.Seed,
		)

		return benchmark.RunAll(ctx, benchmark.RunConfig{
			Runtime: rt,
			Name:    args[0],
			Model:   modelName,
			Options: options,
			Root:    flagRoot,
			Date:    date(),
		})
	},
}

var evaluateCmd = &cobra.Command{
	Use:   "evaluate MODEL",
	Short: "Evaluate benchmark runs",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		return evaluator.EvaluateAll(
			flagRoot,
			date(),
			args[0],
		)
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate MODEL",
	Short: "Validate benchmark responses",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		return validator.ValidateAll(
			flagRoot,
			date(),
			args[0],
		)
	},
}

var reportCmd = &cobra.Command{
	Use:   "report MODEL",
	Short: "Generate benchmark report",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		benchmarks, err := report.LoadResponses(
			flagRoot,
			date(),
			args[0],
		)
		if err != nil {
			return err
		}

		r := report.Generate(
			args[0],
			date(),
			benchmarks,
		)

		if err := report.Write(
			flagRoot,
			date(),
			r,
		); err != nil {
			return err
		}

		fmt.Printf(
			"✓ Report written: %s\n",
			filepath.Join(flagRoot, "reports", date(), args[0]+".md"),
		)

		return nil
	},
}

var (
	flagBaseline string
	flagMaxDrop  int
)

var gateCmd = &cobra.Command{
	Use:   "gate MODEL",
	Short: "Fail if a model's scores regressed against a baseline date",
	Long: "Compare the model's validation scores on --date (default " +
		"today) against --baseline. Exits non-zero when a benchmark " +
		"disappeared or the average score dropped more than --max-drop " +
		"points. Intended as a CI quality gate for model, prompt, or " +
		"quantization changes.",
	Args: cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		if _, err := time.Parse("2006-01-02", flagBaseline); err != nil {
			return fmt.Errorf(
				"invalid --baseline %q: expected YYYY-MM-DD",
				flagBaseline,
			)
		}

		result, err := gate.Compare(
			flagRoot,
			args[0],
			flagBaseline,
			date(),
		)
		if err != nil {
			return err
		}

		fmt.Print(gate.Report(result, flagMaxDrop))

		if !result.Pass(flagMaxDrop) {
			// The report already explains the failure.
			cmd.SilenceErrors = true
			return fmt.Errorf("gate failed")
		}

		return nil
	},
}

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare all benchmarked models across dates",
	Long: "Discover every evaluated run under responses/ and generate a " +
		"cross-model comparison report with per-benchmark scores and " +
		"score history.",
	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {

		c, err := report.Compare(flagRoot)
		if err != nil {
			return err
		}

		outputFile, err := report.WriteComparison(flagRoot, c)
		if err != nil {
			return err
		}

		for _, s := range c.Skipped {
			fmt.Fprintf(os.Stderr, "⚠ skipped %s\n", s)
		}

		fmt.Printf("✓ Comparison written: %s\n", outputFile)

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&flagRoot,
		"root",
		".",
		"benchmark suite root directory",
	)

	rootCmd.PersistentFlags().StringVar(
		&flagDate,
		"date",
		"",
		"date of the benchmark run (YYYY-MM-DD, default today)",
	)

	runCmd.Flags().StringVar(
		&flagEndpoint,
		"endpoint",
		"",
		"Ollama endpoint (default $OLLAMA_HOST or "+defaultEndpoint+")",
	)

	runCmd.Flags().DurationVar(
		&flagTimeout,
		"timeout",
		runtime.DefaultTimeout,
		"per-benchmark generation timeout",
	)

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(evaluateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(compareCmd)

	gateCmd.Flags().StringVar(
		&flagBaseline,
		"baseline",
		"",
		"baseline date to compare against (YYYY-MM-DD, required)",
	)
	gateCmd.MarkFlagRequired("baseline")

	gateCmd.Flags().IntVar(
		&flagMaxDrop,
		"max-drop",
		5,
		"maximum allowed drop of the average score, in points",
	)

	rootCmd.AddCommand(gateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
