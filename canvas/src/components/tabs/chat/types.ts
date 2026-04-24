export interface ChatMessage {
  id: string;
  role: "user" | "agent" | "system";
  content: string;
  timestamp: string; // ISO string for serialization
}

export function createMessage(role: ChatMessage["role"], content: string): ChatMessage {
  return { id: crypto.randomUUID(), role, content, timestamp: new Date().toISOString() };
}

// appendMessageDeduped adds a ChatMessage to `prev` unless the tail
// already contains the same (role, content) from within
// dedupeWindowMs. Collapses the case where two delivery paths race to
// render the same agent reply — e.g. the HTTP .then() handler for
// POST /a2a AND a `send_message_to_user` WebSocket push from the
// runtime, both carrying the same text. Without this guard the user
// sees two or three identical bubbles with identical timestamps.
//
// Why a time-windowed check instead of dedupe-by-id: the three delivery
// paths (HTTP response, WS A2A_RESPONSE, WS send_message_to_user) each
// mint a fresh `createMessage` with a random UUID client-side — there's
// no stable end-to-end message id yet. Content+role+time is the
// pragmatic identity. The window is short (3s) so genuine repeat
// messages ("hi", "hi") from a real user/agent still render.
export function appendMessageDeduped(prev: ChatMessage[], msg: ChatMessage, dedupeWindowMs = 3000): ChatMessage[] {
  const cutoff = Date.now() - dedupeWindowMs;
  const alreadyThere = prev.some((m) => {
    if (m.role !== msg.role || m.content !== msg.content) return false;
    const t = Date.parse(m.timestamp);
    return !Number.isNaN(t) && t >= cutoff;
  });
  if (alreadyThere) return prev;
  return [...prev, msg];
}
