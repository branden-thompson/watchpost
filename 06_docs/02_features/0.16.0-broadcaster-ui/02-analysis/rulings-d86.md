# D-86 — THE CONSOLE'S COLOURS, AND WHY REGISTERING A PAIR IS NOT FREE

**HUM LEAD, 2026-09-11:**

> *"Main Track Cards should have some color BKGs that use tokens so it can be themeable like
> Observer … tint them based on report type and card origin: Operator requested cards get a slightly
> different Tint than Producer created cards.  Alert cards should be color coded to match the most
> severe alert based on the [w] category bkgs in Observer (they should match).  Color BKGs for the
> left RAIL: LIVE = RED, UP NEXT = ORANGE, SCHEDULED LINEUP = BLUE."*

**"Narrow approved"** — one ground for the station's own proposals and one for the operator's
requests, rather than a ground per slot.  Most slots do not exist yet (D-31) and the report type is
already in the card's title.

---

## The alert cards cost no tokens at all

`category.Of(k).Tint` **IS** the background the `[w]` window paints that category on, and `Card.From`
carries each arrival's `Category` — so the most severe is computable from the card alone and the two
surfaces match **by construction**.  A copied value would have drifted; this cannot.

**BY READ RANK**, which is the registry's own severity order and the same ladder the Producer plans a
burst with.  A second notion of "most severe" here would paint a card one colour and read it in
another order.

**A burst with no arrival to read a category from keeps the ordinary card ground.**  Painting a
hazard the wrong severity is worse than painting it none.

---

## Registering an AA pair changes the token EVERYWHERE

The batch's real finding, and it cost a round trip.

`withAA` lifts a foreground until it reads on **every** background it is registered against.  So
registering `TextBase` against the new card grounds did not just measure them — it **moved
`TextBase`**, and Observer's goldens drifted.

**MEASURED RATHER THAN GUESSED.**  Dumping every token per theme, with the registration and without,
named exactly two changes:

| theme | token | before | after |
|---|---|---|---|
| Watchpost | `TableMuted` | `38;2;143;143;143` | `38;2;152;152;152` |
| Solarized Night | `TextBase` | `247` | `38;2;170;170;170` |

Small — and it is **Observer's appearance changing because the Broadcaster gained a card colour**,
which is not a thing this batch is allowed to do.

**So the console's cards carry a tone of their own, `CardText`**, and lifting it reaches nothing else.
The rail needed no such thing: `GroupText` moved in no theme against the three rail grounds, because
they were derived to sit in the Group bands' own family — which is the second reason to derive rather
than pick.

---

## The rail's three grounds are one family, permuted

Observer already has a language for "a band naming a region": the `Group*BG` bands, muted
`66/94/122` mixes.  The rail keeps that family by permuting **the same three channel values** —

| | | |
|---|---|---|
| LIVE | `122;66;66` | red dominant |
| UP NEXT | `122;94;66` | mid |
| SCHEDULED / LINE UP | `66;94;122` | **`GroupTodayBG`'s own value** |

— which gives equal perceived weight by construction, keeps the rail reading as a rail rather than as
a warning, and is what made `GroupText` survive the registration unmoved.

**QUATTRO DERIVES THEM**, at the same `0.45` mix the Group bands use, so seven themes got a rail in
their own palette without anybody choosing twenty-one more colours.

**THEY ARE THE CONSOLE'S OWN TOKENS, NOT BORROWED.**  The rail names a REGION while the alert cards
beside it are painted by HAZARD CATEGORY; borrowing a Ticker lane for "UP NEXT" would make one colour
mean *this is an Advisory* in one column and *this is queued* in the next.

**SCHEDULED AND LINE UP SHARE ONE GROUND**, because they are one stack of cards the rail happens to
name in two halves — the same reason no break is drawn between them.

**MONOCHROME KEEPS THE THREE APART BY LIGHTNESS**, since it has no hue to do it with, and LIVE is the
brightest — the ordering the eye already reads from red / orange / blue.  **THE LIGHT THEME INVERTS
THE RELATIONSHIP, NOT THE HUE**: a band on a light ground is a pale wash, and dark values there would
read as holes punched in the page.

---

## Two extractions the work asked for

- **`render.TintKeeping`.**  A card's row carries chips with SGR of their own, and a plain wrap leaves
  the ground behind every chip as a hole in the card.  `TintDefault` was the first caller and this
  was about to be the second, so the rule lives once.
- **`cardTone` paints the WHOLE box, borders included.**  A card is one object; a ground that stopped
  at the border would draw a coloured window inside a colourless frame.

**AN EMPTY SLOT IS NOT A CARD.**  The waiting placeholder and the LIVE slot on a station at rest
carry no identity, and a ground there is a card that is not there.

---

## Recorded

`mR4` did not compile on its first run: replacing the category lookup orphaned `worst`.  That is the
shape `mA3` cost a gate run to learn — **a mutation that removes the only use of a name stops
compiling rather than stops being true** — and the fix was already written down: prefer `_ = x` to a
deletion.  First time the recorded lesson was applied rather than re-learned.
