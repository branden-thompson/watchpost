# Order the Marine arm ahead of Warning: a Special Marine Warning would file as
# marine furniture instead of the warning it is.
import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
start = s.index('\tcase strings.Contains(product, "Marine"):')
end = s.index("\t\treturn TabMarine, true\n", start) + len("\t\treturn TabMarine, true\n")
arm = s[start:end]
s = s[:start] + s[end:]
anchor = '\tcase strings.Contains(product, "Warning"):'
assert anchor in s, "m39"
p.write_text(s.replace(anchor, arm + anchor, 1))
