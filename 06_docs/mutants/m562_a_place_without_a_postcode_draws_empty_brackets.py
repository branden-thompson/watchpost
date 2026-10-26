import pathlib
# 0.19.0 R2I-2: a place with no postcode is drawn with empty brackets after its name.
# Killed by TestAStoredPostcodeIsDrawnPlain.
p = pathlib.Path('modes/tty/setup_form.go'); s = p.read_text()
old = '\tif ref.Zip == "" {\n\t\treturn render.Plain(ref.Label)\n\t}\n'
assert s.count(old) == 1, 'm562'
p.write_text(s.replace(old, '', 1))
