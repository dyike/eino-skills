package skill

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Registry manages loaded skills and provides lookup functionality.
type Registry struct {
	mu        sync.RWMutex
	skills    map[string]*Skill
	metadata  []SkillMetadata
	loader    *Loader
	watcher   *Watcher
	autoWatch bool
}

// RegistryOption configures the Registry.
type RegistryOption func(*Registry)

// WithAutoWatch enables automatic file watching after Initialize.
// When enabled, the registry will automatically reload when SKILL.md files change.
func WithAutoWatch(enabled bool) RegistryOption {
	return func(r *Registry) {
		r.autoWatch = enabled
	}
}

// NewRegistry creates a new skills registry.
func NewRegistry(loader *Loader, opts ...RegistryOption) *Registry {
	r := &Registry{
		skills: make(map[string]*Skill),
		loader: loader,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// StartWatching begins monitoring skill directories for changes.
// When changes are detected, the registry automatically reloads.
func (r *Registry) StartWatching(ctx context.Context) error {
	if r.watcher != nil {
		return fmt.Errorf("watcher already started")
	}

	dirs := []string{r.loader.globalDir, r.loader.projectDir}
	watcher, err := NewWatcher(r, dirs)
	if err != nil {
		return err
	}

	r.watcher = watcher
	return watcher.Start(ctx)
}

// StopWatching stops monitoring skill directories.
func (r *Registry) StopWatching() error {
	if r.watcher == nil {
		return nil
	}

	err := r.watcher.Stop()
	r.watcher = nil
	return err
}

// Initialize loads all skills from configured directories.
func (r *Registry) Initialize(ctx context.Context) error {
	r.mu.Lock()

	// Load metadata for system prompt
	metadata, err := r.loader.LoadMetadataOnly(ctx)
	if err != nil {
		r.mu.Unlock()
		return fmt.Errorf("failed to load skill metadata: %w", err)
	}
	r.metadata = metadata

	// Clear existing skills
	r.skills = make(map[string]*Skill)

	r.mu.Unlock()

	// Start watching if autoWatch is enabled
	if r.autoWatch && r.watcher == nil {
		if err := r.StartWatching(ctx); err != nil {
			// Log warning but don't fail initialization
			fmt.Printf("Warning: failed to start auto-watch: %v\n", err)
		}
	}

	return nil
}

// Get retrieves a skill by name, loading it on demand if needed.
func (r *Registry) Get(ctx context.Context, name string) (*Skill, error) {
	r.mu.RLock()
	skill, exists := r.skills[name]
	r.mu.RUnlock()

	if exists {
		return skill, nil
	}

	// Load on demand
	skill, err := r.loader.LoadSkill(ctx, name)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.skills[name] = skill
	r.mu.Unlock()

	return skill, nil
}

// GetContent retrieves the full content of a skill.
func (r *Registry) GetContent(ctx context.Context, name string) (string, error) {
	skill, err := r.Get(ctx, name)
	if err != nil {
		return "", err
	}

	return r.loader.LoadSkillContent(ctx, skill)
}

// GetMetadata returns all loaded skill metadata.
func (r *Registry) GetMetadata() []SkillMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metadata
}

// FindMatchingSkill finds a skill that matches the given query.
// This uses simple keyword matching for skill selection.
func (r *Registry) FindMatchingSkill(query string) *SkillMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	var bestMatch *SkillMetadata
	bestScore := 0

	for i := range r.metadata {
		m := &r.metadata[i]
		score := r.calculateMatchScore(query, m)
		if score > bestScore {
			bestScore = score
			bestMatch = m
		}
	}

	// Require minimum score to return a match
	if bestScore < 2 {
		return nil
	}

	return bestMatch
}

// calculateMatchScore computes how well a skill matches a query.
func (r *Registry) calculateMatchScore(query string, m *SkillMetadata) int {
	score := 0
	queryWords := strings.Fields(query)

	// Check skill name
	name := strings.ToLower(m.Name)
	for _, word := range queryWords {
		if strings.Contains(name, word) {
			score += 3
		}
	}

	// Check description
	desc := strings.ToLower(m.Description)
	for _, word := range queryWords {
		if len(word) > 2 && strings.Contains(desc, word) {
			score += 1
		}
	}

	return score
}

// GenerateSystemPromptSection generates the skills section for system prompts.
func (r *Registry) GenerateSystemPromptSection() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.metadata) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")

	for _, m := range r.metadata {
		sb.WriteString("<skill>\n")
		sb.WriteString(fmt.Sprintf("<name>\n%s\n</name>\n", m.Name))
		sb.WriteString(fmt.Sprintf("<description>\n%s\n</description>\n", m.Description))
		sb.WriteString(fmt.Sprintf("<location>\n%s/SKILL.md\n</location>\n", m.Path))
		sb.WriteString("</skill>\n\n")
	}

	sb.WriteString("</available_skills>\n")

	return sb.String()
}

// GenerateSkillsInstructions generates instructions for using skills.
func (r *Registry) GenerateSkillsInstructions() string {
	return `<skills_instructions>
When a task matches one of the available skills, follow these steps:

1. **Discovery**: Check <available_skills> to see if any skill matches the current task
2. **Load Instructions**: Use the view tool or read_file to load the full SKILL.md content from the skill's location
3. **Follow Instructions**: Execute the task according to the loaded skill instructions
4. **Reference Files**: If the skill references additional files (scripts/, references/, assets/), load them as needed

Skills provide specialized workflows and domain knowledge. Always prefer using a relevant skill over improvising when one is available.
</skills_instructions>
`
}

// Reload refreshes the registry with updated skills from disk.
func (r *Registry) Reload(ctx context.Context) error {
	return r.Initialize(ctx)
}

// Count returns the number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.metadata)
}

// Names returns all registered skill names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.metadata))
	for i, m := range r.metadata {
		names[i] = m.Name
	}
	return names
}
