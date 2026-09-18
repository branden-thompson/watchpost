# 0.16.0 P3. Queue the rotation's card and never settle: the card sits in the
# schedule and nothing ever asks for its words, so the station shows a full
# lineup and plays silence.
#
# THIS ONE SURVIVED ITS FIRST RUN and that is why it is here. The test held the
# step's effects and threw them away (`_ = fx`), so deleting the work changed no
# assertion. Third instance in one batch of a gate that watches the STATE and
# not the WORK.
import pathlib
p = pathlib.Path("platform/lineup/rotation.go"); s = p.read_text()
old = "\td.lineup = next\n\treturn d.settle()\n"
assert old in s, "m100"
p.write_text(s.replace(old, "\td.lineup = next\n\treturn d, nil\n"))
