package skills

import (
	"fmt"
	"os"
	"sort"
)

// Registry manages skills from multiple sources with precedence handling.
// Precedence (highest to lowest): project > personal > team > starter
type Registry struct {
	skills   map[string]*Skill // name -> skill (highest precedence wins)
	bySource map[Source][]*Skill
}

// NewRegistry creates an empty skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills:   make(map[string]*Skill),
		bySource: make(map[Source][]*Skill),
	}
}

// Add adds a skill to the registry.
// If a skill with the same name exists from a lower precedence source, it's overridden.
func (r *Registry) Add(skill *Skill) {
	r.bySource[skill.Source] = append(r.bySource[skill.Source], skill)

	// Check precedence before overriding
	existing, exists := r.skills[skill.Name]
	if !exists || sourcePrecedence(skill.Source) > sourcePrecedence(existing.Source) {
		r.skills[skill.Name] = skill
	}
}

// AddAll adds multiple skills to the registry.
func (r *Registry) AddAll(skills []*Skill) {
	for _, skill := range skills {
		r.Add(skill)
	}
}

// Get returns a skill by name (highest precedence version).
func (r *Registry) Get(name string) *Skill {
	return r.skills[name]
}

// All returns all unique skills (highest precedence version of each).
func (r *Registry) All() []*Skill {
	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills
}

// BySource returns all skills from a specific source.
func (r *Registry) BySource(source Source) []*Skill {
	// Make a copy to avoid mutating the original slice during sort
	original := r.bySource[source]
	skills := make([]*Skill, len(original))
	copy(skills, original)
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills
}

// ByTag returns all skills that have a specific tag.
func (r *Registry) ByTag(tag string) []*Skill {
	var result []*Skill
	for _, skill := range r.skills {
		for _, t := range skill.Tags {
			if t == tag {
				result = append(result, skill)
				break
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Names returns all skill names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the total number of unique skills.
func (r *Registry) Count() int {
	return len(r.skills)
}

// CountBySource returns counts per source.
func (r *Registry) CountBySource() map[Source]int {
	counts := make(map[Source]int)
	for source, skills := range r.bySource {
		counts[source] = len(skills)
	}
	return counts
}

// IsOverridden checks if a skill from a lower source is overridden.
func (r *Registry) IsOverridden(name string, source Source) bool {
	skill := r.skills[name]
	if skill == nil {
		return false
	}
	return skill.Source != source
}

// GetAllVersions returns all versions of a skill across sources.
func (r *Registry) GetAllVersions(name string) []*Skill {
	var versions []*Skill
	for _, skills := range r.bySource {
		for _, skill := range skills {
			if skill.Name == name {
				versions = append(versions, skill)
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
	case SourceStarter:
		return 0
	default:
		return -1
	}
}

// LoadRegistry creates a registry by loading skills from all sources.
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
		skills, err := LoadFromDirectory(s.dir, s.source)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s skills: %w", s.source.Label(), err)
		}
		registry.AddAll(skills)
	}

	return registry, nil
}

// LoadRegistryWithMultipleDirs creates a registry by loading skills from multiple team directories.
// This supports multi-source configurations where different skills come from different repos.
func LoadRegistryWithMultipleDirs(teamDirs []string, personalDir, projectDir string) (*Registry, error) {
	registry := NewRegistry()

	// Load team skills from all team directories
	for _, teamDir := range teamDirs {
		if teamDir == "" {
			continue
		}
		skills, err := LoadFromDirectory(teamDir, SourceTeam)
		if err != nil {
			// Log warning but continue - some dirs may not have skills
			fmt.Fprintf(os.Stderr, "Warning: failed to load team skills from %s: %v\n", teamDir, err)
			continue
		}
		registry.AddAll(skills)
	}

	// Load personal skills
	if personalDir != "" {
		skills, err := LoadFromDirectory(personalDir, SourcePersonal)
		if err != nil {
			return nil, fmt.Errorf("failed to load personal skills: %w", err)
		}
		registry.AddAll(skills)
	}

	// Load project skills
	if projectDir != "" {
		skills, err := LoadFromDirectory(projectDir, SourceProject)
		if err != nil {
			return nil, fmt.Errorf("failed to load project skills: %w", err)
		}
		registry.AddAll(skills)
	}

	return registry, nil
}
