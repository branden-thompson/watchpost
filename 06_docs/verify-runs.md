# Verify runs

`tools/runlog` appends one line here per `make verify` or `make verify-docs` run; a line is
written by the tool, never by hand. **Tree** is the git tree the run tested, this file left out, so a line
says exactly what passed.

| When (UTC) | HEAD | Tree | Lane | Result | Seconds |
|---|---|---|---|---|---|
| 2026-10-17T14:47:26Z | 4a3ae688 | 1852b5a4a61d | verify | FAILED (exit 2) | 226 |
| 2026-10-17T15:24:28Z | 4a3ae688 | 25ea7e4d96f5 | verify | green | 2175 |
| 2026-10-17T16:53:03Z | 3c45707b | 77eb2b4514f9 | verify | FAILED (exit 2) | 2 |
| 2026-10-17T16:58:16Z | 3c45707b | 2204c3409f8e | verify | FAILED (exit 2) | 274 |
| 2026-10-17T17:07:12Z | 3c45707b | 92358432c999 | verify | FAILED (exit 2) | 331 |
| 2026-10-17T17:47:23Z | 3c45707b | 9a16d6bfab70 | verify | green | 2335 |
