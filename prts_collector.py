#!/usr/bin/env python3
"""
Fetch a PRTS wiki page and save the raw HTML/text for later parsing.
"""

from __future__ import annotations

import argparse
import json
import re
from html.parser import HTMLParser
from pathlib import Path

import requests


class HTMLTextExtractor(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.parts: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag in {"p", "br", "li", "tr", "div", "h1", "h2", "h3"}:
            self.parts.append("\n")

    def handle_data(self, data: str) -> None:
        if data and data.strip():
            self.parts.append(data)


def html_to_text(html: str) -> str:
    extractor = HTMLTextExtractor()
    extractor.feed(html)
    text = "".join(extractor.parts)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip()


def main() -> None:
    parser = argparse.ArgumentParser(description="PRTS page collector")
    parser.add_argument("--url", required=True)
    parser.add_argument("--output-dir", default="./knowledge/prts_character")
    args = parser.parse_args()

    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    response = requests.get(args.url, timeout=20)
    response.raise_for_status()

    html_path = output_dir / "page.html"
    html_path.write_text(response.text, encoding="utf-8")

    text = html_to_text(response.text)
    (output_dir / "page.txt").write_text(text, encoding="utf-8")

    summary = {
        "url": args.url,
        "html": str(html_path),
        "text_length": len(text),
    }
    (output_dir / "collection_summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == "__main__":
    main()
