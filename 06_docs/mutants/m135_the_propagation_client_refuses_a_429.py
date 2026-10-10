import pathlib
# 0.19.0 FR-4.4: a 429 is handed back as it came, for go-ionomaps to back off from.
# Killed by TestThePropagationClientNeverRetriesA429.
p = pathlib.Path("platform/httpx/plain.go"); s = p.read_text()
old = "\treturn Answer{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil"
assert old in s, "m135"
p.write_text(s.replace(old, "\tif resp.StatusCode == http.StatusTooManyRequests {\n\t\treturn Answer{}, errPlainTooLarge\n\t}\n\treturn Answer{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil"))
