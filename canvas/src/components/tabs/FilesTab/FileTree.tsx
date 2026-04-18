"use client";

import { type TreeNode, getIcon } from "./tree";

interface TreeCallbacks {
  selectedPath: string | null;
  onSelect: (path: string) => void;
  onDelete: (path: string) => void;
  expandedDirs: Set<string>;
  onToggleDir: (path: string) => void;
  loadingDir: string | null;
}

export function FileTree({
  nodes,
  selectedPath,
  onSelect,
  onDelete,
  expandedDirs,
  onToggleDir,
  loadingDir,
  depth = 0,
}: TreeCallbacks & { nodes: TreeNode[]; depth?: number }) {
  return (
    <div>
      {nodes.map((node) => (
        <TreeItem
          key={`${node.path}:${node.isDir ? "dir" : "file"}`}
          node={node}
          selectedPath={selectedPath}
          onSelect={onSelect}
          onDelete={onDelete}
          expandedDirs={expandedDirs}
          onToggleDir={onToggleDir}
          loadingDir={loadingDir}
          depth={depth}
        />
      ))}
    </div>
  );
}

function TreeItem({
  node,
  selectedPath,
  onSelect,
  onDelete,
  expandedDirs,
  onToggleDir,
  loadingDir,
  depth,
}: TreeCallbacks & { node: TreeNode; depth: number }) {
  const isSelected = selectedPath === node.path;
  const expanded = expandedDirs.has(node.path);
  const isLoading = loadingDir === node.path;

  if (node.isDir) {
    return (
      <div>
        <div
          className="group w-full flex items-center gap-1 px-2 py-0.5 text-left hover:bg-zinc-800/40 transition-colors cursor-pointer"
          style={{ paddingLeft: `${depth * 12 + 8}px` }}
          onClick={() => onToggleDir(node.path)}
        >
          <span className="text-[10px] text-zinc-400 w-3" aria-hidden="true">{isLoading ? "…" : expanded ? "▼" : "▶"}</span>
          <span className="text-[10px]">📁</span>
          <span className="text-[10px] text-zinc-400 flex-1">{node.name}</span>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onDelete(node.path);
            }}
            className="text-[10px] text-red-400/0 group-hover:text-red-400/60 hover:!text-red-400 transition-colors focus-visible:ring-2 focus-visible:ring-blue-500/70 focus-visible:outline-none rounded"
          >
            ✕
          </button>
        </div>
        {expanded && (
          <FileTree
            nodes={node.children}
            selectedPath={selectedPath}
            onSelect={onSelect}
            onDelete={onDelete}
            expandedDirs={expandedDirs}
            onToggleDir={onToggleDir}
            loadingDir={loadingDir}
            depth={depth + 1}
          />
        )}
      </div>
    );
  }

  return (
    <div
      className={`group flex items-center gap-1 px-2 py-0.5 cursor-pointer transition-colors ${
        isSelected ? "bg-blue-900/30 text-zinc-100" : "hover:bg-zinc-800/40 text-zinc-400"
      }`}
      style={{ paddingLeft: `${depth * 12 + 20}px` }}
      onClick={() => onSelect(node.path)}
    >
      <span className="text-[10px]" aria-hidden="true">{getIcon(node.name, false)}</span>
      <span className="text-[10px] flex-1 truncate font-mono">{node.name}</span>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onDelete(node.path);
        }}
        className="text-[9px] text-red-400/0 group-hover:text-red-400/60 hover:!text-red-400 transition-colors"
      >
        ✕
      </button>
    </div>
  );
}
