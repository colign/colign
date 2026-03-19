# Contributing to Coalign

Thank you for your interest in contributing to Coalign! This guide will help you get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Reporting Issues](#reporting-issues)
- [License](#license)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/<your-username>/coalign.git
   cd coalign
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/gobenpark/coalign.git
   ```

## Development Setup

### Prerequisites

- Go 1.21+
- Node.js 20+
- Docker & Docker Compose
- [buf](https://buf.build/docs/installation) (for protobuf)
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

### Initial Setup

```bash
# Install dependencies and generate proto code
make setup

# Start PostgreSQL and Redis
make up

# Run database migrations
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/coalign?sslmode=disable" up
```

### Running Locally

```bash
# Terminal 1: API server
make run

# Terminal 2: Frontend
make web-dev

# Terminal 3: Hocuspocus (realtime collaboration)
cd hocuspocus && npm run dev
```

### Useful Commands

| Command | Description |
|---------|-------------|
| `make proto` | Generate protobuf code (Go + TypeScript) |
| `make proto-lint` | Lint protobuf definitions |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |
| `make build` | Build API and MCP server binaries |
| `make web-build` | Build the Next.js frontend |
| `make up` / `make down` | Start/stop Docker services |

## Making Changes

1. Create a branch from `main`:
   ```bash
   git checkout -b feat/your-feature
   ```
2. Make your changes
3. Ensure all checks pass:
   ```bash
   make lint
   make test
   cd proto && buf lint
   ```
4. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat: add workflow template support
   fix: resolve JWT token refresh race condition
   docs: update API authentication guide
   refactor: simplify project service layer
   ```

## Pull Request Process

1. Update your branch with the latest `main`:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```
2. Push your branch and create a Pull Request
3. Fill in the PR template with:
   - **Summary** of what changed and why
   - **Test plan** describing how you verified the changes
4. Wait for CI to pass and a maintainer to review
5. Address review feedback with additional commits (don't force-push during review)

### What We Look For in Reviews

- Does the change solve the stated problem?
- Are there tests for new functionality?
- Does the code follow existing patterns in the codebase?
- Are protobuf changes backward-compatible?
- No secrets, credentials, or `.env` files committed

## Coding Standards

### Go (Backend)

- Follow [Effective Go](https://go.dev/doc/effective_go) conventions
- Use `golangci-lint` before submitting
- Keep functions focused and small
- Handle errors explicitly; avoid ignoring returned errors
- Add tests for new functionality in `*_test.go` files

### TypeScript (Frontend)

- Follow the existing ESLint configuration
- Use TypeScript strict mode; avoid `any` types
- Use server components by default (Next.js App Router)

### Protobuf

- Run `buf lint` before submitting proto changes
- Keep backward compatibility in mind
- Follow [Buf style guide](https://buf.build/docs/best-practices/style-guide/)

## Reporting Issues

### Bug Reports

Please include:
- Steps to reproduce
- Expected vs actual behavior
- Environment (OS, Go version, Node version)
- Relevant logs or screenshots

### Feature Requests

Please include:
- Problem you're trying to solve
- Proposed solution (if any)
- How it fits with Coalign's SDD workflow

### Security Issues

Do **not** open a public issue for security vulnerabilities. See [SECURITY.md](SECURITY.md) for responsible disclosure instructions.

## License

By contributing to Coalign, you agree that your contributions will be licensed under the [AGPL-3.0 License](LICENSE).
