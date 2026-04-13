"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import { useCanvasStore, type PanelTab } from "@/store/canvas";
import { showToast } from "@/components/Toaster";
import { StatusDot } from "./StatusDot";
import { Tooltip } from "./Tooltip";
import { DetailsTab } from "./tabs/DetailsTab";
import { SkillsTab } from "./tabs/SkillsTab";
import { ChatTab } from "./tabs/ChatTab";
import { ConfigTab } from "./tabs/ConfigTab";
import { TerminalTab } from "./tabs/TerminalTab";
import { FilesTab } from "./tabs/FilesTab";
import { MemoryTab } from "./tabs/MemoryTab";
import { TracesTab } from "./tabs/TracesTab";
import { EventsTab } from "./tabs/EventsTab";
import { ActivityTab } from "./tabs/ActivityTab";
import { ScheduleTab } from "./tabs/ScheduleTab";
import { ChannelsTab } from "./tabs/ChannelsTab";
import { summarizeWorkspaceCapabilities } from "@/store/canvas";

const TABS: { id: PanelTab; label: string; icon: string }[] = [
  { id: "chat", label: "Chat", icon: "◈" },
  { id: "activity", label: "Activity", icon: "⊙" },
  { id: "details", label: "Details", icon: "◉" },
  { id: "skills", label: "Skills", icon: "✦" },
  { id: "terminal", label: "Terminal", icon: "▸" },
  { id: "config", label: "Config", icon: "⚙" },
  { id: "schedule", label: "Schedule", icon: "⏲" },
  { id: "channels", label: "Channels", icon: "⇌" },
  { id: "files", label: "Files", icon: "⊞" },
  { id: "memory", label: "Memory", icon: "◇" },
  { id: "traces", label: "Traces", icon: "◎" },
  { id: "events", label: "Events", icon: "◊" },
];

export function SidePanel() {
  const selectedNodeId = useCanvasStore((s) => s.selectedNodeId);
  const panelTab = useCanvasStore((s) => s.panelTab);
  const setPanelTab = useCanvasStore((s) => s.setPanelTab);
  const selectNode = useCanvasStore((s) => s.selectNode);
  const node = useCanvasStore((s) =>
    s.nodes.find((n) => n.id === s.selectedNodeId)
  );

  // Resizable panel width
  const [width, setWidth] = useState(480);
  const dragging = useRef(false);
  const startX = useRef(0);
  const startWidth = useRef(480);

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    dragging.current = true;
    startX.current = e.clientX;
    startWidth.current = width;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  }, [width]);

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!dragging.current) return;
      const delta = startX.current - e.clientX;
      const newWidth = Math.min(Math.max(startWidth.current + delta, 320), window.innerWidth * 0.8);
      setWidth(newWidth);
    };
    const onMouseUp = () => {
      dragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
    return () => {
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);
    };
  }, []);

  if (!selectedNodeId || !node) return null;

  const isOnline = node.data.status === "online";
  const capability = summarizeWorkspaceCapabilities(node.data);

  return (
    <div
      className="fixed top-0 right-0 h-full bg-zinc-950/95 backdrop-blur-xl border-l border-zinc-800/50 flex flex-col z-50 shadow-2xl shadow-black/50 animate-in slide-in-from-right duration-200"
      style={{ width }}
    >
      {/* Resize handle */}
      <div
        onMouseDown={onMouseDown}
        className="absolute left-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-blue-500/30 active:bg-blue-500/50 transition-colors z-10"
      />
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 border-b border-zinc-800/40 bg-zinc-900/30">
        <div className="flex items-center gap-3 min-w-0">
          <div className="relative">
            <StatusDot status={node.data.status} size="md" />
          </div>
          <div className="min-w-0">
            <h2 className="text-[14px] font-semibold text-zinc-100 truncate leading-tight">
              {node.data.name}
            </h2>
            <div className="flex items-center gap-2 mt-0.5">
              {node.data.role && (
                <span className="text-[10px] text-zinc-500 truncate">
                  {node.data.role}
                </span>
              )}
              <span className={`text-[9px] px-1.5 py-0.5 rounded-md font-mono ${
                isOnline ? "text-emerald-400 bg-emerald-950/30" : "text-zinc-500 bg-zinc-800/50"
              }`}>
                T{node.data.tier}
              </span>
            </div>
          </div>
        </div>
        <button
          onClick={() => selectNode(null)}
          className="w-7 h-7 flex items-center justify-center rounded-lg text-zinc-500 hover:text-zinc-200 hover:bg-zinc-800/60 transition-colors"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path d="M1 1l10 10M11 1L1 11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
        </button>
      </div>

      {/* Capability summary */}
      <div className="px-5 py-3 border-b border-zinc-800/40 bg-zinc-900/20">
        <div className="flex flex-wrap gap-2">
          <MetaPill label="Tier" value={`T${node.data.tier}`} />
          <MetaPill label="Runtime" value={capability.runtime || "unknown"} />
          <MetaPill label="Skills" value={capability.skillCount > 0 ? `${capability.skillCount}` : "none"} />
          <MetaPill label="Status" value={node.data.status} tone={isOnline ? "emerald" : "zinc"} />
        </div>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-zinc-800/40 overflow-x-auto bg-zinc-900/20 px-1">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setPanelTab(tab.id)}
            className={`shrink-0 px-3 py-2.5 text-[10px] font-medium tracking-wide transition-all rounded-t-lg mx-0.5 ${
              panelTab === tab.id
                ? "text-zinc-100 bg-zinc-800/40 border-b-2 border-blue-500"
                : "text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800/20"
            }`}
          >
            <span className="mr-1 opacity-50">{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Needs Restart Banner */}
      {node.data.needsRestart && !node.data.currentTask && selectedNodeId && (
        <div className="px-4 py-2 bg-sky-950/20 border-b border-sky-800/20 flex items-center justify-between">
          <span className="text-[10px] text-sky-300/90">Config changed — restart to apply</span>
          <button
            onClick={() => {
              useCanvasStore.getState().restartWorkspace(selectedNodeId).catch(() => showToast("Restart failed", "error"));
            }}
            className="text-[9px] px-2 py-1 bg-sky-800/40 hover:bg-sky-700/50 text-sky-200 rounded transition-colors"
          >
            Restart Now
          </button>
        </div>
      )}

      {/* Current Task Banner */}
      {node.data.currentTask && (
        <Tooltip text={node.data.currentTask as string}>
          <div className="px-4 py-2 bg-amber-950/20 border-b border-amber-800/20 flex items-center gap-2 cursor-default">
            <div className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse shrink-0" />
            <span className="text-[10px] text-amber-300/90 truncate">
              {node.data.currentTask}
            </span>
          </div>
        </Tooltip>
      )}

      {/* Tab Content */}
      <div className="flex-1 overflow-y-auto">
        {panelTab === "details" && <DetailsTab key={selectedNodeId} workspaceId={selectedNodeId} data={node.data} />}
        {panelTab === "skills" && <SkillsTab key={selectedNodeId} data={node.data} />}
        {panelTab === "activity" && <ActivityTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "chat" && <ChatTab key={selectedNodeId} workspaceId={selectedNodeId} data={node.data} />}
        {panelTab === "terminal" && <TerminalTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "config" && <ConfigTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "schedule" && <ScheduleTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "channels" && <ChannelsTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "files" && <FilesTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "memory" && <MemoryTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "traces" && <TracesTab key={selectedNodeId} workspaceId={selectedNodeId} />}
        {panelTab === "events" && <EventsTab key={selectedNodeId} workspaceId={selectedNodeId} />}
      </div>

      {/* Footer — workspace ID */}
      <div className="px-5 py-2 border-t border-zinc-800/40 bg-zinc-900/20">
        <span className="text-[9px] font-mono text-zinc-600 select-all">
          {selectedNodeId}
        </span>
      </div>
    </div>
  );
}

function MetaPill({ label, value, tone = "zinc" }: { label: string; value: string; tone?: "zinc" | "emerald" | "amber" }) {
  const toneClasses = {
    zinc: "border-zinc-700/50 bg-zinc-900/70 text-zinc-400",
    emerald: "border-emerald-500/20 bg-emerald-950/20 text-emerald-300",
    amber: "border-amber-500/20 bg-amber-950/20 text-amber-300",
  }[tone];

  return (
    <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-[9px] ${toneClasses}`}>
      <span className="uppercase tracking-[0.18em] text-[8px] opacity-70">{label}</span>
      <span className="font-medium">{value}</span>
    </span>
  );
}
