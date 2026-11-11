import pathlib
# 0.19.5 Q4.2 (PF-7, D-22): the masthead memo keys on the theme, which every tint reads.
# Killed by TestTheMastheadKeyCoversEverythingTheMastheadShows.
p = pathlib.Path('modes/tty/masthead.go'); s = p.read_text()
old = '\t\ttheme: render.ThemeGeneration(), colour: render.ColorOn()}'
assert s.count(old) == 1, 'm746'
p.write_text(s.replace(old, '\t\tcolour: render.ColorOn()}', 1))
