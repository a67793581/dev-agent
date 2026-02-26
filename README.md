# DevAgent

基于 Go 实现的命令行编程 Agent，借鉴 MetaGPT 的代码修复逻辑，使用 OpenAI 模型驱动。

不使用 OpenAI 的 function calling API，而是通过自定义提示词规则 + 输出解析实现工具调用。

## 架构设计

```
┌──────────────────────────────────────────────────┐
│                    main.go (CLI)                 │
│              命令行参数 / 交互模式                  │
└──────────────┬───────────────────────────────────┘
               │
┌──────────────▼───────────────────────────────────┐
│               agent/agent.go                     │
│          ReAct 循环 (Think → Act → Observe)       │
│          对话历史管理 / 重试机制                     │
└──────┬───────────┬──────────────┬────────────────┘
       │           │              │
┌──────▼──┐  ┌─────▼─────┐  ┌────▼─────┐
│  llm/   │  │  parser/  │  │  tools/  │
│ OpenAI  │  │  输出解析   │  │ 工具注册  │
│ 客户端   │  │ JSON提取   │  │ 与执行    │
└─────────┘  └───────────┘  └──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
         ┌────▼────┐    ┌─────▼─────┐    ┌─────▼─────┐
         │ file.go │    │ shell.go  │    │ prompt.go │
         │文件读写  │    │Shell执行   │    │ 提示词模板 │
         │目录遍历  │    │Grep搜索   │    │ 代码修复   │
         └─────────┘    └───────────┘    └───────────┘
```

## 核心特性

- **自定义工具调用**：通过提示词规则让 AI 输出 JSON 命令块，解析后执行对应工具，不依赖 OpenAI function calling
- **ReAct 模式**：Think → Act → Observe 循环，每步先思考再执行
- **代码修复**：借鉴 MetaGPT 的 DebugError/RunCode 工作流，支持读取代码→分析错误→修复→验证的迭代循环
- **Shell 执行**：完整的命令行操作能力，可以安装依赖、运行测试、构建项目
- **文件操作**：读写文件、目录遍历、文件搜索、Grep 搜索
- **流式输出**：支持 SSE 流式输出，实时显示 AI 思考过程
- **对话历史管理**：自动裁剪过长的对话历史，保留关键上下文
- **JSON 修复**：借鉴 MetaGPT 的 repair_llm_raw_output，自动修复 AI 输出中的 JSON 格式错误

## 安装

```bash
go build -o devagent .
```

## 使用

### 环境变量

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 可选，支持兼容 API
export OPENAI_MODEL="gpt-4o"                         # 可选，默认 gpt-4o
```

### 命令行模式

```bash
# 指定项目目录和任务
devagent -project ./myapp -task "分析项目结构并添加错误处理"

# 详细输出模式
devagent -project ./myapp -task "修复 main.go 中的 bug" -verbose

# 使用自定义模型
devagent -project ./myapp -model gpt-4o-mini -task "添加单元测试"
```

### 交互模式

```bash
devagent -project ./myapp

# 进入交互模式后，直接输入任务：
🤖 > 分析这个项目的代码结构
🤖 > 添加 Docker 支持
🤖 > 安装 golangci-lint 并运行检查
```

### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-project` | 项目目录路径 | `.` |
| `-model` | OpenAI 模型名称 | `gpt-4o` |
| `-base-url` | API 基础 URL | `https://api.openai.com/v1` |
| `-api-key` | API 密钥 | `OPENAI_API_KEY` 环境变量 |
| `-verbose` | 详细输出 | `false` |
| `-task` | 任务描述 | 空 (进入交互模式) |
| `-version` | 显示版本 | - |

## 借鉴 MetaGPT 的设计

### 代码修复流程

```
错误发生 → 读取源码和错误日志 → AI 分析根因 → 生成修复代码 → 执行验证 → 成功/重试
```

对应 MetaGPT 中的:
- `DebugError.run()` → 分析代码和错误，生成修复
- `RunCode.run()` → 执行代码并分析结果
- `DataInterpreter._write_and_exec_code()` → 写入→执行→修复的迭代循环

### 输出解析

- JSON 命令块提取 (类似 MetaGPT 的 `CodeParser.parse_code`)
- JSON 自动修复 (类似 MetaGPT 的 `repair_llm_raw_output`)
- 思考标签提取 (类似 MetaGPT 的 ReAct 模式)

### 工具注册机制

类似 MetaGPT 的 `tool_execution_map`，使用注册表模式管理可用工具。
