# AGENTS.md 自动同步：让 AI 维护自己的上下文

## 背景

AGENTS.md 是 AI coding agent（Claude Code / Windsurf / Codex 等）理解项目的核心上下文文件——描述了代码结构、模块职责、开发规范和调用关系。当代码结构发生变化（新增/删除/重命名模块）时，AGENTS.md 必须同步更新，否则 AI 读到的上下文与代码实际不一致，导致错误的代码生成。

人工维护 AGENTS.md 的问题很明显：容易忘记，且开发者未必清楚该怎么写。

**核心思路：让 AI 自己读代码，自己更新 AGENTS.md。**

## 架构

```
开发者 push 代码到 main
  ↓
GitHub Actions 检测结构性路径变化
  ↓
调用 agents-md-updater reusable workflow
  ↓
claude-code-action 安装 Claude Code CLI
  ↓
通过 Anthropic 兼容 API 调用 LLM
  ↓
LLM 读取代码结构 → 对比现有 AGENTS.md → 最小化编辑
  ↓
检测 diff → 有变化则开 PR，无变化则跳过
```

## 工作原理

### Reusable Workflow

基于开源项目 agents-md-updater，提供可复用的 GitHub Actions workflow：

- 支持 `codex`（OpenAI）和 `claude`（Anthropic）两种 agent
- 自动扫描项目目录，发现所有 AGENTS.md / CLAUDE.md 文件
- 内置 prompt 模板，指导 LLM 做最小化精准更新——只改过时的部分，保留已有结构和语气
- 支持自定义 `anthropic_base_url` 和 `model`，兼容 DeepSeek 等第三方 Anthropic API

### Caller Workflow

在你的项目中添加一个 caller workflow，指定触发条件和配置：

```yaml
on:
  push:
    branches: [main]
    paths:
      - "internal/**"    # Go 包结构
      - "cmd/**"         # 入口变化
      - "go.mod"         # 依赖变化
      - "Makefile"       # 构建命令变化
  workflow_dispatch: {}  # 支持手动触发
```

关键配置项：
- **Agent**: `claude`（通过 claude-code-action SDK 调用）
- **API**: Anthropic 兼容端点（支持原生 Anthropic 或 DeepSeek 等第三方）
- **Model**: 可配置（如 `deepseek-v4-pro`、`claude-sonnet-4-20250514`）
- **输出模式**: `pr`（开 Pull Request 供人审核）或 `commit`（直接推送）
- **Secret**: `ANTHROPIC_API_KEY`

### Prompt 策略

内置 prompt 遵循几个关键原则：

1. **先读后改**：先读现有 AGENTS.md，理解其结构和语气
2. **验证再改**：通过读 `go.mod`、目录结构、配置文件验证声明是否过时
3. **最小化编辑**：只改过时的部分，不重写整个文件
4. **避免过度探索**：不需要读遍整个仓库，够用就停

支持通过 `extra_instructions` 注入项目特定指引（如"保持中文描述"、"关注 internal/ 下的包结构变化"）。

## 第三方 API 兼容

`claude-code-action` 底层使用 Claude Code CLI，通过 `ANTHROPIC_BASE_URL` 路由 API 请求。使用 DeepSeek 等第三方兼容端点时需要注意：

**模型映射**：Claude Code 内部区分 haiku/sonnet/opus 模型角色（用于不同复杂度的子任务），必须通过环境变量将所有角色统一映射：

```
ANTHROPIC_MODEL=deepseek-v4-pro
ANTHROPIC_DEFAULT_HAIKU_MODEL=deepseek-v4-pro
ANTHROPIC_DEFAULT_SONNET_MODEL=deepseek-v4-pro
ANTHROPIC_DEFAULT_OPUS_MODEL=deepseek-v4-pro
```

**Tool Calling**：Claude Code 使用 Anthropic 格式的 tool use 来读写文件。第三方 API 的 tool calling 兼容程度可能有差异，如遇问题可切换到 `agent: codex` + OpenAI 兼容端点。

**成本参考**：每次运行约 25 轮对话，DeepSeek API 成本约 $0.5/次。

## 触发方式

- **自动**：push 到 main 且涉及结构性路径变化时自动触发
- **手动**：GitHub Actions 页面点击 Run workflow
- **CLI**：`gh workflow run "Update AGENTS.md"`

## 核心理念

**让 AI 自己维护自己的上下文。** 代码变了，AI 读代码后自动更新 AGENTS.md，形成闭环。人类只需审核 PR，不需要手动编写 AGENTS.md。

这是 Harness Engineering 中"验证外部化"原则的自然延伸——不仅用外部机制验证 AI 的输出质量，还用 AI 自身来维护那些外部机制所依赖的上下文。
