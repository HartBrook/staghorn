package actions

import (
	"fmt"
	"sort"
)

// Registry manages actions from multiple sources with precedence handling.
// Precedence (highest to lowest): project > personal > team
type Registry struct {
	actions  map[string]*Action // name -> action (highest precedence wins)
	bySource map[Source][]*Action
}

// NewRegistry creates an empty action registry.
func NewRegistry() *Registry {
	return &Registry{
		actions:  make(map[string]*Action),
		bySource: make(map[Source][]*Action),
	}
}

// Add adds an action to the registry.
// If an action with the same name exists from a lower precedence source, it's overridden.
func (r *Registry) Add(action *Action) {
	r.bySource[action.Source] = append(r.bySource[action.Source], action)

	// Check precedence before overriding
	existing, exists := r.actions[action.Name]
	if !exists || sourcePrecedence(action.Source) > sourcePrecedence(existing.Source) {
		r.actions[action.Name] = action
	}
}

// AddAll adds multiple actions to the registry.
func (r *Registry) AddAll(actions []*Action) {
	for _, action := range actions {
		r.Add(action)
	}
}

// Get returns an action by name (highest precedence version).
func (r *Registry) Get(name string) *Action {
	return r.actions[name]
}

// All returns all unique actions (highest precedence version of each).
func (r *Registry) All() []*Action {
	actions := make([]*Action, 0, len(r.actions))
	for _, action := range r.actions {
		actions = append(actions, action)
	}
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Name < actions[j].Name
	})
	return actions
}

// BySource returns all actions from a specific source.
func (r *Registry) BySource(source Source) []*Action {
	// Make a copy to avoid mutating the original slice during sort
	original := r.bySource[source]
	actions := make([]*Action, len(original))
	copy(actions, original)
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Name < actions[j].Name
	})
	return actions
}

// ByTag returns all actions that have a specific tag.
func (r *Registry) ByTag(tag string) []*Action {
	var result []*Action
	for _, action := range r.actions {
		for _, t := range action.Tags {
			if t == tag {
				result = append(result, action)
				break
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Names returns all action names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.actions))
	for name := range r.actions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the total number of unique actions.
func (r *Registry) Count() int {
	return len(r.actions)
}

// CountBySource returns counts per source.
func (r *Registry) CountBySource() map[Source]int {
	counts := make(map[Source]int)
	for source, actions := range r.bySource {
		counts[source] = len(actions)
	}
	return counts
}

// IsOverridden checks if an action from a lower source is overridden.
func (r *Registry) IsOverridden(name string, source Source) bool {
	action := r.actions[name]
	if action == nil {
		return false
	}
	return action.Source != source
}

// GetAllVersions returns all versions of an action across sources.
func (r *Registry) GetAllVersions(name string) []*Action {
	var versions []*Action
	for _, actions := range r.bySource {
		for _, action := range actions {
			if action.Name == name {
				versions = append(versions, action)
			}
		}
	}
	// Sort by precedence (highest first)
	sort.Slice(versions, func(i, j int) bool {
		return sourcePrecedence(versions[i].Source) > sourcePrecedence(versions[j].Source)
	})
	return versions
}

// sourcePrecedence returns the precedence level of a source.
// Higher number = higher precedence.
func sourcePrecedence(s Source) int {
	switch s {
	case SourceProject:
		return 3
	case SourcePersonal:
		return 2
	case SourceTeam:
		return 1
	default:
		return 0
	}
}

// LoadRegistry creates a registry by loading actions from all sources.
func LoadRegistry(teamDir, personalDir, projectDir string) (*Registry, error) {
	registry := NewRegistry()

	// Load in precedence order (lowest first, so higher precedence overwrites)
	sources := []struct {
		dir    string
		source Source
	}{
		{teamDir, SourceTeam},
		{personalDir, SourcePersonal},
		{projectDir, SourceProject},
	}

	for _, s := range sources {
		if s.dir == "" {
			continue
		}
		actions, err := LoadFromDirectory(s.dir, s.source)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s actions: %w", s.source.Label(), err)
		}
		registry.AddAll(actions)
	}

	return registry, nil
}
