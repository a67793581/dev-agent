# DevAgent

[English](#english) | [中文](#中文)

---

## English

AI-powered command-line programming agent built with Go. Uses OpenAI models with a custom ReAct loop and tool calling via prompt engineering (no function calling API).

### Architecture

```
┌──────────────────────────────────────────────────┐
│                    main.go (CLI)                 │
│            Flags / Interactive Mode / i18n       │
└──────────────┬───────────────────────────────────┘
               │
┌──────────────▼───────────────────────────────────┐
│               agent/agent.go                     │
│          ReAct Loop (Think → Act → Observe)      │
│          Conversation History / Retry            │
└──────┬───────────┬──────────────┬────────────────┘
       │           │              │
┌──────▼──┐  ┌─────▼─────┐  ┌────▼─────┐
│  llm/   │  │  parser/  │  │  tools/  │
│ OpenAI  │  │  Output   │  │ Registry │
│ Client  │  │  Parser   │  │ & Exec   │
└─────────┘  └───────────┘  └────┬─────┘
                                 │
               ┌─────────────────┼─────────────────┐
               │                 │                  │
          ┌────▼────┐     ┌─────▼──────┐    ┌──────▼──────┐
          │ file.go │     │ shell.go   │    │ sandbox/    │
          │ R/W/Edit│     │ Docker Exec│    │ Policy +    │
          │ Search  │     │ or Direct  │    │ Docker Env  │
          └─────────┘     └────────────┘    └─────────────┘
```

### Features

- **Custom Tool Calling**: LLM outputs JSON command blocks parsed at runtime — no OpenAI function calling dependency
- **ReAct Loop**: Think → Act → Observe cycle with reasoning traces
- **Sandbox Security**: Two-layer protection
  - **Code-level policy**: Path containment, shell command filtering, risk-based approval (permissive / normal / strict)
  - **Docker container**: Shell commands run in a persistent per-project container with resource limits
- **File Operations**: Read, write, edit (str_replace / insert_line), search, grep
- **Shell Execution**: Full shell access inside Docker sandbox (or direct with `-no-docker`)
- **Code Repair**: MetaGPT-inspired debug workflow: read code → analyze error → fix → verify
- **Skills System**: Extensible via `SKILL.md` files in `.devagent/skills/`
- **Custom Prompts**: Override agent identity (`SOUL.md`) and coding guidelines (`GUIDELINES.md`)
- **Streaming Output**: Real-time SSE streaming of LLM responses
- **i18n**: Chinese / English UI via `-lang` flag or `LANG` env auto-detection

### Installation

```bash
go build -o bin/devagent .
```

### Configuration

#### Environment Variables

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # optional, supports compatible APIs
export OPENAI_MODEL="gpt-4o"                         # optional, default: gpt-4o
```

`.env` file lookup order (first found wins, existing env vars are never overwritten):
1. File specified by `-env` flag
2. `.env` in current working directory
3. `~/.devagent.env` in home directory

#### Sandbox Configuration

Create `.devagent/sandbox.yaml` in your project directory:

```yaml
mode: normal  # permissive / normal / strict

shell:
  block:
    - "sudo *"           # appended to default blocklist
    - "docker rm -f *"
  approve:
    - "git push *"       # require user confirmation
    - "npm publish"
  allow:
    - "go test *"        # always allow (highest priority)
    - "make *"

paths:
  deny:
    - "/etc"
    - "~/.ssh"
  allow_outside_workdir:
    - "~/go/pkg"

docker:
  enabled: true
  image: "ubuntu:22.04"  # base image
  network: "host"        # host / none / bridge
  memory: "512m"
  cpus: "2"
  extra_mounts:
    - source: "~/go/pkg/mod"
      target: "/go/pkg/mod"
      readonly: true
```

### Usage

#### Command Line Mode

```bash
devagent -project ./myapp -task "add error handling to all API endpoints"
devagent -project ./myapp -task "fix the bug in main.go" -verbose
devagent -project ./myapp -model gpt-4o-mini -task "add unit tests"
```

#### Interactive Mode

```bash
devagent -project ./myapp

🤖 > Analyze the project structure
🤖 > Add Docker support
🤖 > quit
```

#### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-project` | Project directory path | `.` |
| `-task` | Task to execute (empty = interactive mode) | |
| `-model` | OpenAI model name | `gpt-4o` |
| `-base-url` | API base URL | `https://api.openai.com/v1` |
| `-api-key` | API key | `OPENAI_API_KEY` env |
| `-verbose` | Show LLM streaming and tool details | `false` |
| `-sandbox` | Sandbox mode: `permissive` / `normal` / `strict` | `normal` |
| `-no-docker` | Disable Docker sandbox | `false` |
| `-lang` | UI language: `en` / `zh` (auto-detect from `LANG` env) | auto |
| `-soul` | Path to custom soul/identity prompt | `.devagent/SOUL.md` |
| `-guidelines` | Path to custom guidelines prompt | `.devagent/GUIDELINES.md` |
| `-skills` | Additional skill directories (comma-separated) | `.devagent/skills/` |
| `-env` | Path to `.env` file | auto-detect |
| `-version` | Show version | |

---

## 中文

基于 Go 实现的命令行编程 Agent，使用 OpenAI 模型驱动，通过自定义提示词规则 + 输出解析实现工具调用（不依赖 function calling API）。

### 核心特性

- **自定义工具调用**：AI 输出 JSON 命令块，解析后执行对应工具
- **ReAct 模式**：Think → Act → Observe 循环，每步先思考再执行
- **双层沙箱安全**
  - **代码层策略**：路径隔离、Shell 命令过滤、分级审批（permissive / normal / strict）
  - **Docker 容器**：Shell 命令在每个项目独立的持久容器内执行，资源隔离
- **文件操作**：读写、编辑（str_replace / insert_line）、搜索、Grep
- **Shell 执行**：在 Docker 沙箱内完整 Shell 访问（或用 `-no-docker` 直接执行）
- **代码修复**：借鉴 MetaGPT 的调试流程：读取代码 → 分析错误 → 修复 → 验证
- **技能系统**：通过 `.devagent/skills/` 下的 `SKILL.md` 文件扩展能力
- **自定义提示词**：覆盖 Agent 身份（`SOUL.md`）和编码规范（`GUIDELINES.md`）
- **流式输出**：实时显示 AI 思考过程
- **中英文切换**：通过 `-lang` 参数或 `LANG` 环境变量自动检测

### 安装

```bash
go build -o bin/devagent .
```

### 配置

#### 环境变量

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 可选，支持兼容 API
export OPENAI_MODEL="gpt-4o"                         # 可选，默认 gpt-4o
```

#### 沙箱配置

在项目目录下创建 `.devagent/sandbox.yaml`：

```yaml
mode: normal  # permissive / normal / strict

shell:
  block:              # 追加到默认黑名单
    - "sudo *"
  approve:            # 需要用户确认
    - "git push *"
  allow:              # 白名单（优先级最高）
    - "go test *"

paths:
  deny:               # 敏感路径黑名单
    - "/etc"
    - "~/.ssh"
  allow_outside_workdir:  # 允许访问项目外路径
    - "~/go/pkg"

docker:
  enabled: true
  image: "ubuntu:22.04"
  network: "host"     # host / none / bridge
  memory: "512m"
  cpus: "2"
  extra_mounts:       # 额外挂载
    - source: "~/go/pkg/mod"
      target: "/go/pkg/mod"
      readonly: true
```

### 使用

#### 命令行模式

```bash
devagent -project ./myapp -task "分析项目结构并添加错误处理"
devagent -project ./myapp -task "修复 main.go 中的 bug" -verbose
```

#### 交互模式

```bash
devagent -project ./myapp -lang zh

🤖 > 分析项目结构
🤖 > 添加 Docker 支持
🤖 > quit
```

#### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-project` | 项目目录路径 | `.` |
| `-task` | 任务描述（空则进入交互模式） | |
| `-model` | OpenAI 模型名称 | `gpt-4o` |
| `-base-url` | API 基础 URL | `https://api.openai.com/v1` |
| `-api-key` | API 密钥 | `OPENAI_API_KEY` 环境变量 |
| `-verbose` | 显示 LLM 流式输出和工具详情 | `false` |
| `-sandbox` | 沙箱模式：`permissive` / `normal` / `strict` | `normal` |
| `-no-docker` | 禁用 Docker 沙箱 | `false` |
| `-lang` | 界面语言：`en` / `zh`（自动检测 `LANG` 环境变量） | 自动 |
| `-soul` | 自定义身份提示词文件路径 | `.devagent/SOUL.md` |
| `-guidelines` | 自定义编码规范文件路径 | `.devagent/GUIDELINES.md` |
| `-skills` | 额外技能目录（逗号分隔） | `.devagent/skills/` |
| `-env` | `.env` 文件路径 | 自动查找 |
| `-version` | 显示版本号 | |

### 沙箱模式说明

| 模式 | 行为 |
|------|------|
| `permissive` | 仅拦截危险命令（`sudo`、`rm -rf /` 等），其余放行 |
| `normal` | 拦截危险命令 + 高风险操作需用户确认（默认） |
| `strict` | 除只读操作外，所有写操作和中/高风险命令均需确认 |

### Docker 沙箱

- 每个项目对应一个持久容器（命名：`devagent-<hash>`）
- 容器在命令间复用，安装的软件包持久保留
- 进程退出时停止容器（不删除），下次启动自动恢复
- 支持配置镜像、网络模式、资源限制、额外挂载
- Docker 不可用时自动降级为直接执行

### 项目结构

```
.devagent/
├── sandbox.yaml     # 沙箱配置
├── SOUL.md          # Agent 身份/人格提示词
├── GUIDELINES.md    # 编码规范提示词
└── skills/          # 技能目录
    └── my-skill/
        └── SKILL.md
```
