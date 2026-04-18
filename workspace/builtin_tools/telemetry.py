"""OpenTelemetry (OTEL) instrumentation for the Molecule AI workspace runtime.

Architecture
------------
* One global ``TracerProvider`` is initialised at startup via ``setup_telemetry()``.
* Up to three exporters are wired in:
    1. **OTLP/HTTP** — activated when ``OTEL_EXPORTER_OTLP_ENDPOINT`` is set.
       Point this at any compatible collector (Jaeger, Tempo, Grafana OTEL, …).
    2. **Langfuse OTLP bridge** — activated when the ``LANGFUSE_HOST``,
       ``LANGFUSE_PUBLIC_KEY`` and ``LANGFUSE_SECRET_KEY`` env vars are all present.
       Langfuse ≥4 accepts OTLP/HTTP at ``<host>/api/public/otel``.
       This is a *second* exporter alongside the existing Langfuse LangChain
       callback handler in agent.py — both paths emit spans simultaneously.
    3. **Console** (debug) — activated when ``OTEL_DEBUG=1``.

* **W3C TraceContext** propagation (``traceparent`` / ``tracestate``) is used for
  cross-workspace context injection and extraction so A2A hops form a single
  distributed trace.

* ``make_trace_middleware()`` returns an ASGI middleware that extracts incoming
  trace context from HTTP headers and stores it in a ``ContextVar`` so the
  A2A executor can access it to parent its spans correctly.

GenAI semantic conventions
--------------------------
Attribute constants for ``gen_ai.*`` follow OpenTelemetry GenAI SemConv 1.26.

Usage example
-------------
    # main.py — call once at startup
    from builtin_tools.telemetry import setup_telemetry, make_trace_middleware
    setup_telemetry(service_name=workspace_id)
    instrumented = make_trace_middleware(app.build())

    # Any module
    from builtin_tools.telemetry import get_tracer
    tracer = get_tracer()
    with tracer.start_as_current_span("my_span") as span:
        span.set_attribute("key", "value")

    # Outgoing HTTP — inject W3C headers
    from builtin_tools.telemetry import inject_trace_headers
    headers = inject_trace_headers({"Content-Type": "application/json"})
    await client.post(url, headers=headers, ...)

    # Incoming HTTP — extract context (done automatically by middleware)
    from builtin_tools.telemetry import extract_trace_context
    ctx = extract_trace_context(dict(request.headers))
"""

from __future__ import annotations

import base64
import logging
import os
from contextvars import ContextVar
from typing import Any, Optional

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# GenAI Semantic Convention attribute keys (OTel SemConv 1.26)
# https://opentelemetry.io/docs/specs/semconv/gen-ai/
# ---------------------------------------------------------------------------
GEN_AI_SYSTEM = "gen_ai.system"
GEN_AI_REQUEST_MODEL = "gen_ai.request.model"
GEN_AI_OPERATION_NAME = "gen_ai.operation.name"
GEN_AI_USAGE_INPUT_TOKENS = "gen_ai.usage.input_tokens"
GEN_AI_USAGE_OUTPUT_TOKENS = "gen_ai.usage.output_tokens"
GEN_AI_RESPONSE_FINISH_REASONS = "gen_ai.response.finish_reasons"

# ---------------------------------------------------------------------------
# Workspace / A2A attribute keys
# ---------------------------------------------------------------------------
WORKSPACE_ID_ATTR = "workspace.id"
A2A_SOURCE_WORKSPACE = "a2a.source_workspace_id"
A2A_TARGET_WORKSPACE = "a2a.target_workspace_id"
A2A_TASK_ID = "a2a.task_id"
MEMORY_SCOPE = "memory.scope"
MEMORY_QUERY = "memory.query"

# ---------------------------------------------------------------------------
# Module-level state
# ---------------------------------------------------------------------------
WORKSPACE_ID: str = os.environ.get("WORKSPACE_ID", "unknown")

_initialized: bool = False
_tracer: Any = None  # opentelemetry.trace.Tracer | _NoopTracer

# ContextVar that carries incoming trace context from the ASGI middleware to
# the A2A executor.  Using a ContextVar (rather than a global) is safe with
# asyncio because each task inherits a copy of the context at creation time.
_incoming_trace_context: ContextVar[Optional[Any]] = ContextVar(
    "otel_incoming_trace_context", default=None
)


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def setup_telemetry(service_name: Optional[str] = None) -> None:
    """Initialise the global ``TracerProvider``.  Safe to call multiple times.

    Reads configuration from environment variables:

    ``OTEL_EXPORTER_OTLP_ENDPOINT``
        Base URL of an OTLP-compatible collector (e.g. ``http://jaeger:4318``).
        Spans are sent to ``<endpoint>/v1/traces``.

    ``LANGFUSE_HOST`` + ``LANGFUSE_PUBLIC_KEY`` + ``LANGFUSE_SECRET_KEY``
        When all three are set, a second OTLP exporter is wired to Langfuse's
        ingest endpoint using HTTP Basic auth.

    ``OTEL_DEBUG``
        Set to ``1`` / ``true`` to also print spans to stdout.
    """
    global _initialized, _tracer

    if _initialized:
        return

    try:
        from opentelemetry import propagate, trace
        from opentelemetry.baggage.propagation import W3CBaggagePropagator
        from opentelemetry.propagators.composite import CompositePropagator
        from opentelemetry.sdk.resources import SERVICE_NAME as OTEL_SERVICE_NAME
        from opentelemetry.sdk.resources import Resource
        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry.sdk.trace.export import BatchSpanProcessor, ConsoleSpanExporter
        from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
    except ImportError as exc:
        logger.warning(
            "OTEL: opentelemetry packages not installed — telemetry disabled. "
            "Add opentelemetry-api, opentelemetry-sdk, "
            "opentelemetry-exporter-otlp-proto-http to requirements.txt. "
            "Error: %s",
            exc,
        )
        return

    svc = service_name or f"molecule-{WORKSPACE_ID}"

    resource = Resource.create(
        {
            OTEL_SERVICE_NAME: svc,
            "service.version": "1.0.0",
            WORKSPACE_ID_ATTR: WORKSPACE_ID,
        }
    )

    provider = TracerProvider(resource=resource)

    # -- Exporter 1: Generic OTLP/HTTP ----------------------------------------
    otlp_endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "").rstrip("/")
    if otlp_endpoint:
        try:
            from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

            exporter = OTLPSpanExporter(endpoint=f"{otlp_endpoint}/v1/traces")
            provider.add_span_processor(BatchSpanProcessor(exporter))
            logger.info("OTEL: OTLP/HTTP exporter → %s", otlp_endpoint)
        except ImportError:
            logger.warning(
                "OTEL: OTEL_EXPORTER_OTLP_ENDPOINT is set but "
                "opentelemetry-exporter-otlp-proto-http is not installed"
            )
        except Exception as exc:
            logger.warning("OTEL: OTLP exporter init failed: %s", exc)

    # -- Exporter 2: Langfuse OTLP bridge -------------------------------------
    # Langfuse ≥4 accepts OTLP at <host>/api/public/otel (Basic auth).
    lf_host = os.environ.get("LANGFUSE_HOST", "").rstrip("/")
    lf_public = os.environ.get("LANGFUSE_PUBLIC_KEY", "")
    lf_secret = os.environ.get("LANGFUSE_SECRET_KEY", "")

    if lf_host and lf_public and lf_secret:
        try:
            from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

            lf_endpoint = f"{lf_host}/api/public/otel/v1/traces"
            token = base64.b64encode(f"{lf_public}:{lf_secret}".encode()).decode()
            lf_exporter = OTLPSpanExporter(
                endpoint=lf_endpoint,
                headers={"Authorization": f"Basic {token}"},
            )
            provider.add_span_processor(BatchSpanProcessor(lf_exporter))
            logger.info("OTEL: Langfuse OTLP bridge → %s", lf_endpoint)
        except ImportError:
            logger.warning(
                "OTEL: Langfuse env vars set but "
                "opentelemetry-exporter-otlp-proto-http is not installed"
            )
        except Exception as exc:
            logger.warning("OTEL: Langfuse OTLP bridge init failed: %s", exc)

    # -- Exporter 3: Console (debug) ------------------------------------------
    if os.environ.get("OTEL_DEBUG", "").lower() in ("1", "true", "yes"):
        provider.add_span_processor(BatchSpanProcessor(ConsoleSpanExporter()))
        logger.info("OTEL: console debug exporter enabled")

    # -- Register global provider + W3C propagators ---------------------------
    trace.set_tracer_provider(provider)
    propagate.set_global_textmap(
        CompositePropagator(
            [
                TraceContextTextMapPropagator(),
                W3CBaggagePropagator(),
            ]
        )
    )

    _tracer = trace.get_tracer(
        "molecule.workspace",
        schema_url="https://opentelemetry.io/schemas/1.26.0",
    )
    _initialized = True
    logger.info("OTEL: telemetry initialised for service '%s'", svc)


def get_tracer() -> Any:
    """Return the global ``Tracer``.  Lazily calls ``setup_telemetry()`` if needed.

    Returns a no-op tracer when the opentelemetry packages are not installed so
    that instrumented code never raises ``ImportError``.
    """
    global _tracer

    if not _initialized:
        setup_telemetry()

    if _tracer is None:
        # Packages unavailable — hand back a no-op implementation
        try:
            from opentelemetry import trace

            return trace.get_tracer("molecule.noop")
        except ImportError:
            return _NoopTracer()

    return _tracer


def inject_trace_headers(headers: dict) -> dict:
    """Inject W3C ``traceparent`` / ``tracestate`` into *headers* and return it.

    Mutates the dict in-place so it can be used directly::

        headers = inject_trace_headers({"Content-Type": "application/json"})
        await client.post(url, headers=headers, ...)
    """
    try:
        from opentelemetry import propagate

        propagate.inject(headers)
    except Exception:
        pass  # Never let telemetry break the caller
    return headers


def extract_trace_context(carrier: dict) -> Any:
    """Extract W3C trace context from a header mapping.

    Returns an OpenTelemetry ``Context`` object suitable for::

        tracer.start_as_current_span("name", context=ctx)

    Returns ``None`` when packages are unavailable or no context is present.
    """
    try:
        from opentelemetry import propagate

        return propagate.extract(carrier)
    except Exception:
        return None


def get_current_traceparent() -> Optional[str]:
    """Return the W3C ``traceparent`` string for the active span, or ``None``."""
    try:
        from opentelemetry import trace

        span = trace.get_current_span()
        ctx = span.get_span_context()
        if not ctx.is_valid:
            return None
        trace_id = format(ctx.trace_id, "032x")
        span_id = format(ctx.span_id, "016x")
        flags = "01" if ctx.trace_flags else "00"
        return f"00-{trace_id}-{span_id}-{flags}"
    except Exception:
        return None


def make_trace_middleware(asgi_app: Any) -> Any:
    """Wrap an ASGI application with W3C trace-context extraction middleware.

    The middleware reads ``traceparent`` / ``tracestate`` from every incoming
    HTTP request and stores the extracted ``Context`` in the
    ``_incoming_trace_context`` ContextVar.  The A2A executor reads that
    ContextVar to parent its ``task_receive`` span correctly, forming an
    unbroken distributed trace across workspace hops.

    Usage::

        built = app.build()
        instrumented = make_trace_middleware(built)
        uvicorn.Config(instrumented, ...)
    """

    async def _middleware(scope: dict, receive: Any, send: Any) -> None:  # type: ignore[override]
        if scope.get("type") != "http":
            await asgi_app(scope, receive, send)
            return

        # Decode byte-headers from the ASGI scope (latin-1 per HTTP/1.1 spec)
        raw_headers: list[tuple[bytes, bytes]] = scope.get("headers", [])
        str_headers: dict[str, str] = {
            k.decode("latin-1"): v.decode("latin-1") for k, v in raw_headers
        }

        ctx = extract_trace_context(str_headers)
        token = _incoming_trace_context.set(ctx)
        try:
            await asgi_app(scope, receive, send)
        finally:
            _incoming_trace_context.reset(token)

    return _middleware


# ---------------------------------------------------------------------------
# Helpers for GenAI attributes
# ---------------------------------------------------------------------------

def gen_ai_system_from_model(model_str: str) -> str:
    """Map a ``provider:model`` string to a ``gen_ai.system`` value."""
    if ":" not in model_str:
        return "unknown"
    provider = model_str.split(":", 1)[0].lower()
    return {
        "anthropic": "anthropic",
        "openai": "openai",
        "openrouter": "openrouter",
        "groq": "groq",
        "google_genai": "google",
        "ollama": "ollama",
    }.get(provider, provider)


def record_llm_token_usage(span: Any, result: dict) -> None:
    """Extract token counts from a LangGraph ainvoke result and set span attrs.

    Handles both Anthropic (``usage``) and OpenAI (``token_usage``) metadata
    shapes.  Silently skips if metadata is absent.
    """
    try:
        messages = result.get("messages", [])
        for msg in reversed(messages):
            meta = getattr(msg, "response_metadata", {}) or {}
            # Anthropic
            usage = meta.get("usage", {})
            if usage:
                inp = usage.get("input_tokens") or usage.get("prompt_tokens")
                out = usage.get("output_tokens") or usage.get("completion_tokens")
                if inp is not None:
                    span.set_attribute(GEN_AI_USAGE_INPUT_TOKENS, int(inp))
                if out is not None:
                    span.set_attribute(GEN_AI_USAGE_OUTPUT_TOKENS, int(out))
                return
            # OpenAI
            token_usage = meta.get("token_usage", {})
            if token_usage:
                inp = token_usage.get("prompt_tokens")
                out = token_usage.get("completion_tokens")
                if inp is not None:
                    span.set_attribute(GEN_AI_USAGE_INPUT_TOKENS, int(inp))
                if out is not None:
                    span.set_attribute(GEN_AI_USAGE_OUTPUT_TOKENS, int(out))
                return
    except Exception:
        pass  # Best-effort — never break the caller


# ---------------------------------------------------------------------------
# No-op fallbacks (used when opentelemetry packages are absent)
# ---------------------------------------------------------------------------

class _NoopSpan:
    """Transparent no-op span that satisfies the context-manager protocol."""

    def set_attribute(self, key: str, value: Any) -> None:  # noqa: ARG002
        pass

    def set_status(self, *args: Any, **kwargs: Any) -> None:
        pass

    def record_exception(self, exc: BaseException, *args: Any, **kwargs: Any) -> None:
        pass

    def add_event(self, name: str, *args: Any, **kwargs: Any) -> None:
        pass

    def __enter__(self) -> "_NoopSpan":
        return self

    def __exit__(self, *args: Any) -> None:
        pass


class _NoopTracer:
    """Transparent no-op tracer returned when the SDK is unavailable."""

    def start_as_current_span(self, name: str, *args: Any, **kwargs: Any) -> _NoopSpan:  # noqa: ARG002
        return _NoopSpan()

    def start_span(self, name: str, *args: Any, **kwargs: Any) -> _NoopSpan:  # noqa: ARG002
        return _NoopSpan()
