# 0.16.0 P3, FR-2.5. Make a rotation card's id vary per call: the lineup's
# identity check then admits the same location twice and it is read twice.
#
# THE ID IS THE WHOLE NO-DOUBLE-SPEAK MECHANISM at the schedule level. The
# duplicate test would stay green under any implementation that remembered refs
# in a set; what makes a second need harmless is that the id is a pure function
# of the ref.
import pathlib
p = pathlib.Path("platform/lineup/rotation.go"); s = p.read_text()
old = '\treturn "read:" + ref\n'
assert old in s, "m101"
p.write_text(s.replace(old, '\treadSeq++\n\treturn "read:" + ref + string(rune(\'a\'+readSeq%3))\n')
             .replace("func ReadID(ref string) string {", "var readSeq int\n\nfunc ReadID(ref string) string {"))
