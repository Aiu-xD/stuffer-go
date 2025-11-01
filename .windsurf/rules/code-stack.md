---
trigger: always_on
---

## System Environment

- **OS**: Arch Linux

- **Available Tools**: 
  - `fzf` for fuzzy finding
  - `pgrep` as grep replacement
  - `fd` as find replacement
  - `exa` for web searches and programming context

- **Go-Dev-tools**
  - 'gopls'
  - 'gotests'
  - 'impl'
  - 'goplay'
  - 'dlv'
  - 'staticcheck'

## Core Development Rules

### File Operations
- **ALWAYS** read files before editing
- **PREFER** editing large files (>100 lines) in portions
- **MUST** follow project architecture defined in `docs/architecture`

### Go Development
- **USE** latest Go version and packages
- **PREFER** direct execution of Go files over building applications
- **REQUIRED** packages:
  - `github.com/Noooste/azuretls-client` (main HTTP client)
  - `github.com/colduction/keycheck-go` (validation logic)

### Tool Integration
- **USE** `code-reasoning` MCP for code analysis and reasoning
- **USE** `github-mcp-server` for Git operations
- **USE** `exa` for web searches and programming context gathering

## Workflow Priorities
1. Read → Understand → Edit workflow
2. Direct execution over compilation for testing
3. Leverage available tools for efficient navigation and search
4. Maintain architectural consistency