"""Claude Code adaptor for molecule-guardrails.

Inherits the generic agentskills installer (which copies skills into
/configs/skills/ and appends rules into CLAUDE.md), then adds Claude
Code-specific install steps:

  1. Hooks  → copied into <configs>/.claude/hooks/
  2. Commands → copied into <configs>/.claude/commands/
  3. settings.json hook fragment → merged into <configs>/.claude/settings.json
     with ${CLAUDE_DIR} placeholder rewritten to the absolute install path.
"""
from __future__ import annotations

import json
import shutil
from pathlib import Path

from plugins_registry.builtins import AgentskillsAdaptor
from plugins_registry.protocol import InstallContext, InstallResult


class Adaptor(AgentskillsAdaptor):
    """Extends the base adapter with hook + slash-command + settings install."""

    async def install(self, ctx: InstallContext) -> InstallResult:
        # Run the standard rules + skills installer first
        result = await super().install(ctx)

        claude_dir = ctx.configs_dir / ".claude"
        claude_dir.mkdir(parents=True, exist_ok=True)

        self._install_dir(ctx.plugin_root / "hooks", claude_dir / "hooks", result, executable_suffix=".sh")
        self._install_dir(ctx.plugin_root / "commands", claude_dir / "commands", result, allow_suffix=".md")
        self._merge_settings(ctx.plugin_root, claude_dir, result, ctx)

        return result

    async def uninstall(self, ctx: InstallContext) -> None:
        await super().uninstall(ctx)
        # Best-effort: remove our hook + command files. settings.json
        # entries we leave (they reference paths that simply won't fire
        # — cleaning them robustly requires marker tracking we don't ship yet).
        claude_dir = ctx.configs_dir / ".claude"
        for sub in ("hooks", "commands"):
            src = ctx.plugin_root / sub
            if not src.is_dir():
                continue
            for f in src.iterdir():
                target = claude_dir / sub / f.name
                if target.exists():
                    target.unlink()
                    ctx.logger.info("%s: removed %s", self.plugin_name, target)

    @staticmethod
    def _install_dir(
        src: Path,
        dst: Path,
        result: InstallResult,
        executable_suffix: str | None = None,
        allow_suffix: str | None = None,
    ) -> None:
        if not src.is_dir():
            return
        dst.mkdir(parents=True, exist_ok=True)
        for f in src.iterdir():
            if not f.is_file():
                continue
            if allow_suffix and f.suffix != allow_suffix:
                # also allow .py companion files when copying hooks
                if not (executable_suffix and f.suffix == ".py"):
                    continue
            target = dst / f.name
            shutil.copy2(f, target)
            if executable_suffix and f.suffix == executable_suffix:
                target.chmod(0o755)
            result.files_written.append(str(target))

    @staticmethod
    def _merge_settings(
        plugin_root: Path,
        claude_dir: Path,
        result: InstallResult,
        ctx: InstallContext,
    ) -> None:
        fragment_path = plugin_root / "settings-fragment.json"
        if not fragment_path.is_file():
            return
        try:
            fragment = json.loads(fragment_path.read_text())
        except Exception as e:
            result.warnings.append(f"settings-fragment.json invalid: {e}")
            return

        settings_path = claude_dir / "settings.json"
        if settings_path.is_file():
            try:
                existing = json.loads(settings_path.read_text())
            except Exception:
                existing = {}
        else:
            existing = {}

        rewritten = Adaptor._rewrite_hook_paths(fragment, claude_dir)
        merged = Adaptor._deep_merge_hooks(existing, rewritten)
        settings_path.write_text(json.dumps(merged, indent=2) + "\n")
        result.files_written.append(str(settings_path))
        ctx.logger.info("%s: merged hook config into %s", "molecule-guardrails", settings_path)

    @staticmethod
    def _rewrite_hook_paths(fragment: dict, claude_dir: Path) -> dict:
        """Replace ${CLAUDE_DIR} placeholder in hook command strings."""
        out = json.loads(json.dumps(fragment))  # deep copy via roundtrip
        for handlers in out.get("hooks", {}).values():
            for handler in handlers:
                for h in handler.get("hooks", []):
                    cmd = h.get("command", "")
                    h["command"] = cmd.replace("${CLAUDE_DIR}", str(claude_dir))
        return out

    @staticmethod
    def _deep_merge_hooks(existing: dict, fragment: dict) -> dict:
        """Append fragment hooks to existing without removing existing ones."""
        out = dict(existing)
        out.setdefault("hooks", {})
        for event, handlers in fragment.get("hooks", {}).items():
            out["hooks"].setdefault(event, [])
            out["hooks"][event].extend(handlers)
        for key, val in fragment.items():
            if key == "hooks":
                continue
            out.setdefault(key, val)
        return out
