package commands

import (
	"fmt"
	"sort"
)

// Registry manages commands from multiple sources with precedence handling.
// Precedence (highest to lowest): project > personal > team
type Registry struct {
	commands map[string]*Command // name -> command (highest precedence wins)
	bySource map[Source][]*Command
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]*Command),
		bySource: make(map[Source][]*Command),
	}
}

// Add adds a command to the registry.
// If a command with the same name exists from a lower precedence source, it's overridden.
func (r *Registry) Add(cmd *Command) {
	r.bySource[cmd.Source] = append(r.bySource[cmd.Source], cmd)

	// Check precedence before overriding
	existing, exists := r.commands[cmd.Name]
	if !exists || sourcePrecedence(cmd.Source) > sourcePrecedence(existing.Source) {
		r.commands[cmd.Name] = cmd
	}
}

// AddAll adds multiple commands to the registry.
func (r *Registry) AddAll(cmds []*Command) {
	for _, cmd := range cmds {
		r.Add(cmd)
	}
}

// Get returns a command by name (highest precedence version).
func (r *Registry) Get(name string) *Command {
	return r.commands[name]
}

// All returns all unique commands (highest precedence version of each).
func (r *Registry) All() []*Command {
	cmds := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name < cmds[j].Name
	})
	return cmds
}

// BySource returns all commands from a specific source.
func (r *Registry) BySource(source Source) []*Command {
	// Make a copy to avoid mutating the original slice during sort
	original := r.bySource[source]
	cmds := make([]*Command, len(original))
	copy(cmds, original)
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name < cmds[j].Name
	})
	return cmds
}

// ByTag returns all commands that have a specific tag.
func (r *Registry) ByTag(tag string) []*Command {
	var result []*Command
	for _, cmd := range r.commands {
		for _, t := range cmd.Tags {
			if t == tag {
				result = append(result, cmd)
				break
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Names returns all command names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the total number of unique commands.
func (r *Registry) Count() int {
	return len(r.commands)
}

// CountBySource returns counts per source.
func (r *Registry) CountBySource() map[Source]int {
	counts := make(map[Source]int)
	for source, cmds := range r.bySource {
		counts[source] = len(cmds)
	}
	return counts
}

// IsOverridden checks if a command from a lower source is overridden.
func (r *Registry) IsOverridden(name string, source Source) bool {
	cmd := r.commands[name]
	if cmd == nil {
		return false
	}
	return cmd.Source != source
}

// GetAllVersions returns all versions of a command across sources.
func (r *Registry) GetAllVersions(name string) []*Command {
	var versions []*Command
	for _, cmds := range r.bySource {
		for _, cmd := range cmds {
			if cmd.Name == name {
				versions = append(versions, cmd)
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

// LoadRegistry creates a registry by loading commands from all sources.
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
		cmds, err := LoadFromDirectory(s.dir, s.source)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s commands: %w", s.source.Label(), err)
		}
		registry.AddAll(cmds)
	}

	return registry, nil
}
