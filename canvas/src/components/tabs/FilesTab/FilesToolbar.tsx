"use client";

import { useRef } from "react";

interface Props {
  root: string;
  setRoot: (r: string) => void;
  fileCount: number;
  onNewFile: () => void;
  onUpload: (files: FileList) => void;
  onDownloadAll: () => void;
  onClearAll: () => void;
  onRefresh: () => void;
}

export function FilesToolbar({
  root,
  setRoot,
  fileCount,
  onNewFile,
  onUpload,
  onDownloadAll,
  onClearAll,
  onRefresh,
}: Props) {
  const uploadRef = useRef<HTMLInputElement>(null);

  return (
    <div className="flex items-center justify-between px-3 py-2 border-b border-zinc-800/40 bg-zinc-900/30">
      <div className="flex items-center gap-2">
        <select
          value={root}
          onChange={(e) => setRoot(e.target.value)}
          className="text-[10px] bg-zinc-800 text-zinc-300 border border-zinc-700 rounded px-1.5 py-0.5 outline-none"
        >
          <option value="/configs">/configs</option>
          <option value="/home">/home</option>
          <option value="/workspace">/workspace</option>
          <option value="/plugins">/plugins</option>
        </select>
        <span className="text-[10px] text-zinc-400">{fileCount} files</span>
      </div>
      <div className="flex gap-1.5">
        {root === "/configs" && (
          <>
            <button onClick={onNewFile} className="text-[10px] text-blue-400 hover:text-blue-300 focus-visible:ring-2 focus-visible:ring-blue-500/70 focus-visible:outline-none rounded" title="Create new file">
              + New
            </button>
            <input
              ref={uploadRef}
              type="file"
              // @ts-expect-error webkitdirectory
              webkitdirectory=""
              multiple
              className="hidden"
              onChange={(e) => e.target.files && onUpload(e.target.files)}
            />
            <button onClick={() => uploadRef.current?.click()} className="text-[10px] text-blue-400 hover:text-blue-300 focus-visible:ring-2 focus-visible:ring-blue-500/70 focus-visible:outline-none rounded" title="Upload folder">
              Upload
            </button>
          </>
        )}
        <button onClick={onDownloadAll} className="text-[10px] text-zinc-400 hover:text-zinc-300 focus-visible:ring-2 focus-visible:ring-blue-500/70 focus-visible:outline-none rounded" title="Download all files">
          Export
        </button>
        {root === "/configs" && (
          <button onClick={onClearAll} className="text-[10px] text-red-400/60 hover:text-red-400 focus-visible:ring-2 focus-visible:ring-blue-500/70 focus-visible:outline-none rounded" title="Delete all files">
            Clear
          </button>
        )}
        <button onClick={onRefresh} className="text-[10px] text-zinc-400 hover:text-zinc-300 focus-visible:ring-2 focus-visible:ring-blue-500/70 focus-visible:outline-none rounded" title="Refresh">
          ↻
        </button>
      </div>
    </div>
  );
}
