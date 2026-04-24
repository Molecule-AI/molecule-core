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
          aria-label="File root directory"
          className="text-[10px] bg-zinc-800 text-zinc-300 border border-zinc-700 rounded px-1.5 py-0.5 outline-none"
        >
          <option value="/configs">/configs</option>
          <option value="/home">/home</option>
          <option value="/workspace">/workspace</option>
          <option value="/plugins">/plugins</option>
        </select>
        <span className="text-[10px] text-zinc-500">{fileCount} files</span>
      </div>
      <div className="flex gap-1.5">
        {root === "/configs" && (
          <>
            <button type="button" onClick={onNewFile} aria-label="Create new file" className="text-[10px] text-blue-400 hover:text-blue-300" title="Create new file">
              + New
            </button>
            <input
              ref={uploadRef}
              type="file"
              aria-label="Upload folder files"
              // @ts-expect-error webkitdirectory
              webkitdirectory=""
              multiple
              className="hidden"
              onChange={(e) => e.target.files && onUpload(e.target.files)}
            />
            <button type="button" onClick={() => uploadRef.current?.click()} aria-label="Upload folder" className="text-[10px] text-blue-400 hover:text-blue-300" title="Upload folder">
              Upload
            </button>
          </>
        )}
        <button type="button" onClick={onDownloadAll} aria-label="Download all files" className="text-[10px] text-zinc-500 hover:text-zinc-300" title="Download all files">
          Export
        </button>
        {root === "/configs" && (
          <button type="button" onClick={onClearAll} aria-label="Delete all files" className="text-[10px] text-red-400/60 hover:text-red-400" title="Delete all files">
            Clear
          </button>
        )}
        <button type="button" onClick={onRefresh} aria-label="Refresh file list" className="text-[10px] text-zinc-500 hover:text-zinc-300" title="Refresh">
          ↻
        </button>
      </div>
    </div>
  );
}
