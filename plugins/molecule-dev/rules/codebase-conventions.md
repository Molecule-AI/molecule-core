# Molecule AI Codebase Conventions

These rules apply to every agent working on the Molecule AI / Molecule AI codebase.
They are lessons learned from real bugs — not style preferences. Violating them
causes production failures.

## Canvas (Next.js 15 App Router)

### `'use client'` — NON-NEGOTIABLE
Every `.tsx` file in `canvas/src/` that uses React hooks (`useState`, `useEffect`,
`useCallback`, `useMemo`, `useRef`), Zustand stores (`useCanvasStore`, `useSecretsStore`),
or event handlers (`onClick`, `onChange`) MUST have `'use client';` as the very first
line. Without it, Next.js renders the component as server HTML and React never hydrates
it — buttons appear but silently don't respond to clicks.

**This has caused two reverted PRs.** Always run this check before reporting done:
```bash
cd canvas
for f in $(grep -rl "useState\|useEffect\|useCallback\|useMemo\|useRef\|useStore\|onClick\|onChange" src/ --include="*.tsx"); do
  head -3 "$f" | grep -q "use client" || echo "MISSING 'use client': $f"
done
```

### Zustand Selectors — No New Objects
Never call a function that returns a new object inside a Zustand selector:
```typescript
// BAD — creates new object every render → infinite re-renders
const grouped = useSecretsStore((s) => s.getGrouped());

// GOOD — use useMemo with stable selector values
const secrets = useSecretsStore((s) => s.secrets);
const grouped = useMemo(() => groupSecrets(secrets), [secrets]);
```

### Dark Zinc Theme — No Light Colors
The canvas is dark-themed. Every new component must use:
- Backgrounds: `zinc-900`, `zinc-950`, `#18181b`, `#09090b`
- Text: `zinc-300`, `zinc-400`, `#d4d4d8`, `#a1a1aa`
- Accents: `blue-500`, `blue-600`, `violet-500`
- Borders: `zinc-700`, `zinc-800`
- Never: `white`, `#ffffff`, `#f5f5f5`, or any light gray

### API Response Shapes
Always verify the actual platform API response format before writing fetch code.
Check the Go handler or test with curl — don't assume. Past bug: FE assumed
`GET /settings/secrets` returned a flat array but it returns `{ secrets: [...] }`.

## Platform (Go)

### SQL Safety
- Always use parameterized queries (`$1`, `$2`), never string concatenation
- Use `ExecContext` / `QueryContext` with context, never bare `Exec` / `Query`
- Always check `rows.Err()` after iterating result sets
- JSONB: convert `[]byte` to `string()` and use `::jsonb` cast

### Access Control
- Every endpoint touching workspace data must verify ownership
- A2A proxy calls go through `CanCommunicate()` — new proxy paths must respect it
- System callers (`webhook:*`, `system:*`, `test:*`) bypass via `isSystemCaller()`

### Container Lifecycle
- Use `ContainerRemove(Force: true)` to stop containers — never `ContainerStop` +
  `ContainerRemove` separately (restart policy race condition causes zombies)
- Always reap zombie processes: `proc.wait()` after `proc.kill()` with a timeout

## Workspace Runtime (Python)

### Error Sanitization
Never emit raw exception messages or subprocess stderr to the user. Use
`sanitize_agent_error()` which exposes the exception class name but strips
the message body (which can leak tokens, paths, and stack traces).

### System Prompt Hot-Reload
System prompts are re-read from `/configs/system-prompt.md` on every message.
Always use `encoding="utf-8", errors="replace"` when reading prompt files.

## Inter-Agent Communication

### Parallel Delegation
Use `delegate_task_async` to send tasks to multiple peers simultaneously.
Don't serialize what can be parallel — fire all async delegations, then poll
`check_task_status` to collect results as they finish.

### Side Questions
Any agent can ask a peer a direct question via `delegate_task` (sync) without
going through the lead. FE can ask BE "what's the API response format?" mid-task.
Use this to avoid guessing — it's faster than getting it wrong.

### Proactive Updates
Use `send_message_to_user` to push status updates to the CEO's chat at any time.
Don't wait until everything is done to report — acknowledge immediately, update
during long work, deliver results when complete.

## Before Reporting Done

Every agent, regardless of role, must verify their own work before claiming completion:
1. Read back every file you changed — confirm it looks right
2. Run the relevant test suite (`npm test`, `go test`, `python -m pytest`)
3. Run the relevant build (`npm run build`, `go build`)
4. Check for framework-specific gotchas (the `'use client'` grep above)
5. If you can imagine a way your change could break, test that scenario
