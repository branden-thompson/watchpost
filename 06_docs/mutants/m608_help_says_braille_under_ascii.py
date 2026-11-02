import pathlib
# 0.19.3 F-191 (A-107 (3)): under --ascii Help does not say the picture is braille.
# Killed by TestTheMapHelpSaysThePictureIsBraille.
p = pathlib.Path('modes/tty/help_about.go'); s = p.read_text()
old = '\tif o.ASCII {\n\t\treturn notes\n\t}\n'
assert s.count(old) == 1, 'm608'
p.write_text(s.replace(old, '\tif o.ASCII && false {\n\t\treturn notes\n\t}\n', 1))
