import pathlib
# READ MANIFEST is centred over the lane's INTERIOR rather than over the cell the
# row is drawn in, so the caption sits three cells left of the middle of the table
# it captions (D-110).
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = """		centerText(bcManifestCaption, lane.lane),"""
assert old in s, "mAD4"
p.write_text(s.replace(old, """		centerText(bcManifestCaption, lane.inner()),""", 1))
