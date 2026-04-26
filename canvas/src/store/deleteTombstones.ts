/**
 * Transient "recently deleted" map used by canvas.ts to short-circuit
 * the hydrate-races-cascade-delete window described in #2069.
 *
 * Problem
 * -------
 * `removeSubtree(rootId)` drops a parent + descendants locally after
 * `DELETE /workspaces/:id?confirm=true` returns 200. `hydrate()` is
 * server-authoritative and rebuilds the entire node array from whatever
 * `/workspaces` returns. If a `GET /workspaces` was IN-FLIGHT before the
 * DELETE completed, its response (still containing the deleted subtree)
 * can land AFTER our local `removeSubtree`, hydrate the store with the
 * stale snapshot, and re-introduce the deleted nodes on the canvas.
 *
 * Fix
 * ---
 * `removeSubtree` calls `markDeleted(ids)` to record a tombstone for every
 * removed id. `hydrate` calls `wasRecentlyDeleted(id)` to filter out any
 * incoming workspace whose id matches a fresh tombstone. After
 * `TOMBSTONE_TTL_MS`, the entry expires and a legitimately-recreated id
 * (template re-import, undo, manual recreate) flows through normally.
 *
 * GC happens lazily at every read AND at write time so the map stays
 * bounded — no separate timer / interval / unmount plumbing.
 *
 * Module-level (not store state) so it doesn't trigger React Flow
 * re-renders and isn't part of the public store surface. The store is a
 * singleton, so module identity ≡ store identity for this purpose.
 */

const TOMBSTONE_TTL_MS = 10_000; // matches the 10s WS-fallback poll cadence

const tombstones = new Map<string, number>();

function gcExpired(now: number): void {
  for (const [id, deletedAt] of tombstones) {
    if (now - deletedAt >= TOMBSTONE_TTL_MS) {
      tombstones.delete(id);
    }
  }
}

export function markDeleted(ids: Iterable<string>): void {
  const now = Date.now();
  gcExpired(now);
  for (const id of ids) {
    tombstones.set(id, now);
  }
}

export function wasRecentlyDeleted(id: string): boolean {
  const deletedAt = tombstones.get(id);
  if (deletedAt === undefined) return false;
  if (Date.now() - deletedAt >= TOMBSTONE_TTL_MS) {
    tombstones.delete(id);
    return false;
  }
  return true;
}

/** Test-only: clear the module-level map between tests. Production code
 *  must not call this — the map is intentionally process-lifetime. */
export function __resetTombstonesForTest(): void {
  tombstones.clear();
}

/** Test-only: inspect the current tombstone count. */
export function __tombstoneCountForTest(): number {
  return tombstones.size;
}
