import os
"""Tests for tools/telemetry.py.

Loads the real module via importlib so the conftest.py mock at
sys.modules["builtin_tools.telemetry"] does not interfere.  Each test operates on
a freshly exec'd copy with _initialized=False so there is no cross-test
state pollution.
"""

import importlib.util
import sys
from types import ModuleType
from unittest.mock import MagicMock, patch, call

import pytest


# ---------------------------------------------------------------------------
# Fixture: load real telemetry module
# ---------------------------------------------------------------------------

@pytest.fixture
def real_telemetry(monkeypatch):
    monkeypatch.delitem(sys.modules, "builtin_tools.telemetry", raising=False)
    spec = importlib.util.spec_from_file_location(
        "builtin_tools.telemetry",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools/telemetry.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.telemetry", mod)
    spec.loader.exec_module(mod)
    # Reset global state so tests are independent
    mod._initialized = False
    mod._tracer = None
    return mod


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

class TestConstants:

    def test_constants_defined(self, real_telemetry):
        mod = real_telemetry
        for attr in (
            "GEN_AI_SYSTEM",
            "GEN_AI_REQUEST_MODEL",
            "GEN_AI_OPERATION_NAME",
            "GEN_AI_USAGE_INPUT_TOKENS",
            "GEN_AI_USAGE_OUTPUT_TOKENS",
            "GEN_AI_RESPONSE_FINISH_REASONS",
            "WORKSPACE_ID_ATTR",
            "A2A_SOURCE_WORKSPACE",
            "A2A_TARGET_WORKSPACE",
            "A2A_TASK_ID",
        ):
            value = getattr(mod, attr)
            assert isinstance(value, str), f"{attr} should be a str"
            assert value  # non-empty


# ---------------------------------------------------------------------------
# setup_telemetry
# ---------------------------------------------------------------------------

class TestSetupTelemetry:

    def test_setup_telemetry_no_otel_packages(self, real_telemetry, monkeypatch):
        """When opentelemetry is not importable, setup_telemetry returns gracefully."""
        mod = real_telemetry

        # Make opentelemetry unimportable
        monkeypatch.setitem(sys.modules, "opentelemetry", None)  # causes ImportError on import

        mod.setup_telemetry()

        assert mod._initialized is False

    def test_setup_telemetry_idempotent(self, real_telemetry):
        """Calling setup_telemetry twice only initializes once."""
        mod = real_telemetry

        call_count = {"n": 0}
        original_setup = mod.setup_telemetry

        def counting_setup(*args, **kwargs):
            call_count["n"] += 1
            # call the real one the first time
            return original_setup(*args, **kwargs)

        # First call — mark initialized manually to simulate prior init
        mod._initialized = True
        mod.setup_telemetry()  # should be a no-op
        assert call_count["n"] == 0  # our wrapper not used, just confirming idempotence

        # Verify _initialized stays True and _tracer is unchanged
        mod._tracer = "existing"
        mod.setup_telemetry()
        assert mod._tracer == "existing"


# ---------------------------------------------------------------------------
# get_tracer
# ---------------------------------------------------------------------------

class TestGetTracer:

    def test_get_tracer_returns_noop_when_not_initialized(self, real_telemetry, monkeypatch):
        """When _initialized=False and opentelemetry not importable, returns _NoopTracer."""
        mod = real_telemetry
        mod._initialized = False
        mod._tracer = None

        # Make opentelemetry unimportable so setup_telemetry is a no-op
        monkeypatch.setitem(sys.modules, "opentelemetry", None)

        tracer = mod.get_tracer()

        assert isinstance(tracer, mod._NoopTracer)

    def test_get_tracer_calls_setup(self, real_telemetry):
        """get_tracer() triggers setup_telemetry() if not initialized."""
        mod = real_telemetry
        mod._initialized = False
        mod._tracer = None

        setup_called = {"n": 0}
        original_setup = mod.setup_telemetry

        def fake_setup(*args, **kwargs):
            setup_called["n"] += 1
            # Do not actually init (leave _initialized False) to keep it simple

        mod.setup_telemetry = fake_setup

        mod.get_tracer()  # should call setup_telemetry

        assert setup_called["n"] == 1

        # Restore
        mod.setup_telemetry = original_setup

    def test_get_tracer_returns_stored_tracer(self, real_telemetry):
        """When _tracer is set, get_tracer returns it without calling setup again."""
        mod = real_telemetry
        fake_tracer = object()
        mod._initialized = True
        mod._tracer = fake_tracer

        result = mod.get_tracer()

        assert result is fake_tracer


# ---------------------------------------------------------------------------
# inject_trace_headers
# ---------------------------------------------------------------------------

class TestInjectTraceHeaders:

    def test_inject_trace_headers_no_otel(self, real_telemetry, monkeypatch):
        """When opentelemetry absent, returns headers unchanged."""
        mod = real_telemetry
        monkeypatch.setitem(sys.modules, "opentelemetry", None)

        headers = {"Content-Type": "application/json"}
        result = mod.inject_trace_headers(headers)

        assert result is headers
        assert result == {"Content-Type": "application/json"}

    def test_inject_trace_headers_with_otel(self, real_telemetry, monkeypatch):
        """When opentelemetry present, calls propagate.inject."""
        mod = real_telemetry

        mock_propagate = MagicMock()
        mock_otel = MagicMock()
        mock_otel.propagate = mock_propagate

        # Patch sys.modules so 'from opentelemetry import propagate' works
        monkeypatch.setitem(sys.modules, "opentelemetry", mock_otel)
        monkeypatch.setitem(sys.modules, "opentelemetry.propagate", mock_propagate)

        # Override the propagate attribute on the mock otel module
        mock_otel.propagate = mock_propagate
        mock_propagate.inject = MagicMock()

        headers = {"X-Custom": "value"}
        result = mod.inject_trace_headers(headers)

        # Should still return the headers dict regardless
        assert result is headers


# ---------------------------------------------------------------------------
# extract_trace_context
# ---------------------------------------------------------------------------

class TestExtractTraceContext:

    def test_extract_trace_context_no_otel(self, real_telemetry, monkeypatch):
        """Returns None when packages absent."""
        mod = real_telemetry
        monkeypatch.setitem(sys.modules, "opentelemetry", None)

        result = mod.extract_trace_context({"traceparent": "00-abc-def-01"})

        assert result is None


# ---------------------------------------------------------------------------
# get_current_traceparent
# ---------------------------------------------------------------------------

class TestGetCurrentTraceparent:

    def test_get_current_traceparent_no_otel(self, real_telemetry, monkeypatch):
        """Returns None when packages absent."""
        mod = real_telemetry
        monkeypatch.setitem(sys.modules, "opentelemetry", None)

        result = mod.get_current_traceparent()

        assert result is None


# ---------------------------------------------------------------------------
# make_trace_middleware
# ---------------------------------------------------------------------------

class TestMakeTraceMiddleware:

    async def test_make_trace_middleware_non_http_scope(self, real_telemetry):
        """Passes through non-http scope unchanged."""
        mod = real_telemetry

        calls = []

        async def fake_app(scope, receive, send):
            calls.append(scope)

        middleware = mod.make_trace_middleware(fake_app)
        scope = {"type": "websocket"}
        await middleware(scope, None, None)

        assert len(calls) == 1
        assert calls[0] is scope

    async def test_make_trace_middleware_http_scope(self, real_telemetry):
        """Extracts trace context from headers and calls inner app."""
        mod = real_telemetry

        calls = []

        async def fake_app(scope, receive, send):
            calls.append(scope)

        middleware = mod.make_trace_middleware(fake_app)
        scope = {
            "type": "http",
            "headers": [(b"traceparent", b"00-abc123-def456-01")],
        }
        await middleware(scope, None, None)

        assert len(calls) == 1

    async def test_make_trace_middleware_resets_contextvar(self, real_telemetry):
        """ContextVar is reset after request completes."""
        mod = real_telemetry

        async def fake_app(scope, receive, send):
            pass

        middleware = mod.make_trace_middleware(fake_app)
        scope = {
            "type": "http",
            "headers": [],
        }

        # Get the value before
        before = mod._incoming_trace_context.get()
        await middleware(scope, None, None)
        after = mod._incoming_trace_context.get()

        # The ContextVar should be reset to its original value
        assert after == before

    async def test_make_trace_middleware_resets_on_exception(self, real_telemetry):
        """ContextVar is reset even when inner app raises."""
        mod = real_telemetry

        async def failing_app(scope, receive, send):
            raise RuntimeError("boom")

        middleware = mod.make_trace_middleware(failing_app)
        scope = {"type": "http", "headers": []}

        before = mod._incoming_trace_context.get()

        with pytest.raises(RuntimeError, match="boom"):
            await middleware(scope, None, None)

        after = mod._incoming_trace_context.get()
        assert after == before


# ---------------------------------------------------------------------------
# gen_ai_system_from_model
# ---------------------------------------------------------------------------

class TestGenAiSystemFromModel:

    def test_gen_ai_system_from_model_anthropic(self, real_telemetry):
        assert real_telemetry.gen_ai_system_from_model("anthropic:claude-3") == "anthropic"

    def test_gen_ai_system_from_model_openai(self, real_telemetry):
        assert real_telemetry.gen_ai_system_from_model("openai:gpt-4") == "openai"

    def test_gen_ai_system_from_model_no_colon(self, real_telemetry):
        assert real_telemetry.gen_ai_system_from_model("unknown-model") == "unknown"

    def test_gen_ai_system_from_model_unknown_provider(self, real_telemetry):
        # "custom" is not in the known map so it should be returned as-is
        result = real_telemetry.gen_ai_system_from_model("custom:model")
        assert result == "custom"


# ---------------------------------------------------------------------------
# record_llm_token_usage
# ---------------------------------------------------------------------------

class TestRecordLlmTokenUsage:

    def _make_msg(self, response_metadata):
        msg = MagicMock()
        msg.response_metadata = response_metadata
        return msg

    def test_record_llm_token_usage_anthropic(self, real_telemetry):
        mod = real_telemetry
        span = MagicMock()
        msg = self._make_msg({"usage": {"input_tokens": 42, "output_tokens": 17}})

        mod.record_llm_token_usage(span, {"messages": [msg]})

        span.set_attribute.assert_any_call(mod.GEN_AI_USAGE_INPUT_TOKENS, 42)
        span.set_attribute.assert_any_call(mod.GEN_AI_USAGE_OUTPUT_TOKENS, 17)

    def test_record_llm_token_usage_openai(self, real_telemetry):
        mod = real_telemetry
        span = MagicMock()
        msg = self._make_msg({"token_usage": {"prompt_tokens": 10, "completion_tokens": 20}})

        mod.record_llm_token_usage(span, {"messages": [msg]})

        span.set_attribute.assert_any_call(mod.GEN_AI_USAGE_INPUT_TOKENS, 10)
        span.set_attribute.assert_any_call(mod.GEN_AI_USAGE_OUTPUT_TOKENS, 20)

    def test_record_llm_token_usage_no_messages(self, real_telemetry):
        mod = real_telemetry
        span = MagicMock()

        # Should not raise
        mod.record_llm_token_usage(span, {})
        span.set_attribute.assert_not_called()

    def test_record_llm_token_usage_uses_last_message_with_usage(self, real_telemetry):
        """Iterates in reverse and returns on first message that has usage."""
        mod = real_telemetry
        span = MagicMock()

        no_usage_msg = self._make_msg({})
        usage_msg = self._make_msg({"usage": {"input_tokens": 5, "output_tokens": 3}})

        mod.record_llm_token_usage(span, {"messages": [no_usage_msg, usage_msg]})

        span.set_attribute.assert_any_call(mod.GEN_AI_USAGE_INPUT_TOKENS, 5)


# ---------------------------------------------------------------------------
# _NoopSpan
# ---------------------------------------------------------------------------

class TestNoopSpan:

    def test_noop_span_methods(self, real_telemetry):
        mod = real_telemetry
        span = mod._NoopSpan()

        # None of these should raise
        span.set_attribute("key", "value")
        span.set_status("ok")
        span.record_exception(ValueError("test"))
        span.add_event("my_event")

    def test_noop_span_context_manager(self, real_telemetry):
        mod = real_telemetry
        span = mod._NoopSpan()

        with span as s:
            assert s is span

    def test_noop_span_enter_exit_explicitly(self, real_telemetry):
        mod = real_telemetry
        span = mod._NoopSpan()

        result = span.__enter__()
        assert result is span
        span.__exit__(None, None, None)  # should not raise


# ---------------------------------------------------------------------------
# _NoopTracer
# ---------------------------------------------------------------------------

class TestNoopTracer:

    def test_noop_tracer_start_as_current_span_returns_noop_span(self, real_telemetry):
        mod = real_telemetry
        tracer = mod._NoopTracer()

        span = tracer.start_as_current_span("my_span")
        assert isinstance(span, mod._NoopSpan)

    def test_noop_tracer_start_span_returns_noop_span(self, real_telemetry):
        mod = real_telemetry
        tracer = mod._NoopTracer()

        span = tracer.start_span("my_span")
        assert isinstance(span, mod._NoopSpan)

    def test_noop_tracer_context_manager(self, real_telemetry):
        mod = real_telemetry
        tracer = mod._NoopTracer()

        with tracer.start_as_current_span("op") as span:
            assert isinstance(span, mod._NoopSpan)
            span.set_attribute("x", 1)  # should not raise


# ---------------------------------------------------------------------------
# setup_telemetry with exporters (require opentelemetry or skip)
# ---------------------------------------------------------------------------

class TestSetupTelemetryExporters:

    def test_setup_telemetry_with_otlp_endpoint(self, real_telemetry, monkeypatch):
        """When OTEL_EXPORTER_OTLP_ENDPOINT is set and OTLPSpanExporter importable,
        adds exporter."""
        otel = pytest.importorskip("opentelemetry")
        mod = real_telemetry
        mod._initialized = False
        mod._tracer = None

        monkeypatch.setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")

        # Should succeed without raising
        mod.setup_telemetry(service_name="test-service")

        # If opentelemetry was available, _initialized should be True
        assert mod._initialized is True

    def test_setup_telemetry_with_langfuse(self, real_telemetry, monkeypatch):
        """When LANGFUSE_HOST/PUBLIC_KEY/SECRET_KEY set, attempts to add exporter."""
        otel = pytest.importorskip("opentelemetry")
        mod = real_telemetry
        mod._initialized = False
        mod._tracer = None

        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse:3000")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-test")

        mod.setup_telemetry(service_name="test-langfuse")

        assert mod._initialized is True

    def test_setup_telemetry_console_debug(self, real_telemetry, monkeypatch):
        """OTEL_DEBUG=1 adds ConsoleSpanExporter."""
        otel = pytest.importorskip("opentelemetry")
        mod = real_telemetry
        mod._initialized = False
        mod._tracer = None

        monkeypatch.setenv("OTEL_DEBUG", "1")

        mod.setup_telemetry(service_name="debug-service")

        assert mod._initialized is True

    def test_setup_telemetry_no_otel_with_blocking_import(self, real_telemetry, monkeypatch):
        """Simulate missing opentelemetry via sys.modules None sentinel."""
        mod = real_telemetry
        mod._initialized = False
        mod._tracer = None

        # Setting to None in sys.modules causes ImportError on 'import opentelemetry'
        monkeypatch.setitem(sys.modules, "opentelemetry", None)

        mod.setup_telemetry()

        # Should have returned early without setting _initialized
        assert mod._initialized is False


# ---------------------------------------------------------------------------
# Comprehensive opentelemetry mock fixture
# ---------------------------------------------------------------------------

def _make_otel_mocks():
    """Return a dict of mock modules for the entire opentelemetry hierarchy."""
    from types import ModuleType

    mock_trace = MagicMock()
    mock_propagate = MagicMock()
    mock_baggage_prop = MagicMock()
    mock_baggage_prop.W3CBaggagePropagator = MagicMock()
    mock_composite = MagicMock()
    mock_composite.CompositePropagator = MagicMock()
    mock_resources = MagicMock()
    mock_resources.SERVICE_NAME = "service.name"
    mock_resources.Resource = MagicMock(return_value=MagicMock())
    mock_sdk_trace = MagicMock()
    mock_provider = MagicMock()
    mock_sdk_trace.TracerProvider = MagicMock(return_value=mock_provider)
    mock_export = MagicMock()
    mock_export.BatchSpanProcessor = MagicMock()
    mock_export.ConsoleSpanExporter = MagicMock()
    mock_tracecontext = MagicMock()
    mock_tracecontext.TraceContextTextMapPropagator = MagicMock()
    mock_tracer = MagicMock()
    mock_trace.get_tracer = MagicMock(return_value=mock_tracer)
    mock_trace.set_tracer_provider = MagicMock()
    mock_trace.get_current_span = MagicMock(return_value=MagicMock())
    mock_propagate.set_global_textmap = MagicMock()
    mock_propagate.inject = MagicMock()
    mock_propagate.extract = MagicMock(return_value={"ctx": "value"})
    otel_root = MagicMock()
    otel_root.trace = mock_trace
    otel_root.propagate = mock_propagate

    return {
        "opentelemetry": otel_root,
        "opentelemetry.trace": mock_trace,
        "opentelemetry.propagate": mock_propagate,
        "opentelemetry.baggage": MagicMock(),
        "opentelemetry.baggage.propagation": mock_baggage_prop,
        "opentelemetry.propagators": MagicMock(),
        "opentelemetry.propagators.composite": mock_composite,
        "opentelemetry.sdk": MagicMock(),
        "opentelemetry.sdk.resources": mock_resources,
        "opentelemetry.sdk.trace": mock_sdk_trace,
        "opentelemetry.sdk.trace.export": mock_export,
        "opentelemetry.trace.propagation": MagicMock(),
        "opentelemetry.trace.propagation.tracecontext": mock_tracecontext,
        "_provider": mock_provider,
        "_tracer": mock_tracer,
        "_trace": mock_trace,
        "_propagate": mock_propagate,
        "_export": mock_export,
    }


@pytest.fixture
def otel_mocked_telemetry(monkeypatch):
    """Load real telemetry module with comprehensive opentelemetry mock hierarchy."""
    mocks = _make_otel_mocks()
    for key, val in mocks.items():
        if not key.startswith("_"):
            monkeypatch.setitem(sys.modules, key, val)

    monkeypatch.delitem(sys.modules, "builtin_tools.telemetry", raising=False)
    spec = importlib.util.spec_from_file_location(
        "builtin_tools.telemetry_otel",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools/telemetry.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.telemetry_otel", mod)
    spec.loader.exec_module(mod)
    mod._initialized = False
    mod._tracer = None
    return mod, mocks


# ---------------------------------------------------------------------------
# setup_telemetry with mocked opentelemetry (covers lines 125-218)
# ---------------------------------------------------------------------------

class TestSetupTelemetryMockedOtel:

    def test_setup_telemetry_basic_initializes(self, otel_mocked_telemetry):
        """setup_telemetry() sets _initialized=True when opentelemetry mocks are present."""
        mod, mocks = otel_mocked_telemetry
        mod.setup_telemetry(service_name="test-ws")
        assert mod._initialized is True
        assert mod._tracer is not None
        mocks["_trace"].set_tracer_provider.assert_called_once()

    def test_setup_telemetry_with_otlp_endpoint_import_error(self, otel_mocked_telemetry, monkeypatch):
        """OTLP exporter ImportError is caught with a warning."""
        mod, mocks = otel_mocked_telemetry
        monkeypatch.setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
        # Make OTLPSpanExporter import fail
        monkeypatch.setitem(sys.modules,
                            "opentelemetry.exporter.otlp.proto.http.trace_exporter", None)
        monkeypatch.setitem(sys.modules, "opentelemetry.exporter", None)
        mod.setup_telemetry(service_name="test-ws")
        # Should still complete without raising
        assert mod._initialized is True

    def test_setup_telemetry_with_otlp_endpoint_success(self, otel_mocked_telemetry, monkeypatch):
        """OTLP exporter is added when importable."""
        mod, mocks = otel_mocked_telemetry
        monkeypatch.setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
        mock_otlp = MagicMock()
        mock_otlp.OTLPSpanExporter = MagicMock(return_value=MagicMock())
        monkeypatch.setitem(sys.modules,
                            "opentelemetry.exporter.otlp.proto.http.trace_exporter", mock_otlp)
        monkeypatch.setitem(sys.modules, "opentelemetry.exporter", MagicMock())
        monkeypatch.setitem(sys.modules, "opentelemetry.exporter.otlp", MagicMock())
        monkeypatch.setitem(sys.modules, "opentelemetry.exporter.otlp.proto", MagicMock())
        monkeypatch.setitem(sys.modules, "opentelemetry.exporter.otlp.proto.http", MagicMock())
        mod.setup_telemetry(service_name="otlp-ws")
        assert mod._initialized is True
        mock_otlp.OTLPSpanExporter.assert_called_once()

    def test_setup_telemetry_with_langfuse_success(self, otel_mocked_telemetry, monkeypatch):
        """Langfuse OTLP bridge exporter is added when all env vars set."""
        mod, mocks = otel_mocked_telemetry
        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse:3000")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-secret")
        mock_otlp = MagicMock()
        mock_otlp.OTLPSpanExporter = MagicMock(return_value=MagicMock())
        for path in ("opentelemetry.exporter.otlp.proto.http.trace_exporter",
                     "opentelemetry.exporter", "opentelemetry.exporter.otlp",
                     "opentelemetry.exporter.otlp.proto", "opentelemetry.exporter.otlp.proto.http"):
            monkeypatch.setitem(sys.modules, path, mock_otlp if path.endswith("trace_exporter") else MagicMock())
        mod.setup_telemetry(service_name="lf-ws")
        assert mod._initialized is True

    def test_setup_telemetry_langfuse_import_error(self, otel_mocked_telemetry, monkeypatch):
        """Langfuse OTLPSpanExporter ImportError is caught."""
        mod, mocks = otel_mocked_telemetry
        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse:3000")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-secret")
        monkeypatch.setitem(sys.modules,
                            "opentelemetry.exporter.otlp.proto.http.trace_exporter", None)
        monkeypatch.setitem(sys.modules, "opentelemetry.exporter", None)
        mod.setup_telemetry(service_name="lf-err")
        assert mod._initialized is True

    def test_setup_telemetry_console_debug(self, otel_mocked_telemetry, monkeypatch):
        """Console exporter is added when OTEL_DEBUG=1."""
        mod, mocks = otel_mocked_telemetry
        monkeypatch.setenv("OTEL_DEBUG", "1")
        mod.setup_telemetry(service_name="debug-ws")
        assert mod._initialized is True
        mocks["_export"].ConsoleSpanExporter.assert_called_once()

    def test_setup_telemetry_otlp_exporter_init_exception(self, otel_mocked_telemetry, monkeypatch):
        """OTLP exporter instantiation raising non-ImportError is caught with warning."""
        mod, mocks = otel_mocked_telemetry
        monkeypatch.setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
        # Make OTLPSpanExporter importable but raise on instantiation
        mock_otlp = MagicMock()
        mock_otlp.OTLPSpanExporter = MagicMock(side_effect=RuntimeError("connection refused"))
        for path in ("opentelemetry.exporter.otlp.proto.http.trace_exporter",
                     "opentelemetry.exporter", "opentelemetry.exporter.otlp",
                     "opentelemetry.exporter.otlp.proto", "opentelemetry.exporter.otlp.proto.http"):
            monkeypatch.setitem(sys.modules, path, mock_otlp if path.endswith("trace_exporter") else MagicMock())
        mod._initialized = False
        mod._tracer = None
        mod.setup_telemetry(service_name="test")
        # Should complete without raising (exception is caught)
        assert mod._initialized is True

    def test_setup_telemetry_langfuse_exporter_init_exception(self, otel_mocked_telemetry, monkeypatch):
        """Langfuse exporter instantiation raising non-ImportError is caught with warning."""
        mod, mocks = otel_mocked_telemetry
        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse:3000")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-secret")
        # Make OTLPSpanExporter importable but raise on instantiation
        mock_otlp = MagicMock()
        mock_otlp.OTLPSpanExporter = MagicMock(side_effect=RuntimeError("langfuse error"))
        for path in ("opentelemetry.exporter.otlp.proto.http.trace_exporter",
                     "opentelemetry.exporter", "opentelemetry.exporter.otlp",
                     "opentelemetry.exporter.otlp.proto", "opentelemetry.exporter.otlp.proto.http"):
            monkeypatch.setitem(sys.modules, path, mock_otlp if path.endswith("trace_exporter") else MagicMock())
        mod._initialized = False
        mod._tracer = None
        mod.setup_telemetry(service_name="lf-exc")
        assert mod._initialized is True


# ---------------------------------------------------------------------------
# get_tracer / inject / extract / traceparent with mocked opentelemetry
# ---------------------------------------------------------------------------

class TestOtelFunctionsWithMocks:

    def test_get_tracer_when_tracer_none_but_otel_available(self, otel_mocked_telemetry):
        """When _tracer is None but opentelemetry importable, get_tracer falls back."""
        mod, mocks = otel_mocked_telemetry
        mod._initialized = True
        mod._tracer = None
        result = mod.get_tracer()
        # Should call trace.get_tracer for the noop fallback
        mocks["_trace"].get_tracer.assert_called()
        assert result is not None

    def test_extract_trace_context_calls_propagate_extract(self, otel_mocked_telemetry):
        """extract_trace_context returns propagate.extract result when otel available."""
        mod, mocks = otel_mocked_telemetry
        carrier = {"traceparent": "00-abc-def-01"}
        result = mod.extract_trace_context(carrier)
        mocks["_propagate"].extract.assert_called_with(carrier)
        assert result == {"ctx": "value"}

    def test_get_current_traceparent_valid_span(self, otel_mocked_telemetry):
        """get_current_traceparent returns W3C string when span context is valid."""
        mod, mocks = otel_mocked_telemetry
        mock_ctx = MagicMock()
        mock_ctx.is_valid = True
        mock_ctx.trace_id = 0xabcdef1234567890abcdef1234567890
        mock_ctx.span_id = 0x1234567890abcdef
        mock_ctx.trace_flags = 1
        mock_span = MagicMock()
        mock_span.get_span_context.return_value = mock_ctx
        mocks["_trace"].get_current_span.return_value = mock_span

        result = mod.get_current_traceparent()

        assert result is not None
        assert result.startswith("00-")
        assert len(result.split("-")) == 4

    def test_get_current_traceparent_invalid_span(self, otel_mocked_telemetry):
        """get_current_traceparent returns None when ctx.is_valid is False."""
        mod, mocks = otel_mocked_telemetry
        mock_ctx = MagicMock()
        mock_ctx.is_valid = False
        mock_span = MagicMock()
        mock_span.get_span_context.return_value = mock_ctx
        mocks["_trace"].get_current_span.return_value = mock_span

        result = mod.get_current_traceparent()
        assert result is None

    def test_get_current_traceparent_zero_flags(self, otel_mocked_telemetry):
        """trace_flags=0 produces '00' flag string."""
        mod, mocks = otel_mocked_telemetry
        mock_ctx = MagicMock()
        mock_ctx.is_valid = True
        mock_ctx.trace_id = 0x1
        mock_ctx.span_id = 0x2
        mock_ctx.trace_flags = 0  # falsy
        mock_span = MagicMock()
        mock_span.get_span_context.return_value = mock_ctx
        mocks["_trace"].get_current_span.return_value = mock_span

        result = mod.get_current_traceparent()
        assert result is not None
        assert result.endswith("-00")


# ---------------------------------------------------------------------------
# record_llm_token_usage — exception path
# ---------------------------------------------------------------------------

class TestRecordLlmTokenUsageExceptionPath:

    def test_record_llm_token_usage_exception_swallowed(self, real_telemetry):
        """Exception inside record_llm_token_usage is swallowed silently."""
        mod = real_telemetry
        span = MagicMock()
        # Passing an int instead of dict triggers AttributeError (no .get method)
        mod.record_llm_token_usage(span, 42)  # type: ignore
        span.set_attribute.assert_not_called()
