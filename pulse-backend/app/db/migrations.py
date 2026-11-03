from __future__ import annotations

import logging
from pathlib import Path
from typing import List
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncEngine

logger = logging.getLogger("pulse-backend")


def split_statements(sql_content: str) -> List[str]:
    """Split SQL file into statements ending with ';' on a line, matching Go migrate.go."""
    stmts: List[str] = []
    current: List[str] = []
    for line in sql_content.splitlines():
        trimmed = line.strip()
        if not trimmed or trimmed.startswith("--"):
            continue
        current.append(line)
        if trimmed.endswith(";"):
            joined = "\n".join(current).strip()
            if joined.endswith(";"):
                joined = joined[:-1].strip()
            stmts.append(joined)
            current = []
    if current:
        rest = "\n".join(current).strip()
        if rest:
            stmts.append(rest)
    return stmts


async def run_migrations(engine: AsyncEngine, migrations_dir: Path | None = None) -> None:
    """Run all unapplied *.sql migrations in filename order."""
    if migrations_dir is None:
        # Check standard locations
        possible_dirs = [
            Path(__file__).resolve().parent.parent.parent / "migrations",
            Path("migrations"),
            Path("/src/migrations"),
        ]
        for p in possible_dirs:
            if p.is_dir():
                migrations_dir = p
                break
        if migrations_dir is None:
            logger.warning("no migrations directory found")
            return

    async with engine.begin() as conn:
        await conn.execute(
            text(
                """
                CREATE TABLE IF NOT EXISTS schema_migrations (
                    version    VARCHAR(255) NOT NULL PRIMARY KEY,
                    applied_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
                """
            )
        )
        res = await conn.execute(text("SELECT version FROM schema_migrations"))
        applied_versions = {row[0] for row in res.fetchall()}

        files = sorted(migrations_dir.glob("*.sql"))
        for file_path in files:
            name = file_path.name
            if name in applied_versions:
                continue

            content = file_path.read_text(encoding="utf-8")
            statements = split_statements(content)
            for idx, stmt in enumerate(statements, start=1):
                try:
                    await conn.execute(text(stmt))
                except Exception as e:
                    logger.error("migrate: %s statement %d failed: %s", name, idx, e)
                    raise RuntimeError(f"migrate: {name} statement {idx} failed: {e}") from e

            await conn.execute(
                text("INSERT INTO schema_migrations (version) VALUES (:version)"),
                {"version": name},
            )
            logger.info("migrate: applied %s", name)

