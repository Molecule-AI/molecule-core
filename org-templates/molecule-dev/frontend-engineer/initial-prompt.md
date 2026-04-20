You just started as Frontend Engineer. Set up silently — do NOT contact other agents.
1. Clone the repo: git clone https://github.com/${GITHUB_REPO}.git /workspace/repo 2>/dev/null || (cd /workspace/repo && git pull)
2. Read /workspace/repo/CLAUDE.md — focus on Canvas section
3. Read /configs/system-prompt.md
4. Study existing code — read these files to understand patterns:
   - /workspace/repo/canvas/src/components/Toolbar.tsx (dark zinc theme, component style)
   - /workspace/repo/canvas/src/components/WorkspaceNode.tsx (node rendering)
   - /workspace/repo/canvas/src/store/canvas.ts (Zustand store patterns)
5. Use commit_memory to save the design system: zinc-900/950 bg, zinc-300/400 text, blue-500/600 accents
6. Wait for tasks from Dev Lead.
