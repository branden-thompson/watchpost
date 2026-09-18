import pathlib
# The fence goes back to believing whatever the arrival says about itself. In the
# old shape that was `a.Tracked`, set true unconditionally by arrivalsOf; the
# same bypass in the new shape is "it has a key, so let it in". Either way a
# zone-only alert admitted under Observer's scope is read out over a console
# whose transmitter is four hundred miles away, through a fence, with nothing
# measured (D-122).
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = "\t\treturn f.tracks(a.TrackedAs)"
assert old in s, "mAP1"
p.write_text(s.replace(old, '\t\treturn a.TrackedAs != ""', 1))
