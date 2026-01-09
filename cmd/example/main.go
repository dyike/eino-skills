package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	skillsmw "github.com/dyike/eino-skills/pkg/middleware"
	skillpkg "github.com/dyike/eino-skills/pkg/skill"
	skilltools "github.com/dyike/eino-skills/pkg/tools"
)

// LoggerCallback 用于打印 Agent 执行过程中的各个步骤
type LoggerCallback struct {
	callbacks.HandlerBuilder // 继承 HandlerBuilder 来辅助实现 callback
	totalInputTokens         int
	totalOutputTokens        int
	totalTokens              int
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	// 打印 run_terminal_command 或其他工具的输入
	if info.Name == "run_terminal_command" || info.Name == "list_skills" || info.Name == "view_skill" {
		inputStr, _ := json.MarshalIndent(input, "", "  ")
		fmt.Printf("\n [%s] 👉 Input: %s\n", info.Name, string(inputStr))
	}
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	// 打印工具执行结果
	if info.Name == "run_terminal_command" || info.Name == "list_skills" || info.Name == "view_skill" {
		outputStr, _ := json.MarshalIndent(output, "", "  ")
		fmt.Printf("✅ [%s] Output: %s\n", info.Name, string(outputStr))
	}
	return ctx
}

// UpdateTokenUsage 用于从流式消息中更新 token 使用统计
// promptTokens: 从流中收集到的 prompt tokens（可能为 0）
// completionTokens: 从流中收集到的 completion tokens
func (cb *LoggerCallback) UpdateTokenUsage(promptTokens, completionTokens int, inputMessages []*schema.Message) (estimated bool) {
	// 如果 API 没有返回 prompt_tokens（代理问题），手动估算
	if promptTokens == 0 && len(inputMessages) > 0 {
		// 简单估算：英文 1 token ≈ 4 字符，中文 1 token ≈ 1.5 字符
		// 这里使用保守估算：总字符数 / 3
		totalChars := 0
		for _, m := range inputMessages {
			totalChars += len(m.Content)
			// 也要计算 system prompt 和其他内容
			for _, tc := range m.ToolCalls {
				totalChars += len(tc.Function.Name) + len(tc.Function.Arguments)
			}
		}
		promptTokens = totalChars / 3
		if promptTokens == 0 {
			promptTokens = 100 // 最小估值
		}
		estimated = true
	}

	cb.totalInputTokens += promptTokens
	cb.totalOutputTokens += completionTokens
	cb.totalTokens += promptTokens + completionTokens

	return estimated
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Printf("🔴 [%s] Error: %v\n", info.Name, err)
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	// 不在 callback 中读取流，以免影响主流程
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	return ctx
}

func main() {
	ctx := context.Background()

	// 1. 初始化 Skills 系统
	loader := skillpkg.NewLoader(
		skillpkg.WithGlobalSkillsDir("~/.claude/skills"),
	)

	registry := skillpkg.NewRegistry(loader, skillpkg.WithAutoWatch(true))
	if err := registry.Initialize(ctx); err != nil {
		fmt.Printf("Failed to initialize skills: %v\n", err)
		return
	}

	// 2. 创建 Skills 中间件
	skillsMiddleware := skillsmw.NewSkillsMiddleware(registry)

	// 3. 创建 Claude Chat Model
	baseURL := "http://127.0.0.1:8045"
	chatModel, err := claude.NewChatModel(ctx, &claude.Config{
		Model:     "gemini-3-flash",
		APIKey:    "sk-d61829b65a1642cd948d0915948f8473",
		BaseURL:   &baseURL,
		MaxTokens: 4096,
	})
	if err != nil {
		fmt.Printf("Failed to create chat model: %v\n", err)
		return
	}

	// 4. 获取 skill tools + 终端命令工具
	tools := skilltools.NewSkillTools(registry)

	// 获取当前工作目录的绝对路径
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current working directory: %v\n", err)
		return
	}
	terminalTool := skilltools.NewRunTerminalCommandTool(cwd)
	tools = append(tools, terminalTool)

	// 5. 构建带 Skills 的 system prompt
	basePrompt := `You are a helpful AI assistant with access to specialized skills.

**CRITICAL INSTRUCTIONS FOR SKILL EXECUTION:**

1. **DISCOVERY & LOADING**: Use 'list_skills' and 'view_skill' to understand the task.
2. **STRICT STEP-BY-STEP EXECUTION**:
   - You MUST follow the loaded skill's workflow exactly as written.
   - **DO NOT SKIP STEPS**: If the skill defines an "Analysis", "Preparation", or "Check" phase, you MUST execute it before moving to the main action.
3. **EXECUTE WITH ROBUSTNESS**:
   - **USE ABSOLUTE PATHS**: Construct **ABSOLUTE PATHS** for scripts referenced in the skill (e.g., '~/.claude/skills/[skill-name]/scripts/[script-name]').
   - **EXECUTE DIRECTLY**: Run the command directly. **DO NOT** use 'ls' or 'stat' to check existence first.
4. **ERROR RECOVERY (Fix & Retry)**:
   - If a command fails (e.g., "No such file"), **ANALYZE the error**.
   - **Retry with Fix**:
     - Did you use a relative path? Retry with the absolute path.
     - Are you in the wrong directory? Retry with correct context or path.
   - **Fallback**: Only if the script is truly broken or missing after retries, fallback to using **equivalent native commands** to accomplish the step's goal.
5. **TRANSPARENCY**: Explicitly state your thinking process (e.g., "Step 1: Running analysis script...").

Always be concise, professional, and act like an expert engineer.`

	systemPrompt := skillsMiddleware.InjectPrompt(basePrompt)

	// 6. 创建 ReAct Agent
	rAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: 100,
	})
	if err != nil {
		fmt.Printf("Failed to create agent: %v\n", err)
		return
	}

	fmt.Println("🚀 Eino Skills Agent Started!")
	fmt.Println("Type 'quit' or 'exit' to exit.")
	fmt.Println("Try: '帮我写一个 git commit message' to test skills")
	fmt.Println("---")

	// 7. 创建共享的 LoggerCallback 实例来累计整个会话的 token 使用
	logger := &LoggerCallback{}

	// 8. 交互式对话循环
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" {
			fmt.Println("\n" + strings.Repeat("=", 50))
			fmt.Println("📈 Session Summary - Total Token Usage:")
			fmt.Printf("   Input Tokens:  %d\n", logger.totalInputTokens)
			fmt.Printf("   Output Tokens: %d\n", logger.totalOutputTokens)
			fmt.Printf("   Total Tokens:  %d\n", logger.totalTokens)
			fmt.Println(strings.Repeat("=", 50))
			fmt.Println("Goodbye!")
			break
		}

		// 构建消息
		messages := []*schema.Message{
			{Role: schema.System, Content: systemPrompt},
			{Role: schema.User, Content: input},
		}

		// 使用共享的 callback 实例来累计 token 使用
		opts := []agent.AgentOption{
			agent.WithComposeOptions(compose.WithCallbacks(logger)),
		}

		fmt.Println("\n🤖 Thinking...")
		streamReader, err := rAgent.Stream(ctx, messages, opts...)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// 读取流式输出
		var fullContent strings.Builder
		seenToolCalls := make(map[string]bool)
		startInputTokens := logger.totalInputTokens
		startOutputTokens := logger.totalOutputTokens
		var promptTokens, completionTokens int // 累积的 token 统计

		for {
			msg, err := streamReader.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("\nError receiving stream: %v\n", err)
				break
			}

			// 处理每条消息的 usage 信息
			if msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
				usage := msg.ResponseMeta.Usage
				// prompt_tokens 通常只在第一条消息中非零，取非零值
				if usage.PromptTokens > 0 {
					promptTokens = usage.PromptTokens
				}
				// completion_tokens 是累积的，取最新的非零值
				if usage.CompletionTokens > 0 {
					completionTokens = usage.CompletionTokens
				}
			}

			// 打印 tool calls
			for _, tc := range msg.ToolCalls {
				key := fmt.Sprintf("%s:%s", tc.Function.Name, tc.Function.Arguments)
				if tc.Function.Name != "" && tc.Function.Arguments != "" && !seenToolCalls[key] {
					seenToolCalls[key] = true
					fmt.Printf("\n🔧 Tool Call: %s\n", tc.Function.Name)
					fmt.Printf("   Args: %s\n", tc.Function.Arguments)
				}
			}

			fullContent.WriteString(msg.Content)
		}

		// 更新 token 统计
		var isEstimated bool
		if promptTokens > 0 || completionTokens > 0 {
			isEstimated = logger.UpdateTokenUsage(promptTokens, completionTokens, messages)
		}

		// 打印最终内容
		if fullContent.Len() > 0 {
			fmt.Println("\n📝 Response:")
			fmt.Println(fullContent.String())
		}

		// 打印本次对话的 token 使用统计
		turnInputTokens := logger.totalInputTokens - startInputTokens
		turnOutputTokens := logger.totalOutputTokens - startOutputTokens
		turnTotalTokens := turnInputTokens + turnOutputTokens
		if turnTotalTokens > 0 {
			estimatedMark := ""
			if isEstimated {
				estimatedMark = " (Input estimated*)"
			}
			fmt.Printf("\n📊 This Turn - Token Usage: Input=%d, Output=%d, Total=%d%s\n",
				turnInputTokens, turnOutputTokens, turnTotalTokens, estimatedMark)
			fmt.Printf("💰 Cumulative Tokens: Input=%d, Output=%d, Total=%d\n",
				logger.totalInputTokens, logger.totalOutputTokens,
				logger.totalInputTokens+logger.totalOutputTokens)
			if isEstimated {
				fmt.Println("    * Input tokens estimated (API didn't return prompt_tokens)")
			}
		}
	}
}
