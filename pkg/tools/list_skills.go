package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	skillpkg "github.com/dyike/eino-skills/pkg/skill"
)

// ListSkillsTool allows agents to discover available skills.
type ListSkillsTool struct {
	registry *skillpkg.Registry
}

// ListSkillsArgs defines the arguments for list_skills tool.
type ListSkillsArgs struct {
	// Filter optionally filters skills by keyword
	Filter string `json:"filter,omitempty"`
	// Source optionally filters by source (global, project)
	Source string `json:"source,omitempty"`
}

// NewListSkillsTool creates a new list_skills tool.
func NewListSkillsTool(registry *skillpkg.Registry) *ListSkillsTool {
	return &ListSkillsTool{registry: registry}
}

// Info returns the tool's schema information.
func (t *ListSkillsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_skills",
		Desc: `List all available skills with their descriptions. Use this tool to:
- Discover what specialized capabilities are available
- Find skills relevant to a specific domain or task
- Check if a skill exists before trying to use it`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"filter": {
				Type:     schema.String,
				Desc:     "Optional: filter skills by keyword in name or description",
				Required: false,
			},
			"source": {
				Type:     schema.String,
				Desc:     "Optional: filter by source - 'global' or 'project'",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool and returns the skills list.
func (t *ListSkillsTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args ListSkillsArgs
	if argumentsInJSON != "" && argumentsInJSON != "{}" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			return "", fmt.Errorf("failed to parse arguments: %w", err)
		}
	}

	metadata := t.registry.GetMetadata()

	if len(metadata) == 0 {
		return "No skills available.", nil
	}

	// Apply filters
	filtered := make([]skillpkg.SkillMetadata, 0, len(metadata))
	for _, m := range metadata {
		// Filter by source
		if args.Source != "" {
			if string(m.Source) != args.Source {
				continue
			}
		}

		// Filter by keyword
		if args.Filter != "" {
			filter := strings.ToLower(args.Filter)
			name := strings.ToLower(m.Name)
			desc := strings.ToLower(m.Description)
			if !strings.Contains(name, filter) && !strings.Contains(desc, filter) {
				continue
			}
		}

		filtered = append(filtered, m)
	}

	if len(filtered) == 0 {
		if args.Filter != "" || args.Source != "" {
			return "No skills match the specified filters.", nil
		}
		return "No skills available.", nil
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d skill(s):\n\n", len(filtered)))

	for _, m := range filtered {
		sb.WriteString(fmt.Sprintf("## %s\n", m.Name))
		sb.WriteString(fmt.Sprintf("- **Source**: %s\n", m.Source))
		sb.WriteString(fmt.Sprintf("- **Location**: %s/SKILL.md\n", m.Path))
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n\n", m.Description))
	}

	return sb.String(), nil
}

// Ensure ListSkillsTool implements tool.InvokableTool
var _ tool.InvokableTool = (*ListSkillsTool)(nil)
