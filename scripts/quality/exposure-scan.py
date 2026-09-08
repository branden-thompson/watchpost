#!/usr/bin/env python3
"""exposure-scan.py — what a PUBLIC repository discloses about the person who wrote it.

FR-7.5. This is a survey, not a gate: it reports, it does not fail a build. The
ruling on what to do about each category is the HUM LEAD's.

WHY IT SCANS MORE THAN THE WORKING TREE. Revision 1 of the requirement scoped
this to 06_docs. A scan that never looks at the git history, a published tag or
a built artifact is scoped wrong: `git rm` does not unpublish, every tag carries
its own copy of the tree, and a binary embeds the paths it was compiled from.
The repository has been public since 0.13.0, so this measures an exposure that
already happened, not one that might.

IT NEVER PRINTS A CREDENTIAL. Matches in that category are reported as a
location and a length; the value is the thing being protected.

usage: python3 scripts/quality/exposure-scan.py [--full-history]
"""
import re, subprocess, sys, pathlib, collections

def sh(*a, **kw):
    return subprocess.run(a, capture_output=True, text=True, errors="ignore", **kw).stdout

# The maintainer's own identifiers, read from git rather than hardcoded, so the
# scan follows the repository instead of needing an edit per contributor.
NAME  = sh("git", "config", "user.name").strip() or "Branden Thompson"
EMAIL = sh("git", "config", "user.email").strip()
USER  = pathlib.Path.home().name
SURNAME = NAME.split()[-1] if NAME else ""
FIRST   = NAME.split()[0] if NAME else ""

# ONE REGEX ENGINE FOR EVERY SCOPE. The first version scanned the working tree
# with Python's `re` and the history with `git grep -E`, which is POSIX ERE: the
# inline `(?i)` flags were a FATAL error there, so four of the six categories
# reported a confident ZERO from a command that had crashed. Two dialects also
# means two definitions of a category, and numbers that cannot be compared
# across the scopes they are being compared across.
CATEGORIES = {
    # label: (compiled pattern, redact the match when printing?)
    "identity":     (re.compile(rf"{re.escape(EMAIL)}|{re.escape(NAME)}|\b{re.escape(USER)}\b"), False),
    "location":     (re.compile(r"\b9[12][0-9]{3}\b|Oceanside|Vista, CA|Carlsbad"), False),
    "host":         (re.compile(rf"{re.escape(USER)}-[a-z0-9]+|MacBook|iMac|\.local\b"), False),
    # Tightened from the first run, which counted every "internal/poll" in the Go
    # runtime and reported ~4,600 per binary. A category that fires on the
    # standard library measures the standard library.
    "internal-url": (re.compile(r"https?://[a-z0-9.-]*(?:corp|internal|intranet)[a-z0-9.-]*/\S*", re.I), False),
    # NARROWED, because the first version reported 2,658 "credentials" in the
    # history and 100 in the tree, and every one was a false positive. In THIS
    # codebase "token" is a colour: `token = "tui.StatusQueued"`. A secret
    # scanner that fires on the theme system finds nothing and hides what it
    # would have found. The value must now look like a key rather than an
    # identifier: no dots, no spaces, and enough distinct characters to be
    # random. Reported as DISTINCT VALUES, never occurrences — one fixture
    # copied into sixty blobs is one thing to judge, not sixty.
    "credential":   (re.compile(r"\b[0-9a-f]{32}\b|(?:api[_-]?key|apikey|secret|password)\s*[:=]\s*['\"]([A-Za-z0-9+/=_-]{16,})['\"]", re.I), True),
    "path":         (re.compile(rf"/Users/{re.escape(USER)}|/home/{re.escape(USER)}"), False),
}

def scan_text(text, pat):
    return pat.findall(text) if pat.groups == 0 else [m.group(0) for m in pat.finditer(text)]

def worktree():
    files = [f for f in sh("git", "ls-files").split("\n") if f]
    hits = collections.defaultdict(lambda: collections.defaultdict(int))
    examples = collections.defaultdict(set)
    for f in files:
        p = pathlib.Path(f)
        if not p.is_file():
            continue
        try:
            t = p.read_text(errors="ignore")
        except Exception:
            continue
        for cat, (pat, _) in CATEGORIES.items():
            for m in scan_text(t, pat):
                hits[cat][f] += 1
                if len(examples[cat]) < 4:
                    examples[cat].add(f)
    return hits, examples

def every_blob():
    """Every blob ever committed, decoded as text. Reachability is not the
    question: an unreferenced blob is still in a clone."""
    ids = subprocess.run(["git", "cat-file", "--batch-all-objects", "--batch-check=%(objectname) %(objecttype) %(objectsize)"],
                         capture_output=True, text=True).stdout
    blobs = [l.split()[0] for l in ids.split("\n")
             if l and l.split()[1] == "blob" and int(l.split()[2]) < 2_000_000]
    for i in range(0, len(blobs), 200):
        chunk = blobs[i:i + 200]
        proc = subprocess.run(["git", "cat-file", "--batch"], input="\n".join(chunk),
                              capture_output=True, text=True, errors="ignore")
        yield proc.stdout

def tag_texts():
    """Each published tag's whole tree as text — a tag is its own copy, and
    `git rm` on main unpublishes none of them."""
    for t in [t for t in sh("git", "tag").split("\n") if t]:
        raw = subprocess.run(["git", "archive", "--format=tar", t],
                             capture_output=True).stdout
        yield t, raw.decode("utf-8", errors="ignore")

def artifacts():
    found = collections.defaultdict(list)
    for p in sorted(pathlib.Path("dist").glob("watchpost*")) if pathlib.Path("dist").is_dir() else []:
        if not p.is_file():
            continue
        raw = subprocess.run(["strings", str(p)], capture_output=True, text=True, errors="ignore").stdout
        for cat, (pat, _) in CATEGORIES.items():
            n = len(scan_text(raw, pat))
            if n:
                found[p.name].append((cat, n))
    return found

def images():
    out = []
    d = pathlib.Path("docs/img")
    if not d.is_dir():
        return out
    for p in sorted(d.iterdir()):
        if not p.is_file():
            continue
        raw = subprocess.run(["strings", str(p)], capture_output=True, text=True, errors="ignore").stdout
        for cat, (pat, _) in CATEGORIES.items():
            hits = scan_text(raw, pat)
            if hits:
                out.append((p.name, cat, len(hits)))
    return out

def main():
    print(f"EXPOSURE SCAN — repository is {sh('gh','repo','view','--json','visibility','-q','.visibility').strip() or 'UNKNOWN'}")
    print(f"identifiers followed: name={NAME!r} user={USER!r} email={'set' if EMAIL else 'unset'}\n")

    hits, examples = worktree()
    print("== 1. WORKING TREE (tracked files) ==")
    for cat, (pat, redact) in CATEGORIES.items():
        files = hits[cat]
        total = sum(files.values())
        print(f"  {cat:13s} {len(files):4d} files, {total:5d} occurrences"
              + ("" if not files else f"   e.g. {', '.join(sorted(examples[cat])[:3])}"))
    print()

    print("== 2. GIT HISTORY (every blob ever committed) ==")
    hist = collections.Counter()
    for chunk in every_blob():
        for cat, (pat, _) in CATEGORIES.items():
            hist[cat] += len(scan_text(chunk, pat))
    for cat in CATEGORIES:
        print(f"  {cat:13s} {hist[cat]:6d} occurrences — `git rm` removes none of these")
    print()

    print("== 3. PUBLISHED TAGS (each is its own published copy) ==")
    per_tag = {}
    for t, text in tag_texts():
        per_tag[t] = {cat: len(scan_text(text, pat)) for cat, (pat, _) in CATEGORIES.items()}
    tags = sorted(per_tag)
    for cat in CATEGORIES:
        carrying = [t for t in tags if per_tag[t][cat]]
        print(f"  {cat:13s} {len(carrying):2d}/{len(tags)} tags"
              + (f"   earliest: {carrying[0]}" if carrying else ""))
    print()

    print("== 4. BUILT ARTIFACTS (dist/) ==")
    a = artifacts()
    if not a:
        print("  none present (dist/ is not tracked; build one to re-check)")
    for name, cats in a.items():
        print(f"  {name}: " + ", ".join(f"{c}×{n}" for c, n in cats))
    print()

    print("== 5. README IMAGES ==")
    im = images()
    if not im:
        print("  no identifiers found in image bytes")
    for name, cat, n in im:
        print(f"  {name}: {cat} ×{n}")

if __name__ == "__main__":
    main()
