import pathlib
# D-134/D-136. The UP NEXT LABEL CELL goes back to the report's own tile ground,
# so it no longer matches the blue under D I R E C T I O N and T O D A Y — two
# answers to what the app's band blue is, drifting apart on the next theme edit.
#
# THE CELL IS MOVED ONTO THE GROUND IT WORE BEFORE THE RULING, not deleted: the
# box still draws and still has a ground everywhere. Only the SAMENESS with the
# bands is gone, which is the whole of the rule.
p = pathlib.Path("modes/tty/broadcaster_upnext.go"); s = p.read_text()
old = '\tlabelGround := render.Tok(render.GroupText) + ";" + render.Tok(render.GroupTodayBG)\n\tfg, bg := render.ModalTone(b.darkBG)\n\tground := fg + ";" + bg'
new = '\tfg, bg := render.ModalTone(b.darkBG)\n\tground := fg + ";" + bg\n\tlabelGround := ground'
assert old in s, "mCG2"
p.write_text(s.replace(old, new, 1))
