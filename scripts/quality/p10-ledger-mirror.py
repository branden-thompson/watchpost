#!/usr/bin/env python3
"""Mirror the P10 exemption ledger into the public record, curated.

WHY THIS EXISTS. The ledger itself is machine-local by policy: `.gitignore`
keeps `.a2dh*` out of the tree under "Local development harness — not part of
the public project", and that is correct — the harness is not this project. The
CONSEQUENCE is that every HUM LEAD ratification lives on one disk. A fresh clone
reports every finding as unratified, CI cannot run the gate meaningfully, and
the record of what was approved and why is one disk failure from gone.

So the ledger stays out and its DECISIONS come in, curated. This script is the
only supported way to produce that mirror: hand-editing the output is how a
machine path reaches a public repository.

WHAT IS CARRIED: the rule, the repo-relative file and symbol, the reason, and
the ratification date.

WHAT IS NEVER CARRIED, and why each one matters on a PUBLIC repository:
  - absolute machine paths (/Users/..., /home/..., /Volumes/...) — they name a
    person's account and a directory layout nobody outside can use;
  - paths to the harness (`_a2dh/`, `.a2dh*`, `AGENTS.md`, `CLAUDE.md`,
    `.github/copilot-instructions.md`) — all git-ignored, so a public reader
    follows them to a 404 and learns the project depends on something it cannot
    see;
  - A2DH skill paths (`0N_skills/...`), which appear in the CHECKER'S OWN JSON
    under `skill_path` and `next_step`. This is the most likely leak: a future
    agent drafting a reason copies the finding wholesale and the skill path
    rides along with it;
  - any internal project tree, or the A2DH CLI's install location — naming
    where the tool lives tells a reader nothing and exposes an internal tree.
    The class is defined once, in Go (`tools/internaltrees`), by shape and by
    the names read from the disk at run time, so a workspace rename does not
    leave this filter describing the old layout;
  - usernames and email addresses.

HOW TO ADD A ROW (for the next session):
  1. A P10 exemption is RATIFIED, NEVER SELF-ISSUED. Draft the reason, present
     it to the HUM LEAD, and write it only once they have said so.
  2. Write it into the machine-local ledger, with `ratified:` set to the date
     they said it.
  3. Re-run this script. Do not edit the mirror by hand.
  4. Run `scripts/quality/lint-ledger.sh`, which fails on every pattern above.
     It is a gate rather than a convention because a convention is a rule the
     next worker has not read.
"""
import os
import re
import subprocess
import sys

LEDGER = ".a2dh-p10-exemptions.yml"
OUT = "06_docs/p10-ledger.md"


def internal_trees():
    """The internal-tree expression, asked of the one Go command that defines it.

    A missing or failing command stops the generator: writing a public file with
    one of its refusals silently absent is the failure this file exists to stop.
    """
    root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    try:
        out = subprocess.run(["go", "run", "./tools/internaltrees"], cwd=root,
                             capture_output=True, text=True, check=True).stdout.strip()
    except (OSError, subprocess.CalledProcessError) as err:
        sys.exit(f"p10-ledger-mirror: cannot read the internal-tree rule from tools/internaltrees: {err}")
    if not out:
        sys.exit("p10-ledger-mirror: tools/internaltrees produced no pattern")
    return out


# The same patterns lint-ledger.sh enforces. Kept here so a bad row is refused
# at WRITE time, not only at lint time — the earlier a leak is stopped the fewer
# places it has been copied to.
FORBIDDEN = [
    (r"/Users/|/home/|/Volumes/", "an absolute machine path"),
    (r"(^|[^A-Za-z0-9_])_a2dh/|\.a2dh[-.\w]*", "a path to the local harness"),
    (r"\b\d\d_skills/", "an A2DH skill path (these ride in on the checker's own JSON)"),
    (internal_trees(), "an internal project tree"),
    (r"\bAGENTS\.md\b|\bCLAUDE\.md\b|copilot-instructions", "a git-ignored harness file"),
    (r"[\w.+-]+@[\w-]+\.[\w.]+", "an email address"),
]


def offending(text):
    for pat, why in FORBIDDEN:
        m = re.search(pat, text)
        if m:
            return f"{why} ({m.group(0)!r})"
    return None


def main():
    try:
        import yaml
    except ImportError:
        sys.exit("p10-ledger-mirror: PyYAML is required to read the ledger")
    if not os.path.exists(LEDGER):
        sys.exit(f"p10-ledger-mirror: {LEDGER} not found — run from the repository root")
    rows = yaml.safe_load(open(LEDGER))["exemptions"]

    bad = []
    for r in rows:
        blob = " ".join(str(v) for v in r.values())
        why = offending(blob)
        if why:
            bad.append((r.get("file", "?"), r.get("rule_id", "?"), why))
        # A `file:` that is not in this repository is either a typo or a row
        # about something the public tree does not contain. Both are wrong here.
        f = r.get("file", "")
        if f and not os.path.exists(f):
            bad.append((f, r.get("rule_id", "?"), "a path that does not exist in this repository"))
    if bad:
        print("p10-ledger-mirror: REFUSING to write; these rows would leak:", file=sys.stderr)
        for f, rule, why in bad:
            print(f"  {f}  [{rule}]  -> {why}", file=sys.stderr)
        print("\nFix the ROW in the ledger, not this script's filters.", file=sys.stderr)
        return 1

    by_rule = {}
    for r in rows:
        by_rule.setdefault(r["rule_id"], []).append(r)

    out = []
    out.append("<!-- GENERATED by scripts/quality/p10-ledger-mirror.py — do not edit by hand. -->")
    out.append("<!-- Hand-editing is how a machine path reaches a public repository. -->")
    out.append("")
    out.append("# P10 exemption ledger — the public mirror")
    out.append("")
    out.append("**Every row here was RATIFIED by the HUM LEAD, never self-issued.** That is the "
               "standing rule for a P10 exemption: an agent may draft a reason and must not accept it.")
    out.append("")
    out.append("**Why this file exists.** The ledger the gate actually reads is machine-local — the "
               "harness is deliberately kept out of this repository — so without this mirror every "
               "ratification would live on one disk, a fresh clone would report every finding as "
               "unratified, and the record of what was approved and why would be one disk failure from "
               "gone. The ledger stays out; its DECISIONS come in.")
    out.append("")
    out.append("**It is generated.** Add a row by ratifying it with the HUM LEAD, writing it to the "
               "ledger, re-running `scripts/quality/p10-ledger-mirror.py`, and running "
               "`scripts/quality/lint-ledger.sh`. The generator refuses to write a row carrying an "
               "absolute path, a harness path, an A2DH skill path, an internal project tree, or an "
               "email address — and the lint refuses the file if one ever gets in another way.")
    out.append("")
    out.append(f"**{len(rows)} rows.**")
    out.append("")
    for rule in sorted(by_rule):
        rs = by_rule[rule]
        out.append(f"## {rule} ({len(rs)})")
        out.append("")
        out.append("| File | Symbol | Ratified | Reason |")
        out.append("|---|---|---|---|")
        for r in sorted(rs, key=lambda x: (str(x.get("ratified", "")), x["file"])):
            reason = " ".join(str(r.get("reason", "")).split()).replace("|", "\\|")
            out.append(f"| `{r['file']}` | `{r.get('symbol','package')}` | "
                       f"{r.get('ratified','—')} | {reason} |")
        out.append("")
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    with open(OUT, "w") as fh:
        fh.write("\n".join(out) + "\n")
    print(f"p10-ledger-mirror: wrote {OUT} ({len(rows)} rows, {len(by_rule)} rules)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
