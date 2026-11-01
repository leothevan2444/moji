#!/usr/bin/env python3
"""Check the shared agent guide without third-party dependencies or network IO."""

import re
from pathlib import Path
from urllib.parse import unquote, urlsplit


ROOT = Path(__file__).resolve().parents[2]
GUIDES = ROOT / ".agents"
LINK = re.compile(r"\[[^\]\n]*\]\((<[^>]+>|[^\s)]+)(?:\s+\"[^\"]*\")?\)")


def prose(text):
    """Preserve line numbers while ignoring fenced examples."""
    fence = None
    for line in text.splitlines():
        marker = re.match(r"^\s{0,3}(`{3,}|~{3,})", line)
        if fence is None and marker:
            fence = marker[1]
            yield ""
        elif fence is not None:
            if re.fullmatch(r"\s{0,3}" + re.escape(fence[0]) + "{" + str(len(fence)) + r",}\s*", line):
                fence = None
            yield ""
        else:
            yield line


def anchors(path):
    found, counts = set(), {}
    for line in prose(path.read_text(encoding="utf-8")):
        found.update(re.findall(r'<(?:a|span)\s+id=[\"\']([^\"\']+)', line))
        heading = re.match(r"^#{1,6}\s+(.+?)\s*#*\s*$", line)
        if not heading:
            continue
        slug = re.sub(r"[^\w\- ]", "", heading[1].lower()).replace(" ", "-")
        count = counts.get(slug, 0)
        found.add(slug if count == 0 else f"{slug}-{count}")
        counts[slug] = count + 1
    return found


def main():
    documents = sorted(GUIDES.rglob("*.md"))
    documents += [ROOT / "AGENTS.md", ROOT / ".github/copilot-instructions.md"]
    required = [
        "AGENTS.md", "README.md", "architecture.md", "TODO.md",
        "standards/product.md", "standards/engineering.md",
        "workflows/development.md", "workflows/review.md",
    ]
    errors = [f"missing required guide: {name}" for name in required if not (GUIDES / name).is_file()]
    edges = {}
    for document in documents:
        if not document.is_file():
            errors.append(f"missing entry: {document.relative_to(ROOT)}")
            continue
        edges[document] = set()
        for number, line in enumerate(prose(document.read_text(encoding="utf-8")), 1):
            for match in LINK.finditer(line):
                raw = match[1].strip("<>")
                url = urlsplit(raw)
                if url.scheme or url.netloc:
                    continue
                label = f"{document.relative_to(ROOT)}:{number}"
                if url.path.startswith("/"):
                    errors.append(f"{label}: use a relative link: {raw}")
                    continue
                target = (document.parent / unquote(url.path)).resolve()
                if not target.is_relative_to(ROOT) or not target.exists():
                    errors.append(f"{label}: missing or out-of-repository target: {raw}")
                    continue
                if target.suffix == ".md":
                    edges[document].add(target)
                    if url.fragment and unquote(url.fragment) not in anchors(target):
                        errors.append(f"{label}: missing anchor: {raw}")
    entry = GUIDES / "AGENTS.md"
    for folder in ("modules", "standards"):
        for guide in (GUIDES / folder).glob("*.md"):
            if guide not in edges.get(entry, set()):
                errors.append(f"{folder} guide not routed from main entry: {guide.relative_to(ROOT)}")
    for pointer in (ROOT / "AGENTS.md", ROOT / ".github/copilot-instructions.md"):
        if entry not in edges.get(pointer, set()):
            errors.append(f"entry must point to .agents/AGENTS.md: {pointer.relative_to(ROOT)}")
    seen, pending = set(), [entry]
    while pending:
        current = pending.pop()
        if current not in seen:
            seen.add(current)
            pending.extend(edges.get(current, set()) - seen)
    for document in documents:
        if document.is_relative_to(GUIDES) and document not in seen:
            errors.append(f"unreachable guide: {document.relative_to(ROOT)}")
    if errors:
        print("\n".join(errors))
        return 1
    print(f"Agent docs OK: {len(documents)} Markdown files; local links, anchors and entry routing verified.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
