# Google ADK Adapter

Molecule AI workspace adapter for [Google Agent Development Kit (ADK)](https://github.com/google/adk-python) — Google's official multi-agent Python SDK (~19k ⭐, Apache-2.0).

## Overview

This adapter bridges the A2A protocol used by the Molecule AI platform to Google ADK's runner/session model. Agents are backed by Google Gemini models via AI Studio or Vertex AI. Each workspace gets an `LlmAgent` wrapped in a `Runner` with an `InMemorySessionService`; sessions are tied to A2A task context IDs for stable, isolated per-conversation state.

**Runtime key:** `google-adk`

## Installation

The adapter dependencies are installed automatically by `entrypoint.sh` from this directory's `requirements.txt`:

```bash
pip install -r adapters/google-adk/requirements.txt
```

You'll also need a Google API key (AI Studio) or Vertex AI credentials.

## Configuration

### `config.yaml`

```yaml
runtime: google-adk
model: google:gemini-2.0-flash        # or gemini-1.5-pro, gemini-2.5-flash, etc.
runtime_config:
  agent_name: my-agent                # optional, default: molecule-adk-agent
  max_output_tokens: 8192             # optional, default: 8192
  temperature: 1.0                    # optional, default: 1.0
```

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_API_KEY` | Yes (unless Vertex AI) | Google AI Studio API key |
| `GOOGLE_GENAI_USE_VERTEXAI` | No | Set to `"1"` to use Vertex AI instead of AI Studio |
| `GOOGLE_CLOUD_PROJECT` | When using Vertex AI | GCP project ID |
| `GOOGLE_CLOUD_LOCATION` | When using Vertex AI | GCP region, e.g. `"us-central1"` |

## Usage Example

```python
import asyncio
from adapter_base import AdapterConfig
from adapters.google_adk.adapter import GoogleADKAdapter

async def main():
    config = AdapterConfig(
        model="google:gemini-2.0-flash",
        system_prompt="You are a helpful assistant.",
        runtime_config={
            "agent_name": "demo-agent",
            "max_output_tokens": 1024,
            "temperature": 0.7,
        },
        workspace_id="ws-demo",
    )

    adapter = GoogleADKAdapter()
    await adapter.setup(config)              # validates keys, loads plugins/skills

    executor = await adapter.create_executor(config)  # returns GoogleADKA2AExecutor
    # executor.execute(context, event_queue) is called by the A2A server per turn
    print(f"Adapter: {adapter.display_name()} — model {config.model}")

asyncio.run(main())
```

### Running via A2A

Once the workspace is provisioned, send A2A messages as normal:

```bash
curl -X POST http://localhost:8000 \
  -H 'Content-Type: application/json' \
  -d '{
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"kind": "text", "text": "What is 2 + 2?"}]
      }
    }
  }'
```

## Supported Models

Any model supported by Google ADK and available through your credential path:

| Model | Notes |
|-------|-------|
| `gemini-2.0-flash` | Recommended — fast, cost-effective |
| `gemini-2.5-flash` | Latest preview, strong reasoning |
| `gemini-1.5-pro` | Higher capability, higher latency |
| `gemini-1.5-flash` | Fast, lower cost |

Use the `google:` prefix in `config.yaml` — the adapter strips it before passing the model name to ADK.

## Architecture

```
A2A Request
    │
    ▼
GoogleADKA2AExecutor.execute()
    │
    ├── extract_message_text()   ← shared_runtime helper
    ├── _ensure_session()        ← create/reuse InMemorySessionService session
    ├── _build_content()         ← wrap text in google.genai.types.Content
    │
    ▼
runner.run_async(session_id, user_id, new_message)
    │
    ▼
ADK Event stream → filter is_final_response() → extract text
    │
    ▼
event_queue.enqueue_event(new_agent_text_message(reply))
    │
    ▼
A2A Response
```

## License

Apache-2.0 — same as [google/adk-python](https://github.com/google/adk-python).
