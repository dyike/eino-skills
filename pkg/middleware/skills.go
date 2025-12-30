// Package middleware provides eino middleware for skills integration.
package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	skillpkg "github.com/dyike/eino-skills/pkg/skill"
	skilltools "github.com/dyike/eino-skills/pkg/tools"
)

// SkillsMiddleware injects skills metadata into agent prompts
// and provides skill-related tools.
type SkillsMiddleware struct {
	registry *skillpkg.Registry
	tools    []tool.BaseTool
}

// NewSkillsMiddleware creates a new skills middleware.
func NewSkillsMiddleware(registry *skillpkg.Registry) *SkillsMiddleware {
	mw := &SkillsMiddleware{
		registry: registry,
		tools:    skilltools.NewSkillTools(registry),
	}

	return mw
}

// InjectPrompt adds skills information to the system prompt.
func (m *SkillsMiddleware) InjectPrompt(basePrompt string) string {
	skillsSection := m.registry.GenerateSystemPromptSection()
	instructions := m.registry.GenerateSkillsInstructions()

	if skillsSection == "" {
		return basePrompt
	}

	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n")
	sb.WriteString(skillsSection)
	sb.WriteString("\n")
	sb.WriteString(instructions)

	return sb.String()
}

// GetTools returns skill-related tools to add to the agent.
func (m *SkillsMiddleware) GetTools() []tool.BaseTool {
	return m.tools
}

// ProcessMessages can modify messages before they reach the model.
// This is useful for auto-detecting when to suggest relevant skills.
func (m *SkillsMiddleware) ProcessMessages(ctx context.Context, messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return messages
	}

	// Check the last user message for skill relevance
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != schema.User {
		return messages
	}

	// Find potentially relevant skill
	content := lastMsg.Content
	if match := m.registry.FindMatchingSkill(content); match != nil {
		// Add a system hint about the relevant skill
		hint := &schema.Message{
			Role:    schema.System,
			Content: fmt.Sprintf("[Hint: The '%s' skill may be relevant for this task. Consider reading %s/SKILL.md for specialized instructions.]", match.Name, match.Path),
		}
		// Insert hint before the user message
		result := make([]*schema.Message, 0, len(messages)+1)
		result = append(result, messages[:len(messages)-1]...)
		result = append(result, hint, lastMsg)
		return result
	}

	return messages
}

// SkillsConfig holds configuration for the skills middleware.
type SkillsConfig struct {
	// GlobalSkillsDir is the global skills directory
	GlobalSkillsDir string

	// ProjectSkillsDir is the project-level skills directory
	ProjectSkillsDir string

	// AutoDetect enables automatic skill suggestion based on user input
	AutoDetect bool

	// AddTools determines whether to add skill tools to the agent
	AddTools bool
}

// DefaultConfig returns the default skills middleware configuration.
func DefaultConfig() *SkillsConfig {
	return &SkillsConfig{
		GlobalSkillsDir:  "~/.eino/agent/skills",
		ProjectSkillsDir: ".eino/skills",
		AutoDetect:       true,
		AddTools:         true,
	}
}

// CreateMiddleware creates a fully configured skills middleware.
func CreateMiddleware(ctx context.Context, config *SkillsConfig) (*SkillsMiddleware, error) {
	if config == nil {
		config = DefaultConfig()
	}

	loader := skillpkg.NewLoader(
		skillpkg.WithGlobalSkillsDir(config.GlobalSkillsDir),
		skillpkg.WithProjectSkillsDir(config.ProjectSkillsDir),
	)

	registry := skillpkg.NewRegistry(loader)
	if err := registry.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize skills registry: %w", err)
	}

	return NewSkillsMiddleware(registry), nil
}
