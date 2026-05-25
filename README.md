# Pathfinder

A Go-based AI agent framework for building LLM-powered applications with tool calling, multi-step reasoning, and pluggable backends. Built on top of the OpenAI API (via OpenRouter) and inspired by state-of-the-art agent architectures.

## Overview

Pathfinder is an experimental agent framework that enables an LLM to autonomously plan, reason, and execute tasks by combining:

- **Chain-of-thought / reasoning** capabilities of modern LLMs
- A **graph-based execution engine** for orchestrating multi-step agent workflows
- **Tool calling** with automatic schema generation from Go function signatures
- Pluggable **execution backends** (filesystem, shell, internet search, etc.)

The agent iterates between generating LLM responses (including reasoning and tool calls) and executing the requested tools, forming an intelligent loop capable of solving complex tasks.

## Project Structure

```
pathfinder/
├── main.go                          # Entry point – sets up the agent, backends, and REPL loop
├── agent/
│   ├── agent.go                     # Core Agent struct, session management, backend registration
│   ├── llmNode.go                   # Graph node: generates LLM responses from conversation history
│   └── toolNode.go                  # Graph node: executes function calls from the LLM response
├── graph/
│   ├── graph.go                     # Generic graph execution engine with state management
│   ├── command.go                   # Graph commands (Goto, Exit) for controlling execution flow
│   └── node.go                      # Node type definition
├── llms/
│   ├── llm.go                       # LLM interface (ILlm, IToolCallingLlm)
│   └── openai_llm.go                # OpenAI-compatible LLM implementation (via OpenRouter)
├── backends/
│   ├── backend.go                   # Backend interfaces (IFileSystemBackend, IExecutionBackend)
│   ├── definitions.go               # Predefined tool definitions for backends
│   ├── local_filesystem.go          # Local filesystem backend (ls, read, grep, glob, write, edit)
│   ├── local_shell.go               # Local shell execution backend
│   └── models.go                    # Backend input/output models
├── messages/
│   └── models.go                    # Message types (Human, AI, Tool, System) and helper methods
├── stores/
│   └── session.go                   # Session store interface & in-memory implementation
├── tools/
│   ├── executor.go                  # Tool executor that dispatches function calls
│   ├── function_call.go             # FunctionDefinition, schema generation via reflection
│   ├── basic.go                      # Built-in tools: get_date_time
│   └── internet.go                   # Built-in tools: internet_search, open_url
├── .env                             # Environment variables (API keys, config)
├── .gitignore
├── go.mod
└── go.sum
```

## How It Works

1. The agent is initialized with an LLM, a tool executor, and a session store.
2. A **directed graph** with two nodes — `llmNode` and `toolNode` — controls execution flow.
3. **`llmNode`**: Receives the full conversation history, sends it to the LLM, and reads the response. If the LLM wants to call a tool, the graph transitions to `toolNode`. Otherwise, the conversation ends.
4. **`toolNode`**: Extracts tool calls from the last LLM message, executes them via the tool executor, appends the results to the conversation, and routes back to `llmNode` for the next reasoning step.
5. The loop continues until the LLM decides not to make any more tool calls or a maximum iteration count is reached.

## Built-in Tools

| Tool | Description |
|---|---|
| `get_date_time` | Returns the current date and time |
| `internet_search` | Performs a web search using Tavily API |
| `open_url` | Reads the contents of a web URL |
| `ls` | Lists files in a directory (via filesystem backend) |
| `read` | Reads a file's content (via filesystem backend) |
| `grep` | Searches for a pattern in files (via filesystem backend) |
| `glob` | Finds files matching a glob pattern (via filesystem backend) |
| `write` | Writes content to a file (via filesystem backend) |
| `edit` | Performs string replacement in a file (via filesystem backend) |
| `execute` | Runs shell commands and returns their output (via execution backend) |

## Configuration

Create a `.env` file with the following variables:

```bash
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_API_KEY=your-api-key
MODEL=openai/gpt-4o
WORK_DIR=/path/to/working/directory
TAVILY_API_KEY=your-tavily-api-key   # Required for internet tools
LOG_LEVEL=0                           # 0=Debug, 1=Info, 2=Warn, 3=Error
```

## Running the Project

You can use `make` to manage the project:

- `make build`: Build the project and its TUI component.
- `make run`: Build and run the main CLI application.
- `make run-tui`: Build and run the Terminal User Interface (TUI).
- `make test`: Run the project's test suite.
- `make clean`: Clean up build artifacts.

## Roadmap – Future Features

Here are some features that would be nice to have:

- **Persistent session storage** – Move beyond the in-memory store to a database-backed session store (e.g., SQLite or PostgreSQL) for resilience across restarts.
- **Streaming responses** – Support streaming LLM output to the user in real-time via Server-Sent Events (SSE) or WebSockets.
- **Agent memory & planning** – Add long-term memory and explicit planning nodes so the agent can break down complex multi-step tasks more reliably.
- **Tool result caching** – Cache tool call results to avoid redundant executions and reduce API costs for repeated queries.
- **Multi-agent orchestration** – Support multiple specialized agents that can collaborate, with one orchestrator delegating subtasks.
- **Custom tool definitions via config** – Allow users to register custom tools from a config file or DSL without writing Go code.
- **Graph visualization** – Add a way to export and visualize the agent's execution graph for debugging and observability.
- **Error recovery & retry policies** – Implement configurable retry logic and fallback strategies for tool call failures.
