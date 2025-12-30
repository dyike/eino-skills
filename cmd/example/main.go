package main

import (
	"context"
	"fmt"

	skillpkg "github.com/dyike/eino-skills/pkg/skill"
	skilltools "github.com/dyike/eino-skills/pkg/tools"
)

func main() {
	ctx := context.Background()

	// 1. 加载 Skills
	loader := skillpkg.NewLoader(
		skillpkg.WithGlobalSkillsDir("~/.claude/skills"),
		skillpkg.WithProjectSkillsDir(".eino/skills"),
	)

	registry := skillpkg.NewRegistry(loader)
	registry.Initialize(ctx)

	// 2. 创建 Skills 中间件
	// skillsMiddleware := skillsmw.NewSkillsMiddleware(registry)

	// systemPrompt := skillsMiddleware.InjectPrompt("{This is your system prompt}")
	// fmt.Println("systemPrompt", systemPrompt)

	// 3. 获取 skill tools
	// skillTools := skilltools.NewSkillTools(registry)

	// for _, t := range skillTools {
	// 	info, _ := t.Info(ctx)
	// 	fmt.Println("Tool:", info.Name)
	// 	fmt.Println("  Desc:", info.Desc)
	// }

	// 测试 list_skills 工具
	fmt.Println("\n=== 测试 list_skills 工具 ===")
	listSkillsTool := skilltools.NewListSkillsTool(registry)
	result, err := listSkillsTool.InvokableRun(ctx, "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Skills list:")
		fmt.Println(result)
	}

	// 测试 view_skill 工具
	fmt.Println("\n=== 测试 view_skill 工具 ===")
	viewSkillTool := skilltools.NewViewSkillTool(registry)
	// 调用 view_skill 查看 git-commit skill
	result, err = viewSkillTool.InvokableRun(ctx, `{"name": "git-commit"}`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("git-commit skill content:")
		fmt.Println(result)
	}

	// 测试获取特定 section
	fmt.Println("\n=== 测试获取 Workflow section ===")
	result2, err := viewSkillTool.InvokableRun(ctx, `{"name": "git-commit", "section": "Workflow"}`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("git-commit Workflow section content:")
		fmt.Println(result2)
	}
}
