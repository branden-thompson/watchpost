import pathlib
# D-158. FR-1.5's exit sentence is "an override in the user's key table changes
# the chord", and this is the defect as it shipped: `broadcasterKeyMap()` reached
# the Router raw, so no `[keys]` entry could change a single console binding —
# including `ctrl+b`, tmux's own default prefix and the exact key the multiplexer
# survey says an operator will need to rebind. The gate that stood beside it
# asserted the actions were IN the map, which an unreachable map satisfies.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = "keys: o.consoleKeyMap(), onSurface:"
assert old in s, "mKB1"
p.write_text(s.replace(old, "keys: broadcasterKeyMap(), onSurface:", 1))
