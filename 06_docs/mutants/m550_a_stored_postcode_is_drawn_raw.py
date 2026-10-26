import pathlib
# 0.19.0 R2I-2: a stored postcode is drawn as it was saved, an escape sequence in it live.
# Killed by TestAStoredPostcodeIsDrawnPlain.
p = pathlib.Path('modes/tty/setup_form.go'); s = p.read_text()
old = '\treturn render.Plain(ref.Label) + " (" + render.Plain(ref.Zip) + ")"\n'
assert s.count(old) == 1, 'm550'
p.write_text(s.replace(old, '\treturn render.Plain(ref.Label) + " (" + ref.Zip + ")"\n', 1))
