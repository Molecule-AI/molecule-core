"""Fallback adaptor used when no per-runtime adaptor is found.

Behaviour: copy the plugin's content into ``/configs/plugins/<name>/`` so a
user can still inspect or hand-wire it, then surface a warning that no tools
or sub-agents were registered.

This preserves the "power users can drop raw files" escape hatch without
silently breaking — the warning is propagated up via :class:`InstallResult`
so the API can surface it to the user.
"""

from __future__ import annotations

import shutil

from .protocol import InstallContext, InstallResult, PluginAdaptor


class RawDropAdaptor:
    """Filesystem-only fallback. Implements :class:`PluginAdaptor`."""

    def __init__(self, plugin_name: str, runtime: str) -> None:
        self.plugin_name = plugin_name
        self.runtime = runtime

    async def install(self, ctx: InstallContext) -> InstallResult:
        dst = ctx.configs_dir / "plugins" / self.plugin_name
        files_written: list[str] = []

        if ctx.plugin_root.exists() and ctx.plugin_root.is_dir():
            dst.parent.mkdir(parents=True, exist_ok=True)
            if dst.exists():
                # Idempotent — leave existing copy alone.
                ctx.logger.info(
                    "raw_drop: %s already present at %s, skipping copy",
                    self.plugin_name, dst,
                )
            else:
                shutil.copytree(ctx.plugin_root, dst)
                for p in dst.rglob("*"):
                    if p.is_file():
                        files_written.append(str(p.relative_to(ctx.configs_dir)))
                ctx.logger.info(
                    "raw_drop: copied %s → %s (%d files)",
                    self.plugin_name, dst, len(files_written),
                )

        warning = (
            f"plugin '{self.plugin_name}' has no adaptor for runtime "
            f"'{self.runtime}' — files dropped at /configs/plugins/{self.plugin_name} "
            f"but no tools/sub-agents were wired in"
        )
        ctx.logger.warning(warning)

        return InstallResult(
            plugin_name=self.plugin_name,
            runtime=self.runtime,
            source="raw_drop",
            files_written=files_written,
            warnings=[warning],
        )

    async def uninstall(self, ctx: InstallContext) -> None:
        dst = ctx.configs_dir / "plugins" / self.plugin_name
        if dst.exists():
            shutil.rmtree(dst)
            ctx.logger.info("raw_drop: removed %s", dst)


# Static check: RawDropAdaptor satisfies PluginAdaptor.
_: PluginAdaptor = RawDropAdaptor("_", "_")
