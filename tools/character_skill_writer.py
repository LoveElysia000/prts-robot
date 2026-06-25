#!/usr/bin/env python3
"""Character skill writer for wiki-derived character profiles."""

from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path

DEFAULT_GLOBAL_CUSTOM_PATH = Path(__file__).resolve().parents[1] / "custom_global.md"
DEFAULT_CHARACTER_CUSTOM = "# Custom\n\n"
DEFAULT_RELATIONSHIP = "# Relationship\n\n"
EMPTY_GLOBAL_CUSTOM = "当前没有额外的通用补充规则。"
EMPTY_CHARACTER_CUSTOM = "当前没有额外的角色补充规则。"
EMPTY_RELATIONSHIP = "当前没有额外的关系身份规则。"
DEFAULT_OPTIONAL_SECTION_MARKERS = {
    "# Custom",
    "# Character Custom",
    "# Relationship",
}

SKILL_TEMPLATE = """\
---
name: character_{slug}
description: {name} 的角色 Skill
user-invocable: true
---

# {name}

## 运行原则

1. 优先像角色一样自然对话，而不是机械背设定
2. 允许进行日常聊天、安慰、鼓励、闲聊和陪伴式回应，只要不违背角色气质
3. 涉及设定事实的问题时优先依据 Lore
4. 页面未提供的信息，不要编造成确定事实，但可以用符合角色的方式表达“不确定”或“目前无法确认”
5. 生成结果需要同时遵守角色本体、通用补充规则、关系身份规则和角色补充规则
6. 关系身份规则只约束角色如何面向用户互动，不改变网页提炼出的角色本体

---

{persona}

---

## 通用补充规则

{custom_global}

---

## 关系身份规则

{relationship}

---

## 角色补充规则

{custom}

---

{lore}
"""


def write_json(path: Path, payload: dict) -> None:
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def _normalize_custom(content: str, empty_text: str) -> str:
    normalized = content.strip()
    if not normalized or normalized in DEFAULT_OPTIONAL_SECTION_MARKERS:
        return empty_text
    return normalized


def _read_optional_markdown(path: str | Path | None, fallback: str = "") -> str:
    if not path:
        return fallback
    file_path = Path(path)
    if not file_path.exists():
        return fallback
    return file_path.read_text(encoding="utf-8")


def create_skill(
    base_dir: Path,
    slug: str,
    name: str,
    persona: str,
    lore: str,
    source_url: str = "",
    custom_global: str = "",
    relationship: str = DEFAULT_RELATIONSHIP,
    custom: str = DEFAULT_CHARACTER_CUSTOM,
) -> Path:
    skill_dir = base_dir / slug
    skill_dir.mkdir(parents=True, exist_ok=True)
    (skill_dir / "versions").mkdir(exist_ok=True)

    global_text = custom_global if custom_global else _read_optional_markdown(DEFAULT_GLOBAL_CUSTOM_PATH)
    relationship_text = relationship if relationship else DEFAULT_RELATIONSHIP
    custom_text = custom if custom else DEFAULT_CHARACTER_CUSTOM

    (skill_dir / "persona.md").write_text(persona, encoding="utf-8")
    (skill_dir / "lore.md").write_text(lore, encoding="utf-8")
    (skill_dir / "relationship.md").write_text(relationship_text, encoding="utf-8")
    (skill_dir / "custom.md").write_text(custom_text, encoding="utf-8")
    (skill_dir / "SKILL.md").write_text(
        SKILL_TEMPLATE.format(
            slug=slug,
            name=name,
            persona=persona,
            lore=lore,
            custom_global=_normalize_custom(global_text, EMPTY_GLOBAL_CUSTOM),
            relationship=_normalize_custom(relationship_text, EMPTY_RELATIONSHIP),
            custom=_normalize_custom(custom_text, EMPTY_CHARACTER_CUSTOM),
        ),
        encoding="utf-8",
    )

    now = datetime.now(timezone.utc).isoformat()
    meta = {
        "name": name,
        "slug": slug,
        "source_url": source_url,
        "created_at": now,
        "updated_at": now,
        "version": "v1",
        "corrections_count": 0,
        "has_global_custom": bool(global_text.strip()),
        "relationship": _normalize_custom(relationship_text, ""),
        "has_relationship": bool(
            relationship_text.strip() and relationship_text.strip() not in DEFAULT_OPTIONAL_SECTION_MARKERS
        ),
        "has_character_custom": bool(
            custom_text.strip() and custom_text.strip() not in DEFAULT_OPTIONAL_SECTION_MARKERS
        ),
    }
    write_json(skill_dir / "meta.json", meta)
    return skill_dir


def main() -> None:
    parser = argparse.ArgumentParser(description="Character skill writer")
    parser.add_argument("--action", required=True, choices=["create"])
    parser.add_argument("--slug", required=True)
    parser.add_argument("--name", required=True)
    parser.add_argument("--persona", default="")
    parser.add_argument("--lore", default="")
    parser.add_argument("--custom-global", default="")
    parser.add_argument("--relationship", default="")
    parser.add_argument("--custom", default="")
    parser.add_argument("--source-url", default="")
    parser.add_argument("--base-dir", default="./characters")
    args = parser.parse_args()

    persona = Path(args.persona).read_text(encoding="utf-8") if args.persona else "# Persona\n"
    lore = Path(args.lore).read_text(encoding="utf-8") if args.lore else "# Lore\n"
    custom_global = _read_optional_markdown(args.custom_global)
    relationship = _read_optional_markdown(args.relationship, DEFAULT_RELATIONSHIP)
    custom = _read_optional_markdown(args.custom, DEFAULT_CHARACTER_CUSTOM)
    skill_dir = create_skill(
        Path(args.base_dir).expanduser(),
        args.slug,
        args.name,
        persona,
        lore,
        args.source_url,
        custom_global,
        relationship,
        custom,
    )
    print(f"created: {skill_dir}")


if __name__ == "__main__":
    main()
