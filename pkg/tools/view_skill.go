// Package tools provides skill-related tools for eino agents.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	skillpkg "github.com/dyike/eino-skills/pkg/skill"
)

// ViewSkillTool allows agents to load full skill content on demand.
type ViewSkillTool struct {
	registry *skillpkg.Registry
}

// ViewSkillArgs defines the arguments for view_skill tool.
type ViewSkillArgs struct {
	// Name is the skill name to view
	Name string `json:"name"`
	// Section optionally specifies a specific section to extract
	Section string `json:"section,omitempty"`
}

// NewViewSkillTool creates a new view_skill tool.
func NewViewSkillTool(registry *skillpkg.Registry) *ViewSkillTool {
	return &ViewSkillTool{registry: registry}
}

// Info returns the tool's schema information.
func (t *ViewSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "view_skill",
		Desc: `View the full content of a skill's instructions. Use this tool when:
- A task matches an available skill from <available_skills>
- You need detailed instructions for a specific workflow
- The skill description indicates it's relevant to the current task

The tool loads the complete SKILL.md content including instructions, examples, and best practices.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name": {
				Type:     schema.String,
				Desc:     "The name of the skill to view (must match a name from <available_skills>)",
				Required: true,
			},
			"section": {
				Type:     schema.String,
				Desc:     "Optional: extract only a specific section by heading (e.g., 'Instructions', 'Examples')",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool and returns the skill content.
func (t *ViewSkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args ViewSkillArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Name == "" {
		return "", fmt.Errorf("skill name is required")
	}

	// Load skill content
	content, err := t.registry.GetContent(ctx, args.Name)
	if err != nil {
		return "", fmt.Errorf("failed to load skill '%s': %w", args.Name, err)
	}

	// Extract specific section if requested
	if args.Section != "" {
		parser := skillpkg.NewParser()
		sectionContent := parser.ExtractSection(content, args.Section)
		if sectionContent == "" {
			return "", fmt.Errorf("section '%s' not found in skill '%s'", args.Section, args.Name)
		}
		return sectionContent, nil
	}

	return content, nil
}

// Ensure ViewSkillTool implements tool.InvokableTool
var _ tool.InvokableTool = (*ViewSkillTool)(nil)
