# Answer key — T4: add a new spoken phrase

> **0.19.5 M5 answer key (D-24), hidden from the blind reviewer.** Built on a scratch copy of the
> tree at batch 41 (`e25a9ea9`) by doing the task and leaving out one site at a time. Batch 42 acted on
> it: the doc findings below are fixed, and the guards it added are named in the build log
> (`TestEveryFetchKindHasADomain`, `TestAllAsksEveryFamily`, `TestEveryChipWordIsRead`,
> `TestReduceMotionsChipsNameIt`). The site lists stand as the key; a row's "NO GUARD" is as measured
> at batch 41. Scratch paths are written as `<scratch>/`.


Scratch copy: `<scratch>/m5key/t4-phrase` (git `base` = `84e304d`).
Every claim below was verified in the code or by running a test in that copy, on 2026-11-08.

## The change

**A closing line, read after the sign-off at the end of every synthesized broadcast cycle:**
"Stay safe, and stay tuned to Watchpost Weather Radio for the next update."

- Script part `weather-radio/closing` (no data).
- Spoken by a new segment, key `closing`, role `cast.Station`. It is appended **after** the tail
  (`tail:{{voice}}`) in `synth.Composer.Compose`. It is left out when the phrase renders empty, for example
  under a blank override.
- **Why after the tail, in the Station role.** The tail is the one `SelfIntro` segment
  (`compose.go`, `Segment.SelfIntro`; `source.go:renderAhead` skips the hand-over into it). A Station
  line placed *before* the tail would follow the last report's role (seismic, fire…). When those roles
  have different voices, the Source would speak a hand-over into the new line ("This is Eddie, taking
  over for Karen."). The tail would then introduce Eddie again: the double introduction that
  `SelfIntro` exists to prevent. Placing the line after the tail in the same role means the voice
  does not change, so no hand-over is spoken.

### The two shapes of "a new phrase" (a junior may choose either)

| Shape | What it costs | Verified |
|---|---|---|
| **Re-word an existing part** (e.g. edit `weather-radio/tail.txt`) | the `.txt` only, plus any test that pins the old wording (`TestComposeReadsLikeNWR` pins the tail's full text at `synth_test.go:47`). No golden moves: `cycle.golden` holds keys/roles/pauses, not text (`golden_test.go:26`). | by reading |
| **Add a new part** (this key) | a `.txt` **and a Go call site that asks for it by (report, part)**: a new segment, the golden, and the tests that pin the segment list. | by test runs below |

Adding only the `.txt` file for a new part, against the base code, leaves **every test green and
nothing is spoken**. I ran it: base `compose.go` and base tests plus the new `closing.txt` gave
`ok domains/radio/script`, `ok domains/radio/synth`. The repo already has a precedent:
`scripts/transition/*.txt` exist and have no caller, and README.md:546 says they are "not yet on air".

## 1–2. Every site, and the guard that fails when it is omitted

Method: I finished the change, copied every touched file aside, and then omitted one site at a time
(restoring by `cp` and checking with `cmp`). After each omission I ran
`go test ./domains/radio/script ./domains/radio/synth`.

| # | File | Symbol | The change | Guard when omitted (verified) |
|---|---|---|---|---|
| 1 | `domains/radio/script/scripts/weather-radio/closing.txt` (new) | part `weather-radio/closing` | First line `{{/* weather-radio/closing: the last line of a cycle, after the sign-off. No data. */}}`, then the sentence. | With the tests updated (rows 6–8): `TestTheAppsPartsExist` "missing weather-radio/closing"; `TestComposeWithZeroReportsIsTodaysBroadcast` "composed 6 segments, want 7"; `TestComposedCycleGolden`; `TestComposeReadsLikeNWR` "missing \"Stay safe…\""; `TestReportsAreSeparatedByAir`; `TestTheClosingFollowsTheSignOffInItsVoice`. **Against base tests: NO GUARD.** A missing part is silence by design (`script.Library.Say`). |
| 2 | `domains/radio/synth/compose.go` | `Composer.Compose` (after the tail append) | `if closing := c.Closing(); closing != "" { segs = append(segs, Segment{Key: "closing", Text: closing, Role: cast.Station}) }` | `TestComposeWithZeroReportsIsTodaysBroadcast` (6 vs 7), `TestComposedCycleGolden`, `TestComposeReadsLikeNWR` (missing wording), `TestReportsAreSeparatedByAir`, `TestTheClosingFollowsTheSignOffInItsVoice`. `domains/radio/script` stays green. **With only the file + row 7 and no call site, the script package passes and the line is dead.** |
| 2a | same | `Composer.Closing` (new method) | `return c.say("weather-radio", "closing", nil)` | the compiler (row 2 calls it). |
| 2b | same | the segment's `Role: cast.Station` | Without it the role is the zero value `cast.All` ("All reads"). | `TestComposedCycleGolden` (`role=voice` vs `role=station`), `TestComposeReadsLikeNWR` "segment 6 (closing) has role All reads, want Station", `TestTheClosingFollowsTheSignOffInItsVoice`. |
| 2c | same | placement after the tail | (wrong variant: the block placed before the tail) | `TestComposeWithZeroReportsIsTodaysBroadcast` (segment 5/6 key order), `TestComposedCycleGolden`, `TestComposeReadsLikeNWR`, `TestReportsAreSeparatedByAir`, `TestTheClosingFollowsTheSignOffInItsVoice`. **These fail only because the fixtures pin "tail then closing".** The real hazard (a double introduction across a voice change) has **NO GUARD** at the Compose level. Re-capturing the golden accepts either order. |
| 2d | same | the `closing != ""` guard | (omitted: append unconditionally) | `TestTheClosingFollowsTheSignOffInItsVoice` only ("an empty closing override still composed a segment"). **Without that new test (row 9): NO GUARD.** Note the tail itself is appended even when empty. The guard copies `app/rotation_script.go:scriptFromSegments`, which drops blank segments on the card path. |
| 3 | same | doc comments of `Compose` ("…then the tail naming the correspondent voice and the closing line") and `Composer` ("…lead, conditions, alerts, tail and closing") | keeps the comments current (comment rule 1, AP-HIST-01's intent) | **NO GUARD.** Reverted, tests and `go vet` stay green. `tools/authoring` "cannot see a comment that is merely wrong" (code-standards.md:304-305). |
| 4 | `domains/radio/synth/testdata/cycle.golden` | `TestComposedCycleGolden` | one new last line `closing … role=station … pause=0s`. Re-capture: `go test ./domains/radio/synth -run TestComposedCycleGolden -args -update-golden` (flag after the path; `-update-golden` without `-args` also works after the path). | `TestComposedCycleGolden` "the composed cycle changed:" (want/got differ by the `closing` line). |
| 5 | `README.md:542` | "What the correspondent says" | `weather-radio/` "(the broadcast's lead, conditions, alerts, sign-off and closing line)" | **NO GUARD.** The list is hand-written. `go test ./cmd/watchpost` passed with the edit, and nothing reads that sentence. |
| 6 | `domains/radio/synth/composer_test.go` | `TestComposeWithZeroReportsIsTodaysBroadcast` | `wantKeys` gains `"closing"`; "segment before the sign-off" becomes `zero[len(zero)-3]` | It is a guard itself. Base version: "zero Reports composed 7 segments, want 6". With only the index edit reverted: "the segment before the sign-off pauses 0s, want the tail pause 1s (UAT 115)". |
| 7 | `domains/radio/script/script_test.go` | `TestTheAppsPartsExist` | the list gains `"weather-radio/closing"` | **NO GUARD.** Reverted, both packages pass. It is a hand-written list ("the contract between the library and its callers"). Nothing derives it from the call sites. |
| 8 | `domains/radio/synth/synth_test.go` | `TestComposeReadsLikeNWR` (tail at `len-2`, closing at `len-1`) and `TestReportsAreSeparatedByAir` (tail at `len-2`) | Both pinned the tail as the **last** segment (base `synth_test.go:52` and `:784`). | Base version: `TestComposeReadsLikeNWR` "segment keys carry issuance identity: [alert:a1:0 ZFP:p1:0 closing]", `TestReportsAreSeparatedByAir` "no fire report: the forecast pauses 1 s then the tail: {Key:ZFP:p1:0 …}". That message is misleading: it prints the ZFP segment, but the failing condition is the tail's position. |
| 8a | same | `TestComposeReadsLikeNWR` | added a wording pin for the closing sentence, and a 7th role `cast.Station` | **NO GUARD** for either addition (removed one at a time: green). They are optional strengthening. The role loop only checks as many entries as the list has. |
| 9 | `domains/radio/synth/composer_test.go` | `TestTheClosingFollowsTheSignOffInItsVoice` (new) | asserts: the cycle ends tail then closing; the closing is Station, not SelfIntro, and has text; an override `weather-radio/closing.txt` of only a comment leaves it out | Removing it: **NO GUARD** for 2d (the empty-override behaviour). Placement and role stay covered by rows 4, 6 and 8. |

**Site count:** 9 numbered sites (2 data/fixture files, 1 production file with 4 sub-parts, 1 doc,
4 test files/tests). **NO GUARD rows:** 1 (against base tests), 2c's real hazard, 2d (without row 9),
3, 5, 7, 8a, 9.

### Conditional rules (they apply depending on the phrase chosen; each verified)

| If the phrase… | Then | Guard (verified) |
|---|---|---|
| takes data (`{{.Field}}`) | the field must be in `TestBuiltinScriptsFollowTheConvention`'s shared `data` map (`script_test.go:20-30`). The template runs with `missingkey=error`. | `{{.Cadence}}`: `TestBuiltinScriptsFollowTheConvention` "map has no entry for key \"Cadence\"". **Gotcha:** the map is the union of every script's fields, so a name already used elsewhere (`{{.Next}}`, from marine) **passes** even when your caller never passes it. At run time `Say` then returns "" in silence. |
| has no `{{/* … */}}` first-line comment | the convention says each file's first line documents its data (`script.go:11-12`, README.md:540) | **NO GUARD** (tried: green) |
| starts a new report **folder** | the two hard-coded `Reports()` lists | `TestBuiltinScriptsFollowTheConvention` (`script_test.go:44`) and `TestOverrideFileWinsPhraseByPhrase` (`:119`) both fail |
| is a new `global/` part | the `global parts` assertion `head,tail` (`script_test.go:47`) | by reading |
| renders empty, contains `{{`, or ends in a newline; or its name is not `^[a-z][a-z0-9-]*$` | | `TestBuiltinScriptsFollowTheConvention` (by reading `script_test.go:32-41`) |
| says a quantity | word it through `domains/radio/synth/spoken_units.go` (the one owner of spoken units; `TestTheConditionsSayTheListenersUnits`). Spell units out ("miles per hour"). | by reading |
| names a place or state | pass it through `ExpandStates` as `Compose` does for `.Location` (`compose.go:251-253` in the changed file) | by reading |
| contains abbreviations / agency codes | nothing to do in the script: `Pronounce` runs on every rendered segment (`source.go:625`, `Source.say`), reading `domains/radio/pronounce/rules/*.txt` (e.g. `NOAA` → "Noah"). A new spelling is a row in `rules/words.txt`. Pronounce is **not** applied by `script.Library.Say`, which only strips terminal escapes (`plaintext.Text`). | by reading |
| comes from provider text | `Say` strips terminal escapes (NFR-6); template data is never re-parsed, so `{{` in provider prose is prose (`script.go:91`) | by reading |

### Narrator role / voice

- The role is set on the `Segment` (`cast.Station`, "Station" in Settings, under Standard; `cast.go:73`).
  An untagged segment is read by the root voice (`cast.All`).
- **The role survives only on the Observer's synth stream** (`app/radio.go:startSynth` →
  `synth.NewSource` → `d.segments` → `Compose`). On the Broadcaster's card path,
  `app/rotation_script.go:scriptFromSegments` keeps only `Text`. `segmentsFromScript` rebuilds the
  segments with `Role` unset, so the main track reads every line in the root voice (its comment
  cites G-7/F-92). The closing line is spoken on both paths. Its role matters only on the first.
- The card's manifest (`contentsFromSegments`) skips segments without `Source`, so the closing adds
  no card line. That is right: it is connective tissue, not a source.

### Gates not run here (forbidden or heavy in this exercise)

- `make verify`, `make lint-authoring` (`go run ./tools/authoring -base auto`), `make p10`.
  `domains/radio/synth` has a ratified P10-05 FLOOR of 0.87 over changed functions
  (`06_docs/p10-ledger.md:98`). This change adds two check-free changed functions (`Compose`, `Closing`).
  Whether the package stays above the floor needs `make p10`, which needs the external harness.
  P10-04: `Compose` is 49 code lines, under the 60 limit, with complexity 12 by hand count.

## 3. Doc accuracy (for the doc steps used)

| Doc:line | Says | Reality (code) |
|---|---|---|
| README.md:623-634 ("Start here") | routes to architecture, where-things-happen, extending, code-standards | Correct files, but **none of the four has a "spoken phrase" route.** The only place that explains the script tree is README.md:539-552, outside "Start here". |
| docs/extending.md:10-11 | extend "by touching **one folder plus one wiring line**" | For a new part: one file + a call site + the golden + 3 pinned tests (rows 4, 6, 8). |
| docs/extending.md:89-106, architecture.md:104 | the extension list: provider, radio control, report kind, map layer, Setting | **Step left out:** there is no "add a spoken phrase" walkthrough. The nearest is Walkthrough 4 step 5. |
| docs/extending.md:103 | "**Wired by folder name, not by code** — a missing phrase is simply not spoken, and no gate reads this tree." | (a) **Wrong for a new part:** a part is spoken only when Go asks for it by (report, part) (`Library.Text/Say`; `Composer.say` in `compose.go`). The folder name selects a file, but code does the wiring. (b) **"no gate reads this tree" is false:** `TestBuiltinScriptsFollowTheConvention` walks and renders every built-in file and pins the report-folder list. `TestTheAppsPartsExist` checks a hand-written part list. What is true: no gate checks that a file **is spoken**. (c) "a missing phrase is simply not spoken": true (verified). |
| docs/extending.md:120 | "The script folder is wired by name, not by a registry." | Same as above: right that there is no registry, but it hides the Go call site. |
| docs/extending.md:185-186 | "Re-capture … `go test ./modes/tty -update-golden`. Before the path, `go test` refuses a flag it does not know." | After the path: correct (verified for synth, with and without `-args`). Before the path, `go test -update-golden ./domains/radio/synth` from the repo root printed `no Go files in <root>` / `FAIL . [setup failed]`. It took the package path as the flag's value and tested `.`. It does fail, but not by "refusing the flag", and the message never names the flag. |
| docs/extending.md:187 | `-update-golden` for `domains/radio/synth` | correct (`golden_test.go:16`) |
| docs/where-things-happen.md:17 | "the voice-only pass `…/normalize.go:Pronounce` (every Say)" | Ambiguous: "Say" here means `Voice.Say` (e.g. `source.go:625`), **not** `script.Library.Say`, which does not pronounce. A junior may look for it in the script package. |
| docs/where-things-happen.md:18 | "A report's wording is chosen by file name in `script.go:Text` … the event read … `app/severe_read.go:eventScript`, the takeover … `app/burst_words.go:breakingLine`" | Correct symbols, but **left out** where the broadcast cycle asks for its parts: `domains/radio/synth/compose.go:Compose` (and `LeadParts`, `Tail`, `conditions`, `Sample`, `HandoffLine`). That is the file a cycle phrase needs. |
| docs/where-things-happen.md:59, :123 | hand-over wording `compose.go:HandoffLine`; "segment … `domains/radio/synth/compose.go`" | correct |
| docs/architecture.md:26, :80-95 | the diagram names `radio/… synth · cast · script · pronounce`; the package table has no `domains/radio` row | Not wrong, but there is no route from the package table to `domains/radio/script` or `synth`. |
| 06_docs/code-standards.md | comment rules, AP-*, P10 | Nothing about scripts, the golden or spoken wording (no wrong names found). |
| `domains/radio/script/script.go:15-17` (package comment, the code's own doc) | "Adding a phrase is adding a file; adding a report is adding a folder." | **False for a new part or report:** proven above. With the file alone everything stays green and nothing is spoken (precedent: `transition/`, README.md:546). True only for re-wording or overriding an existing part. |
| `script.go:6` | "the tree IS the wiring" | same disagreement |
| `script.go:97-98` (`Say`) | "the built-in tree is pinned by this package's tests, so that only happens with a broken override" | Not only: a built-in phrase using a field its caller does not pass is silenced too. The convention test's union map can hide that (the `.Next` case above). |
| `script.go:228-229` (`BuiltinFiles`) | "(the convention's test and the README's listing)" | The README listing (README.md:542-547) is hand-written. Nothing generates it from `BuiltinFiles`. |
| `synth/golden_test.go:26-28` | Text is deliberately not in the golden | correct (verified: re-wording does not move the golden; a new segment does) |

## 4. IDs met in the docs followed, resolved through 06_docs/ids.md

Docs scanned: README.md:539-552 and :623-641, docs/extending.md, docs/where-things-happen.md,
docs/architecture.md, 06_docs/code-standards.md. The ID shape and families come from
`tools/ids/idset/idset.go`.

**Do not resolve:**

| ID | Where cited | Why |
|---|---|---|
| `F-111` (cited) | docs/extending.md:94 | No row in ids.md. It appears only inside cells of `0.16.0-broadcaster-ui/08-reports/red-team-build.md:87` and `p10-ledger.md:143`, never as a defining first cell. |
| `0.15.0 FR-7.1` | docs/where-things-happen.md:68 | ids.md has FR-7.1 only for 0.16.0 ("Broadcaster uses the platform breakpoint vocabulary"), an unrelated row. No 0.15.0 FR-7.1. |
| `MVS-D-3`, `MVS-D-15` | where-things-happen.md:54, :60 | The `MVS-D-n` prefix is not indexed. The bare `D-3`/`D-15` multi-voice rows in ids.md are red-team findings ("Important"/"Minor"), not these rulings. |
| `MVS-D-80` | code-standards.md:225, where-things-happen.md:147 | Not indexed. There is no D-80 multi-voice row. |
| `R3-A-01` | where-things-happen.md:34 | Not an ID family. Defined in `severe-alerts-modals/08-reports/red-team-build.md:21`. |
| `AP-HIST-01`, `AP-CITE-01`, `AP-CAPS-01`, `AP-DEAD-01`, `AP-OK-01`, `AP-SHELL-01`, `SN-02`, `SN-03`, `P10-00`…`P10-10` | code-standards.md (and AP-CITE-01 at where-things-happen.md:150) | Not ID families, so not in ids.md **by design**. Each is defined in code-standards.md itself (§ headings :57, :109, :124, :171, :192, :210, :236, :267; P10 table :326-338). |

**Resolve** (a row exists; qualified ones match their release): `0.19.0 D-142`, `0.19.5 A-2`,
`0.19.5 D-4/D-10/D-11/D-12/D-13/D-27/D-30/D-33/D-34`, `D-10`, `D-15`, `D-30`, `D-56`, `D-61`, `D-124`,
`D-160`, `D-292`, `DC-14`, `DR-21`, `F-21`, `FR-1.1`, `FR-1.2`, `FR-3.8`, `FR-9`, `FR-9.4`, `JM-3`.
The bare ones (`D-10`, `D-15`, `D-30`, `D-56`, `D-61`, `D-124`, `D-160`, `FR-1.1`, `FR-1.2`, `FR-9`)
each match 2–11 rows across releases. They resolve, but ambiguously, which is the problem AP-CITE-01
names.

## Final run with the complete change in place

- `go test -race -count=1 ./domains/radio/script ./domains/radio/synth`: **ok / ok** (the touched packages).
- `go test ./app`: **ok** (140.7 s). `go test ./cmd/watchpost`: **ok** (74.4 s; the doc-symbol tests, run with the README edit). `go vet ./domains/radio/script ./domains/radio/synth`: clean.

Final diff: `README.md`, `domains/radio/script/script_test.go`, `domains/radio/synth/compose.go`,
`domains/radio/synth/composer_test.go`, `domains/radio/synth/synth_test.go`,
`domains/radio/synth/testdata/cycle.golden`, and the new
`domains/radio/script/scripts/weather-radio/closing.txt`.
