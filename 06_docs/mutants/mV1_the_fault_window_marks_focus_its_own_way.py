import pathlib
# The window marks focus with a bare glyph and no tint instead of going through
# render/list.go, which owns how every list marks focus (D-1). The pointer still
# moves, so the model is right and the LISTENER cannot tell: nothing on the line
# changes colour when the arrows are pressed.
#
# RE-ANCHORED at the wrap (UAT 2026-09-05 round 4): the row's head is built as
# its own value now so that WrapHanging can measure it. The rule is unchanged.
p = pathlib.Path("modes/tty/relayfault.go"); s = p.read_text()
old = '\t\thead := o.ListMark(focused) + " " + render.ListLabel(render.PadTo(r.label, relayFaultLabelW)+":", focused) + "  "'
new = '''\t\tmark := "  "
\t\tif focused {
\t\t\tmark = "\\u203a "
\t\t}
\t\thead := mark + " " + render.PadTo(r.label, relayFaultLabelW) + ":  "'''
assert old in s, "mV1"
p.write_text(s.replace(old, new, 1))
