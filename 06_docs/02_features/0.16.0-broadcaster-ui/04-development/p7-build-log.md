> **Reconstructed 2026-09-13**, from the commits, the tests and a re-run of the corpus.  See
> `p4-build-log.md` for why the record ran a month behind the code.

# P7 — instruments and gates

**Plan:** *"FR-11 — runs THROUGHOUT, closed here."*  It did run throughout; this records what it
closed with.

## The performance work, and the number that made it honest

**D-120 — the budget was measuring the path that does not cost.**  `bcFrameAllocs` had spent its whole
history measuring a console with **no pool and no snapshot**: no row ever joined its weather, no pool
table drew a location, and the expensive path was never in the number at all.

HUM LEAD, on being shown that: **re-base it.**  The figure went UP because the measurement got honest,
not because anything got slower — and the budget now **validates its own fixture** (`loadedJoins`)
before it reports, because a budget that cannot tell whether it measured the expensive path is a
budget reporting on nothing.

**Only half of D-120's optimisation paid, and the other half was RETRACTED.**  The day-cell skip saved
351 allocations a frame — two tables building five `DayCell`s per row that neither draws.  The
map-once change measured **alloc-neutral (9296 → 9296) and time-neutral (503 µs vs 519 µs,
overlapping)**, and the claim was withdrawn in the code comment and to the HUM LEAD.

**AND ONE MEASUREMENT IN THAT ROUND WAS VOID.**  A `git stash` round-trip mid-measurement reverted the
optimised files, so the same tree was benchmarked twice and 5% of noise was read as signal.  Caught
only by validating the fixture.  It is recorded because the lesson is not "be careful with stash" — it
is that **a measurement whose instrument was not checked is not evidence**, and this one looked
exactly like a result.

## D-123 — the console memoises its two tables

Measured on the loaded fixture before anything was changed, which is the whole method:

| | before | after (hit) | after (miss) |
|---|---|---|---|
| time | 537 µs | **197 µs** | 508 µs |
| bytes | 421 KB | **107 KB** | 420 KB |
| allocations | 9298 | **530** | 9298 |

**The prize was sized before it was built for**: a throwaway probe put the two spans at 320 µs and
8796 allocs — 60% of the frame's time and 95% of its allocations.  **Nothing came off the miss path**,
which is the claim that matters when a cache is added.

**What makes a memo safe is not the key, it is the guard.**  Every one of the twelve key fields was
deleted and the completeness guard watched to go red.  **Six were not caught the first time**, and
every one was a real hole — `used` and `theme` cannot be reached by perturbing the model at all;
`lineupGen`/`areaGen` stand for values the walk cannot perturb; `recent` is a pointer; `fireBoldMW`
reached nothing because the fixture had no fire; `shimmer` needs something LOADING before the frame is
an input.

**The engine is generic now rather than copied**, and it perturbs FLOATS, which it never did — it
listed the kinds it handled and fell through the rest in silence, the same shape as the hand-written
key it exists to check.  Observer's own guard passes with them included: 88 perturbations became 91,
no new findings, one blind spot fewer.

## D-122 — the fence bypass, and the direction of a wrong fix

A zone-only alert has no point, so no fence can measure it; the only thing that admits one is the app
already following that hazard at a watched location.  `arrivalsOf` stamped `Tracked: true` on every
arrival — true of the scope that PLANNED the card, false of every scope after it.  **It is the one
bypass a narrower radius could not close, because narrowing is not what it consulted.**

**The over-correction is the worse failure and is pinned separately.**  If the arrival's key and the
tie set's key normalise differently, NOTHING zone-only is admitted and a flood warning at the
listener's own location goes unread.

## `make mutant-anchors` — the gate this release added

**RATIFIED BY THE HUM LEAD 2026-09-13** after four anchors drifted in one session.  306 mutants, 0.18
seconds; `mutant-check` answers the same question in ~400 because it also compiles every mutation.

**It runs each mutant's OWN assertion** — executed with `write_text` neutered — rather than grepping
`old = "…"`.  A second parser for the corpus can disagree with the harness, and the first draft did:
it decoded anchors with `unicode_escape` and reported a false drift on the one anchor containing a
`◆`.  Four mutants also COMPUTE their anchor, which no regex sees.

Four controls, all watched firing; unconditional in CI, where `mutant-check` is policy-scheduled and
therefore absent from an ordinary PR.

## D-128 — a key is one half of a conversation

**The window on top owned the keyboard and nothing else, so every answer it was owed went to the
wrong surface.**  HUM LEAD, UAT 2026-09-14: *"now location search doesn't work at all - no suggestion
or error for invalid location; pressing `<enter>` does nothing … tried multiple valid/invalid
locations multiple times - same (lack) of behavior."*

**What actually happened, end to end.**  `[l]` on the console opened Observer's search window, D-58
forwarded the key presses to it, and the window did everything right: it built the query, took the
`enter`, and issued its resolve command.  The `resolvedMsg` that came back was then routed by the
Router's ordinary rule — *program-scoped to both, console-scoped to the console, everything else to
the ACTIVE surface* — and the active surface was the console.  **The console received an answer to a
question it had not asked, did not recognise the type, and dropped it.**  The window stayed open with
no result and no error, which is indistinguishable from a dead key.

**The fix is a third scope, not a wider forward.**  `observerScoped` names the four unexported replies
— `resolvedMsg`, `committedMsg`, `castSavedMsg`, `uiSavedMsg` — and the Router carries them to
Observer wherever the operator is looking.  They are unexported answers to unexported commands, so a
Dashboard is the only thing that can have issued one and the only thing that can read one: the
routing is decidable from the type alone.  It is **unconditional** rather than gated on an open
window, because when Observer is the active surface the switch below already delivers them, and a
gate would only be a second place for the two paths to disagree.

**Three more windows were broken and nobody had pressed them yet.**  Settings save, the UI-prefs
save, and the watchlist commit all reply through the same three types.  Opened from the console, each
would have saved and then said nothing — the same silence, in three places the UAT had not reached.

**The guard is derived, because a hand-kept list of types is the F-30 failure.**
`TestEveryWindowReplyIsRoutedBackToTheWindow` walks the package's AST for every unexported `*Msg` and
requires each one to be either routed or **excused by name with its reason**.  The two tick messages
are excused: they are Observer's own animation cadence, and nothing asked for them, so nothing is
owed an answer.  Validated by dropping `uiSavedMsg` from the switch — the guard names it and fails.

**What the tests measure.**  Both UAT tests drive real key presses through `Router.Update`, run the
command that comes back, and feed its message in the way bubbletea does, then assert on the PAINTED
FRAME.  A test that stopped at the key press would have measured that a command was returned, not
that anybody received its answer — which is the whole of the defect.

## D-129 — on the console, a place the station cannot reach is not a location

**HUM LEAD, UAT 2026-09-14:** `[l]`, type *"Lone Pine, CA"*, wait, press enter.  *"EXPECTED: A. …
should trigger 'invalid location message'  B. `<enter>` should be disabled for invalid location.
ACTUAL: i. No invalid location message  ii. `<enter>` opens location details modal for Lone Pine,
CA."*

**The ruling was already on the record and I argued against it.**  Asked on 2026-09-14 whether `[l]`
should be pool-scoped, the HUM LEAD answered *"either what they type is a valid location within the
service radius or not"* — which defines validity **for the console** as reachability.  I measured the
two lookup paths, found the pool 15 µs faster and the difference immaterial, and recommended NOT
scoping it on the grounds that it would cost out-of-radius lookup.  **That reasoning took a property
of OBSERVER and applied it to the console.**  Observer exists to look anywhere; the station can only
broadcast about what it can reach, and the request window three feet away had been enforcing exactly
that since R4.

**D-56 is what makes this consistent rather than contradictory.**  One key, one meaning **per
surface**: `[l]` means "find me a place" on both, and what differs is what a place IS.  The console's
refusal points at Observer, which is where an operator who wants Lone Pine should go.

**What it does now.**  The search window, while serving the console, asks `PoolLookup` on every
keystroke, shows the refusal as it is typed, names the pooled location when there IS one, and draws
`enter` unavailable until there is.  Enter is refused at the handler too — a chip drawn unavailable
and a key that acts anyway is worse than either alone.  **A pool hit skips the resolver entirely**:
the answer is already held, so the console's lookup no longer makes a network call at all.

**Live feedback is only affordable because the question got cheaper.**  The pool scan is a prefix
compare per pooled location; the resolver is a ~200 ms round trip whenever the query is not an exact
city name.  Per-keystroke validation against the resolver would have been the per-keystroke network
call the pool-only ruling removed from the request window.

### The extraction, and two mutants that moved with it

Two windows now ask the pool the same question, so the wording and the tint live in `poolnote.go` —
the second caller is where the helper gets extracted.  `mBB1` and `mBB2` were re-pointed there, and
**they now guard both windows with one mutation**, which is the argument for the extraction rather
than a consequence of it.

### Three instrument failures found on the way, all the same shape

1. **`modalAdd` had no fixture arm in the F-30 guard**, so the window was exercised with an empty
   query on Observer's surface — where the pool verdict never reaches the frame.  The guard reported
   the window covered.  The `modalRequest` arm next to it already says why in as many words: *a
   fixture that does not exercise the state is a hole shaped exactly like coverage.*
2. **`walk` never emitted POINTER fields at all** — the third instance of the hole its own `Float64`
   comment documents.  Pointers were excluded for a real reason: `perturbEach` copies the MODEL, and
   a copy shares every pointer in it, so bumping through one moves the baseline too and the guard
   passes by changing nothing.  **Cloning the pointee is what makes including them safe.**
3. **My own validation silently matched nothing.**  Checking that the guard catches a missing
   `addRef` meant deleting it from the key — and the anchor string carried two spaces where gofmt had
   left one, so the deletion never applied and the green result described an unmodified build.  I
   reported "still not caught" from it before re-running with an assertion on the anchor.  **This is
   the mutant-corpus lesson arriving in a one-off check: an edit that did not apply is not evidence.**

### Recorded

`mCB1` (the console looks up anywhere) and `mCB2` (the refused key acts anyway) — both CAUGHT.
Corpus is 336.

## D-129a — "Broadcast Radius" again, from the other end of the same rule

**HUM LEAD, UAT 2026-09-14, second screenshot:** *"'Broadcast Radius' bug is back"* — the caveat's
first line red and italic, its second line plain grey.

**Not a regression of the first fix; the half of the rule the first fix did not state.**  The tint
survives a wrap only if the text is wrapped BEFORE it is coloured — that was mBB2, and it still
holds.  What this adds: **it must be wrapped to the width of the window it is going into.**  The
console's lookup window is 56 cells inside a 133-cell terminal, and it was being wrapped to 112,
because it borrowed `requestHelperWidth` — which derives from the TERMINAL width, correct for the
request window and wrong for this one.  The lines came back tinted and 62 runes long, the frame
wrapped them a second time, and the second wrap landed after the colour.

**Wrapping to the wrong width is the same defect as not wrapping at all.**  One formula now —
`modalHelperWidth(width)` — with the request window passing `o.Width` and the lookup window passing
its own `modalWidth()`.

### The test that passed against the broken build

The first version asserted that each caveat line `Contains("\x1b[")`.  **Every line in the window
contains that**: the panel tints its own background on all of them.  The assertion was true of the
build in the screenshot, and it was true of a blank line.  It now asserts the **italic** — which is
the caveat's own and which the frame never adds — and it fails against the screenshot's build and
passes against the fix.

**This is the third time in two days that a test measured something adjacent to the thing reported.**
The pattern is the same each time: an assertion that is cheap to write and true for a reason other
than the one intended.  The counter is the one already in `06_docs/quality-observations.md` —
**prove the instrument fails before quoting it as green** — and it is what turned this one up, one
step later than it should have.

`mBB3` records it: the helper wrapped to the terminal's width instead of the window's.  Corpus 337.

## D-130 — the pool was never the test, and the fix is a debounce

**HUM LEAD, UAT 2026-09-14:** *"location search should accept any value WITHIN the service radius, not
just the 25 slot location pool.  Example: 'Rainbow, CA' is a valid location within a 25 mi radius of
Oceanside, but now it says that it's not a valid location."*

**D-129 scoped the console's lookup to the POOL and called that the service radius.  They are not the
same set.**  `locations.PoolCap` is 25 and the comment on it says why — deliberately narrower than
the fence, so the Director always has more to choose from than it needs.  Testing membership of that
25 answers "is this one of the places the Director already offers", which is a different question
from "can the station broadcast about this".

### What the measurement settled

| Query | Embedded index | Truth |
| --- | --- | --- |
| **Rainbow, CA** | **in neither the city table nor the zip table** | **14.7 mi from Oceanside — inside 25** |
| Lone Pine, CA | absent | ~200 mi — outside |
| Bonsall, Pala, Julian | absent from the city table | Bonsall and Pala are in the pool via the zip tier |
| Ramona, CA | present | 28.6 mi |
| 92028 | present | 9.8 mi |

34,106 cities and 41,490 zips, and **Rainbow is in none of them** — an unincorporated community with
no postal code of its own.  So this is not a case where the network is an optimisation to avoid: for
a whole class of small places the geocoder is the only thing that knows they exist.

### Which collided with a rule that was already ratified

`setup.go` carries it in as many words: **never the network per keystroke (AI-8, ToS).**  Asking the
geocoder as the operator types would have traded one defect for a worse one.

**HUM LEAD, same day:** *"instead of trying to resolve on *every keystroke* let's set a timer to
'wait' for the user pause … Everytime the human user presses an alpha-numeric key in the text field,
we can infer they're still typing, and we can reset that timer.  This way we're not slamming the
resolver, but we're also not forcing the user to hit `<enter>` before we give *any* feedback at
all."*  That is what reconciles the two, and it is what shipped: 300 ms, reset on every edit.

### `platform/debounce`, and why it is a package

**HUM LEAD:** *"let's ensure we build the debounce as a helper that can be applied to *any* user
input we want in the future, or retrofit into existing ones later to help improve performance."*

It is **framework-free**: nothing in it knows about bubbletea, a timer, or a surface.  What lives
there is the part that is easy to get wrong and identical everywhere — WHICH answer is still the one
the operator is waiting for.  The tick wiring stays at the bubbletea edge in `modes/tty/locate.go`,
generic over the message so any future field can wait on its own.

**A SEQUENCE NUMBER, NOT A TIMER HANDLE**, and that is the load-bearing choice.  Cancelling a pending
timer is the obvious design and the one that goes wrong: the cancel races the fire, and **a resolve
already in flight has no timer left to cancel at all.**  Counting edits makes staleness a comparison
— every pause and every answer carries the sequence it was asked for, and anything that does not
match is discarded on arrival.  Nothing has to be stopped, so nothing can fail to stop.

### Both windows, one mechanism

The console's `[l]` and the Line-Up Request window's Location field ask the same question of the same
hook and draw the same sentence.  `requestState`'s own `ref`/`outside` pair is gone; both now hold a
`locateState`.  Two copies of the bookkeeping would have become two ideas of what "valid" means —
which is how this defect started.

**Stated assumption, open to reversal:** applying it to the request window means the operator can
request a card for any place inside the service radius, not only a pooled one.  That follows ruling 6
— the Director executes the will of the operator — but it was not separately ruled.

`enter` is refused only on a **definite** no.  "Not yet known" leaves it enabled, because the check
can take 300 ms plus a round trip and a key that went inert while the field was thinking is the dead
control D-129 exists to prevent.

### Two "survivors" that were neither

`mCC2` (drop the fence test) and `mCC4` (zero the pause) first reported SURVIVED.  **Both were build
failures** — `within := true` orphans the `globalfeed` import and a bare `0` orphans `time` — so the
harness compiled nothing and my crude verdict loop, which counted `--- FAIL` lines, read silence as
survival.  That is the INVALID-vs-SURVIVED distinction this project already writes down, arriving in
a one-off loop rather than in `run.sh`, which gets it right.  Both mutations now keep their imports
used, and both are CAUGHT.

`mCC1` first survived for a real reason: **nothing in `app` tested the radius check at all.**
`app/locate_radius_test.go` now covers the five answers — pooled, inside-but-unpooled, real-but-far,
nonexistent, and a station with no epicentre.

### Recorded

`mCC1` pool-is-the-radius, `mCC2` fence-admits-everywhere, `mCC3` stale-answer-taken, `mCC4`
every-keystroke-asks — all CAUGHT.  `mCA1`, `mCB1` and `mCB2` re-pointed.  Corpus 341.

### D-130 ratified, with the reason that makes it a requirement rather than a preference

**HUM LEAD, 2026-09-14:** *"That's correct - the human operator should be able to lookup any valid
location within their service radius, even if the initial sorting didn't include it into the default
location pool.  That value here again is exactly the hyper-local (Rainbow, CA) use case for human
operator broadcasting a short range FRS/GMRS/CBRS station."*

**The assumption stated above is ratified**: the request window accepts any in-radius location, not
only pooled ones.

**And the reason closes the question for good.**  `locations.Pool` fills from a POPULATION-FILTERED
table, nearest first — so the 25 it keeps are structurally the LARGEST places in range.  A
short-range FRS/GMRS/CBRS station transmits a few miles, and the people listening are standing in the
small places that sort discards.  **Rainbow was not an unlucky 26th.  A pool ordered by population
can never surface it, at any cap** — which is why raising `PoolCap` would have been the wrong fix and
why the service radius is the only correct test.

This is recorded at the decision site in `app/pool.go`, because it is exactly the kind of reasoning
that gets re-derived wrongly from the code alone — as it was, on the way to D-129.

## D-131 — a label that outlived its control

**HUM LEAD, UAT 2026-09-14:** *"since we removed per card presenters, we need to remove the PRESENTER
- N/A from the Up Next Card."*

The v3 layout plan's item 6 was **"the per-card PRESENTER control"**.  It was dropped; the LABEL was
not.  Every card went on drawing `PRESENTER: N/A` — the name of a control that no longer exists,
reporting nothing, with no key that could ever change it.  D-65 already rules that a control which
cannot act is worse than an absent one, and this was a step past that: not a dead control but the
caption of one.

**THE VOICE IS NOT GONE, AND THAT IS THE DISTINCTION.** `Card.ReadBy` is still resolved from the role
cast and is still said where it is a FACT rather than a control — the LIVE NOW row names who is
presenting the card on air, and the detail window carries `READ BY:`.  What was removed is the
per-card OVERRIDE and its caption, never the answer to "who reads this".

**Pinned as an ABSENCE, which is the unusual half.**  The field it named still exists and is still
populated, so restoring the label compiles, renders, and reads perfectly plausibly — the recurrence
is the risk, not the survival.  `TestTheUpNextCardCarriesItsHandle` now fails if the word `PRESENTER`
returns to the footer, and `mCD1` puts it back to prove that.

`mAD3` re-pointed: `cardControls` no longer takes the card.  CAUGHT before and after.  Corpus 342.

## D-132 — the UP NEXT caption

**HUM LEAD, 2026-09-14:** *"Let's make 'UP NEXT' in the up next box BOLD WHITE so it contrasts a bit
more."*

Done, through `TextBright` rather than a literal white: that token is what this app already calls
emphasized plain text, so the caption follows the theme.  The Light theme's bright is not the dark
one's, and a colour written here would be a second answer to a question the theme owns.

**Why it needed a mutant rather than an eyeball.**  D-114 paints the whole box — borders included —
so the caption row carries escape codes whether it is styled or not.  A test asking whether it "has
colour" passes against a completely plain caption.  **The bold is the discriminator**, because the
ground never sets it; `mCE1` strips the styling and is CAUGHT on exactly that.

### A mutant written, measured, and thrown away

I also wrote `mCE2`: pad the caption AFTER tinting it, on the reasoning that escape codes would be
counted as cells and shift the card's column.  **It changed no frame**, because `render.PadTo` and
`centerText` both measure with `displayWidth`, which skips escapes.  It was an EQUIVALENT mutant —
not a rule nothing measures — so it was deleted rather than committed as a survivor owing a
retirement.

**The comment it came from was corrected too.**  I had already written the false reasoning into
`broadcaster_upnext.go` as the justification for the ordering, where it would have read as
established fact to whoever came next.  Running the mutation is what disproved it.  A plausible
rationale in a comment is not evidence, and this is the cheapest possible demonstration: the claim
survived review in my own head and died in one measurement.

`mCE1` CAUGHT.  Corpus 343.

## D-133 — the two editions stop sharing a colour

**HUM LEAD, 2026-09-14:** *"Let's make the 'Broadcaster' text in the mastHead Orange vs. the Bold
Light Blue - so: Observer - Bold Light Blue / Broadcaster - Bold Orange."*

**The token's own doc had already called this.** `TitleEdition` was written in 0.14.0 with the note:
*"Its own token, not FocusCell borrowed … and the second edition will want to differ."*  It does, and
`TitleEditionBroadcaster` is that difference arriving as planned rather than as a retrofit.

**Why it matters more than a colour preference.** Both surfaces draw the same wordmark, the same
ladders and the same stamp, and `ctrl+o` / `ctrl+b` swap between them IN PLACE.  The word beside the
wordmark is the only thing that says which one you are looking at — and with both words in one tone
that was a distinction you had to READ rather than see.

**Each theme's own orange, never a colour introduced here.** Default takes 208 — already its `TempHi`
and `FireMark`; High Contrast 214; Synthwave 215; Solarized Night its own 166; Light a dark orange
that reads on white, the same move its blue already makes.  Every Quattro palette declares an
`orange`, so the mapper derives it exactly as it derives the blue — no per-theme decision at all.

**Monochrome is the honest exception and says so at its entry.**  The theme has no orange, and
inventing one is the thing it exists to refuse, so both editions share a tone there and the WORD
carries the distinction — the fallback every surface in that theme already relies on.

**`Wordmark` picks the token, not the two call sites.** The masthead and the About window both go
through it precisely so they cannot disagree about which edition the build is; after this they cannot
disagree about its colour either.

### Two guards, and one test that was measuring the encoding

`TestEachEditionWearsItsOwnToneInEveryTheme` walks every registered theme and requires the two tones
to differ and both to be bold — with Monochrome excused **by name, with its reason**, so a theme added
later that simply leaves the token unset fails rather than passing quietly.  The AA register carries
the new token on both grounds, and the existing completeness guard is what forced that.

**My first version of the wordmark test failed against a correct build.**  It asserted the rendered
string contained `Tok(TitleEditionBroadcaster)` — the raw table value, `"1;208"` — but `Tint` expands
that into `"1;38;5;208"`, a 256-colour index rather than a literal SGR.  The test was measuring the
ENCODING and not the tone.  Building the expected segment through `Tint`, the way the painter does,
is the comparison that actually means something.

`mCF1` — the painter reaches for one token again while the other stays declared, registered and
themed — CAUGHT.  Corpus 344.

## D-134 — the live state, and the box that borrows the bands' blue

**HUM LEAD, 2026-09-15:** *"When Broadcaster is Actively broadcasting, let's make this string `*** ON
AIR · BROADCASTING ***` BOLD WHITE … and we'll make the cell background color of the UP NEXT box the
same 'blue' token color used as the bkg for 'DIRECTION' and 'TODAY' column - this will help add a bit
of visual distinction that will also be themeable."*

**The on-air state is bolded in the section's OWN white.** `stationTone` already paints that band
`AlertModalText` on `TickerEmergencyBG`, and `AlertModalText` **is** white — so this states the tone
the row already wears and adds the weight.  Registering `TextBright` against the emergency ground
instead would have lifted `TextBright` **everywhere it is painted**, which `aaPairs` warns about in as
many words.  **Weight is not contrast**, so no AA answer changes: the pair is already in production
and is unchanged.  Only the LIVE state is bolded — bolding all three would spend the emphasis on the
one thing it exists to single out.

**The UP NEXT box wears `GroupTodayBG`, which is the same OBJECT**, not a colour matched by eye: the
ground under `D I R E C T I O N` in the console's line-up table and under `T O D A Y` in Observer's.
`GroupText` comes with it, and `aaPairs` already registers `{GroupText, bands}` — so the box inherits
a contrast answer measured in every theme rather than asserting a new one.  D-114's *"the same Blue
as the modal FOR NOW"* ends exactly where that note said it would.

**The test compares against the rendered `D I R E C T I O N` band, not against the token's name**, so
it still means something on the day the table moves to a different token.  Its first draft took the
FIRST background on that row — a grey cell three bands away — and reported a mismatch that was
entirely its own; the escape that paints a run of text is the last one opened before it.

`mCG1` (the state loses its weight) and `mCG2` (the box returns to the modal's tile) — both CAUGHT.

## D-135 — the Help window answers for the surface you are on

**HUM LEAD, UAT 2026-09-15:** *"we also need to make the ? (help) modal display the correct key
bindings and hints depending on the mode that currently active - rigth now the help window only shows
Observer key bindings, and it doesnt show the user how to swap between Observer and Broadcaster.
Once the user in the Broadcaster UI, they key bindings share/rempapped for that mode do not update
their help (like 'r')."*

**Three defects, one report.**

1. **The console was handed the listener's manual.**  `helpBlocks` read `d.keys` — the Dashboard's
   map — and the console's bindings live on the *Router*.  The Help window is Observer's, forwarded
   from the console, so it documented the wrong surface entirely: a watchlist and a visualizer the
   console does not have, and nothing about the station, the line-up or the bed.
2. **Neither surface said how to reach the other.**  The swap is live on BOTH — the Router looks it
   up before either surface sees the key — and precisely because the Router owns it, **neither
   keymap carries it**, so neither help could show it.  `SURFACES` now leads both windows.
3. **`[r]` was not a binding at all.**  It was a bare `case "r":` in a key switch, which breaks D-15
   — *keys are data* — in both directions that rule exists for: it could not be rebound, and, since
   the Help window is built from the keymap, **a control the console DRAWS could not be documented.**
   It is `actRequest` now, gated on the console owning the keys so Observer's repeat survives.

**The legend followed the surface too.**  The row marks describe a table only Observer draws.  The
console's legend carries what the keymap cannot: the ten slot ADDRESSES are deliberately not
bindings — *"ten ADDRESSES of one action, not ten actions"* — so the legend is the only place they
can be explained at all.

### A test that passed on the defect, and the mutant that found it

`mCH1` swaps the GROUPING back to Observer's while leaving the keymap correct — and it **survived**.
`helpBlocks` sweeps anything the grouping does not claim into an `OTHER` block, so every console row
was still present, just heaped under one heading.  My test asserted the ROWS and never the SECTIONS,
so the organisation — which is most of what a help window is for — was unmeasured.  It now requires
`STATION`, `LINE UP` and `BED` to exist and Observer's sections to be absent.

**And an existing test broke for an honest reason.**  `TestHelpLaysOutOneOrTwoColumns` decided "two
columns" by asking whether `NAVIGATE` sat beside `WATCHLIST` — a proxy for the layout that broke the
day a group was ADDED and the balance point moved, reporting one column on a window plainly drawing
two.  It now asks whether ANY line carries two group headers, which is the thing it is named for.

`mAZ1` re-pointed: `[r]` is a binding now, so killing it means refusing the ACTION, not renaming the
key — and `mCH4` is the separate mutant that removes the binding.  `mCH1`–`mCH4` all CAUGHT.
Corpus 350.

## D-136 — "cell" meant cell

**HUM LEAD, UAT 2026-09-15, with a diagram:** the label cell is TODAY BLUE, the report beside it is
"STANDARD MODAL BLUE", and its *"content … should not be all bold and white, but the standard text
color"*.

**I read "the cell background color of the UP NEXT box" as the BOX and painted all of it** — label
and report together — in the bands' blue and the bands' bold white.  The word was *cell*, and a label
cell beside a report is exactly the shape `D I R E C T I O N` has beside its rows, which is why that
band was the colour named: **the point was the CELL, not the box.**

**Two grounds now.**  `TintKeeping` paints the cell first and the row second, and the row's paint
keeps the cell's — it rewrites an inner RESET to the outer ground, and the cell's opening escape is
not a reset.  So the label block survives and the report returns to the modal tone at the rail.

### The test passed the over-application, and says so

The D-134 test asked whether EVERY row carried the bands' blue — which a box painted entirely in it
passes perfectly.  It measured the presence of a colour, not its BOUNDARY, so the defect the HUM LEAD
could see in a screenshot was invisible to it.

**Rewritten to measure the ground IN FORCE where the text is**, which is the same technique the
DIRECTION comparison needed: every body row carries a label cell, so `Contains(row, blue)` answers
the wrong question — what paints a run of text is the last ground opened before it.  Verified by
painting the whole box blue again: the test fails, as it should have the first time.

### READ MANIFEST was already centred, and the measurement says so

The report asked for it centred in the data cell.  **It is**: 26 cells of padding to its left, 27 to
its right, inside a 66-cell interior — the 1-cell asymmetry is the integer rounding of an odd
remainder and cannot be divided away.  It is also centred over the manifest TABLE, whose own span is
narrower.  Nothing was changed; the number is recorded so the next reading of that line starts from a
measurement rather than from the impression a screenshot gives.

*(My first attempt to measure it said "3 cells right of centre" — `strings.Index` returns a BYTE
offset and `┃` is three bytes, so the arithmetic mixed bytes with cells. A rune-accurate count is what
settled it.)*

`mCG2` re-pointed at the label cell — CAUGHT.  Corpus 350.

## D-137 — the card's two facts, readable at a glance

**HUM LEAD, 2026-09-15:** *"Let's make 'Ready for Read-Out' Green, and color code the other status
messages … Color code the (n MIN AGO): < 2 min BLUE, < 5 min GREEN, < 10 min YELLOW, < 15 min ORANGE
( we refresh at 15m max so It should never get to 'red' )."*

**STATUS uses tokens already registered on this ground.** `ProviderOK`, `AlertLabel` and
`ProviderDown` are in `aaPairs`' `onBoth` list, so they read at AA on the modal tile the card is
painted on without widening anything.  The waiting state is yellow and **not bold**, said in the same
breath: a card waiting on a fetch is a caution, not a fault, and weight would make every unfilled
slot shout.

### The measurement that decided the ladder's shape

The obvious move is to reuse `FocusCell`, `ProviderOK`, `AlertLabel` and `FireMark` for the four
rungs.  **Measured before writing it:** registering those against the MODAL ground lifts them until
they read there, and it **moved `FireMark` in five themes and `FocusCell` in one**.  Observer's ▲ fire
marks would have changed colour to make a console caption legible — precisely what `aaPairs` warns
about.  So the ladder owns its tokens, and a re-run of the same probe confirms **no existing token
moved at all**.

**They are filled from each theme's own palette** rather than listed per theme, so every theme —
including the Quattro palettes nobody hand-writes — gets an on-palette ladder for free, and a theme
may still name all five.

### Named for the data, never for the colour

**HUM LEAD, correcting my first names:** *"make the tokens semantic … dataNew / dataFresh /
dataUsable / dataAged / dataStale … that way the color/implementation detail isn't in the token name
(and can be remapped to other colors later)."*

My `PullFresh`/`PullDue` carried no colour either, but the family was mine rather than the domain's.
The rungs now say **how much the operator can trust what they are reading**; the palette says what
that looks like.  **Five rungs for four thresholds**, so every duration maps to a named rung instead
of falling into an implicit "everything else" — `DataStale` is the state the cadence exists to
prevent, and naming it is what lets a theme distinguish it later with no code change.  It is not red
today: no rung promises a colour the ruling said the operator should never meet.

### A bug the Monochrome guard caught immediately

The first fill tested the slot for EMPTINESS — but a registered theme starts from a **copy of the
default**, so the slot was already full of Watchpost's blue and no theme ever re-derived its own.
Monochrome, whose whole purpose is to have none, was handed four colours and said so at once.  The
fill now asks *"did this theme name it"*, and `mCJ3` restores the emptiness test to keep that pinned.

### Still owed: "Data Pull failed" has nothing behind it

The ruling asks for a red `Data Pull failed`.  **`lineup.State` has no such state** — Proposed,
Admitted, Refused, Standby, OnAir, Done, Discarded — and `cardStatus` has no arm that can produce
those words.  Colouring a string the console cannot emit would be painting a state that does not
exist, which FR-3.3 refuses in the direction that matters most.  **Recorded, not invented**: making
it real means the fetch path reporting a failure onto the card, which is a pipeline change and a
ruling of its own.

`mCJ1` (status goes plain), `mCJ2` (a rung's threshold moves), `mCJ3` (the ladder is borrowed, not
owned) — all CAUGHT.  Corpus 353.

## D-138 — the notice nobody's eye stopped on

**HUM LEAD, 2026-09-15:** *"Found a good usage that can use some UI attention … Let's make that look
like the ticker I almost missed this: 3 lines, message in the center, 3 line bkg should be the darker
yellow (not orange, not red) use the same tint as the 'LOCAL ALERT Advisory BKG'."*

**"I almost missed this" is the whole finding.**  The held-hazard notice was one UNPAINTED row among
painted regions — the least visible thing a frame can contain — and it is the one row that says a
hazard is being held OFF THE AIR while the station is silent.  Its words were right and nobody's eye
stopped on them.  NFR-7's bound was met in text and missed in practice.

**`AlertModalAdvBG` is the tint named**: the LOCAL ALERT window's advisory tile, "muted yellow" in its
own comment — **not** `TickerAdvisoryBG`, which is the burnt orange of the tape's advisory lane and
exactly the colour the ruling excludes.  `AlertModalText` comes with it, a pair `aaPairs` already
registers, so the band inherits a contrast answer measured in every theme.  A test names the three
tape grounds it must NOT be wearing, so "not orange, not red" is enforced rather than remembered.

**Built the way the station section is built** — padded to the frame less the inset, the inset added,
then painted in one `Block` call.  Two adjacent bands assembled two different ways is how a
three-cell disagreement gets in.

### The prettier version clipped the worst message

Centring alone CUTS.  The `!!!` rung's sentence is ~149 cells against a 127-cell band at the HUM
LEAD's width, and the console's most severe notice lost its ending: *"may be drop"*.  **The single
line it replaced wrapped in the terminal**, so this would have been a regression introduced by making
the notice prettier — the exact shape this project keeps writing down, found this time by rendering
the band and reading it rather than by reading the code.

**Wrapped before it is styled**, which is D-129a's rule applied in the other direction, and the
emphasis applied PER LINE afterwards: a styling span does not survive being cut in half.  Three lines
is the common case, not a promise the band breaks to keep — where the sentence does not fit, the band
GROWS rather than the message shrinking, because a truncated hazard notice is the one outcome this
notice exists to prevent.

### Two of my own assertions were measuring the encoding again

`TestTheHeldMessageIsCentredInTheBand` ran with colour OFF, where `Block` trims trailing spaces — so
every row looked right-aligned and the measurement meant nothing.  And
`TestTheBandBoldsTheCountAndTheAction` compared against `render.Bold(text)` as a literal, but painting
**rewrites every inner reset** to the band's ground, so that literal cannot survive into the output:
the test failed against a perfectly bold band.  It now checks the WEIGHT IN FORCE where the text is —
the nearest escape opened before it — which is the same technique the DIRECTION and the report-ground
comparisons needed.  **Third instance of one shape in two days**, and the counter is the one already
written down: assert the property, never the encoding.

`mCK1` (the band goes back to a bare line) and `mCK2` (the band clips instead of wrapping) — both
CAUGHT, and `mCK1` needed `_ = o` to stay buildable, or it would have read INVALID rather than as
evidence.  Corpus 355.

### D-137 addendum — "Data Pull failed" is CLOSED, not deferred

**HUM LEAD, 2026-09-15:** *"No need for now if that state actually doesn't exist."*

It does not.  `lineup.State` is Proposed / Admitted / Refused / Standby / OnAir / Done / Discarded, and
`cardStatus` has no arm that can emit those words — so the red rung of the status colour coding has
nothing behind it and is not built.  **Recorded so it is not re-derived as a phantom requirement**:
the ruling asked for it, the state was checked, and it was dropped on the evidence rather than
half-implemented against a string the console can never produce.

### F-102 re-verified at the HUM LEAD's request — not present

**245 clean runs and no sighting.**  200 consecutive runs of the named test, 6 further full-`app`
runs, and 39 app-suite runs across 13 `make verify` logs — all green, with the string `TempDir
RemoveAll` appearing in **no log in `dist/` at all**.  The single occurrence was in an ad-hoc `go test
./...`, never in a gate.

**Left OPEN rather than closed.**  One unexplained occurrence is not disproved by 245 clean ones, and
the mechanism was never identified — closing it would delete the only record of it for whoever meets
it next.  Re-open on a second sighting; carrying it costs one line.
