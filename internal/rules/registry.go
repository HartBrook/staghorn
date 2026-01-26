package rules

import "sort"

// Registry manages rules from multiple sources with precedence handling.
// Precedence (highest to lowest): project > personal > team
// Key is the relative path (e.g., "api/rest.md", "security.md")
type Registry struct {
	rules    map[string]*Rule // relPath -> rule (highest precedence wins)
	bySource map[Source][]*Rule
}

// NewRegistry creates an empty rule registry.
func NewRegistry() *Registry {
	return &Registry{
		rules:    make(map[string]*Rule),
		bySource: make(map[Source][]*Rule),
	}
}

// Add adds a rule to the registry.
// Higher precedence sources (project > personal > team) override lower ones.
func (r *Registry) Add(rule *Rule) {
	existing, exists := r.rules[rule.RelPath]

	// Determine if new rule should override existing
	shouldOverride := !exists || sourcePrecedence(rule.Source) > sourcePrecedence(existing.Source)

	if shouldOverride {
		r.rules[rule.RelPath] = rule
	}

	// Always track by source
	r.bySource[rule.Source] = append(r.bySource[rule.Source], rule)
}

// sourcePrecedence returns the precedence level for a source.
// Higher values = higher precedence (wins in conflicts).
func sourcePrecedence(s Source) int {
	switch s {
	case SourceTeam:
		return 1
	case SourceStarter:
		return 1 // Same as team
	case SourcePersonal:
		return 2
	case SourceProject:
		return 3
	default:
		return 0
	}
}

// All returns all unique rules (highest precedence version of each),
// sorted by relative path for deterministic ordering.
func (r *Registry) All() []*Rule {
	result := make([]*Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		result = append(result, rule)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].RelPath < result[j].RelPath
	})
	return result
}

// Get returns a rule by its relative path.
func (r *Registry) Get(relPath string) *Rule {
	return r.rules[relPath]
}

// BySource returns all rules from a specific source.
func (r *Registry) BySource(source Source) []*Rule {
	return r.bySource[source]
}

// Count returns the number of unique rules.
func (r *Registry) Count() int {
	return len(r.rules)
}

// LoadRegistry creates a registry by loading rules from all sources.
// Empty string for any directory means that source is not available.
func LoadRegistry(teamDir, personalDir, projectDir string) (*Registry, error) {
	registry := NewRegistry()

	// Load team rules (lowest precedence)
	if teamDir != "" {
		teamRules, err := LoadFromDirectory(teamDir, SourceTeam)
		if err != nil {
			return nil, err
		}
		for _, rule := range teamRules {
			registry.Add(rule)
		}
	}

	// Load personal rules (middle precedence)
	if personalDir != "" {
		personalRules, err := LoadFromDirectory(personalDir, SourcePersonal)
		if err != nil {
			return nil, err
		}
		for _, rule := range personalRules {
			registry.Add(rule)
		}
	}

	// Load project rules (highest precedence)
	if projectDir != "" {
		projectRules, err := LoadFromDirectory(projectDir, SourceProject)
		if err != nil {
			return nil, err
		}
		for _, rule := range projectRules {
			registry.Add(rule)
		}
	}

	return registry, nil
}
