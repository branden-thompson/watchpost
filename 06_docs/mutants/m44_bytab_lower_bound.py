# m44 — the bucket walk's LOWER bound.
#
# RE-POINTED 2026-09-08. This used to patch domains/severe/severe.go, where
# ByTab carried its own copy of the guard. Metric D collapsed that walk and
# tty's identical one into platform/bucket.ByIndex, and the mutant stopped
# matching the tip — which the harness reported as "the rule it guards is
# UNMEASURED", correctly: a mutant that no longer applies is not a passing
# mutant, it is an absent one.
#
# It now patches the single owner, so it guards the rule for BOTH callers
# instead of one.
import pathlib
p = pathlib.Path("platform/bucket/bucket.go"); s = p.read_text()
old = "if b := of(it); b >= 0 && b < n {"
assert old in s, "m44"
p.write_text(s.replace(old, "if b := of(it); b >= -1 && b < n {"))
