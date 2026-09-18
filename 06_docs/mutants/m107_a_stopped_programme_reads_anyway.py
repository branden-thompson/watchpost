# 0.14.0 PD-1. Neutralise airOnce's advances() guard: a card takes the air on a
# station the listener has stopped.
#
# THE ASYMMETRY IS THE POINT — stopping the radio stops the PROGRAMME, not the
# hazards — and this guard is the half that holds the programme. The property
# test asks the DIRECTOR's own power rather than its own bookkeeping, so a
# driver that lost track of the station cannot make it pass.
import pathlib
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\tif !d.advances(track) {\n\t\treturn d, nil, false // the listener stopped the programme (PD-1)\n\t}"
assert old in s, "m107"
p.write_text(s.replace(old, "\tif false && !d.advances(track) {\n\t\treturn d, nil, false // the listener stopped the programme (PD-1)\n\t}"))
