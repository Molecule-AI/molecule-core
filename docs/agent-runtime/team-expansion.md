# Team Expansion (Recursive Workspaces)

When a workspace is expanded into a team, it gains sub-workspaces while its own agent remains as the **team lead** (coordinator). This is recursive — sub-workspaces can themselves be expanded into teams, infinitely deep.

## How It Works

When Developer PM is expanded into a team:

```
Business Core
   |
   +-- Developer PM (agent stays, becomes coordinator)
          |
          +-- Frontend Agent (sub-workspace, private scope)
          +-- Backend Agent  (sub-workspace, private scope)
          +-- QA Agent       (sub-workspace, private scope)
```

- Developer PM's agent **still exists** and acts as coordinator
- Developer PM receives incoming A2A messages from Business Core
- Developer PM's agent decides how to delegate to sub-workspaces
- Sub-workspaces talk to Developer PM and to each other (same level)
- Sub-workspaces **cannot** talk to Business Core or any workspace outside the team

## Communication Rules

| Direction | Allowed? | Example |
|-----------|----------|---------|
| Parent level -> team lead | Yes | Business Core -> Developer PM |
| Team lead -> sub-workspaces | Yes | Developer PM -> Frontend Agent |
| Sub-workspace -> team lead | Yes | Frontend Agent -> Developer PM |
| Sub-workspace <-> sibling | Yes | Frontend Agent <-> Backend Agent |
| Outside -> sub-workspace directly | No (403) | Business Core -> Frontend Agent |
| Sub-workspace -> outside directly | No | Frontend Agent -> Business Core |

The team lead (Developer PM) is the **only** bridge between the team's internal world and the outside.

## Scoped Registry

Sub-workspaces register in the platform registry but with a **private scope**. The registry knows about them but enforces access control.

```
Registry:
  Business Core      :8001   scope: public
  Developer PM       :8002   scope: public
  Frontend Agent     :8010   scope: private, parent=Developer PM
  Backend Agent      :8011   scope: private, parent=Developer PM
  QA Agent           :8012   scope: private, parent=Developer PM
```

- The platform can always discover any workspace (for provisioning, monitoring)
- The parent workspace can discover its sub-workspaces
- Sub-workspaces can discover their siblings (same parent)
- Outside workspaces get a **403 Forbidden** if they try to discover a private sub-workspace

## How to Expand

Expansion is triggered via `POST /workspaces/:id/expand`. The platform reads the `sub_workspaces` list from the workspace's config and provisions each one. On the canvas, users right-click a workspace node and select "Expand into team."

Collapsing is the inverse: `POST /workspaces/:id/collapse`. Sub-workspaces are stopped and removed.

## What Happens on Expansion

When Developer PM is expanded into a team, the hierarchy changes but the outside view doesn't. Business Core's parent/child relationship to Developer PM is unaffected — Developer PM still responds to the same A2A endpoint.

The events fired:
- `WORKSPACE_EXPANDED` with the new `sub_workspace_ids` in the payload
- `WORKSPACE_PROVISIONING` for each new sub-workspace
- `WORKSPACE_ONLINE` for each sub-workspace as they come up

Communication rules are automatically derived from the new hierarchy — no manual wiring needed.

## Canvas Behavior

- Children render as embedded mini-cards (`TeamMemberChip`) inside the parent node, not as separate canvas nodes
- Each mini-card shows full status: gradient bar, name, tier badge, skills pills, active tasks, descendant count
- **Recursive rendering** up to 3 levels deep (`MAX_NESTING_DEPTH = 3`) — sub-cards can contain their own "Team" sections
- Parent node dynamically resizes: 210-280px (no children), 320-450px (children), 400-560px (grandchildren)
- Eject button (sky-blue arrow icon) on hover extracts a child from the team
- "Extract from Team" also available in the right-click context menu
- Double-click a team node to zoom/fit to the parent area
- The parent workspace node shows a badge with total descendant count

## Collapsing a Team

The inverse of expansion, triggered via `POST /workspaces/:id/collapse`:

1. Each sub-workspace agent wraps up current work and writes a handoff document to memory
2. Sub-workspaces are stopped and removed
3. The team lead's agent goes back to handling everything directly
4. A `WORKSPACE_COLLAPSED` event fires

Sub-workspace memory is cleaned up based on backend (see [Memory — Cleanup](../architecture/memory.md#cleanup-on-workspace-deletion)).

## Deleting a Team Workspace

When a team workspace is deleted:
1. Platform shows a warning listing all sub-workspaces that will be deleted
2. User can **drag sub-workspaces out** of the team before confirming (promotes them to the parent level)
3. On confirmation, cascade delete removes the parent and all remaining sub-workspaces
4. `WORKSPACE_REMOVED` events fire for each deleted workspace

## Related Docs

- [Communication Rules](../api-protocol/communication-rules.md) — Full access control model
- [Core Concepts](../product/core-concepts.md) — Workspace fundamentals
- [System Prompt Structure](./system-prompt-structure.md) — How peer capabilities are injected
- [Provisioner](../architecture/provisioner.md) — How sub-workspaces are deployed
- [Registry & Heartbeat](../api-protocol/registry-and-heartbeat.md) — How registration works
- [Event Log](../architecture/event-log.md) — Events fired during expansion
- [Canvas UI](../frontend/canvas.md) — Visual behavior of teams
