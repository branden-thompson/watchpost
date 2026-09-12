import pathlib
# The columns stop being padded to their own widths before the join, so a row
# whose alert column ran short pulls the running order leftward on that row
# alone — which is occlusion arriving by accident rather than by design.
#
# SUPERSEDES the four mN mutants that guarded D-83's script window, deleted at
# D-87: the window was replaced by a manifest, so they guarded a rule that no
# longer exists. HUM LEAD, 2026-09-11: "no more card occlusion."
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\t\t\treturn render.PadTo(render.TruncateCells(col[i], w), w)"
new = "\t\t\t\treturn col[i]"
assert old in s, "mS0"
p.write_text(s.replace(old, new, 1))
