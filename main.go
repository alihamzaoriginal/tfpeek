package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alihamzaoriginal/tfpeek/internal/summary"
)

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

func run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(argv) >= 2 {
		switch argv[1] {
		case "-h", "--help":
			printUsage(stdout)
			return 0
		}
	}
	args := argv[1:]
	if len(args) < 2 || args[0] != "plan" {
		fmt.Fprintf(stderr, "usage: tfpeek plan (apply|destroy) [terraform/tofu plan options…]\n")
		return 2
	}
	mode := args[1]
	switch mode {
	case "apply", "destroy":
		cli, err := resolveCLI()
		if err != nil {
			fmt.Fprintf(stderr, "tfpeek: %v\n", err)
			return 1
		}
		return runPlanDryRun(cli, mode, args[2:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "usage: tfpeek plan (apply|destroy) [terraform/tofu plan options…]\n")
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `usage: tfpeek plan (apply|destroy) [terraform/tofu plan options…]

Dry-run summary from terraform or tofu plan JSON (never runs apply or destroy).

Environment:
  TFPEEK_CLI           terraform or tofu (optional; see README)
  TFPEEK_VERBOSE_PLAN  set (non-empty) to stream full plan stdout

`)
}

func runPlanDryRun(cli, mode string, planArgs []string, stdin io.Reader, stdout, stderr io.Writer) int {
	planFile, injected, err := resolvePlanOut(planArgs)
	if err != nil {
		fmt.Fprintf(stderr, "tfpeek: %v\n", err)
		return 1
	}
	if injected {
		defer func() { _ = os.Remove(planFile) }()
	}

	var fullArgs []string
	switch mode {
	case "apply":
		fullArgs = append([]string{"plan"}, planArgs...)
	case "destroy":
		fullArgs = append([]string{"plan", "-destroy"}, stripDestroyFlag(planArgs)...)
	default:
		return 2
	}
	if injected {
		fullArgs = append(fullArgs, "-out="+planFile)
	}

	code := runPlanCaptureExit(cli, fullArgs, stdin, planStdoutWriter(stdout), stderr)
	if code != 0 && code != 2 {
		return code
	}

	if summaryCode := writePlanSummary(stdout, stderr, planFile, mode, cli); summaryCode != 0 {
		return summaryCode
	}
	return code
}

func stripDestroyFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "-destroy" || a == "--destroy" {
			continue
		}
		out = append(out, a)
	}
	return out
}

// resolveCLI selects terraform or tofu:
//   - TFPEEK_CLI=tofu or terraform forces that binary (must exist on PATH)
//   - otherwise prefers terraform if found on PATH, else tofu
func resolveCLI() (string, error) {
	v := strings.TrimSpace(os.Getenv("TFPEEK_CLI"))
	if v != "" {
		if v != "terraform" && v != "tofu" {
			return "", fmt.Errorf("TFPEEK_CLI must be terraform or tofu, got %q", v)
		}
		if _, err := exec.LookPath(v); err != nil {
			return "", fmt.Errorf("%s (TFPEEK_CLI): not found on PATH", v)
		}
		return v, nil
	}
	if _, err := exec.LookPath("terraform"); err == nil {
		return "terraform", nil
	}
	if _, err := exec.LookPath("tofu"); err == nil {
		return "tofu", nil
	}
	return "", fmt.Errorf("neither terraform nor tofu found on PATH (install one or set TFPEEK_CLI)")
}

// planStdoutWriter sends plan stdout to Discard by default so huge human-readable
// diffs are not formatted and written to the terminal (often the dominant cost after the
// provider refresh). Set TFPEEK_VERBOSE_PLAN=1 to stream the normal plan output.
func planStdoutWriter(stdout io.Writer) io.Writer {
	if os.Getenv("TFPEEK_VERBOSE_PLAN") != "" {
		return stdout
	}
	return io.Discard
}

func runPlanCaptureExit(cli string, fullArgs []string, stdin io.Reader, planStdout, stderr io.Writer) int {
	cmd := exec.Command(cli, fullArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = planStdout
	cmd.Stderr = stderr
	exitErr := cmd.Run()
	code := 0
	if exitErr != nil {
		if ee, ok := exitErr.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			fmt.Fprintf(stderr, "tfpeek: %v\n", exitErr)
			return 1
		}
	}
	if code != 0 && code != 2 {
		return code
	}
	return code
}

func writePlanSummary(stdout, stderr io.Writer, planFile, mode, cli string) int {
	show := exec.Command(cli, "show", "-json", planFile)
	jsonOut, err := show.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(stderr, "tfpeek: %s show -json: %s\n", cli, strings.TrimSpace(string(ee.Stderr)))
			return ee.ExitCode()
		}
		fmt.Fprintf(stderr, "tfpeek: %s show -json: %v\n", cli, err)
		return 1
	}

	b, err := summary.Parse(strings.NewReader(string(jsonOut)))
	if err != nil {
		fmt.Fprintf(stderr, "tfpeek: parse plan JSON: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, strings.Repeat("─", 48))
	fmt.Fprint(stdout, summary.Format(b, mode, cli))
	fmt.Fprintln(stdout)
	return 0
}

// resolvePlanOut returns (planFilePath, injectedTemp, error).
// If the user already passed -out, we use that path and do not remove it.
func resolvePlanOut(args []string) (string, bool, error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-out" || a == "--out":
			if i+1 < len(args) {
				return filepath.Clean(args[i+1]), false, nil
			}
		case strings.HasPrefix(a, "-out="):
			return filepath.Clean(strings.TrimPrefix(a, "-out=")), false, nil
		case strings.HasPrefix(a, "--out="):
			return filepath.Clean(strings.TrimPrefix(a, "--out=")), false, nil
		}
	}
	f, err := os.CreateTemp("", "tfpeek-plan-*.tfplan")
	if err != nil {
		return "", false, err
	}
	path := f.Name()
	_ = f.Close()
	return path, true, nil
}
