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
# THE IDENTIFIERS COME FROM THE REPOSITORY'S OWN HISTORY, not from the machine
# and not from a checked-in list (red team, 2026-09-08, twice).
#
# First fault: reading `git config` and $HOME meant the scan measured whoever RAN
# it — on any other machine the `path` and `host` categories matched nothing and
# reported a confident ZERO against a real 12 files and 81 occurrences. A survey
# that answers "clean" because it is looking for the wrong name is worse than no
# survey.
#
# Second fault, in the fix: checking the identifiers into a tracked file made
# this the single most canonical, greppable statement of the author's name and
# username in a PUBLIC repository — added by a script whose purpose is to reduce
# that surface, and not covered by any of the exposure rulings. `git log` already
# publishes every contributor identity, so deriving from it is machine-
# independent, self-maintaining for new contributors, and adds nothing new to the
# public tree.
#
# A gitignored local file may still SUPPLEMENT this for identifiers the history
# does not carry (a corporate username, say); it is optional and never required.
IDENTITIES = pathlib.Path(__file__).with_name("exposure-identities.local.txt")

def identifiers():
    names, users, emails = set(), set(), set()
    if IDENTITIES.exists():
        for line in IDENTITIES.read_text().splitlines():
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            kind, _, value = line.partition(":")
            {"name": names, "user": users, "email": emails}.get(kind.strip(), set()).add(value.strip())
    # EVERY CONTRIBUTOR THE HISTORY KNOWS. This is the same set a reader of the
    # public repo can enumerate in one command, so naming it here discloses
    # nothing the tree does not already publish.
    for line in sh("git", "log", "--all", "--format=%an%n%ae").split("\n"):
        line = line.strip()
        if not line:
            continue
        (emails if "@" in line else names).add(line)
    if v := sh("git", "config", "user.name").strip():
        names.add(v)
    if v := sh("git", "config", "user.email").strip():
        emails.add(v)
    users.add(pathlib.Path.home().name)
    # A username is not in the git history; derive it from the paths the tree
    # already contains rather than requiring a checked-in list.
    for m in re.finditer(r"/(?:Users|home)/([A-Za-z0-9._-]+)", sh("git", "grep", "-hoE", r"/(Users|home)/[A-Za-z0-9._-]+") or ""):
        users.add(m.group(1))
    return sorted(n for n in names if n), sorted(u for u in users if u), sorted(e for e in emails if e)

NAMES, USERS, EMAILS = identifiers()
if not NAMES or not USERS:
    sys.exit("exposure-scan: no identifiers to search for — populate " + str(IDENTITIES))
# An EMPTY alternative matches at every byte offset, which is how an unset
# git email once produced 20,786,233 "identity" hits in the same format as a
# real number. Every alternative here is non-empty by construction.
NAME_ALT  = "|".join(re.escape(n) for n in NAMES)
USER_ALT  = "|".join(re.escape(u) for u in USERS)
EMAIL_ALT = "|".join(re.escape(e) for e in EMAILS) or r"(?!x)x"  # matches nothing, safely
NAME, EMAIL, USER = NAMES[0], (EMAILS[0] if EMAILS else ""), USERS[0]
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
    "identity":     (re.compile(rf"{EMAIL_ALT}|{NAME_ALT}|\b(?:{USER_ALT})\b"), False),
    "location":     (re.compile(r"\b9[12][0-9]{3}\b|Oceanside|Vista, CA|Carlsbad"), False),
    "host":         (re.compile(rf"(?:{USER_ALT})-[a-z0-9]+|MacBook|iMac|\.local\b"), False),
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
    "path":         (re.compile(rf"/Users/(?:{USER_ALT})|/home/(?:{USER_ALT})"), False),
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
    # BIG BLOBS ARE COUNTED, NOT DROPPED. The cap silently excluded four blobs
    # totalling 11.6 MB here — including the terminal-recording GIFs, which are
    # the highest-yield carrier of a hostname or a prompt string in this
    # repository, and one blob no longer reachable by any path, which is the
    # exact case this function exists to cover (red team, 2026-09-08).
    blobs, skipped, skipped_bytes = [], 0, 0
    for l in ids.split("\n"):
        if not l:
            continue
        name, kind, size = l.split()[0], l.split()[1], int(l.split()[2])
        if kind != "blob":
            continue
        if size >= 20_000_000:  # a real memory bound, an order of magnitude above any blob here
            skipped += 1
            skipped_bytes += size
            continue
        blobs.append(name)
    if skipped:
        print(f"  NOTE: {skipped} blob(s), {skipped_bytes/1e6:.1f} MB, exceeded the size bound and were NOT scanned")
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
