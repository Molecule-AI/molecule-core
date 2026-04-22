-- Platform-level configurable instructions with global/team/workspace scope.
-- Injected into every agent's system prompt at startup and refreshed
-- periodically, so platform operators can enforce rules without editing
-- template files.
CREATE TABLE IF NOT EXISTS platform_instructions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       TEXT NOT NULL CHECK (scope IN ('global', 'team', 'workspace')),
    scope_target TEXT,  -- NULL for global, team slug for team, workspace_id for workspace
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    priority    INT DEFAULT 0,  -- higher = shown first within scope
    enabled     BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_instructions_scope
    ON platform_instructions (scope, scope_target) WHERE enabled = true;
