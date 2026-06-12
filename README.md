# Pathfinder

A modular AI agent framework built in Go, featuring a graph-based execution engine, pluggable middleware pipeline, and extensible backends for tool execution.

## Features

- **Graph-based execution** — The agent processes user requests through a directed graph of nodes (before agent → before LLM → LLM → after LLM → tool → before LLM → ... → after agent), enabling complex, multi-step reasoning with tool use.
- **Pluggable middleware** — Extend agent behavior at key lifecycle points. Built-in middlewares include:
  - **Memory** — Persistent memory backed by SQLite FTS for long-term knowledge retention.
  - **Skills** — Inject specialized domain knowledge and capabilities into the agent's system prompt.
  - **Summarization** — Automatically summarize lengthy conversations to stay within the model's context window.
  - **HITL (Human-in-the-Loop)** — Intercept tool calls for user approval before execution.
- **Tool-calling LLMs** — Uses OpenAI-compatible APIs (OpenAI, OpenRouter, etc.) with built-in support for function/tool calling.
- **Backends** — Abstracted execution and file system backends:
  - **Shell backend** — Execute arbitrary shell commands in a working directory.
  - **File system backend** — Read, write, edit, list, grep, and glob files (with `.gitignore`-aware ignore patterns).
- **Built-in tools** — Internet search (via Tavily), URL content extraction, and date/time retrieval.
- **Resumable sessions** — Session state can be persisted via a pluggable store interface (in-memory included, SQLite planned).

## Architecture

```
User Input
    │
    ▼
┌─────────────────┐
│ beforeAgentNode │  ← Middlewares: Memory, Skills inject system prompts
└────────┬────────┘
         ▼
┌─────────────────┐
│  beforeLlmNode  │  ← Middlewares: Summarization, HITL prepare state
└────────┬────────┘
         ▼
┌─────────────────┐
│    llmNode      │  ← Call LLM with conversation + tools
└────────┬────────┘
         ▼
┌─────────────────┐
│  afterLlmNode   │  ← Middlewares: HITL intercepts tool calls
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌─────────┐  ┌─────────────────┐
│ toolNode │  │ afterAgentNode  │  → Exit
└────┬────┘  └─────────────────┘
     │           (no tool calls)
     ▼
  (back to beforeLlmNode)
```

## Getting Started

### Prerequisites

- Go 1.26+
- An API key for an OpenAI-compatible provider (e.g., OpenAI, OpenRouter)
- (Optional) A Tavily API key for internet search

### Installation

```bash
git clone https://github.com/ParthPant/pathfinder.git
cd pathfinder
```

### Configuration

Create a `.env` file in the project root:

```bash
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_API_KEY=your_api_key_here
MODEL=your-model-name          # e.g., anthropic/claude-sonnet-4-20250514
SUMMARY_MODEL=your-summary-model
WORK_DIR=/path/to/workspace
TAVILY_API_KEY=your_tavily_key   # optional
LOG_LEVEL=4                      # 4 = DEBUG, 0 = ERROR (optional)
```

### Build & Run

```bash
make build
make run
```

Or use Go directly:

```bash
go build -o build/pathfinder .
./build/pathfinder
```

## Usage

### Demo

![Pathfinder TUI Demo](demo.gif)

Once running, Pathfinder provides an interactive REPL. Type your requests at the `User:` prompt and the agent will reason, use tools, and respond.

```
User: What files are in the current directory?
AI: Let me check.
AI [toolCall]: ls
Tool Response: ...
AI: The current directory contains ...
```

### Human-in-the-Loop

When the agent wants to execute an operation, you'll be prompted for approval:

```
Agent wants to call execute with args {"command":"rm -rf /tmp/test"} (y/n):
```

## Project Structure

```
├── agent/          — Agent core, graph nodes, middlewares, and memory store
│   ├── memory/     — SQLite-backed persistent memory with full-text search
├── backends/       — Execution and file system backend abstractions
├── graph/          — Generic graph-based execution engine
├── llms/           — LLM abstraction and OpenAI-compatible implementation
├── messages/       — Message and conversation models
├── prompts/        — System prompt templates (embedded via Go embed)
├── stores/         — Session state persistence (in-memory, extensible)
├── tools/          — Built-in tools (internet search, URL reading, date/time)
├── main.go         — Entry point and middleware wiring
├── Makefile        — Build, test, and run targets
└── TODO.md         — Upcoming improvements
```

## Built-in Tools

| Tool | Description |
|---|---|
| `execute` | Run a Unix shell command |
| `ls` | List files and directories |
| `read` | Read file content |
| `write` | Create a new file |
| `edit` | Edit a file by replacing string occurrences |
| `grep` | Search for a literal text pattern in files |
| `glob` | Find files matching a glob pattern |
| `internet_search` | Search the web (via Tavily) |
| `open_url` | Read content from a URL |
| `get_date_time` | Get the current date and time |
| `create_memory` | Save a persistent memory entry |

## Makefile Targets

```bash
make build       # Build the project binary
make run         # Build and run
make test        # Run all tests
make clean       # Remove build artifacts
make help        # Show all targets
```

## TODO

- [x] File system tools with `.gitignore`-aware ignore patterns
- [x] Ability to spawn subagents
- [ ] Atomic file edit via swap files
- [ ] Resumable sessions with SQLite store
- [ ] Retry failed agent runs
- [ ] Execution and FS backend via containers (docker/podman)
