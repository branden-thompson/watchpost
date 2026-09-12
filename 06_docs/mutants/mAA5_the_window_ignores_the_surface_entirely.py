import pathlib
# rowVisible stops asking the scope, so the Settings window draws every row on
# every surface — the pre-D-92 state, and the one the HUM LEAD reported: "certain
# Observer settings don't make sense when in Broadcaster and vice-versa." M4
# ("settings bleed", target 0) counts exactly this.
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = "\treturn setupTable()[id].scope.shownOn(d.surface)"
new = "\treturn true"
assert old in s, "mAA5"
p.write_text(s.replace(old, new, 1))
