package main

import (
	"context"
	"fmt"
	"log"

	// 【核心修正】: 引入正确的 MCP 客户端和适配器库
	langchaingo_mcp_adapter "github.com/i2y/langchaingo-mcp-adapter"
	"github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	// --- 1. 连接到正在运行的 MCP 服务 ---

	// 定义您的 Fetcher MCP 服务地址
	mcpServiceURL := "http://localhost:3000/mcp"

	// 【核心修正】: 使用 mark3labs/mcp-go/client 中正确的 NewStreamableHttpClient 方法
	// 这个客户端会通过网络连接到您已经启动的 Fetcher MCP 服务
	mcpClient, err := client.NewStreamableHttpClient(mcpServiceURL)
	if err != nil {
		log.Fatalf("无法创建 MCP HTTP 客户端，请确保 Fetcher MCP 服务正在运行于 %s : %v", mcpServiceURL, err)
	}
	// HTTP 客户端通常不需要手动 Close()

	// --- 2. 使用适配器从 MCP 服务获取所有可用工具 ---

	// 使用 i2y/langchaingo-mcp-adapter 来创建适配器
	adapter, err := langchaingo_mcp_adapter.New(mcpClient)
	if err != nil {
		log.Fatalf("无法创建 MCP 适配器: %v", err)
	}

	// 通过适配器自动获取 MCP 服务器提供的所有工具
	toolList, err := adapter.Tools()
	if err != nil {
		log.Fatalf("无法从 MCP 服务获取工具列表: %v", err)
	}

	log.Printf("从 MCP 服务成功加载了 %d 个工具。\n", len(toolList))
	for _, tool := range toolList {
		log.Printf("- 工具名称: %s", tool.Name())
	}

	// --- 3. 初始化 LLM 和 Agent ---

	ctx := context.Background()

	// 初始化 Google AI (Gemini) LLM 客户端
	llm, err := openai.New(
		openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		openai.WithModel("qwen-plus"),
	)
	if err != nil {
		log.Fatalf("创建 Google AI 客户端失败: %v", err)
	}
	agent := agents.NewOneShotAgent(
		llm,
		toolList,
		agents.WithMaxIterations(5),
	)
	// 创建一个支持函数调用（工具使用）的 Agent 执行器
	executor := agents.NewExecutor(agent)

	if err != nil {
		log.Fatalf("初始化 Agent 失败: %v", err)
	}

	// --- 4. 使用 Agent 完成任务 ---

	// 提出一个需要使用 fetch_url 工具的问题
	question := "请帮我访问 https://juejin.cn/post/7364409188074782771 这个网址，并用中文总结一下它的核心内容是什么？"
	fmt.Println(">> 用户问题:", question)

	// 使用 chains.Run 来执行 Agent
	result, err := chains.Run(ctx, executor, question)
	if err != nil {
		log.Fatalf("Agent 执行出错: %v", err)
	}

	fmt.Println("\n>> Agent 的最终回答:")
	fmt.Println(result)
}
