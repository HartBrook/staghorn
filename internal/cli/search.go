package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/HartBrook/staghorn/internal/github"
	"github.com/spf13/cobra"
)

const (
	// searchTimeout is the maximum time allowed for search operations.
	searchTimeout = 30 * time.Second
)

type searchOptions struct {
	tag      string
	language string
	limit    int
}

// NewSearchCmd creates the search command.
func NewSearchCmd() *cobra.Command {
	opts := &searchOptions{}

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for public staghorn configs",
		Long: `Search for public staghorn configurations on GitHub.

Searches for repositories with the 'staghorn-config' topic. Results are sorted
by star count.

You can filter by language or additional tags using the --lang and --tag flags.`,
		Example: `  staghorn search              # List all public configs
  staghorn search python       # Search for "python" in config name/description
  staghorn search --lang go    # Filter to configs supporting Go
  staghorn search --tag web    # Filter by topic/tag`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			return runSearch(query, opts)
		},
	}

	cmd.Flags().StringVar(&opts.tag, "tag", "", "Filter by topic/tag")
	cmd.Flags().StringVar(&opts.language, "lang", "", "Filter by language")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum results to show")

	return cmd
}

func runSearch(query string, opts *searchOptions) error {
	fmt.Println("Searching for public configs...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	defer cancel()

	// Try unauthenticated first for public repos
	client, err := github.NewUnauthenticatedClient()
	if err != nil {
		// Fall back to authenticated
		client, err = github.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
	}

	results, err := client.SearchConfigs(ctx, query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Apply filters
	if opts.language != "" {
		results = github.FilterByLanguage(results, opts.language)
	}
	if opts.tag != "" {
		results = github.FilterByTag(results, opts.tag)
	}

	if len(results) == 0 {
		fmt.Println("No configs found matching your criteria.")
		fmt.Println()
		fmt.Println("Tips:")
		fmt.Println("  - Try a broader search query")
		fmt.Println("  - Remove --lang or --tag filters")
		fmt.Println("  - Repos must have the 'staghorn-config' topic to be discoverable")
		return nil
	}

	// Limit results
	if opts.limit > 0 && len(results) > opts.limit {
		results = results[:opts.limit]
	}

	// Display results
	fmt.Printf("Found %d configs:\n\n", len(results))

	for i, r := range results {
		// Header: number, name, stars
		fmt.Printf("  %d. %s/%s", i+1, r.Owner, r.Repo)
		if r.Stars > 0 {
			fmt.Printf(" ★ %d", r.Stars)
		}
		fmt.Println()

		// Description
		if r.Description != "" {
			fmt.Printf("     %s\n", truncate(r.Description, 65))
		}

		// Topics (excluding staghorn-config)
		if len(r.Topics) > 0 {
			displayTopics := make([]string, 0, len(r.Topics))
			for _, t := range r.Topics {
				if t != "staghorn-config" {
					displayTopics = append(displayTopics, t)
				}
			}
			if len(displayTopics) > 0 {
				fmt.Printf("     %s\n", dim(joinTopics(displayTopics, 60)))
			}
		}

		fmt.Println()
	}

	fmt.Println("Install with:")
	fmt.Printf("  %s\n", info("staghorn init --from owner/repo"))

	return nil
}

// joinTopics joins topics with commas, truncating if too long.
func joinTopics(topics []string, maxLen int) string {
	result := ""
	for i, t := range topics {
		if i > 0 {
			result += ", "
		}
		if len(result)+len(t) > maxLen {
			result += "..."
			break
		}
		result += t
	}
	return result
}
