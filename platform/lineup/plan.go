package lineup

import (
	"fmt"
	"sort"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// DisasterFreshWindow is how recent a disaster must be to outrank a warning
// where no service radius is set. TWENTY-FOUR HOURS, ratified (R-5): it matches
// the listener's own words — "I care more about today's severe thunderstorm
// warning than a landslide that happened four days ago" — and sits well inside
// the seven-day window the seismic feed already keeps, so a stale disaster still
// appears on the tape and in [w]. It merely stops leading the burst.
const DisasterFreshWindow = 24 * time.Hour

// Band is the half of the ladder an alert sorts into. The Close band leads; the
// Remaining band is what a demotion drops into.
type Band int

const (
	CloseBand Band = iota
	RemainingBand

	// numBands bounds the ladder; it is not itself a band.
	numBands
)

// String names the band for the transition log (DR-23).
func (b Band) String() string {
	if b < 0 || b >= numBands {
		return ""
	}
	if b == RemainingBand {
		return "REMAINING"
	}
	return "CLOSE"
}

// Rung is one step of the ratified ladder: a band and a category.
type Rung struct {
	Band     Band
	Category category.Category
}

// Ladder is the read order the burst is sorted by — the thirteen rungs the HUM
// LEAD ratified, Emergency Orders leading and Forecasts absent.
//
// DERIVED FROM THE REGISTRY, NOT WRITTEN OUT AGAIN. The per-category rank lives
// in category.Spec (DR-10), and the Director must not become a fifth list of
// categories (NFR-D-6, F-21) — so this composes the two bands from
// category.ReadOrder() rather than repeating it.
//
// The Remaining band is one rung shorter: EMERGENCY ORDERS ARE NEVER DEMOTED.
// Only Disasters carry the freshness rule, so there is nothing that could sort
// an evacuation order below a warning, and no rung for it to sort to.
func Ladder() []Rung {
	read := category.ReadOrder()
	if err := invariant.Check(len(read) > 0, "the registry names at least one readable category"); err != nil {
		return nil
	}
	if err := invariant.Check(read[0] == category.Emergency, "the read order leads with the emergency orders"); err != nil {
		return nil
	}
	out := make([]Rung, 0, 2*len(read))
	for _, c := range read { // bounded by the registry (P10-02)
		out = append(out, Rung{Band: CloseBand, Category: c})
	}
	for _, c := range read {
		if c == category.Emergency {
			continue
		}
		out = append(out, Rung{Band: RemainingBand, Category: c})
	}
	if err := invariant.Check(len(out) == 2*len(read)-1, "the ladder is both bands, less the emergency rung that does not exist"); err != nil {
		return nil
	}
	return out
}

// rungOf is where a band and category sit on the ladder, counting from 1.
// ZERO MEANS NEVER READ AS AN ALERT — a forecast, or anything else the registry
// gives no read rank.
func rungOf(b Band, c category.Category) int {
	if b < 0 || b >= numBands {
		return 0
	}
	for i, r := range Ladder() { // bounded by the ladder (P10-02)
		if r.Band == b && r.Category == c {
			return i + 1
		}
	}
	return 0
}

// bandOf is which half of the ladder an arrival sorts into (DR-12).
//
// ONLY DISASTERS ARE EVER DEMOTED, and only where no fence is in force. With a
// radius set, the fence already governs entry, so everything that got in is
// close and Disasters outrank Warnings unconditionally — the Broadcaster case.
// With no fence, a Disaster must also be fresh, which is the Observer case and
// THE DEFAULT PATH: config.TickerRadiusMi defaults to 0 = All
// (platform/config/config.go:260), so every fresh install lands here.
//
// The question "is a radius set" is asked through Fence.InForce, which is also
// what admission asks. One carrier, two askers.
func bandOf(c category.Category, s Settings, at, now time.Time) Band {
	if c != category.Disasters {
		return CloseBand
	}
	if s.Fence.InForce() {
		return CloseBand
	}
	// Up to and including the window is fresh; past it, the disaster sorts down.
	// A clock skewed so the disaster is dated ahead reads as fresh, which is the
	// safe direction: a hazard is never demoted by a bad clock.
	if now.Sub(at) <= DisasterFreshWindow {
		return CloseBand
	}
	return RemainingBand
}

// Arrival is one alert offered to the burst. It carries only what the ordering
// needs, and nothing from any domain: the Director pre-screens what reaches it
// and the composer decides what it will say.
type Arrival struct {
	// ID addresses the alert, and becomes the card's identity.
	ID string

	// Category places it on the ladder. Forecasts, and anything the registry
	// gives no read rank, never reach a burst.
	Category category.Category

	// Headline and Subject are the shape the card carries from creation (DR-7):
	// the one line that names the hazard, and what it is about.
	Headline, Subject string

	// Severity and At are the sort dimensions used inside one rung — today's
	// ordering, unchanged (the severity-then-recency order T3.1 folded into sortCandidates). Making the
	// dimension chain configurable is T4.1's, with the settings surface that
	// produces the configuration.
	Severity int
	At       time.Time

	// Lat, Lon and HasPoint are where the hazard is, for the fence. A zone-only
	// alert has no point (globalfeed.Event.HasPoint), and Tracked is whether it
	// is one the app already follows at a watched location — the only way such
	// an alert reaches a scoped surface today.
	Lat, Lon float64
	HasPoint bool
	Tracked  bool

	// ReachMi is how far this hazard carries its own effects, and ZERO IS THE
	// ORDINARY CASE. It is honoured only for a Disaster (DR-13); QuakeReachMi
	// turns a magnitude into one.
	ReachMi float64
}

// Settings is what the listener has set, as the planner needs it.
//
// ONE MAX FOR THE WHOLE BURST across all categories (R-1, G-5). Read order is
// about sorting, not per-category budgets: the per-category Max row and TOTAL
// READS are gone.
type Settings struct {
	// Max is how many alert reads one burst may spend. Emergency Orders spend it
	// first and may overrun it (DR-11).
	Max int

	// Fence is the listener's service radius. It governs ENTRY (DR-13) and it
	// also decides the ordering rule in DR-12 — one carrier for both, asked
	// through Fence.InForce.
	Fence Fence

	// Watchlist is the listener's rotation, in their order, and Dwell is how
	// long a live relay holds the bed before the next one takes it. A zero Dwell
	// means never advance — see Programme (T3.2b).
	Watchlist []string
	Dwell     time.Duration
}

// Burst is one planned takeover: what is read, in order, and how much was not.
type Burst struct {
	// Takeover is the ONE card the schedule holds for this burst (MVS-D-77), and
	// its Refs are the SELECTION: the alerts it reads, in read order.
	//
	// THERE IS ONE ORDERING, AND IT IS THIS ONE (red team 2026-09-05). A
	// `Cards []Card` used to sit beside it, derived from the same selection by a
	// second walk, and every rule about the order — the ladder, emergency orders
	// leading, the fence, freshness, ties, the divert count — was asserted
	// against THAT. Production read only the takeover. Renaming the field and
	// building the tree proved it had no production consumer at all, so
	// takeoverOf could have dropped, reordered or truncated its refs with all
	// forty pins still green. The pins are on this now.
	Takeover Card

	// Divert is what the listener is told they did not hear — arrivals the burst
	// could have read, less the ones it did (DR-14). It is spoken aloud, so it
	// is the one figure here that must not be approximate.
	Divert int
}

// placed is an arrival with its position on the ladder worked out.
type placed struct {
	arrival Arrival
	rung    int
}

// Plan is the burst: a pure function of the arrivals, the listener's settings
// and the clock (DR-2). The same inputs produce the same lineup, byte for byte,
// which is what makes the schedule something a test can state rather than
// observe.
func Plan(arrivals []Arrival, s Settings, now time.Time) (Burst, error) {
	if err := invariant.Check(s.Max >= 0, "the burst's Max is never negative"); err != nil {
		return Burst{}, err
	}
	if err := checkArrivals(arrivals); err != nil {
		return Burst{}, err
	}
	cand := candidates(arrivals, s, now)
	sortCandidates(cand)
	chosen := selectBurst(cand, s.Max)
	takeover, err := takeoverOf(chosen)
	if err != nil {
		return Burst{}, err
	}
	divert := len(cand) - len(chosen)
	if err := invariant.Check(divert >= 0, "the burst reads no more than arrived"); err != nil {
		return Burst{}, err
	}
	// EVERY CANDIDATE WAS EITHER READ OR DIVERTED — the arithmetic the listener
	// is actually told, stated as a conservation law rather than trusted to the
	// subtraction above. It is checked against the TAKEOVER'S OWN REFS, so a
	// ref lost between the selection and the card leaves the count wrong here
	// rather than plausible-looking and quietly short: that count is the one
	// figure in the burst that is spoken aloud.
	if err := invariant.Check(len(takeover.Refs)+divert == len(cand), "every candidate was either read or diverted"); err != nil {
		return Burst{}, err
	}
	// THE TWO HALVES AGREE. A selection with alerts in it must yield a takeover,
	// and an empty one must not — otherwise the schedule holds a card for a
	// burst that reads nothing, or reads a burst the schedule never got.
	// THE COUNT RIDES ON THE CARD, so the Composer never has to ask a second
	// time and can never get a different answer (DR-14).
	takeover.Divert = divert
	b := Burst{Takeover: takeover, Divert: divert}
	if err := invariant.Check(b.HasTakeover() == (len(chosen) > 0), "a burst with alerts has a takeover, and an empty one has none"); err != nil {
		return Burst{}, err
	}
	return b, nil
}

// checkArrivals is the burst's input contract. A malformed arrival is found
// HERE, where the Director can fault it, rather than halfway through a takeover:
// the burst is planned once and then spoken, and there is no second chance to
// notice partway down the rail.
func checkArrivals(arrivals []Arrival) error {
	seen := map[string]bool{}
	for _, a := range arrivals { // bounded by the feed's own cap (DR-11)
		if err := invariant.Check(a.ID != "", "every arrival carries an identity"); err != nil {
			return err
		}
		if err := invariant.Check(!seen[a.ID], "no two arrivals share an identity"); err != nil {
			return err
		}
		if err := invariant.Check(a.Headline != "", "every arrival carries a headline"); err != nil {
			return err
		}
		if err := invariant.Check(a.Subject != "", "every arrival says what it is about"); err != nil {
			return err
		}
		seen[a.ID] = true
	}
	return nil
}

// candidates is the arrivals this burst could read, each with its rung.
//
// A FORECAST IS NOT ONE, and neither is anything else the registry gives no read
// rank. It is therefore neither read NOR counted as diverted: the notice says
// "these and N other ALERTS", and an outlook the listener never lost is not one
// of them. It stays part of the location report, which is most of the broadcast.
func candidates(arrivals []Arrival, s Settings, now time.Time) []placed {
	out := make([]placed, 0, len(arrivals))
	for _, a := range arrivals { // bounded by the arrivals (P10-02)
		// THE FENCE COMES FIRST, because it governs entry rather than order
		// (DR-13). What it keeps out is not read, and not counted either: the
		// divert number tells the listener what they can go and read, and [w] is
		// scoped by the same radius, so counting something the fence removed
		// would point them at a page that does not have it.
		if !s.Fence.Admits(a) {
			continue
		}
		rung := rungOf(bandOf(a.Category, s, a.At, now), a.Category)
		if rung == 0 {
			continue
		}
		out = append(out, placed{arrival: a, rung: rung})
	}
	if err := invariant.Check(len(out) <= len(arrivals), "no arrival becomes two candidates"); err != nil {
		return nil
	}
	return out
}

// sortCandidates puts the burst in read order: the rung first, then — inside one
// rung — severity and recency, which is today's ordering unchanged, and finally
// the identity.
//
// THE IDENTITY IS WHAT MAKES THE ORDER TOTAL. Without it two alerts alike in
// every dimension would keep whatever order they arrived in, and the arrival
// order would reach the listener — which is precisely the determinism DR-2
// forbids.
func sortCandidates(p []placed) {
	sort.SliceStable(p, func(i, j int) bool {
		if p[i].rung != p[j].rung {
			return p[i].rung < p[j].rung
		}
		if p[i].arrival.Severity != p[j].arrival.Severity {
			return p[i].arrival.Severity > p[j].arrival.Severity
		}
		if !p[i].arrival.At.Equal(p[j].arrival.At) {
			return p[i].arrival.At.After(p[j].arrival.At)
		}
		return p[i].arrival.ID < p[j].arrival.ID
	})
	for i := 1; i < len(p); i++ { // bounded by the slice (P10-02)
		if err := invariant.Check(p[i-1].rung <= p[i].rung, "the burst is in rung order once sorted"); err != nil {
			return
		}
	}
}

// selectBurst chooses what is read. p must already be in read order, so the
// emergency orders lead it.
//
// EMERGENCY ORDERS ARE READ FIRST AND ALWAYS, AND THEY SPEND THE BUDGET
// (DR-11, Q-1). The remaining budget is filled by read order. Only when the
// emergency orders ALONE exceed Max does the burst overrun — then all of them
// are read and everything else is diverted, because the sheer amount of DO THIS
// NOW means the warnings matter less than the listener hearing the whole
// emergency. The overrun is bounded by the feed, never by a constant: this
// ranges over an already-capped slice (globalfeed.MaxPerLane = 30,
// domains/globalfeed/stack.go:51), which is P10-02's required form.
func selectBurst(p []placed, max int) []placed {
	if err := invariant.Check(max >= 0, "the budget is never negative"); err != nil {
		return nil
	}
	out := make([]placed, 0, len(p))
	spent := 0
	for _, c := range p { // bounded by the candidates (P10-02)
		emergency := c.arrival.Category == category.Emergency
		if !emergency && spent >= max {
			continue
		}
		out = append(out, c)
		spent++
	}
	if err := invariant.Check(spent == len(out), "the budget counts exactly what was read"); err != nil {
		return nil
	}
	if err := invariant.Check(len(out) <= len(p), "the burst reads no more than was offered"); err != nil {
		return nil
	}
	return out
}

// HasTakeover says whether this burst gives the rail a card.
//
// DERIVED FROM THE CARD'S OWN IDENTITY, not stored beside it. A boolean field
// saying the same thing as Takeover.ID would be a second carrier of one rule
// (D-1), and the copy nobody updated is the one that lies — here it would lie
// by putting an empty card on the rail, which wedges it.
func (b Burst) HasTakeover() bool { return b.Takeover.ID != "" }

// BurstID is the identity of a takeover made from these arrivals, and the ONE
// owner of that rule.
//
// The Producer and the Director must agree on it without either passing it to
// the other: the Director derives the card from the arrivals, and the app has
// to find the same burst again when the card's words are asked for. Two copies
// of "how a burst is named" is two carriers of one rule (D-1) — and the one
// nobody edited is the one that stops matching. So it takes the LEAD's id
// rather than the selection, which the app cannot see.
//
// The lead names it because the rail's order is deterministic and its head is
// the burst's lead. Prefixed so a burst id can never collide with a single
// alert's own id in the same schedule.
func BurstID(lead string) string {
	if lead == "" {
		return ""
	}
	return "burst:" + lead
}

// takeoverOf is the ONE card the schedule holds for this burst (MVS-D-77).
//
// The selection above is the Producer's ORDERING — which alerts are read, in
// what order, and how many were diverted — and every rule that governs it (the
// ladder, emergency orders leading, the fence, freshness, ties) is asserted
// against it. That observability is worth keeping: those are safety rules with
// their own UAT history, and collapsing them into one card would leave forty
// pins with nothing to look at.
//
// What is ONE is what the SCHEDULE holds. The tone, the header, the alert lines
// and the tail are this card's content, composed by the Composer; the operator
// promotes or drops the burst, and there is nothing inside it to address
// separately. The Broadcaster mock draws it that way — one panel, one slot
// number — and DROP / DELAY / PROMOTE only make sense against it.
//
// Its words are still empty: they materialise at standby (DR-7), which for a
// takeover means composed just before air rather than when the alerts arrived.
func takeoverOf(p []placed) (Card, error) {
	if len(p) == 0 {
		return Card{}, nil
	}
	// The SUBJECT is the burst's lead, and the HEADLINE says how many follow —
	// the operator reads the card before its script exists.
	head := p[0].arrival
	headline := head.Headline
	if len(p) > 1 {
		headline = fmt.Sprintf("%s + %d more", headline, len(p)-1)
	}
	// THE ORDER TRAVELS ON THE CARD (MVS-D-77). The Director planned it and the
	// Composer needs it; passing it any other way would mean a second planner.
	refs := make([]string, 0, len(p)) // bounded by the selection (P10-02)
	for _, c := range p {
		refs = append(refs, c.arrival.ID)
	}
	card, err := Propose(Card{ID: BurstID(head.ID), Slot: BreakingAlert, Origin: FromObserver,
		Subject: head.Subject, Headline: headline, Refs: refs})
	if err != nil {
		return Card{}, err
	}
	if card, err = card.To(Admitted); err != nil {
		return Card{}, err
	}
	return card, nil
}
