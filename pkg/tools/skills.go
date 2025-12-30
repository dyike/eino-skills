// Package tools provides skill-related tools for eino agents.
//
// This package implements tools that allow agents to discover and load skills
// following the progressive disclosure pattern:
//
//   - list_skills: Discover available skills
//   - view_skill: Load full skill content on demand
//
// Usage:
//
//	registry := skill.NewRegistry(loader)
//	registry.Initialize(ctx)
//
//	skillTools := tools.NewSkillTools(registry)
//	agent, _ := react.NewAgent(ctx, &react.AgentConfig{
//	    Tools: append(baseTools, skillTools...),
//	})
package tools

import (
	"github.com/cloudwego/eino/components/tool"

	skillpkg "github.com/dyike/eino-skills/pkg/skill"
)

// NewSkillTools creates all skill-related tools for an agent.
// Returns a slice of tools that can be added to the agent's tool list.
func NewSkillTools(registry *skillpkg.Registry) []tool.BaseTool {
	return []tool.BaseTool{
		NewViewSkillTool(registry),
		NewListSkillsTool(registry),
	}
}

// ToolNames returns the names of all skill-related tools.
func ToolNames() []string {
	return []string{"view_skill", "list_skills"}
}
