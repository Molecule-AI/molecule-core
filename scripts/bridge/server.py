"""A2A HTTP server for the external workspace bridge."""

import json
import logging
import uuid
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path
from typing import Callable

from .processor import MessageProcessor

logger = logging.getLogger("bridge.server")


class A2AHandler(BaseHTTPRequestHandler):
    """Handles incoming A2A JSON-RPC requests, delegates to a MessageProcessor."""

    processor: MessageProcessor
    inbox_path: Path
    resolve_name: Callable[[str], str]

    def log_message(self, format, *args):
        pass

    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
        try:
            request = json.loads(body)
        except json.JSONDecodeError:
            self.send_error(400)
            return

        method = request.get("method", "")
        req_id = request.get("id", str(uuid.uuid4()))

        if method == "message/send":
            params = request.get("params", {})
            message = params.get("message", {})
            parts = message.get("parts", [])
            text = parts[0].get("text", "") if parts else ""
            sender_id = self.headers.get("X-Workspace-ID", "")
            sender_name = A2AHandler.resolve_name(sender_id) if sender_id else "canvas"

            logger.info(f"📨 {sender_name}: {text[:80]}")

            # Log to inbox
            entry = {
                "id": req_id, "sender_id": sender_id,
                "sender_name": sender_name, "text": text,
            }
            with open(A2AHandler.inbox_path, "a") as f:
                f.write(json.dumps(entry) + "\n")

            # Process with the configured backend
            context = {"sender_id": sender_id, "sender_name": sender_name}
            response_text = A2AHandler.processor.process(text, sender_name, context)

            self._send_a2a_response(req_id, response_text)

        elif method == "agent/card":
            self._send_json(200, {
                "jsonrpc": "2.0", "id": req_id,
                "result": {
                    "name": A2AHandler.processor.name,
                    "description": f"External agent powered by {A2AHandler.processor.name}",
                    "version": "1.0.0",
                },
            })
        else:
            self._send_json(200, {
                "jsonrpc": "2.0", "id": req_id,
                "error": {"code": -32601, "message": f"Unsupported: {method}"},
            })

    def _send_a2a_response(self, req_id, text):
        self._send_json(200, {
            "jsonrpc": "2.0", "id": req_id,
            "result": {
                "kind": "message", "messageId": str(uuid.uuid4()),
                "role": "agent",
                "parts": [{"kind": "text", "text": text}],
            },
        })

    def _send_json(self, status, data):
        body = json.dumps(data).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


def create_server(port: int, processor: MessageProcessor, inbox_path: Path, resolve_name) -> HTTPServer:
    """Create an A2A HTTP server with the given processor."""
    A2AHandler.processor = processor
    A2AHandler.inbox_path = inbox_path
    A2AHandler.resolve_name = resolve_name
    return HTTPServer(("0.0.0.0", port), A2AHandler)
