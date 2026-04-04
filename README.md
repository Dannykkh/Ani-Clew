<div align="center">

# AniClew

### Any Model, One Coding Agent

**Use Claude Code's coding agent experience with ANY model — Ollama, OpenAI, Gemini, Groq, and more.**

**Free with local models. Zero API cost. Single binary.**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](#installation)
[![Binary Size](https://img.shields.io/badge/Binary-11MB-green)](#installation)
[![Providers](https://img.shields.io/badge/Providers-7+-blueviolet)](#supported-providers)
[![License](https://img.shields.io/badge/License-MIT-yellow)](#license)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)](#installation)

[English](#features) | [한국어](#한국어)

<img src="https://github.com/user-attachments/assets/placeholder-screenshot.png" alt="AniClew Dashboard" width="800" />

</div>

---

## Why AniClew?

| | Claude Code | Cursor | AniClew |
|---|---|---|---|
| **Price** | $20/mo + API costs | $20/mo | **Free** (with Ollama) |
| **Models** | Claude only | Limited | **Any model** (7+ providers) |
| **Interface** | Terminal only | IDE only | **Web browser** |
| **Smart Routing** | No | No | **Auto-routes by task** |
| **Self-hosted** | No | No | **Single binary** |
| **Background Agent** | Unreleased (KAIROS) | No | **Built-in** |
| **Team Management** | No | Enterprise only | **Built-in** |
| **Offline** | No | No | **Yes** (Ollama) |

## Features

- **15 Coding Tools** — Bash, Read, Write, Edit, Glob, Grep, Git, WebSearch, WebFetch, LS, TaskCreate/Update/List, NotebookRead/Edit
- **7 Providers** — Anthropic, OpenAI, Gemini, Groq, Ollama (local), GitHub Copilot, xAI (Grok)
- **Smart Router** — Automatically routes requests to the best model by task type (explain → cheap model, refactor → powerful model)
- **KAIROS Daemon** — Background agent that monitors, reviews PRs, runs tests autonomously
- **Web UI** — Full browser-based coding agent with 7 pages (Chat, Settings, Routes, Costs, KAIROS, Memory, Team)
- **Permission System** — Blocks dangerous commands (`rm -rf`, `sudo`, etc.) before execution
- **Project Context** — Auto-loads `CLAUDE.md`, `AGENTS.md`, `.claude/skills/` for project-aware responses
- **Session Management** — Per-project chat history, auto-save, workspace isolation
- **Team Gateway** — Multi-user auth, per-user budgets, audit logging, PII masking
- **Cost Tracking** — Real-time cost breakdown by model, savings vs cloud comparison
- **AutoDream Memory** — Cross-session learning with automatic memory consolidation
- **i18n** — Korean / English UI with configurable AI response language
- **Auth Passthrough** — Forward existing Claude Code OAuth tokens through the proxy
- **Single Binary** — 11MB Go executable, zero dependencies, auto-opens browser

## Quick Start

### Option 1: Download Binary (Recommended)

```bash
# Download for your platform from Releases
# Windows
./aniclew.exe

# macOS / Linux
chmod +x aniclew
./aniclew
```

Browser opens automatically → Select provider → Start coding.

### Option 2: Build from Source

```bash
git clone https://github.com/Dannykkh/Ani-Clew.git
cd Ani-Clew/proxy-go

# Build
go build -o aniclew ./cmd/proxy/

# Run
./aniclew
```

### Option 3: With Ollama (Free, Local)

```bash
# 1. Install Ollama (https://ollama.com)
ollama pull qwen3:14b

# 2. Run AniClew
./aniclew --provider ollama --model qwen3:14b --router

# 3. Open browser → http://localhost:4000/app
```

### Option 4: Docker

```bash
docker-compose up -d
# Ollama + AniClew proxy, ready to go
```

### Option 5: As Claude Code Proxy

```bash
# Run AniClew as a proxy, then connect Claude Code to it
./aniclew --provider ollama --model qwen3:14b &

ANTHROPIC_BASE_URL=http://localhost:4000 claude
```

## Supported Providers

| Provider | Models | Cost | Auth |
|----------|--------|------|------|
| **Ollama** | Qwen3, Llama 4, Gemma 4, Codestral, DeepSeek R1, Mistral | **Free** | None |
| **OpenAI** | GPT-5.4, GPT-5.3 Codex, GPT-4.1, o3, o4-mini | Pay-per-use | API Key |
| **Anthropic** | Claude Opus 4.6, Sonnet 4.6, Haiku 4.5 | Pay-per-use / Subscription | API Key / OAuth |
| **Google Gemini** | Gemini 3 Pro, 3 Flash, 2.5 Pro/Flash | Free tier available | API Key |
| **Groq** | GPT-OSS 120B, Qwen3 32B, Llama 4 Scout | Free tier | API Key |
| **GitHub Copilot** | GPT-4o, o3 Mini, Claude 3.5 Sonnet | $10/mo subscription | GitHub Token |
| **xAI (Grok)** | Grok 4, Grok 4.1, Grok 3 | Pay-per-use | API Key |

## Smart Router

Requests are automatically classified and routed to the optimal model:

```
User: "what does this function do?"
  → Role: explain → Ollama/qwen3:8b (free, fast)

User: "fix this bug"
  → Role: debug → Gemini/gemini-3-pro (long context)

User: "refactor this module"
  → Role: refactor → Claude/opus-4.6 (highest quality)

User: "run npm install"
  → Role: bash-only → Ollama/qwen3:8b (free, instant)
```

**Result: 70-90% cost reduction** — expensive models only used when needed.

## Architecture

```
aniclew (11MB binary)
├── Go HTTP Server
│   ├── /v1/messages    → Anthropic-compatible proxy
│   ├── /api/agent      → Coding agent loop (tool execution)
│   ├── /api/routes     → Smart router config
│   ├── /api/kairos     → Background daemon
│   ├── /api/sessions   → Chat history (per-project)
│   ├── /api/gateway    → Team management
│   └── /app            → React web UI
├── Agent Engine
│   ├── 15 Tools (Bash, Read, Write, Edit, Git, Web, Tasks...)
│   ├── Permission System (dangerous command blocking)
│   ├── Context Loader (CLAUDE.md, skills, MCP config)
│   └── Plan Mode & Context Compression
├── Smart Router
│   ├── Request Classifier (16 roles)
│   ├── Cost Tracker
│   └── Auto-escalation on failure
├── KAIROS Daemon
│   ├── Tick Loop (periodic wake-up)
│   ├── AutoDream Memory (cross-session learning)
│   └── Background Tasks (PR review, tests, lint)
├── 7 LLM Providers
│   ├── Anthropic (OAuth passthrough)
│   ├── OpenAI-compatible (OpenAI, Ollama, Groq, GitHub, xAI)
│   └── Gemini (native API)
└── React Web UI (embedded in binary)
    ├── Chat (coding agent with tool visualization)
    ├── Settings (provider/model selection)
    ├── Routes (role → model mapping)
    ├── Costs (real-time cost tracking)
    ├── KAIROS (daemon control)
    ├── Memory (AutoDream)
    └── Team (gateway management)
```

## CLI Options

```bash
aniclew [flags]

Flags:
  --provider    Provider name (ollama, openai, gemini, groq, anthropic, github-copilot, zai)
  --model       Model ID (e.g., qwen3:14b, gpt-4o, claude-sonnet-4-6-20250217)
  --port        Listen port (default: 4000)
  --router      Enable smart router (auto-route by task type)
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/messages` | Anthropic-compatible proxy |
| POST | `/api/agent` | Coding agent with tool execution |
| GET | `/api/config` | Current provider/model config |
| PUT | `/api/config` | Switch provider/model at runtime |
| GET | `/api/providers` | List all providers and models |
| GET/PUT | `/api/routes` | Smart router rules |
| GET | `/api/costs` | Cost breakdown |
| GET | `/api/sessions` | Chat history (per workspace) |
| GET | `/api/workspaces` | List project workspaces |
| POST | `/api/kairos/start` | Start background daemon |
| GET | `/api/kairos` | Daemon status |
| GET | `/api/memory` | AutoDream memory state |
| POST | `/api/gateway/users` | Add team member |
| GET | `/api/gateway/audit` | Audit log |
| GET | `/api/context` | Project context (CLAUDE.md, skills) |
| GET | `/app` | Web UI |

## Cross-Platform Build

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o aniclew.exe ./cmd/proxy/

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o aniclew-mac ./cmd/proxy/

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o aniclew-mac-intel ./cmd/proxy/

# Linux
GOOS=linux GOARCH=amd64 go build -o aniclew-linux ./cmd/proxy/

# Linux ARM (Raspberry Pi, Mini PC)
GOOS=linux GOARCH=arm64 go build -o aniclew-linux-arm ./cmd/proxy/
```

## Inspired By

AniClew is inspired by [Claude Code](https://claude.com/code) by Anthropic. The name comes from:
- **Ani** = Any (any model)
- **Clew** = A ball of thread (Greek mythology: Theseus used a clew to escape the Minotaur's labyrinth)

AniClew is your thread through the labyrinth of AI models — connecting any model to a unified coding agent experience.

The internal background daemon is named **KAIROS** (Greek: "the opportune moment"), inspired by an unreleased feature discovered in Claude Code's source.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

**[Download](https://github.com/Dannykkh/Ani-Clew/releases) · [Documentation](#api-endpoints) · [Issues](https://github.com/Dannykkh/Ani-Clew/issues)**

Made with Go, React, and too much coffee.

</div>

---

## 한국어

### AniClew — 어떤 모델이든, 하나의 코딩 에이전트

Claude Code처럼 코딩하되, **아무 모델이나 쓸 수 있습니다.** Ollama로 로컬에서 무료로, OpenAI/Gemini/Groq으로 클라우드에서.

#### 빠른 시작

```bash
# Ollama 설치 후
ollama pull qwen3:14b

# AniClew 실행
./aniclew.exe

# 브라우저에서 http://localhost:4000/app 접속
```

#### 주요 기능

- **15개 코딩 도구** — 파일 읽기/쓰기/편집, 터미널, Git, 웹 검색, 태스크 관리
- **7개 프로바이더** — Ollama(무료), OpenAI, Gemini, Groq, Anthropic, GitHub Copilot, xAI
- **스마트 라우터** — 작업 유형별 자동 모델 선택 (비용 70-90% 절감)
- **KAIROS 데몬** — 백그라운드 자율 에이전트
- **웹 UI** — 브라우저에서 전부 관리 (설정, 라우팅, 비용, 메모리, 팀)
- **권한 시스템** — 위험한 명령 자동 차단
- **프로젝트 컨텍스트** — CLAUDE.md, 스킬 자동 로드
- **한국어/영어** UI 전환 + AI 응답 언어 설정

#### AniClew 이름의 뜻

- **Ani** = Any (어떤 모델이든)
- **Clew** = 실타래 (그리스 신화: 테세우스가 미궁을 탈출할 때 쓴 실)

AI 모델의 미궁을 통과하는 실마리 — 그것이 AniClew입니다.
