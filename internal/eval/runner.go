package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner executes evals using Promptfoo.
type Runner struct {
	// WorkDir is the working directory for Promptfoo.
	WorkDir string

	// Verbose enables verbose output.
	Verbose bool

	// Debug enables debug mode (preserves temp files, shows full responses).
	Debug bool

	// Timeout is the timeout for eval execution.
	Timeout time.Duration
}

// DefaultTimeout is the default timeout for eval execution.
const DefaultTimeout = 10 * time.Minute

// NewRunner creates a new eval runner.
func NewRunner(workDir string) *Runner {
	return &Runner{
		WorkDir: workDir,
		Timeout: DefaultTimeout,
	}
}

// CheckPromptfoo verifies that Promptfoo is installed and accessible.
func CheckPromptfoo() error {
	cmd := exec.Command("npx", "promptfoo", "--version")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("promptfoo not found. Install with: npm install -g promptfoo")
	}
	return nil
}

// RunConfig holds configuration for a single eval run.
type RunConfig struct {
	// Eval is the eval definition to run.
	Eval *Eval

	// ClaudeConfig is the merged CLAUDE.md content to use as system prompt.
	ClaudeConfig string
}

// RunResult holds the result of an eval run.
type RunResult struct {
	EvalName   string
	TotalTests int
	Passed     int
	Failed     int
	Duration   time.Duration
	Results    []TestResult
	RawOutput  string
	DebugDir   string // Set when debug mode is enabled
}

// TestResult holds the result of a single test.
type TestResult struct {
	Name        string
	Description string
	Passed      bool
	Duration    time.Duration
	Error       string
	Output      string
}

// Run executes a single eval and returns results.
func (r *Runner) Run(ctx context.Context, cfg RunConfig) (*RunResult, error) {
	// Create temp directory for this run
	tempDir, err := os.MkdirTemp(r.WorkDir, "eval-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	if !r.Debug {
		defer os.RemoveAll(tempDir)
	}

	// Generate Promptfoo config
	promptfooConfig, err := GeneratePromptfooConfig(cfg.Eval, cfg.ClaudeConfig, GenerateOptions{
		OutputDir:   tempDir,
		ResultsPath: filepath.Join(tempDir, "results.json"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate promptfoo config: %w", err)
	}

	// Write config to file
	configPath := filepath.Join(tempDir, "promptfooconfig.yaml")
	if err := WritePromptfooConfig(promptfooConfig, configPath); err != nil {
		return nil, fmt.Errorf("failed to write promptfoo config: %w", err)
	}

	// Prepare output path
	outputPath := filepath.Join(tempDir, "output.json")

	// Build command
	args := []string{
		"promptfoo", "eval",
		"--config", configPath,
		"--output", outputPath,
		"--no-progress-bar",
	}

	if r.Verbose {
		args = append(args, "--verbose")
	}

	// Apply timeout (use default if not set)
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	// Run promptfoo
	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	// Read results even if command failed (partial results may exist)
	result := &RunResult{
		EvalName:  cfg.Eval.Name,
		Duration:  duration,
		RawOutput: stdout.String() + stderr.String(),
	}

	if r.Debug {
		result.DebugDir = tempDir
	}

	// Try to parse output file
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		if r.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: could not read output file %s: %v\n", outputPath, err)
		}
	} else {
		if parseErr := r.parseOutput(outputData, result); parseErr != nil {
			// Log but don't fail - we might still have useful info
			if r.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse output: %v\n", parseErr)
			}
		}
	}

	// If we couldn't parse output, try to infer from command result
	if result.TotalTests == 0 {
		result.TotalTests = len(cfg.Eval.Tests)
		if runErr != nil {
			result.Failed = result.TotalTests
			if r.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: promptfoo command failed: %v\nOutput: %s\n", runErr, result.RawOutput)
			}
		} else {
			result.Passed = result.TotalTests
		}
	}

	// Return error only if command failed and we have no results
	if runErr != nil && result.TotalTests == 0 {
		return result, fmt.Errorf("promptfoo execution failed: %w\nOutput: %s", runErr, result.RawOutput)
	}

	return result, nil
}

// parseOutput parses Promptfoo JSON output.
func (r *Runner) parseOutput(data []byte, result *RunResult) error {
	var output PromptfooOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	result.TotalTests = 0
	result.Passed = 0
	result.Failed = 0

	for _, res := range output.Results {
		testResult := TestResult{
			Name:   res.Description,
			Passed: res.Success,
			Output: res.Response,
		}

		if !res.Success && len(res.AssertionResults) > 0 {
			// Collect failure reasons
			var failures []string
			for _, ar := range res.AssertionResults {
				if !ar.Pass {
					failures = append(failures, ar.Reason)
				}
			}
			testResult.Error = strings.Join(failures, "; ")
		}

		result.Results = append(result.Results, testResult)
		result.TotalTests++
		if res.Success {
			result.Passed++
		} else {
			result.Failed++
		}
	}

	return nil
}

// PromptfooOutput represents Promptfoo's JSON output format.
type PromptfooOutput struct {
	Results []PromptfooResult `json:"results"`
	Stats   struct {
		Successes int `json:"successes"`
		Failures  int `json:"failures"`
	} `json:"stats"`
}

// PromptfooResult represents a single result in Promptfoo output.
type PromptfooResult struct {
	Description      string                     `json:"description"`
	Success          bool                       `json:"success"`
	Response         string                     `json:"response"`
	AssertionResults []PromptfooAssertionResult `json:"gradingResult"`
}

// PromptfooAssertionResult represents an assertion result.
type PromptfooAssertionResult struct {
	Pass   bool   `json:"pass"`
	Reason string `json:"reason"`
}

// RunAll executes multiple evals and returns all results.
func (r *Runner) RunAll(ctx context.Context, evals []*Eval, claudeConfig string) ([]*RunResult, error) {
	var results []*RunResult

	for i, e := range evals {
		fmt.Printf("  [%d/%d] Running %s (%d tests)...\n", i+1, len(evals), e.Name, len(e.Tests))

		result, err := r.Run(ctx, RunConfig{
			Eval:         e,
			ClaudeConfig: claudeConfig,
		})
		if err != nil {
			// Continue with other evals even if one fails
			if result == nil {
				result = &RunResult{
					EvalName:   e.Name,
					TotalTests: len(e.Tests),
					Failed:     len(e.Tests),
				}
			}
		}
		results = append(results, result)
	}

	return results, nil
}
