# Eino Skills - Claude Skills 集成方案

基于 [Anthropic Agent Skills](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) 和 [deepagents-cli](https://github.com/langchain-ai/deepagents/tree/master/libs/deepagents-cli) 的设计，为 [cloudwego/eino](https://github.com/cloudwego/eino) 框架实现 Skills 支持。

## 核心概念

### 什么是 Skills？

Skills 是包含 `SKILL.md` 文件的文件夹，提供：
- **渐进式披露 (Progressive Disclosure)**：只在需要时加载完整指令
- **Token 效率**：启动时仅加载元数据（name + description）
- **认知负担降低**：Agent 使用少量原子工具 + 按需加载的技能指令

### SKILL.md 结构

```yaml
---
name: skill-name
description: Brief description of what this skill does and when to use it
---

# Skill Name

## Instructions
[具体操作指令]

## Examples
[使用示例]
```

## 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                        Eino Agent                                │
├─────────────────────────────────────────────────────────────────┤
│  System Prompt                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ <available_skills>                                         │  │
│  │   <skill name="git-commit" description="..." />            │  │
│  │   <skill name="web-research" description="..." />          │  │
│  │ </available_skills>                                        │  │
│  └───────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│  Tools                                                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐   │
│  │ ReadFile │ │ WriteFile│ │  Bash    │ │ view_skill (新增)│   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  Skills Loader (Middleware)                                      │
│  - 扫描 skills 目录                                              │
│  - 解析 YAML frontmatter                                         │
│  - 注入 system prompt                                            │
└─────────────────────────────────────────────────────────────────┘
```

## 项目结构

```
eino-skills/
├── README.md
├── go.mod
├── pkg/
│   ├── skill/
│   │   ├── types.go          # Skill 类型定义
│   │   ├── loader.go         # Skills 加载器
│   │   ├── parser.go         # SKILL.md 解析器
│   │   └── registry.go       # Skills 注册中心
│   ├── tools/
│   │   ├── tools.go          # 工具包入口
│   │   ├── view_skill.go     # view_skill Tool
│   │   └── list_skills.go    # list_skills Tool
│   └── middleware/
│       └── skills.go         # Skills 中间件
├── cmd/
│   ├── agent/
│   │   └── main.go           # 完整 Agent 示例
│   └── skills-cli/
│       └── main.go           # CLI 管理工具
├── skills/                   # 示例 Skills
│   ├── git-commit/
│   │   └── SKILL.md
│   └── web-research/
│       └── SKILL.md
└── docs/
    └── DESIGN.md             # 设计文档
```

## 快速开始

```go
package main

import (
    "context"
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
    
    skillpkg "github.com/yourname/eino-skills/pkg/skill"
    skilltools "github.com/yourname/eino-skills/pkg/tools"
    skillsmw "github.com/yourname/eino-skills/pkg/middleware"
)

func main() {
    ctx := context.Background()
    
    // 1. 加载 Skills
    loader := skillpkg.NewLoader(
        skillpkg.WithGlobalSkillsDir("~/.eino/skills"),
        skillpkg.WithProjectSkillsDir(".eino/skills"),
    )
    
    registry := skillpkg.NewRegistry(loader)
    registry.Initialize(ctx)
    
    // 2. 创建 Skills 中间件
    skillsMiddleware := skillsmw.NewSkillsMiddleware(registry)
    
    // 3. 获取 skill tools
    skillTools := skilltools.NewSkillTools(registry)
    
    // 4. 创建 Agent（带 Skills 支持）
    agent, _ := react.NewAgent(ctx, &react.AgentConfig{
        Model:         chatModel,
        Tools:         append(baseTools, skillTools...),
        MessageModifier: func(ctx context.Context, msgs []*schema.Message) []*schema.Message {
            systemPrompt := skillsMiddleware.InjectPrompt(basePrompt)
            return append([]*schema.Message{
                {Role: schema.System, Content: systemPrompt},
            }, msgs...)
        },
    })
    
    // 5. 运行
    msg, _ := agent.Generate(ctx, []*schema.Message{
        {Role: schema.User, Content: "帮我写一个 git commit message"},
    })
}
```

