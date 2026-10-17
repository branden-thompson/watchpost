import pathlib
# 0.19.0 AX-12: a dimmed badge is bracketed.
# Killed by TestTheSourcesBadgesSayWhatFedTheSnapshot.
p = pathlib.Path('modes/tty/map_parts.go'); s = p.read_text()
old = '\treturn render.Tint(" ("+face+") ", render.Tok(render.TableMuted))'
assert old in s, 'm422'
p.write_text(s.replace(old, '\treturn render.Tint("  "+face+"  ", render.Tok(render.TableMuted))'))
