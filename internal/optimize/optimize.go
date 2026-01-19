package optimize

import (
	"context"
	"log"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/errors"
)

// Options controls optimization behavior.
type Options struct {
	Target        int    // Target token count (0 = auto ~50% reduction)
	Deterministic bool   // Only apply deterministic transforms (no LLM)
	Force         bool   // Re-optimize even if cache is fresh
	NoCache       bool   // Skip cache read/write
	Model         string // Model to use for optimization
}

// Result contains the optimization outcome.
type Result struct {
	OriginalContent  string
	OptimizedContent string
	Stats            TokenStats
	PreprocessStats  PreprocessStats
	PreservedAnchors []string
	MissingAnchors   []string // All missing (strict + soft), for backward compatibility
	MissingStrict    []string // Strict anchors missing (file paths, commands) - causes failure
	MissingSoft      []string // Soft anchors missing (tool names) - warnings only
	FromCache        bool
	Deterministic    bool
}

// Optimizer handles the optimization pipeline.
type Optimizer struct {
	paths       *config.Paths
	cache       *OptimizationCache
	client      *Client
	clientModel string // tracks which model the client was created with
}

// NewOptimizer creates a new optimizer.
func NewOptimizer(paths *config.Paths) *Optimizer {
	return &Optimizer{
		paths: paths,
		cache: NewOptimizationCache(paths),
	}
}

// Optimize runs the full optimization pipeline on content.
func (o *Optimizer) Optimize(ctx context.Context, content string, owner, repo string, opts Options) (*Result, error) {
	result := &Result{
		OriginalContent: content,
		Deterministic:   opts.Deterministic,
	}

	// Calculate original tokens
	originalTokens := CountTokens(content)
	result.Stats.Before = originalTokens

	// Check cache unless forced or disabled
	if !opts.Force && !opts.NoCache {
		sourceHash := HashContent(content)
		cached, meta, err := o.cache.Read(owner, repo)
		// Cache hit requires matching source hash and compatible model/deterministic settings
		if err == nil && cached != "" && meta != nil && meta.SourceHash == sourceHash {
			// Model must match if specified, or cache must be from same model type
			modelMatches := opts.Model == "" || opts.Model == meta.Model
			deterministicMatches := opts.Deterministic == meta.Deterministic
			if modelMatches && deterministicMatches {
				result.OptimizedContent = cached
				result.Stats.After = meta.OptimizedTokens
				result.FromCache = true
				result.Deterministic = meta.Deterministic
				return result, nil
			}
		}
	}

	// Step 1: Pre-processing (always runs)
	preprocessed, preprocessStats := Preprocess(content)
	result.PreprocessStats = preprocessStats

	// Step 2: Extract anchors for validation
	anchors := ExtractAnchors(content)

	// Step 3: Determine target tokens
	targetTokens := opts.Target
	if targetTokens == 0 {
		// Default to ~50% reduction
		targetTokens = originalTokens / 2
	}

	var optimized string

	if opts.Deterministic {
		// Skip LLM, use only preprocessed content
		optimized = preprocessed
	} else {
		// Step 4: LLM optimization
		client, err := o.getClient(opts.Model)
		if err != nil {
			return nil, err
		}

		optimized, err = client.Optimize(ctx, preprocessed, targetTokens, anchors)
		if err != nil {
			return nil, err
		}
	}

	// Step 5: Validate anchors with categorized strictness
	validation := ValidateAnchorsCategorized(content, optimized)
	result.PreservedAnchors = validation.Preserved
	result.MissingAnchors = validation.AllMissing()
	result.MissingStrict = validation.MissingStrict
	result.MissingSoft = validation.MissingSoft

	// Only fail on strict anchor violations (file paths, commands)
	// Soft violations (tool names) are warnings only
	if validation.HasStrictFailures() && !opts.Force {
		return nil, errors.ValidationFailed(validation.MissingStrict)
	}

	result.OptimizedContent = optimized
	result.Stats.After = CountTokens(optimized)

	// Save to cache unless disabled
	if !opts.NoCache {
		meta := &OptimizationMeta{
			SourceHash:      HashContent(content),
			OptimizedAt:     time.Now(),
			OriginalTokens:  result.Stats.Before,
			OptimizedTokens: result.Stats.After,
			Model:           opts.Model,
			Deterministic:   opts.Deterministic,
		}
		if err := o.cache.Write(owner, repo, optimized, meta); err != nil {
			log.Printf("debug: failed to write optimization cache: %v", err)
		}
	}

	return result, nil
}

// getClient returns a Claude API client, creating one if needed.
// If the requested model differs from the cached client's model, a new client is created.
func (o *Optimizer) getClient(model string) (*Client, error) {
	// Normalize empty model to default
	requestedModel := model
	if requestedModel == "" {
		requestedModel = defaultModel
	}

	// Reuse existing client if model matches
	if o.client != nil && o.clientModel == requestedModel {
		return o.client, nil
	}

	var opts []ClientOption
	if model != "" {
		opts = append(opts, WithModel(model))
	}

	client, err := NewClient(opts...)
	if err != nil {
		return nil, err
	}

	o.client = client
	o.clientModel = requestedModel
	return client, nil
}

// SetClient allows injecting a client for testing.
func (o *Optimizer) SetClient(client *Client) {
	o.client = client
}
