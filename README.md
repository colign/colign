<div align="center">

# Colign

**Align your team before AI writes the code.**

An open-source spec-driven development platform where teams discuss, write, and review specs together — then let AI implement them.

<p>
  <a href="https://github.com/gobenpark/colign/actions/workflows/ci.yml"><img src="https://github.com/gobenpark/colign/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue.svg" alt="License: AGPL-3.0"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="https://nextjs.org"><img src="https://img.shields.io/badge/Next.js-16-000000?logo=next.js&logoColor=white" alt="Next.js"></a>
  <a href="CONTRIBUTING.md"><img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome"></a>
</p>

</div>

<img src="docs/assets/dashboard.png" width="100%" alt="Colign Dashboard"/>

<!-- TODO: Add demo video -->

---

## Why Colign?

AI vibe coding has made individual developers incredibly productive. Claude Code, Cursor, Copilot — there are already powerful tools for writing code on your own.

But real software is never built alone.

Working in a team means aligning on **"what to build"** before writing any code. You gather requirements, discuss specs, define acceptance criteria, get reviews, and only then start implementing. Most AI development tools don't address this **upstream collaboration** — the discussion, alignment, and spec writing that happens before code.

That's where Colign comes in.

- Discuss and write specs together with AI
- Co-edit specs in real-time with your team
- Structure the alignment process with a Draft → Design → Review → Ready workflow
- Once specs are finalized, AI helps implement based on them

**AI that writes code already exists. Colign makes sure your team is looking at the same thing before the code gets written.**

## Features

- **AI Spec Generation** — Generate structured specs from a single prompt using your own API key
- **Real-time Co-editing** — Collaborate on specs simultaneously with your team
- **Structured Proposals** — Problem, Scope, Approach, Acceptance Criteria in a consistent format
- **Project Memory** — Persistent context (domain rules, constraints, decisions) shared across all specs
- **Workflow States** — Draft → Design → Review → Ready pipeline for every change
- **MCP Server** — Connect Claude Code, Cursor, or any MCP-compatible AI tool to read/write specs
- **Dashboard & Inbox** — Track spec status, reviews, and notifications in one place

## Spec-Driven Development

Colign follows a **Spec-Driven Development (SDD)** approach. When AI can generate code in minutes, the bottleneck shifts from "can we build it?" to "have we defined it correctly?"

### Two-layer spec architecture

Traditional PRDs tried to put everything in one 30-page document. Colign splits this into two layers:

**Project Memory** — Strategic context that rarely changes. Domain rules, business constraints, target users, and technical decisions. Written once, referenced by every Change.

**Structured Proposal** — Tactical spec for each Change. Lightweight, structured, and designed for both humans and AI agents to read:

| Section | Required | Purpose |
|---------|:--------:|---------|
| **Problem** | Yes | Why is this change needed? |
| **Scope** | Yes | What specifically will change? |
| **Out of Scope** | No | What is explicitly NOT part of this? |
| **Approach** | No | Technical direction and rationale |
| **Acceptance Criteria** | Yes | Given/When/Then scenarios |

### Two paths for AI integration

1. **Platform AI** — Use your own API key to generate structured specs directly in Colign
2. **External AI** — Connect Claude Code, Cursor, or other AI tools via MCP Server to read/write specs in Colign

> For design decisions and competitive analysis behind this structure, see [docs/structured-proposal.md](docs/structured-proposal.md).

## MCP Integration

Colign exposes an MCP (Model Context Protocol) server so any AI tool — Claude Code, Cursor, Windsurf, VS Code Copilot — can read and write specs directly.

### Streamable HTTP (SaaS)

No binary to install. Just add the URL and your API token:

```json
{
  "mcpServers": {
    "colign": {
      "url": "https://api.colign.co/mcp",
      "headers": {
        "Authorization": "Bearer col_your_token_here"
      }
    }
  }
}
```

Generate an API token at **Settings > AI & API Keys** in the Colign web app.

### stdio (Local / Self-hosted)

For local development or self-hosted instances:

```bash
go build -o colign-mcp ./cmd/mcp

COLIGN_API_TOKEN=col_... COLIGN_API_URL=http://localhost:8080 ./colign-mcp
```

### MCP Tools

| Tool | Description |
|------|-------------|
| `list_projects` | List all accessible projects |
| `get_change` | Get change details including stage |
| `read_spec` | Read a spec document (proposal, design, spec, tasks) |
| `write_spec` | Write or update a spec document |
| `list_tasks` | List implementation tasks for a change |
| `update_task` | Update a task's status (todo, in_progress, done) |
| `suggest_spec` | Get suggestions for improving a spec |
| `list_acceptance_criteria` | List acceptance criteria (Given/When/Then) |
| `create_acceptance_criteria` | Create BDD-style acceptance criteria |

## Claude Code Plugin

Colign ships with a [Claude Code plugin](plugins/claude-code/) that adds workflow skills on top of MCP.

### Install

```bash
# 1. Register marketplace
/plugin marketplace add https://github.com/colign/plugin

# 2. Install plugin
/plugin install colign@colign

# 3. Reload and connect (browser opens for OAuth login)
/reload-plugins
```

### Workflow Skills

6 skills that follow the change lifecycle:

```
onboard → explore → propose → plan → implement → complete
```

| Skill | Stage | Description |
|-------|-------|-------------|
| `/colign:onboard` | Setup | Verify MCP connection and API token |
| `/colign:explore` | Any | Browse projects, read specs, check status |
| `/colign:propose` | Draft → Problem | Write a structured proposal |
| `/colign:plan` | Problem → Solution | Break proposal into architecture and tasks |
| `/colign:implement` | Solution → Review | Code against the spec, update task progress |
| `/colign:complete` | Review → Done | Verify all tasks done, advance workflow |

Skills trigger automatically by context (e.g., "implement the next task") or explicitly via `/colign:implement`.

## Getting Started

```bash
# Start all services (API + DB + Redis)
docker-compose up --build

# Run frontend (separate terminal)
cd web && npm install && npm run dev
```

Open http://localhost:3000 to access the app.

## Development

```bash
# Generate proto (Go + TypeScript)
cd proto && buf generate

# Run API server locally
go run ./cmd/api

# Run tests
go test ./...
```

## Prerequisites

- Go 1.26+
- Node.js 20+
- Docker & Docker Compose
- [buf](https://buf.build/docs/installation)

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0).
