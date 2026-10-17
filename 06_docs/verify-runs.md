# Verify runs

`tools/runlog` appends one line here per `make verify` or `make verify-docs` run; a line is
written by the tool, never by hand. **Tree** is the git tree the run tested, this file left out, so a line
says exactly what passed.

| When (UTC) | HEAD | Tree | Lane | Result | Seconds |
|---|---|---|---|---|---|
| 2026-10-17T14:47:26Z | 4a3ae688 | 1852b5a4a61d | verify | FAILED (exit 2) | 226 |
| 2026-10-17T15:24:28Z | 4a3ae688 | 25ea7e4d96f5 | verify | green | 2175 |
