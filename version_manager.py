#!/usr/bin/env python3
"""
Simple version listing/rollback helper for generated character skills.
"""

from __future__ import annotations

import argparse
import json
import shutil
from datetime import datetime, timezone
from pathlib import Path


def list_versions(skill_dir: Path) -> list[str]:
    versions_dir = skill_dir / "versions"
    if not versions_dir.exists():
        return []
    return sorted([item.name for item in versions_dir.iterdir() if item.is_dir()])


def rollback(skill_dir: Path, version: str) -> None:
    version_dir = skill_dir / "versions" / version
    if not version_dir.exists():
        raise SystemExit(f"missing version: {version}")

    for name in ("persona.md", "lore.md", "relationship.md", "custom.md", "SKILL.md"):
        src = version_dir / name
        if src.exists():
            shutil.copy2(src, skill_dir / name)

    meta_path = skill_dir / "meta.json"
    meta = json.loads(meta_path.read_text(encoding="utf-8"))
    meta["updated_at"] = datetime.now(timezone.utc).isoformat()
    meta["version"] = f"{version}_restored"
    meta_path.write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description="Character skill version manager")
    parser.add_argument("--action", required=True, choices=["list", "rollback"])
    parser.add_argument("--slug", required=True)
    parser.add_argument("--version")
    parser.add_argument("--base-dir", default="./characters")
    args = parser.parse_args()

    skill_dir = Path(args.base_dir).expanduser() / args.slug
    if args.action == "list":
        for version in list_versions(skill_dir):
            print(version)
    else:
        if not args.version:
            raise SystemExit("--version is required for rollback")
        rollback(skill_dir, args.version)
        print(f"rolled back {args.slug} to {args.version}")


if __name__ == "__main__":
    main()
