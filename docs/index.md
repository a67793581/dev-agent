---
layout: default
title: DevAgent — AI 命令行编程 Agent
---

# DevAgent

**AI 驱动的命令行编程 Agent**，使用 Go 实现，基于 OpenAI 模型与自定义 ReAct 循环，通过提示词工程实现工具调用（不依赖 Function Calling API）。

[English](#english) \| [中文](#中文)

---

## English

DevAgent is an open-source CLI agent that helps you automate coding tasks in your project. It runs a **ReAct loop** (Think → Act → Observe), executes file and shell tools inside a **configurable sandbox**, and supports **custom skills** and prompts.

### Why DevAgent?

| Feature | Description |
|--------|-------------|
| **No Function Calling API** | Tool calls are driven by structured LLM output parsing — works with any OpenAI-compatible API |
| **ReAct Loop** | Transparent reasoning: the agent thinks, acts, then observes before the next step |
| **Two-Layer Sandbox** | Code-level policy (block/approve/allow) plus optional Docker container for shell execution |
| **Extensible** | Add skills via `SKILL.md` in `.devagent/skills/`, override identity (`SOUL.md`) and guidelines (`GUIDELINES.md`) |
| **Streaming & i18n** | Real-time streaming output; UI in English or Chinese |

### Quick Start

```bash
# Build
go build -o bin/devagent .

# Set API key (or use .env)
export OPENAI_API_KEY="your-api-key"

# Run a task
./bin/devagent -project ./myapp -task "add error handling to all API endpoints"

# Interactive mode
./bin/devagent -project ./myapp
```

### Architecture (High Level)

- **CLI** (`main.go`): flags, interactive mode, i18n
- **Agent** (`agent/`): ReAct loop, history, retry
- **LLM** (`llm/`): OpenAI client
- **Parser** (`parser/`): extract tool calls from model output
- **Tools** (`tools/`): file (read/write/edit/search), shell (Docker or direct), sandbox policy

### Documentation

- Full usage, configuration, and sandbox options: see the [README](https://github.com/a67793581/dev-agent) in the repository root.
- License: [MIT](https://github.com/a67793581/dev-agent/blob/main/LICENSE).

---

## 中文

DevAgent 是一个**开源命令行编程 Agent**：在本地用 Go 编译运行，通过 OpenAI 兼容接口驱动，用**自定义 ReAct 循环**和**输出解析**实现工具调用，无需依赖官方的 Function Calling API。

### 为什么选 DevAgent？

| 特性 | 说明 |
|------|------|
| **不依赖 Function Calling** | 通过解析模型输出的结构化内容驱动工具调用，兼容任意 OpenAI 兼容 API |
| **ReAct 循环** | 先思考再执行再观察，推理过程可追溯 |
| **双层沙箱** | 代码层策略（拦截/需确认/放行）+ 可选 Docker 容器执行 Shell，安全可控 |
| **可扩展** | 通过 `.devagent/skills/` 下的 `SKILL.md` 扩展技能，用 `SOUL.md`、`GUIDELINES.md` 自定义身份与规范 |
| **流式输出与多语言** | 实时流式输出；界面支持英文/中文 |

### 快速开始

```bash
# 编译
go build -o bin/devagent .

# 配置 API Key（也可使用 .env）
export OPENAI_API_KEY="your-api-key"

# 执行任务
./bin/devagent -project ./myapp -task "为所有 API 接口添加错误处理"

# 交互模式
./bin/devagent -project ./myapp -lang zh
```

### 架构概览

- **CLI**（`main.go`）：参数、交互模式、多语言
- **Agent**（`agent/`）：ReAct 循环、历史、重试
- **LLM**（`llm/`）：OpenAI 客户端
- **解析器**（`parser/`）：从模型输出中解析工具调用
- **工具**（`tools/`）：文件（读/写/编辑/搜索）、Shell（Docker 或直接执行）、沙箱策略

### 文档与许可

- 完整使用说明、环境变量、沙箱配置等见仓库根目录 [README](https://github.com/a67793581/dev-agent)。
- 本作品采用 [MIT](https://github.com/a67793581/dev-agent/blob/main/LICENSE) 许可证。

---

*本介绍页由项目 `docs/` 目录提供，通过 **GitHub Pages** 从 `docs` 分支目录发布，零运行成本。*
