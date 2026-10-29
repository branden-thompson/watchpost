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
| 2026-10-22T18:14:25Z | bd50604e | bec6f413366c | verify | green | 2883 |
| 2026-10-22T21:47:33Z | a3e943a0 | 6669cadd4356 | verify | green | 2985 |
| 2026-10-22T21:51:56Z | 1beb2ed6 | b10d473a7425 | verify-docs | green | 178 |
| 2026-10-24T06:22:18Z | 31610af6 | 9ade3fbb7e01 | verify | green | 2327 |
| 2026-10-24T06:25:44Z | ec72037c | d3cd88797113 | verify-docs | green | 154 |
| 2026-10-24T13:57:35Z | 95b81078 | ac5de17ff57b | verify-docs | green | 144 |
| 2026-10-24T14:26:05Z | 6ad1c098 | 6b8275879424 | verify-docs | green | 143 |
| 2026-10-24T16:25:09Z | 210b8f10 | 9617e8f33b36 | verify-docs | green | 143 |
| 2026-10-24T16:37:35Z | 0a8d578b | fbf48de55822 | verify-docs | green | 141 |
| 2026-10-24T16:40:30Z | b0b4b35e | 2c791e9315ba | verify-docs | green | 143 |
| 2026-10-24T18:53:51Z | 4b4f1c58 | 44d73c06bcfb | verify | FAILED (exit 2) | 0 |
| 2026-10-24T20:02:22Z | 4b4f1c58 | abe9c1488178 | verify | green | 2590 |
| 2026-10-24T22:08:26Z | 2449ae00 | 10ca7110527c | verify | green | 2595 |
| 2026-10-25T00:32:49Z | dd74f0a9 | 2888fca2924a | verify | green | 2592 |
| 2026-10-25T00:54:54Z | 85f7046e | 803979dbd431 | verify | FAILED (exit 2) | 357 |
| 2026-10-25T01:36:40Z | 85f7046e | 803979dbd431 | verify | green | 2458 |
| 2026-10-25T02:32:39Z | 59ed3997 | 742d0700d802 | verify | FAILED (exit 2) | 2423 |
| 2026-10-25T03:51:45Z | 59ed3997 | db1b77718245 | verify | green | 2651 |
| 2026-10-25T04:48:38Z | 8092d107 | b2340162ed2d | verify | FAILED (exit 2) | 370 |
| 2026-10-25T05:29:20Z | 8092d107 | b2340162ed2d | verify | green | 2381 |
| 2026-10-25T13:43:08Z | a444b000 | 8a657ba29ccf | verify | green | 2347 |
| 2026-10-25T21:20:19Z | 5b71df2c | 31dd227a2e83 | verify-docs | green | 144 |
| 2026-10-25T23:47:08Z | 27049f3b | 18af1d620483 | verify | FAILED (exit 2) | 0 |
| 2026-10-25T23:51:54Z | 27049f3b | c0c6bbf3c6a3 | verify | FAILED (exit 2) | 240 |
| 2026-10-26T00:37:25Z | 27049f3b | 43d2152cf292 | verify | green | 2680 |
| 2026-10-26T02:04:11Z | 03ad4e97 | cfa0b48dfe68 | verify-docs | FAILED (exit 2) | 143 |
| 2026-10-26T02:07:01Z | 03ad4e97 | e9f9aebf5a2c | verify-docs | green | 143 |
| 2026-10-26T03:57:17Z | fb08a256 | 74270bf92a0e | verify | green | 2642 |
| 2026-10-26T04:39:05Z | d63f499e | 2c832a551a66 | verify-docs | green | 146 |
| 2026-10-26T06:25:33Z | fe828be1 | cd8f059e1b64 | verify | FAILED (exit 2) | 237 |
| 2026-10-26T07:10:34Z | fe828be1 | 6c8edab4c326 | verify | green | 2644 |
| 2026-10-26T13:58:57Z | af661bf5 | 9925a3a8e898 | verify-docs | green | 145 |
| 2026-10-26T16:01:37Z | 480fcd75 | ef5120c4cb63 | verify | green | 2600 |
| 2026-10-27T00:15:27Z | 2ec275b5 | 6d3cb46c8f32 | verify-docs | green | 167 |
| 2026-10-27T02:26:45Z | 322805b4 | b41bfaade176 | verify | green | 2687 |
| 2026-10-27T03:45:45Z | 9d9dd651 | 479383403b8a | verify | green | 2780 |
| 2026-10-27T05:00:24Z | b16e5411 | 53184d90cb5d | verify-docs | green | 218 |
| 2026-10-27T13:42:47Z | 9a45334c | 93b1e9468dbd | verify-docs | green | 147 |
| 2026-10-27T15:07:06Z | a70d0bbd | 42c40d75958f | verify | green | 2606 |
| 2026-10-27T15:11:17Z | c8a99e48 | 53f4e4976d47 | verify-docs | green | 150 |
| 2026-10-27T15:51:17Z | 1275b161 | f75482295472 | verify-docs | green | 146 |
| 2026-10-27T16:43:24Z | 49c65df4 | b62e544c5057 | verify-docs | green | 143 |
| 2026-10-27T18:35:36Z | e49260d0 | d9137c8d654b | verify | green | 2818 |
| 2026-10-27T18:46:23Z | 6b85f268 | ceb223e4356e | verify | FAILED (exit 2) | 239 |
| 2026-10-27T19:34:01Z | 6b85f268 | 8d58c160aa95 | verify | green | 2827 |
| 2026-10-27T19:37:39Z | 61a326a4 | 926620d49548 | verify-docs | green | 148 |
| 2026-10-27T20:35:45Z | 32324c39 | f4c2b0efee1a | verify-docs | green | 149 |
| 2026-10-27T22:25:55Z | e3044ba9 | ac8c3a6aa69c | verify | green | 2789 |
| 2026-10-27T22:30:23Z | fdb4d35b | e88c17e4d37f | verify-docs | green | 148 |
| 2026-10-28T01:23:35Z | e1c491ac | b55f12002419 | verify | green | 3040 |
| 2026-10-28T01:36:07Z | b434e021 | 2953d6459900 | verify-docs | green | 146 |
| 2026-10-28T03:31:57Z | b9ca4c61 | 326a91f39ef5 | verify-docs | green | 146 |
| 2026-10-28T14:13:12Z | 306fe5ce | 12e8425829a6 | verify | green | 2710 |
| 2026-10-28T14:16:05Z | a2bf3137 | 859f961ac334 | verify-docs | green | 153 |
| 2026-10-28T22:02:32Z | 11a21a84 | 6affbb4c7224 | verify-docs | green | 144 |
| 2026-10-29T16:06:46Z | 8a8c1859 | d2d102692180 | verify-docs | green | 150 |
| 2026-10-29T18:03:28Z | 6368617f | fc90727c0e55 | verify-docs | green | 145 |
| 2026-10-29T18:15:41Z | 5fd5f444 | 6cee413f6232 | verify-docs | green | 147 |
| 2026-10-29T19:10:18Z | ba84f94f | 6cd0971fa3f7 | verify | green | 1906 |
| 2026-10-29T19:13:27Z | bca1705e | 217986899d2f | verify-docs | green | 150 |
| 2026-10-29T21:01:41Z | 243982c4 | 6b72f4d58b36 | verify | green | 1885 |
| 2026-10-29T21:20:14Z | 1eb6c413 | 4185c5ebf3c1 | verify-docs | green | 142 |
