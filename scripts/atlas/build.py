#!/usr/bin/env python3
"""Build the Watchpost architecture atlas from the tracked documents.

**The documents are the source of truth and this page is generated from them.**
It reads every ```mermaid block under 06_docs/ and docs/, keeps the heading
above each one as its title and the paragraph between as its lead, and groups
them by the feature they belong to.

Two things worth knowing before editing a diagram and wondering why the atlas
did not change:

  * only fenced mermaid blocks are read. A change written in the prose beside a
    diagram never reaches this page - that was learned the hard way on the
    sibling project's atlas.
  * the order below is the reading order. A feature not listed is still built,
    after the listed ones, so nothing is silently dropped.

Run from the repository root:  python3 scripts/atlas/build.py
"""

import html
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
OUT = ROOT / "06_docs" / "architecture-atlas.html"
TEMPLATE = pathlib.Path(__file__).with_name("template.html")

# Reading order: what a person should meet first, not alphabetical.
ORDER = [
    ("watchpost-cli", "The shape of the program"),
    ("map-ready-geometry", "Map-ready geometry (0.17.0)"),
    ("severe-alerts-modals", "Severe weather"),
    ("seismic-data", "Earthquakes"),
    ("global-ticker", "The national feed"),
    ("multi-voice-support", "The Director and the voices"),
    ("0.16.0-broadcaster-ui", "The Broadcaster console"),
    ("0.15.0-pre-broadcaster-ui-improvements", "Before the console"),
    ("watchpost-performance-quality-pass", "The quality pass"),
]


def feature_of(path: pathlib.Path) -> str:
    parts = path.relative_to(ROOT).parts
    if "02_features" in parts:
        return parts[parts.index("02_features") + 1]
    return "docs"


def inline(text: str) -> str:
    text = html.escape(text)
    text = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", text)
    text = re.sub(r"\*(.+?)\*", r"<em>\1</em>", text)
    return re.sub(r"`([^`]+)`", r"<code>\1</code>", text)


def collect():
    found = []
    for md in sorted(ROOT.glob("06_docs/**/*.md")) + sorted(ROOT.glob("docs/**/*.md")):
        text = md.read_text(encoding="utf-8")
        if "```mermaid" not in text:
            continue
        for m in re.finditer(r"```mermaid\n(.*?)```", text, re.S):
            heads = list(re.finditer(r"^#+ (.+)$", text[: m.start()], flags=re.M))
            title = heads[-1].group(1).strip() if heads else md.stem
            lead = text[heads[-1].end(): m.start()].strip() if heads else ""
            lead = re.sub(r"\n+", " ", lead)
            if lead.startswith("|") or lead.startswith("```") or len(lead) > 420:
                lead = ""
            found.append(
                {
                    "feature": feature_of(md),
                    "file": str(md.relative_to(ROOT)),
                    "title": title,
                    "lead": inline(lead) if lead else "",
                    "src": m.group(1),
                }
            )
    return found


def main() -> int:
    blocks = collect()
    if not blocks:
        print("no mermaid blocks found", file=sys.stderr)
        return 1
    named = [f for f, _ in ORDER]
    order = named + sorted({b["feature"] for b in blocks} - set(named))
    titles = dict(ORDER)

    nav, body = [], []
    for feat in order:
        mine = [b for b in blocks if b["feature"] == feat]
        if not mine:
            continue
        label = titles.get(feat, feat.replace("-", " "))
        nav.append(
            f'<li><a href="#{feat}">{html.escape(label)}<span class="n">{len(mine)}</span></a></li>'
        )
        body.append(
            f'<section class="level" id="{feat}"><header class="lvl">'
            f"<h2>{html.escape(label)}</h2></header>"
        )
        for i, b in enumerate(mine):
            lead = f'<p class="lead">{b["lead"]}</p>' if b["lead"] else ""
            body.append(
                f'<article class="dia" id="{feat}-{i}"><div class="meta">'
                f'<h3>{html.escape(b["title"])}</h3>'
                f'<p class="file"><code>{html.escape(b["file"])}</code></p>{lead}</div>'
                f'<div class="canvas"><pre class="mermaid">{html.escape(b["src"])}</pre></div></article>'
            )
        body.append("</section>")

    page = TEMPLATE.read_text(encoding="utf-8")
    page = page.replace("<!--NAV-->", "".join(nav))
    page = page.replace("<!--BODY-->", "".join(body))
    page = page.replace("<!--COUNT-->", str(len(blocks)))
    OUT.write_text(page, encoding="utf-8")
    print(f"{len(blocks)} diagrams across {len(set(b['feature'] for b in blocks))} areas -> {OUT.relative_to(ROOT)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
