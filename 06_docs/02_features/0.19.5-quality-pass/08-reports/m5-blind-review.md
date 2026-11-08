# M5 — the blind junior review, and its grade (D-24)

**What this is.** M5 asks whether a junior can find their way. D-24 grades it this way:

- **The answer key.** The agent built it before REVIEW, by doing four tasks on scratch copies and
  leaving out one site at a time. It is in `m5-key/`.
- **The reviewer.** A blind reviewer with a junior's persona traced the same four tasks from README's
  "Start here", on a copy of the tree at batch 43 (`1792b87a`) with this release's folder deleted.
  It could read only:
  - the documents "Start here" links;
  - `06_docs/ids.md`;
  - the files and symbols those documents name, with `go doc`.

  It ran no repository-wide search.
- **The pass, per task:**
  - every site is reached from the docs, or its omission fails a guard that names it;
  - no sentence routes to a wrong or missing file or symbol;
  - every ID met resolves through `ids.md`.
- **A fail** is fixed before SHIP or ruled.

## The grade

| Task | Key's sites | Reached by the reviewer | Wrong routes | IDs | At batch 43 | Fixed in batch 44 |
|---|---|---|---|---|---|---|
| T1, a toggle Setting | 32 (27 code and test, 5 goldens) | all, plus 2 the change would need to show the Setting (the Router's mirror onto the console, the masthead's memo key) | none; a line count read two ways | "0.19.5 Q6.4": the prefix table sent a release-qualified Q to the wrong plan | **pass**, with notes | the three notes Walkthrough 6 lacked (surface is ruled, the Router's mirror, the memo key); the Q prefix row; the count says which span it is |
| T2, a keyless provider | 47 | every production and registry site; the rest fail a guard that names them | 3: seismic does not model the report line or the attribution case; `defaultTheme`'s file; the schema version policy | FR-3.8 resolved to "W1.13, W3.7" | **pass**, with fixes owed | all three routes; FR-3.8 now resolves to what it requires; the merge test's model and the provider host list named |
| T3, a console card line | 7 plus 3 sub-sites | 3 of 10. Not reached, and with no guard: the words (`reportTypeOf`, since `Describe` says nothing for an empty set), the width (`lane.inner()-len(bcCardInset)`), the glyph-set ellipsis, the goldens' test | 3: the required reading's two stale quotation sources; `ids.md` sending the card's 0.16.0 rulings to 0.18.0 and 0.19.0 rows | D-87, D-97, D-110, D-114 and D-119 wrong; F-84 and F-97 absent | **FAIL** | Walkthrough 8, add a line to a Broadcaster card; D-37 (0.16.0's untabled rulings indexed by mention, the 13 commit-only ones restated without the ID); the two quotation sources |
| T4, a spoken phrase | 9 | all 9 | none | all resolved | **pass**, with notes | `TestTheAppsPartsExist`'s file named; `composeFor` noted, since a cycle-end line also ends every console card |

**T3 failed and is fixed in batch 44.** D-24 says a fail is fixed before SHIP.

## The T3 re-trace, after batch 45

A second blind reviewer traced T3 alone, with the same persona and limits, on a copy of the tree after
batch 45.

**Result: pass.** Walkthrough 8 routed every site the key has, and every symbol it names exists:
- the row in `readBody`, with the fixed height left alone;
- the helper beside `cardPulledShort`;
- the width, derived to 31 cells at the floor;
- the glyph set's ellipsis;
- `TestConsoleGoldens` and its 12 goldens;
- the line's own tests, with the cases the key has;
- the allocation budget.

The one site it did not reach, `CHANGELOG.md`, has no guard, and Walkthrough 8 now has it as step 7.

**What it still found, and batch 46's answer:**
- Walkthrough 8 implied `reportTypeOf` gives "whole report" words for an empty set. It gives the
  slot's name, the running order's words. Step 2 now says so, and says other words are a choice
  to ask about.
- The width's description did not match its formula. The formula is the safe side, and step 3 now
  says why.
- `platform/report/report.go` said "there is no card in that state" of an empty set; every card the
  Director makes carries one. Restated.
- Closed follow-ups cited in code (F-84, F-97 and others) resolved nowhere. D-38: the closed rows
  are kept in `06_docs/follow-ups-closed.md`, and the follow-ups only ever mentioned are indexed by
  mention. All 81 F-IDs the code cites now resolve, which `TestEveryFollowUpTheCodeCitesIsIndexed`
  holds.
- `readBody`'s callers: one, the UP NEXT card. Step 1 says so.
- Mutant anchors: the intro says how to check them before moving code.
- Still open, as judgements the walkthrough says to ask about: the empty set's words; whether a
  fourth line is wanted beside the reference's three questions; a transition card's line; the
  reference mock, which is not in the tree.

**Found by the review and not owed by M5's criteria**, recorded so they are not lost:
- T1: no doc names where the masthead is drawn.
- T2:
  - whether a reading is a new block or a field on `Conditions`;
  - whether a provider's readings join the history store;
  - the Status window's feature for a new kind.

  Each is a judgement for the change's author, and the walkthrough says to ask where a ruling decides it.
- T3:
  - the "reference" and "mock" the console's comments cite are not in the tree;
  - whether a card line is drawing or what the card says is unruled. Walkthrough 8 says to ask.
- T4: a pause between the sign-off and a closing line is unruled.
- IDs outside the M5 inputs: F-7 and R-12d in the watchpost-cli design, which the schema note links.
- Batch labels such as B5, Q5 and UAT 12 are prefixes, listed in where-things-happen's prefix table,
  not IDs.

## The reviewer's report, condensed

The reviewer could not write its file and returned the report as text. Its summary table is kept as
it came. The four task sections are condensed by the agent, keeping every place, wrong route and
stuck point the reviewer named, without its line-by-line citations.

> **Per task:**
>
> | Task | Places | Wrong routes | Stuck points |
> |---|---|---|---|
> | T1 Setting | 21 (19 from Walkthrough 6, 2 by following code comments) | 0 real (1 minor drift) | 5 |
> | T2 Provider | 28 | 3 | 6 |
> | T3 Console line | 3 code places, plus 3 gates/tests I could not pin to a target | 3 | 4 |
> | T4 Spoken phrase | 9 | 0 | 6 |
>
> The repo copy has no `.git`. Both ids.md and where-things-happen.md say to resolve a bare ID's
> release from `git log`, so every multi-row bare ID below is a guess based on its subject.
>
> **T1 — an on/off "Compact header" Setting.** Walkthrough 6 is the best walkthrough in the set: every
> step named a real file and symbol, and almost every step named the check that fails. Places: the
> config field; `Config` and `UIPrefs`; `livePipelines.ttyConfig`; `setUIHook`; `uiChoices` and
> `uiChoicesOf`; `uiForSave`; `applyUISaved`; the row id (it renumbers every later row); its
> `setupTable` row (`rowPicker`, `picker: true`, not `rowToggle`); `pickerChipWords`; `uiLines` with
> `at`; `uiLabelW`; a `cycle…` ending in `uiTouched`; the `cycleFocused` case returning `.flashed`;
> `everySettingChanged`; both settings-kept fixtures; the five goldens; and, by following code
> comments, `Router.Update`'s mirror onto the console and the masthead's memo key. Wrong: none; the
> test names were all real but most of their files unnamed; code-standards' `setupTable` "101 lines"
> read as 103; `rowToggle`'s own comment says "←→ or space flip it", where the walkthrough's warning
> is the truth. Stuck: where the header is drawn; the row's surface is a HUM LEAD ruling; the
> console mirror and the memo key are not in the walkthrough; which file holds the new `cycle…`.
> IDs: "0.19.5 Q6.4" is not in ids.md, and the prefix table says Qn is the performance pass's.
>
> **T2 — a keyless pollen provider.** Walkthrough 2 is accurate and frank about where "nothing
> fails". 28 places, each with its sending line: the provider; `KindPollen`; `domainFor`
> (`TestEveryFetchKindHasADomain`); the tiers; `kindName` and the cadence golden; `reportFetch`'s
> kinds (nothing fails); `pollenProviders`; `providerFamilies`, `newProviderFamilies` and `all`
> (`TestAllAsksEveryFamily`); `providerEndpoints`; `providerSources`; `creditGroups`; the attribution
> case (nothing fails); `PollenState` and its fields; the assembler's map, init, store, delete and
> copy (only the merge test); `clonePollen`; `make schema`; the declsets; `RenderPlain` and its
> fixture and golden; `pollenRows` and `detailLines` (nothing fails but the section's test);
> `chipGrounds`; the chip token in three themes; `ChipGrounds` (nothing fails); a glyph token if one;
> the tests; P10-05's density. Wrong: "the same five steps" for seismic, which has no report line and
> no attribution case; `defaultTheme` is in theme.go, and "Monochrome overrides" is an entry of
> `builtinOverrides`; §10.3 never says whether `SchemaVersion` moves, and `make schema` overwrites the
> current rc file. Stuck: a new block or a `Conditions` field; whether a provider joins a host
> allow-list (FR-3.8 resolves to "W1.13, W3.7"); the cadence golden's "freshness argument (plan
> §0.3)"; the merge test's model; the history store; the Status feature.
>
> **T3 — a line on the UP NEXT card.** where-things-happen.md:75 names `upNextBox` → `readBody` →
> `cardLane`. Places: a row in `readBody` (it pads to `bcReadCardRows-1`, so a filler row takes it);
> the words (`report.Everything().Describe()` for an empty set, by following `go doc`); `bcReadCardRows`
> +1 only if a real new row is needed; the frame golden, though no doc names its test; `make
> alloc-budget`; `make mutant-anchors`. Wrong: ids.md resolves D-87, D-110, D-97, D-114 and D-119 to
> 0.18.0 and 0.19.0 rows, and F-84 and F-97 are absent; the required reading quotes `app/radio.go:875-876`
> and "`app/director.go`'s header", both moved; the required reading has nothing on cards. Stuck: the
> "v2 reference" and "v3 mock" cannot be reached; whether the Composer owns the line; which test pins
> the UP NEXT box; label width is unruled.
>
> **T4 — a closing line.** Walkthrough 7 is the most exact walkthrough. 9 places: `closing.txt`; the
> segment in `Compose` after the sign-off, role Station; `cycle.golden`; the three tests that pin the
> segment list; `TestTheAppsPartsExist` (its file not named); a new test; README's list. Wrong: none.
> Stuck: `weather-radio/` or `global/`; `composeFor` means the line ends every console card; FR-2's
> zero-value anchor; code that assumes the tail is last; the cache key for an overridable phrase; a
> pause before the closing line.
>
> **What this says about the docs.** Walkthroughs 6 and 7 are excellent. Walkthrough 2 is accurate,
> but its seismic example does not model two of its steps. The Broadcaster has no walkthrough, and
> its required reading covers audio and roles, not cards. ids.md is the weakest link: it cannot
> disambiguate without git history, it misses the 0.16.0 rows the console code cites, and several IDs
> resolve to subjects that say nothing. Test names are reliably real, but their files are mostly
> unnamed; that was the one recurring reason to search.
