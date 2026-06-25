#!/usr/bin/env python3
"""
Parse a saved PRTS page into a lightweight character profile JSON.
"""

from __future__ import annotations

import argparse
import json
import re
from html.parser import HTMLParser
from html import unescape
from pathlib import Path


def normalize_text(value: str) -> str:
    return re.sub(r"\s+", " ", value).strip()


def clean_page_title(title: str) -> str:
    cleaned = normalize_text(title)
    cleaned = re.sub(r"\s*[-|_]\s*PRTS.*$", "", cleaned)
    return cleaned


def strip_html(value: str) -> str:
    value = re.sub(r"<br\s*/?>", "\n", value, flags=re.I)
    value = re.sub(r"</p>|</div>|</tr>|</li>|</h[1-6]>", "\n", value, flags=re.I)
    value = re.sub(r"<[^>]+>", " ", value)
    value = unescape(value)
    value = re.sub(r"[ \t\r\f\v]+", " ", value)
    value = re.sub(r"\n{3,}", "\n\n", value)
    return value.strip()


class PRTSHTMLParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.blocks: list[dict] = []
        self._title_parts: list[str] = []
        self._active_heading: str | None = None
        self._heading_parts: list[str] = []
        self._active_text_tag: str | None = None
        self._text_parts: list[str] = []
        self._in_table = False
        self._in_row = False
        self._active_cell: str | None = None
        self._cell_parts: list[str] = []
        self._current_row: list[tuple[str, str]] = []
        self._current_table: list[list[tuple[str, str]]] = []
        self._all_text_parts: list[str] = []

    @property
    def page_title(self) -> str:
        return clean_page_title("".join(self._title_parts))

    @property
    def raw_text(self) -> str:
        lines = [normalize_text(part) for part in self._all_text_parts]
        lines = [line for line in lines if line]
        return "\n".join(lines)

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag == "title":
            self._active_text_tag = "title"
            self._text_parts = []
            return

        if tag in {"h1", "h2", "h3"}:
            self._flush_text_block()
            self._active_heading = tag
            self._heading_parts = []
            return

        if tag in {"p", "li", "dd", "blockquote"}:
            self._flush_text_block()
            self._active_text_tag = tag
            self._text_parts = []
            return

        if tag == "table":
            self._flush_text_block()
            self._in_table = True
            self._current_table = []
            return

        if tag == "tr" and self._in_table:
            self._in_row = True
            self._current_row = []
            return

        if tag in {"th", "td"} and self._in_row:
            self._active_cell = tag
            self._cell_parts = []

    def handle_endtag(self, tag: str) -> None:
        if tag == "title" and self._active_text_tag == "title":
            self._title_parts.extend(self._text_parts)
            self._active_text_tag = None
            self._text_parts = []
            return

        if self._active_heading == tag:
            text = normalize_text("".join(self._heading_parts))
            if text:
                self.blocks.append(
                    {"type": "heading", "level": int(tag[1]), "text": text}
                )
            self._active_heading = None
            self._heading_parts = []
            return

        if self._active_text_tag == tag:
            self._flush_text_block()
            return

        if tag in {"th", "td"} and self._active_cell == tag:
            value = normalize_text("".join(self._cell_parts))
            self._current_row.append((tag, value))
            self._active_cell = None
            self._cell_parts = []
            return

        if tag == "tr" and self._in_row:
            if self._current_row:
                self._current_table.append(self._current_row)
            self._current_row = []
            self._in_row = False
            return

        if tag == "table" and self._in_table:
            if self._current_table:
                self.blocks.append({"type": "table", "rows": self._current_table})
            self._current_table = []
            self._in_table = False

    def handle_data(self, data: str) -> None:
        if not data or not data.strip():
            return

        self._all_text_parts.append(data)

        if self._active_heading:
            self._heading_parts.append(data)
            return

        if self._active_cell:
            self._cell_parts.append(data)
            return

        if self._active_text_tag:
            self._text_parts.append(data)

    def _flush_text_block(self) -> None:
        if self._active_text_tag and self._active_text_tag != "title":
            text = normalize_text("".join(self._text_parts))
            if text:
                self.blocks.append({"type": "text", "text": text})
        if self._active_text_tag != "title":
            self._active_text_tag = None
            self._text_parts = []


def rows_to_mapping(rows: list[list[tuple[str, str]]]) -> dict[str, str]:
    mapping: dict[str, str] = {}
    for row in rows:
        if len(row) < 2:
            continue
        key = row[0][1]
        value = " ".join(cell_text for _, cell_text in row[1:] if cell_text)
        key = normalize_text(key)
        value = normalize_text(value)
        if key and value:
            mapping[key] = value
    return mapping


def extract_char_info_block(html: str) -> str:
    start = html.find("var char_info=")
    if start == -1:
        return ""
    brace_start = html.find("{", start)
    if brace_start == -1:
        return ""

    depth = 0
    for idx in range(brace_start, len(html)):
        char = html[idx]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return html[brace_start : idx + 1]
    return ""


def extract_simple_field(block: str, field_name: str) -> str:
    pattern = rf'"{re.escape(field_name)}"\s*:\s*"([^"]*)"'
    match = re.search(pattern, block)
    return normalize_text(match.group(1)) if match else ""


def extract_cv_map(block: str) -> dict[str, str]:
    results: dict[str, str] = {}
    for lang, name in re.findall(r'"([^"]+)"\s*:\s*\{[^{}]*?"name"\s*:\s*"([^"]*)"', block, re.S):
        lang = normalize_text(lang)
        name = normalize_text(name)
        if lang and name:
            results[lang] = name
    return results


def extract_core_profile(html: str) -> dict:
    block = extract_char_info_block(html)
    if not block:
        return {}

    tag_value = extract_simple_field(block, "tag")
    tags = [part for part in re.split(r"[/,，、\s]+", tag_value) if part]

    profile = {
        "name": extract_simple_field(block, "name"),
        "name_en": extract_simple_field(block, "nameEn"),
        "faction": extract_simple_field(block, "group"),
        "class": extract_simple_field(block, "class"),
        "branch": extract_simple_field(block, "branch"),
        "position": extract_simple_field(block, "pos"),
        "tags": tags,
        "painter": extract_simple_field(block, "painter"),
        "cv": extract_cv_map(block),
    }
    return {key: value for key, value in profile.items() if value}


def extract_voice_items(html: str) -> list[dict]:
    voices: list[dict] = []
    start_pattern = re.compile(
        r'<div class="voice-data-item"[^>]*data-title="([^"]+)"[^>]*>',
        re.S,
    )
    matches = list(start_pattern.finditer(html))
    for index, match in enumerate(matches):
        title = normalize_text(match.group(1))
        body_start = match.end()
        body_end = matches[index + 1].start() if index + 1 < len(matches) else len(html)
        body = html[body_start:body_end]
        zh_match = re.search(
            r'data-kind-name="中文"[^>]*>(.*?)</div>',
            body,
            re.S,
        )
        if not zh_match:
            continue
        text = normalize_text(re.sub(r"<[^>]+>", " ", zh_match.group(1)))
        if not text:
            continue
        voices.append({"title": normalize_text(title), "text": text})
    return voices


def extract_section_html(html: str, section_title: str) -> str:
    patterns = [
        rf'<h2[^>]*>.*?id="{re.escape(section_title)}".*?</h2>(.*?)(?=<h2[^>]*>)',
        rf"<h2[^>]*>\s*{re.escape(section_title)}\s*</h2>(.*?)(?=<h2[^>]*>|$)",
    ]
    for pattern in patterns:
        match = re.search(pattern, html, re.S)
        if match:
            return match.group(1)
    return ""


def extract_archive_records(html: str) -> list[dict]:
    section_html = extract_section_html(html, "干员档案")
    if not section_html:
        return []

    section_html = re.sub(r"<style.*?</style>", "", section_html, flags=re.S | re.I)
    records: list[dict] = []
    current_title = ""
    current_unlock = ""

    row_pattern = re.compile(r"<tr[^>]*>(.*?)</tr>", re.S | re.I)
    cell_pattern = re.compile(r"<(th|td)[^>]*>(.*?)</\1>", re.S | re.I)

    for row_html in row_pattern.findall(section_html):
        cells = cell_pattern.findall(row_html)
        if not cells:
            continue
        first_kind, first_html = cells[0]
        first_text = normalize_text(strip_html(first_html))
        if not first_text:
            continue

        if first_kind == "th" and len(cells) == 1:
            if "初始开放" in first_text or "解锁" in first_text:
                current_unlock = first_text
            elif "档案" in first_text or "履历" in first_text or "分析" in first_text or "测试" in first_text:
                current_title = first_text
                current_unlock = ""
            continue

        if first_kind == "td":
            body = normalize_text(strip_html(first_html))
            if body:
                records.append(
                    {
                        "title": current_title or "档案片段",
                        "unlock": current_unlock,
                        "content": body,
                    }
                )
                current_unlock = ""

    return records


def build_info_section(core_profile: dict) -> str:
    lines = []
    if core_profile.get("name"):
        lines.append(f"名称：{core_profile['name']}")
    if core_profile.get("name_en"):
        lines.append(f"英文名：{core_profile['name_en']}")
    if core_profile.get("faction"):
        lines.append(f"所属势力：{core_profile['faction']}")
    if core_profile.get("class"):
        lines.append(f"职业：{core_profile['class']}")
    if core_profile.get("branch"):
        lines.append(f"分支：{core_profile['branch']}")
    if core_profile.get("position"):
        lines.append(f"位置：{core_profile['position']}")
    if core_profile.get("tags"):
        lines.append(f"标签：{'、'.join(core_profile['tags'])}")
    if core_profile.get("painter"):
        lines.append(f"画师：{core_profile['painter']}")
    return "\n".join(lines)


def build_attributes_section(attributes: dict[str, str]) -> str:
    lines = []
    for key, value in attributes.items():
        if not key or not value:
            continue
        lines.append(f"{key}：{value}")
    return "\n".join(lines)


def build_archive_section(records: list[dict]) -> str:
    chunks = []
    for record in records:
        title = record["title"]
        unlock = f"（{record['unlock']}）" if record.get("unlock") else ""
        chunks.append(f"{title}{unlock}\n{record['content']}")
    return "\n\n".join(chunks)


def build_voice_section(voices: list[dict]) -> str:
    return "\n".join(f"{voice['title']}：{voice['text']}" for voice in voices)


def dedupe_repeated_key(key: str) -> str:
    key = normalize_text(key)
    if len(key) % 2 == 0:
        half = len(key) // 2
        if key[:half] == key[half:]:
            return key[:half]
    return key


def is_noisy_text(text: str) -> bool:
    suspicious_markers = (
        "document.addEventListener",
        "function(",
        "switchDisplay",
        "RLQ.push",
        "Cookies.set",
        "mw.",
        "{{",
        "}}",
    )
    return any(marker in text for marker in suspicious_markers)


def clean_table_mapping(mapping: dict[str, str]) -> dict[str, str]:
    cleaned: dict[str, str] = {}
    for raw_key, raw_value in mapping.items():
        key = dedupe_repeated_key(raw_key)
        value = normalize_text(raw_value)
        if not key or not value:
            continue
        if key in {"分支", "等级", "条件"} and value in {
            "描述",
            "图标 技能2 房间 描述",
        }:
            continue
        if key in {"？？？"}:
            continue
        if is_noisy_text(key) or is_noisy_text(value):
            continue
        cleaned[key] = value
    return cleaned


def build_generic_section(mapping: dict[str, str]) -> str:
    return "\n".join(f"{key}：{value}" for key, value in mapping.items())


def collect_quotes(section_map: dict[str, str], raw_text: str) -> list[str]:
    quotes: list[str] = []

    for title, content in section_map.items():
        if "语音" not in title and "台词" not in title:
            continue
        for line in content.splitlines():
            line = normalize_text(line)
            if line:
                quotes.append(line)

    if not quotes:
        for line in raw_text.splitlines():
            line = normalize_text(line)
            if not line:
                continue
            if any(marker in line for marker in ("博士", "罗德岛", "行动", "干员")):
                quotes.append(line)
            if len(quotes) >= 20:
                break

    deduped: list[str] = []
    seen: set[str] = set()
    for quote in quotes:
        if quote in seen:
            continue
        seen.add(quote)
        deduped.append(quote)
    return deduped[:20]


def parse_html(html: str) -> dict:
    parser = PRTSHTMLParser()
    parser.feed(html)

    section_map: dict[str, str] = {}
    section_tables: dict[str, dict[str, str]] = {}
    sections: list[dict] = []
    identity: dict[str, str] = {}
    attributes: dict[str, str] = {}

    current_section: str | None = None
    for block in parser.blocks:
        if block["type"] == "heading":
            current_section = block["text"]
            section_map.setdefault(current_section, "")
            sections.append(
                {
                    "title": current_section,
                    "level": block["level"],
                    "content": "",
                }
            )
            continue

        if block["type"] == "text":
            if not current_section:
                continue
            existing = section_map.get(current_section, "")
            merged = "\n".join(part for part in (existing, block["text"]) if part)
            section_map[current_section] = merged
            if sections:
                sections[-1]["content"] = merged
            continue

        if block["type"] == "table":
            table_map = rows_to_mapping(block["rows"])
            if not table_map:
                continue

            if current_section:
                section_tables[current_section] = {
                    **section_tables.get(current_section, {}),
                    **table_map,
                }
                if current_section in {"干员信息", "基础信息", "角色信息", "档案资料"}:
                    identity.update(table_map)
                elif "属性" in current_section:
                    attributes.update(table_map)
                elif not identity:
                    identity.update(table_map)
            elif not identity:
                identity.update(table_map)

    page_title = parser.page_title
    if not page_title:
        first_heading = next((s["title"] for s in sections if s["level"] == 1), "")
        page_title = clean_page_title(first_heading or "unknown")

    core_profile = extract_core_profile(html)
    voices = extract_voice_items(html)
    archive_records = extract_archive_records(html)
    clean_tables = {
        title: clean_table_mapping(table_map)
        for title, table_map in section_tables.items()
    }
    clean_tables = {title: table for title, table in clean_tables.items() if table}

    if core_profile.get("name") and "名称" not in identity:
        identity["名称"] = core_profile["name"]
    if core_profile.get("faction") and "所属势力" not in identity:
        identity["所属势力"] = core_profile["faction"]
    if core_profile.get("class") and "职业" not in identity:
        identity["职业"] = core_profile["class"]

    quote_seed = [voice["text"] for voice in voices]
    quote_seed.extend(collect_quotes(section_map, parser.raw_text))
    deduped_quotes: list[str] = []
    seen_quotes: set[str] = set()
    for quote in quote_seed:
        quote = normalize_text(quote)
        if not quote or quote in seen_quotes:
            continue
        seen_quotes.add(quote)
        deduped_quotes.append(quote)
        if len(deduped_quotes) >= 20:
            break

    info_section = build_info_section(core_profile)
    if info_section:
        section_map["干员信息"] = info_section
        for section in sections:
            if section["title"] == "干员信息":
                section["content"] = info_section
                break

    attributes_section = build_attributes_section(attributes)
    if attributes_section:
        section_map["属性"] = attributes_section
        for section in sections:
            if section["title"] == "属性":
                section["content"] = attributes_section
                break

    archive_section = build_archive_section(archive_records)
    if archive_section:
        section_map["干员档案"] = archive_section
        for section in sections:
            if section["title"] == "干员档案":
                section["content"] = archive_section
                break

    voice_section = build_voice_section(voices)
    if voice_section:
        section_map["语音记录"] = voice_section
        for section in sections:
            if section["title"] == "语音记录":
                section["content"] = voice_section
                break

    for title in ("特性", "获得方式", "后勤技能", "相关道具", "精英化材料", "潜能提升"):
        mapping = clean_tables.get(title)
        if not mapping:
            continue
        content = build_generic_section(mapping)
        if not content:
            continue
        section_map[title] = content
        for section in sections:
            if section["title"] == title:
                section["content"] = content
                break

    return {
        "page_title": page_title,
        "identity": identity,
        "attributes": attributes,
        "core_profile": core_profile,
        "voices": voices,
        "archive_records": archive_records,
        "clean_tables": clean_tables,
        "sections": sections,
        "section_map": section_map,
        "section_tables": section_tables,
        "quotes": deduped_quotes,
        "raw_excerpt": parser.raw_text[:4000],
    }


def main() -> None:
    cli = argparse.ArgumentParser(description="PRTS parser")
    cli.add_argument("--input", required=True)
    cli.add_argument("--output", default="")
    args = cli.parse_args()

    html = Path(args.input).read_text(encoding="utf-8")
    profile = parse_html(html)
    payload = json.dumps(profile, ensure_ascii=False, indent=2)

    if args.output:
        Path(args.output).write_text(payload, encoding="utf-8")
    else:
        print(payload)


if __name__ == "__main__":
    main()
