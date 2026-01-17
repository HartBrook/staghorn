// Package cli implements the staghorn command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time.
	Version = "dev"

	// Output helpers.
	successIcon = color.New(color.FgGreen).Sprint("✓")
	warningIcon = color.New(color.FgYellow).Sprint("⚠")
	errorIcon   = color.New(color.FgRed).Sprint("✗")

	success = color.New(color.FgGreen).SprintFunc()
	warning = color.New(color.FgYellow).SprintFunc()
	info    = color.New(color.FgCyan).SprintFunc()
	dim     = color.New(color.Faint).SprintFunc()
)

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "staghorn",
		Short: "A shared team layer for Claude Code",
		Long: `Staghorn manages hierarchical CLAUDE.md configurations.

It fetches shared team configs from GitHub, layers personal and project-level
configs on top, and outputs a merged file for Claude to consume.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add subcommands
	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewSyncCmd())
	rootCmd.AddCommand(NewSearchCmd())
	rootCmd.AddCommand(NewEditCmd())
	rootCmd.AddCommand(NewInfoCmd())
	rootCmd.AddCommand(NewProjectCmd())
	rootCmd.AddCommand(NewCommandsCmd())
	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewLanguagesCmd())
	rootCmd.AddCommand(NewTeamCmd())
	rootCmd.AddCommand(NewVersionCmd())

	return rootCmd
}

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("staghorn %s\n", Version)
		},
	}
}

// Execute runs the CLI.
func Execute() error {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		// Print error with hint if available
		if se, ok := err.(interface{ Hint() string }); ok {
			fmt.Fprintf(os.Stderr, "%s %s\n", errorIcon, err.Error())
			if hint := se.Hint(); hint != "" {
				fmt.Fprintf(os.Stderr, "  %s\n", dim(hint))
			}
		} else {
			fmt.Fprintf(os.Stderr, "%s %s\n", errorIcon, err.Error())
		}
		return err
	}
	return nil
}

// printSuccess prints a success message.
func printSuccess(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", successIcon, fmt.Sprintf(format, args...))
}

// printWarning prints a warning message.
func printWarning(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", warningIcon, fmt.Sprintf(format, args...))
}

// printError prints an error message.
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s %s\n", errorIcon, fmt.Sprintf(format, args...))
}

// printInfo prints an info line.
func printInfo(label, value string) {
	fmt.Printf("  %s: %s\n", dim(label), value)
}
