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
| 2026-10-17T19:37:21Z | a4989670 | 09b5a6b665f5 | verify | FAILED (exit 2) | 301 |
| 2026-10-17T19:47:39Z | a4989670 | db6db7a42027 | verify | FAILED (exit 2) | 539 |
| 2026-10-17T20:29:24Z | a4989670 | 0ef64003e32a | verify | green | 2342 |
| 2026-10-17T22:09:50Z | c5e56294 | 6380fa151680 | verify | green | 2397 |
| 2026-10-17T23:02:29Z | 84d74514 | ea1c08095f2a | verify | green | 2287 |
| 2026-10-17T23:30:39Z | 5691d77e | 2b8dadd59caf | verify | FAILED (exit 2) | 262 |
| 2026-10-18T00:11:02Z | 5691d77e | 6975eb630d58 | verify | green | 2311 |
| 2026-10-18T00:49:12Z | 438dc980 | a81f7091e04b | verify | FAILED (exit 2) | 251 |
| 2026-10-18T01:35:02Z | 438dc980 | e52ee7847728 | verify | green | 2403 |
| 2026-10-18T02:24:11Z | 9906c61c | d648454831a3 | verify | green | 2438 |
| 2026-10-18T02:27:51Z | f487782f | f152705a4c39 | verify-docs | green | 152 |
| 2026-10-18T02:34:04Z | 3d4fa87a | cab156e63010 | verify-docs | green | 248 |
| 2026-10-18T03:32:27Z | d6fba065 | 9d032e2d08c5 | verify-docs | green | 152 |
| 2026-10-18T04:28:48Z | e5459d28 | 989b3294a27d | verify | FAILED (exit 2) | 258 |
| 2026-10-18T05:10:33Z | e5459d28 | 989b3294a27d | verify | green | 2321 |
| 2026-10-18T06:00:06Z | 573b5bcd | 03c10a46c1c4 | verify | green | 2333 |
| 2026-10-18T06:45:30Z | 997f8471 | 0b3625e4ac0b | verify | green | 2357 |
| 2026-10-18T06:51:16Z | d56225bf | 5ae0e125d256 | verify-docs | green | 152 |
| 2026-10-18T08:49:45Z | 1b3223a7 | 237e90e01164 | verify | green | 2338 |
| 2026-10-18T08:52:51Z | 181aaf31 | 916a3742a632 | verify-docs | green | 152 |
| 2026-10-18T08:57:26Z | 181aaf31 | 916a3742a632 | verify-docs | FAILED (exit 2) | 263 |
| 2026-10-18T09:00:22Z | 181aaf31 | 916a3742a632 | verify-docs | green | 149 |
| 2026-10-18T13:42:06Z | 559f22db | 80c0f8e15373 | verify-docs | green | 157 |
| 2026-10-18T13:56:43Z | 30191458 | 9231f6599577 | verify-docs | green | 152 |
| 2026-10-18T16:28:53Z | a554c982 | 92d4533c61a5 | verify | FAILED (exit 2) | 2465 |
| 2026-10-18T17:28:07Z | a554c982 | 31f3bd53e0b4 | verify | green | 2678 |
| 2026-10-18T18:18:54Z | 1e6c60b2 | ae49f5012c55 | verify | green | 2709 |
| 2026-10-18T18:47:59Z | a9401314 | bda2d1056921 | verify | FAILED (exit 2) | 2 |
| 2026-10-18T19:35:28Z | a9401314 | bf712f12ceee | verify | green | 2711 |
| 2026-10-18T21:32:11Z | e889e72a | 41412436bc8e | verify-docs | green | 150 |
| 2026-10-18T21:58:13Z | 40c55306 | 91df9b7e292b | verify-docs | FAILED (exit 2) | 0 |
| 2026-10-18T22:44:32Z | 40c55306 | f9278ba257db | verify | green | 2604 |
| 2026-10-18T23:28:00Z | 9eedc35d | d19434492057 | verify-docs | FAILED (exit 2) | 146 |
| 2026-10-18T23:30:56Z | 9eedc35d | 6fab22f2ebbc | verify-docs | green | 159 |
| 2026-10-19T06:30:01Z | 2ecdaa36 | 4d6eea2ac118 | verify | FAILED (exit 2) | 366 |
| 2026-10-19T14:44:11Z | 2ecdaa36 | af3106ad1b62 | verify | green | 2598 |
| 2026-10-19T18:01:28Z | 015fe9b4 | 61a5d07658bd | verify | green | 2523 |
| 2026-10-19T20:20:04Z | 494bede6 | 9b81b72a24e0 | verify | green | 2981 |
| 2026-10-19T23:06:24Z | c72f7468 | 66de3835e1e1 | verify | green | 2900 |
| 2026-10-21T15:01:38Z | c5b63877 | 2ffcda6a334d | verify-docs | green | 155 |
| 2026-10-21T15:18:30Z | c6890b88 | 64c19511a2b7 | verify-docs | green | 150 |
