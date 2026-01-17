package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
)

// OutputFormat specifies the output format for results.
type OutputFormat string

const (
	OutputFormatTable  OutputFormat = "table"
	OutputFormatJSON   OutputFormat = "json"
	OutputFormatGitHub OutputFormat = "github"
)

// Formatter formats eval results for output.
type Formatter struct {
	Writer io.Writer
	Format OutputFormat
	Debug  bool // Show full LLM responses for failures
}

// NewFormatter creates a new result formatter.
func NewFormatter(w io.Writer, format OutputFormat) *Formatter {
	return &Formatter{
		Writer: w,
		Format: format,
	}
}

// FormatResults formats multiple eval results.
func (f *Formatter) FormatResults(results []*RunResult) error {
	switch f.Format {
	case OutputFormatJSON:
		return f.formatJSON(results)
	case OutputFormatGitHub:
		return f.formatGitHub(results)
	default:
		return f.formatTable(results)
	}
}

// formatTable outputs results in human-readable table format.
func (f *Formatter) formatTable(results []*RunResult) error {
	successIcon := color.New(color.FgGreen).Sprint("✓")
	failIcon := color.New(color.FgRed).Sprint("✗")
	dimColor := color.New(color.Faint)
	boldColor := color.New(color.Bold)

	totalTests := 0
	totalPassed := 0
	totalFailed := 0

	for _, result := range results {
		// Print eval name
		fmt.Fprintf(f.Writer, "\n%s\n", boldColor.Sprint(result.EvalName))

		// Print individual test results
		for _, test := range result.Results {
			icon := successIcon
			if !test.Passed {
				icon = failIcon
			}

			// Format duration if available
			durationStr := ""
			if test.Duration > 0 {
				durationStr = dimColor.Sprintf(" (%s)", test.Duration.Round(100*time.Millisecond).String())
			}

			name := test.Name
			if test.Description != "" && test.Description != test.Name {
				name = test.Description
			}

			fmt.Fprintf(f.Writer, "  %s %s%s\n", icon, name, durationStr)

			// Show error details for failures
			if !test.Passed && test.Error != "" {
				for _, line := range strings.Split(test.Error, "; ") {
					fmt.Fprintf(f.Writer, "    %s\n", dimColor.Sprint(line))
				}
			}

			// Show full LLM response in debug mode for failures
			if f.Debug && !test.Passed && test.Output != "" {
				fmt.Fprintf(f.Writer, "\n    %s\n", dimColor.Sprint("─── Claude Response ───"))
				// Indent each line of the response
				for _, line := range strings.Split(test.Output, "\n") {
					fmt.Fprintf(f.Writer, "    %s\n", line)
				}
				fmt.Fprintf(f.Writer, "    %s\n\n", dimColor.Sprint("───────────────────────"))
			}
		}

		totalTests += result.TotalTests
		totalPassed += result.Passed
		totalFailed += result.Failed
	}

	// Print summary
	fmt.Fprintf(f.Writer, "\n")
	passRate := 0.0
	if totalTests > 0 {
		passRate = float64(totalPassed) / float64(totalTests) * 100
	}

	statusColor := color.New(color.FgGreen)
	if totalFailed > 0 {
		statusColor = color.New(color.FgRed)
	}

	fmt.Fprintf(f.Writer, "%s %d/%d passed (%.0f%%)\n",
		statusColor.Sprint("Results:"),
		totalPassed,
		totalTests,
		passRate,
	)

	if totalFailed > 0 {
		fmt.Fprintf(f.Writer, "  %d failure(s)\n", totalFailed)
	}

	// Show debug directories if available
	if f.Debug {
		var debugDirs []string
		for _, result := range results {
			if result.DebugDir != "" {
				debugDirs = append(debugDirs, result.DebugDir)
			}
		}
		if len(debugDirs) > 0 {
			fmt.Fprintf(f.Writer, "\n%s\n", dimColor.Sprint("Debug artifacts preserved at:"))
			for _, dir := range debugDirs {
				fmt.Fprintf(f.Writer, "  %s\n", dir)
			}
			fmt.Fprintf(f.Writer, "%s\n", dimColor.Sprint("Contains: promptfooconfig.yaml, output.json"))
		}
	}

	return nil
}

// JSONOutput represents the JSON output format.
type JSONOutput struct {
	Summary struct {
		Total    int     `json:"total"`
		Passed   int     `json:"passed"`
		Failed   int     `json:"failed"`
		PassRate float64 `json:"passRate"`
	} `json:"summary"`
	Results []JSONEvalResult `json:"results"`
}

// JSONEvalResult represents a single eval's results in JSON.
type JSONEvalResult struct {
	Eval     string           `json:"eval"`
	Total    int              `json:"total"`
	Passed   int              `json:"passed"`
	Failed   int              `json:"failed"`
	Duration string           `json:"duration"`
	Tests    []JSONTestResult `json:"tests"`
}

// JSONTestResult represents a single test result in JSON.
type JSONTestResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Duration string `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
	Output   string `json:"output,omitempty"`
}

// formatJSON outputs results in JSON format for CI/CD.
func (f *Formatter) formatJSON(results []*RunResult) error {
	output := JSONOutput{}

	for _, result := range results {
		evalResult := JSONEvalResult{
			Eval:     result.EvalName,
			Total:    result.TotalTests,
			Passed:   result.Passed,
			Failed:   result.Failed,
			Duration: result.Duration.String(),
		}

		for _, test := range result.Results {
			testResult := JSONTestResult{
				Name:   test.Name,
				Passed: test.Passed,
			}
			if test.Duration > 0 {
				testResult.Duration = test.Duration.String()
			}
			if !test.Passed && test.Error != "" {
				testResult.Error = test.Error
			}
			evalResult.Tests = append(evalResult.Tests, testResult)
		}

		output.Results = append(output.Results, evalResult)
		output.Summary.Total += result.TotalTests
		output.Summary.Passed += result.Passed
		output.Summary.Failed += result.Failed
	}

	if output.Summary.Total > 0 {
		output.Summary.PassRate = float64(output.Summary.Passed) / float64(output.Summary.Total)
	}

	encoder := json.NewEncoder(f.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// formatGitHub outputs results in GitHub Actions annotation format.
func (f *Formatter) formatGitHub(results []*RunResult) error {
	for _, result := range results {
		for _, test := range result.Results {
			if !test.Passed {
				// GitHub Actions annotation format
				// ::error file={name},line={line}::{message}
				fmt.Fprintf(f.Writer, "::error file=%s::Test '%s' failed: %s\n",
					result.EvalName+".yaml",
					test.Name,
					strings.ReplaceAll(test.Error, "\n", " "),
				)
			}
		}
	}

	// Also output summary
	totalTests := 0
	totalFailed := 0
	for _, result := range results {
		totalTests += result.TotalTests
		totalFailed += result.Failed
	}

	if totalFailed > 0 {
		fmt.Fprintf(f.Writer, "::error::Eval failed: %d/%d tests failed\n", totalFailed, totalTests)
	} else {
		fmt.Fprintf(f.Writer, "::notice::All %d eval tests passed\n", totalTests)
	}

	return nil
}

// Summary generates a summary of results.
type Summary struct {
	TotalEvals  int
	TotalTests  int
	Passed      int
	Failed      int
	PassRate    float64
	Duration    float64
	FailedTests []FailedTest
}

// FailedTest holds info about a failed test.
type FailedTest struct {
	Eval  string
	Test  string
	Error string
}

// Summarize generates a summary from results.
func Summarize(results []*RunResult) *Summary {
	s := &Summary{}

	for _, result := range results {
		s.TotalEvals++
		s.TotalTests += result.TotalTests
		s.Passed += result.Passed
		s.Failed += result.Failed
		s.Duration += result.Duration.Seconds()

		for _, test := range result.Results {
			if !test.Passed {
				s.FailedTests = append(s.FailedTests, FailedTest{
					Eval:  result.EvalName,
					Test:  test.Name,
					Error: test.Error,
				})
			}
		}
	}

	if s.TotalTests > 0 {
		s.PassRate = float64(s.Passed) / float64(s.TotalTests) * 100
	}

	return s
}
