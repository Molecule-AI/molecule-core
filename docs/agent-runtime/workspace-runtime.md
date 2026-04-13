# Workspace Runtime

The `workspace-template/` directory is Molecule AI's unified runtime image. Every provisioned workspace starts from this image, loads its own config, selects a runtime adapter, registers an Agent Card, exposes A2A, and joins the platform heartbeat/activity loop.

## Runtime Matrix In Current `main`

Current `main` ships six adapters:

- `langgraph`
- `deepagents`
- `claude-code`
- `crewai`
- `autogen`
- `openclaw`

This is the merged runtime surface today. Branch-level experiments such as NemoClaw are separate and should be treated as roadmap/WIP, not merged support.

Adapter-specific behavior is documented in [Agent Runtime Adapters](./cli-runtime.md).

## What The Runtime Is Responsible For

- loading `config.yaml`
- running preflight checks before the workspace goes live
- selecting an adapter based on `runtime`
- loading local skills plus plugin-mounted shared rules/skills
- constructing an Agent Card
- serving A2A over HTTP
- registering with the platform and sending heartbeats
- reporting activity and task state
- integrating with awareness-backed memory when configured
- hot-reloading skills while the workspace is running

## Environment Model

Common runtime environment variables:

```bash
WORKSPACE_ID=ws-123
WORKSPACE_CONFIG_PATH=/configs
PLATFORM_URL=http://platform:8080
PARENT_ID=
AWARENESS_URL=http://awareness:37800
AWARENESS_NAMESPACE=workspace:ws-123
LANGFUSE_HOST=http://langfuse-web:3000
LANGFUSE_PUBLIC_KEY=...
LANGFUSE_SECRET_KEY=...
```

Important behavior:

- `WORKSPACE_CONFIG_PATH` points at the mounted config directory for that workspace.
- `AWARENESS_URL` + `AWARENESS_NAMESPACE` enable workspace-scoped awareness-backed memory.
- If awareness is absent, runtime memory tools fall back to the platform memory endpoints for compatibility.

## Startup Sequence

At a high level, `workspace-template/main.py` does this:

1. Initialize telemetry.
2. Load `config.yaml`.
3. Run preflight validation.
4. Build the heartbeat loop.
5. Resolve the adapter from `config.runtime`.
6. Let the adapter run `setup()` and build an executor.
7. Build the Agent Card from loaded skills and runtime config.
8. Register the workspace with `POST /registry/register`.
9. Start heartbeats.
10. Start the skill watcher when skills are configured.
11. Serve the A2A app through Uvicorn.

## Core Runtime Pieces

| File | Responsibility |
|---|---|
| `main.py` | Entry point, adapter bootstrap, Agent Card registration, heartbeat startup, initial prompt execution |
| `config.py` | Parses `config.yaml` into the runtime config dataclasses |
| `adapters/` | Adapter registry and adapter implementations |
| `claude_sdk_executor.py` | `ClaudeSDKExecutor` — Claude Code runtime via `claude-agent-sdk` (replaces subprocess) |
| `executor_helpers.py` | Shared helpers for all executors: memory, delegation, heartbeat, system prompt, error sanitization |
| `a2a_executor.py` | Shared LangGraph execution bridge and current-task reporting |
| `cli_executor.py` | `CLIAgentExecutor` — subprocess executor for Codex, Ollama, custom runtimes |
| `skills/loader.py` | Parses `SKILL.md`, loads tool modules, returns loaded skill metadata |
| `skills/watcher.py` | Hot reload path for skill changes |
| `plugins.py` | Scans mounted plugins for shared rules, prompt fragments, and extra skills |
| `tools/memory.py` | Agent memory tools |
| `tools/awareness_client.py` | Awareness-backed persistence wrapper |
| `coordinator.py` | Coordinator-only delegation path for team leads |

## Skills, Plugins, And Hot Reload

The runtime combines three sources of capability:

1. **workspace-local skills** from `skills/<skill>/SKILL.md`
2. **plugin-mounted rules and shared skills** from `/plugins`
3. **built-in tools** like delegation, approval, memory, sandbox, and telemetry helpers

Hot reload matters because the runtime is designed to keep a workspace alive while its capability surface evolves:

- edit `SKILL.md`
- add/remove skill files
- update tool modules
- modify config prompt references

The watcher rescans the skill package, rebuilds the agent tool surface, and updates the Agent Card so peers and the canvas reflect the new capabilities.

## Awareness And Memory Integration

The runtime keeps the agent-facing contract stable:

- `commit_memory(content, scope)`
- `search_memory(query, scope)`

When awareness is configured:

- the tools route durable facts to the workspace's own awareness namespace
- the namespace defaults to `workspace:<workspace_id>` unless explicitly overridden

When awareness is not configured:

- the same tools fall back to the platform memory endpoints

That design lets the platform improve the backend memory boundary without forcing every agent prompt or tool signature to change.

## Coordinator Enforcement

`coordinator.py` is not a generic “smart agent” mode. It is intentionally strict:

- coordinators delegate
- coordinators synthesize
- coordinators do not quietly do the child work themselves

This matters because Molecule AI wants hierarchy to remain operationally real, not cosmetic.

## A2A And Registration

Each workspace exposes an A2A server, builds an Agent Card, and registers with the platform. The platform is used for:

- discovery
- liveness
- event fanout
- proxying browser-initiated A2A calls

But the long-term collaboration model remains direct workspace-to-workspace communication via A2A.

## Related Docs

- [Agent Runtime Adapters](./cli-runtime.md)
- [Skills](./skills.md)
- [Config Format](./config-format.md)
- [System Prompt Structure](./system-prompt-structure.md)
- [Memory Architecture](../architecture/memory.md)
