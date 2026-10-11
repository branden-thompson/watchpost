import pathlib
# FR-4.7: a reply past the propagation client's cap is refused, as the data client's 413.
# Killed by TestThePropagationClientIsHardened.
p = pathlib.Path('platform/httpx/plain.go'); s = p.read_text()
old = '\tif !whole {\n\t\treturn Answer{}, &StatusError{URL: safe, Status: http.StatusRequestEntityTooLarge}\n\t}\n'
assert old in s, 'm177'
p.write_text(s.replace(old, '\tif !whole && false {\n\t\treturn Answer{}, &StatusError{URL: safe, Status: http.StatusRequestEntityTooLarge}\n\t}\n'))
