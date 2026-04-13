'use client';

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";

interface ChannelAdapter {
  type: string;
  display_name: string;
}

interface Channel {
  id: string;
  workspace_id: string;
  channel_type: string;
  config: Record<string, string>;
  enabled: boolean;
  allowed_users: string[];
  message_count: number;
  last_message_at?: string;
  created_at: string;
}

interface Props {
  workspaceId: string;
}

function relativeTime(iso: string | null | undefined): string {
  if (!iso) return "never";
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60000) return `${Math.round(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.round(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.round(diff / 3600000)}h ago`;
  return `${Math.round(diff / 86400000)}d ago`;
}

export function ChannelsTab({ workspaceId }: Props) {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [adapters, setAdapters] = useState<ChannelAdapter[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [testing, setTesting] = useState<string | null>(null);

  // Form state
  const [formType, setFormType] = useState("telegram");
  const [formBotToken, setFormBotToken] = useState("");
  const [formChatId, setFormChatId] = useState("");
  const [formAllowedUsers, setFormAllowedUsers] = useState("");
  const [formError, setFormError] = useState("");
  const [discovering, setDiscovering] = useState(false);
  const [discoveredChats, setDiscoveredChats] = useState<{ chat_id: string; name: string; type: string }[]>([]);
  const [selectedChats, setSelectedChats] = useState<Set<string>>(new Set());
  const [showManualInput, setShowManualInput] = useState(false);

  const load = useCallback(async () => {
    try {
      const [chRes, adRes] = await Promise.all([
        api.get<Channel[]>(`/workspaces/${workspaceId}/channels`),
        api.get<ChannelAdapter[]>(`/channels/adapters`),
      ]);
      setChannels(Array.isArray(chRes) ? chRes : []);
      setAdapters(Array.isArray(adRes) ? adRes : []);
    } catch {
      /* ignore */
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => { load(); }, [load]);

  // Auto-refresh every 15s
  useEffect(() => {
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, [load]);

  const handleDiscover = async () => {
    if (!formBotToken) {
      setFormError("Enter a bot token first");
      return;
    }
    setDiscovering(true);
    setFormError("");
    setDiscoveredChats([]);
    try {
      const res = await api.post<{ chats: { chat_id: string; name: string; type: string }[]; hint: string }>(
        `/channels/discover`,
        { channel_type: formType, bot_token: formBotToken }
      );
      const chats = res.chats || [];
      setDiscoveredChats(chats);
      if (chats.length === 0) {
        setFormError("No chats found. For groups: add the bot and send a message. For DMs: send /start to the bot first. Then retry.");
      } else {
        // Auto-select all discovered chats
        setSelectedChats(new Set(chats.map((c) => c.chat_id)));
        setFormChatId(chats.map((c) => c.chat_id).join(", "));
      }
    } catch (e) {
      setFormError(String(e));
    } finally {
      setDiscovering(false);
    }
  };

  const toggleChat = (chatId: string) => {
    setSelectedChats((prev) => {
      const next = new Set(prev);
      if (next.has(chatId)) next.delete(chatId);
      else next.add(chatId);
      setFormChatId(Array.from(next).join(", "));
      return next;
    });
  };

  const handleCreate = async () => {
    setFormError("");
    if (!formBotToken || !formChatId) {
      setFormError("Bot token and chat ID are required");
      return;
    }
    try {
      const allowed = formAllowedUsers
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      await api.post(`/workspaces/${workspaceId}/channels`, {
        channel_type: formType,
        config: { bot_token: formBotToken, chat_id: formChatId },
        allowed_users: allowed,
      });
      setShowForm(false);
      setFormBotToken("");
      setFormChatId("");
      setFormAllowedUsers("");
      load();
    } catch (e) {
      setFormError(String(e));
    }
  };

  const handleToggle = async (ch: Channel) => {
    await api.patch(`/workspaces/${workspaceId}/channels/${ch.id}`, {
      enabled: !ch.enabled,
    });
    load();
  };

  const handleDelete = async (ch: Channel) => {
    if (!confirm(`Delete ${ch.channel_type} channel?`)) return;
    await api.del(`/workspaces/${workspaceId}/channels/${ch.id}`);
    load();
  };

  const handleTest = async (ch: Channel) => {
    setTesting(ch.id);
    try {
      await api.post(`/workspaces/${workspaceId}/channels/${ch.id}/test`, {});
    } catch {
      /* ignore — error shown on platform side */
    } finally {
      setTimeout(() => setTesting(null), 2000);
    }
  };

  if (loading) {
    return (
      <div className="p-4 text-zinc-500 text-xs">Loading channels...</div>
    );
  }

  return (
    <div className="p-4 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold text-zinc-300 tracking-wide uppercase">
          Channels
        </h3>
        <button
          onClick={() => setShowForm(!showForm)}
          className="text-[10px] px-2.5 py-1 rounded bg-blue-600/20 text-blue-400 hover:bg-blue-600/30 transition"
        >
          {showForm ? "Cancel" : "+ Connect"}
        </button>
      </div>

      {/* Create form */}
      {showForm && (
        <div className="space-y-2 p-3 bg-zinc-800/40 rounded border border-zinc-700/50">
          <div>
            <label className="text-[10px] text-zinc-500 block mb-1">Platform</label>
            <select
              value={formType}
              onChange={(e) => setFormType(e.target.value)}
              className="w-full text-xs bg-zinc-900 border border-zinc-700 rounded px-2 py-1.5 text-zinc-300"
            >
              {adapters.map((a) => (
                <option key={a.type} value={a.type}>{a.display_name}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-[10px] text-zinc-500 block mb-1">Bot Token</label>
            <input
              type="password"
              value={formBotToken}
              onChange={(e) => setFormBotToken(e.target.value)}
              placeholder="123456:ABC-DEF..."
              className="w-full text-xs bg-zinc-900 border border-zinc-700 rounded px-2 py-1.5 text-zinc-300 placeholder-zinc-600"
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-[10px] text-zinc-500">Chat IDs</label>
              <button
                onClick={handleDiscover}
                disabled={discovering || !formBotToken}
                className="text-[10px] px-2 py-0.5 rounded bg-blue-600/20 text-blue-400 hover:bg-blue-600/30 transition disabled:opacity-40"
              >
                {discovering ? "Detecting..." : "Detect Chats"}
              </button>
            </div>
            {discoveredChats.length > 0 && (
              <div className="space-y-1 mb-2">
                {discoveredChats.map((chat) => (
                  <label
                    key={chat.chat_id}
                    className="flex items-center gap-2 px-2 py-1.5 bg-zinc-900/50 rounded border border-zinc-700/50 cursor-pointer hover:bg-zinc-800/50"
                  >
                    <input
                      type="checkbox"
                      checked={selectedChats.has(chat.chat_id)}
                      onChange={() => toggleChat(chat.chat_id)}
                      className="rounded border-zinc-600"
                    />
                    <span className="text-xs text-zinc-300">{chat.name || "Unknown"}</span>
                    <span className="text-[10px] text-zinc-500 ml-auto">{chat.type} {chat.chat_id}</span>
                  </label>
                ))}
              </div>
            )}
            {(discoveredChats.length === 0 || showManualInput) && (
              <input
                value={formChatId}
                onChange={(e) => setFormChatId(e.target.value)}
                placeholder="-100123456789, -100987654321"
                className="w-full text-xs bg-zinc-900 border border-zinc-700 rounded px-2 py-1.5 text-zinc-300 placeholder-zinc-600"
              />
            )}
            <p className="text-[9px] text-zinc-600 mt-0.5">
              {discoveredChats.length > 0 ? (
                <>
                  Chats: <span className="text-zinc-400">{formChatId || "(none selected)"}</span>
                  {" · "}
                  <button
                    onClick={() => setShowManualInput(!showManualInput)}
                    className="text-blue-400 hover:underline"
                  >
                    {showManualInput ? "hide manual input" : "edit manually"}
                  </button>
                </>
              ) : (
                "Click Detect Chats after adding the bot to groups or sending /start in DMs."
              )}
            </p>
          </div>
          <div>
            <label className="text-[10px] text-zinc-500 block mb-1">
              Allowed Users <span className="text-zinc-600">(optional, comma-separated)</span>
            </label>
            <input
              value={formAllowedUsers}
              onChange={(e) => setFormAllowedUsers(e.target.value)}
              placeholder="123456789, 987654321"
              className="w-full text-xs bg-zinc-900 border border-zinc-700 rounded px-2 py-1.5 text-zinc-300 placeholder-zinc-600"
            />
            <p className="text-[9px] text-zinc-600 mt-0.5">
              Telegram user IDs. Leave empty to allow everyone.
            </p>
          </div>
          {formError && (
            <p className="text-[10px] text-red-400">{formError}</p>
          )}
          <button
            onClick={handleCreate}
            className="w-full text-xs py-1.5 rounded bg-blue-600 hover:bg-blue-500 text-white transition"
          >
            Connect Channel
          </button>
        </div>
      )}

      {/* Channel list */}
      {channels.length === 0 && !showForm && (
        <div className="text-center py-8">
          <p className="text-zinc-500 text-xs">No channels connected</p>
          <p className="text-zinc-600 text-[10px] mt-1">
            Connect Telegram, Slack, or Discord to chat with this agent from social platforms.
          </p>
        </div>
      )}

      {channels.map((ch) => (
        <div
          key={ch.id}
          className="p-3 bg-zinc-800/30 rounded border border-zinc-700/40 space-y-2"
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span
                className={`w-2 h-2 rounded-full ${
                  ch.enabled ? "bg-emerald-500" : "bg-zinc-600"
                }`}
              />
              <span className="text-xs font-medium text-zinc-200">
                {ch.channel_type.charAt(0).toUpperCase() + ch.channel_type.slice(1)}
              </span>
              <span className="text-[10px] text-zinc-500">
                {ch.config.chat_id}
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <button
                onClick={() => handleTest(ch)}
                disabled={testing === ch.id}
                className="text-[10px] px-2 py-0.5 rounded bg-zinc-700/50 text-zinc-400 hover:text-zinc-200 transition disabled:opacity-50"
              >
                {testing === ch.id ? "Sent!" : "Test"}
              </button>
              <button
                onClick={() => handleToggle(ch)}
                className={`text-[10px] px-2 py-0.5 rounded transition ${
                  ch.enabled
                    ? "bg-emerald-900/30 text-emerald-400 hover:bg-emerald-900/50"
                    : "bg-zinc-700/50 text-zinc-500 hover:text-zinc-300"
                }`}
              >
                {ch.enabled ? "On" : "Off"}
              </button>
              <button
                onClick={() => handleDelete(ch)}
                className="text-[10px] px-2 py-0.5 rounded bg-red-900/20 text-red-400 hover:bg-red-900/40 transition"
              >
                Remove
              </button>
            </div>
          </div>
          <div className="flex items-center gap-4 text-[10px] text-zinc-500">
            <span>{ch.message_count} messages</span>
            <span>Last: {relativeTime(ch.last_message_at)}</span>
            {ch.allowed_users.length > 0 && (
              <span>{ch.allowed_users.length} allowed user(s)</span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
