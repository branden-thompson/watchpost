// Semantic color tokens (UAT 16.4): every color the UI emits resolves
// through the theme table below — the theming surface. Swapping palettes
// (the theme picker in Settings, CLIAmp-style) replaces this table; call sites
// never change. Values are SGR parameter lists (256-palette codes ride
// go-studs ColorSequence; truecolor entries are pre-expanded 38;2/48;2
// params for the raw-SGR path; window/gradient entries are #RRGGBB hex).
package render

// Token names a semantic color role.
type Token string

// The token vocabulary. Add roles, not colors: a new UI element gets a new
// token here (and a value in every theme), never an inline code.
const (
	TextBase   Token = "text.base"   // default body text
	TextBright Token = "text.bright" // emphasized plain text (timestamps)

	TempHi    Token = "temp.hi"    // high temperatures
	TempLo    Token = "temp.lo"    // low temperatures
	TrendUp   Token = "trend.up"   // ↗ (muted orange)
	TrendDown Token = "trend.down" // ↘ (muted cyan)

	FocusName    Token = "name.focus"       // focused row name
	FocusCell    Token = "cell.focus"       // focused row: grey data cells read light blue (UAT 50.1)
	FocusPointer Token = "pointer.focus"    // focused row pointer: bold white (UAT 50.2)
	ListPointer  Token = "pointer.list"     // a LIST's focused row: bold yellow — see render/list.go for why this is not FocusPointer
	ListFocus    Token = "label.list.focus" // and its label: the same yellow, NOT bold
	NameAdvisory Token = "name.advisory"    // location under advisory/statement
	NameWarning  Token = "name.warning"     // location under warning/alert

	ProviderOK   Token = "provider.ok"
	ProviderDown Token = "provider.down"

	KeyChip       Token = "chip.key"        // key-cap chips (composite style)
	KeyChipMuted  Token = "chip.key.muted"  // disabled controls: ~50% opacity (UAT 21.1)
	ChipFlashUp   Token = "chip.flash.up"   // [+] acknowledged: green blink (UAT 41)
	ChipFlashDown Token = "chip.flash.down" // [-] acknowledged: red blink
	GroupText     Token = "group.text"      // group band text (composite style)

	GroupLocationBG Token = "group.location.bg"
	GroupTodayBG    Token = "group.today.bg"
	GroupTomorrowBG Token = "group.tomorrow.bg"
	GroupExtendedBG Token = "group.extended.bg"
	GroupSectionBG  Token = "group.section.bg" // RECENT / SEARCHED section band (UAT 43)

	// The Broadcaster console's own grounds (D-86, HUM LEAD 2026-09-11).
	//
	// THEY ARE THE CONSOLE'S, NOT BORROWED, and that is the whole reason they
	// exist. The rail names a REGION — live, next, queued — while the alert
	// cards beside it are painted by HAZARD CATEGORY (`category.Of(c).Tint`,
	// the same tints the [w] window uses). Borrowing a Ticker lane for "UP
	// NEXT" would make one colour mean "this is an Advisory" in one column and
	// "this is queued" in the next.
	//
	// THE VALUES STAY IN THE REGION-BAND FAMILY, though, because that is the
	// language Observer already speaks: the Group bands are muted 66/94/122
	// mixes, and these are the SAME three channel values permuted — red
	// dominant, mid, blue dominant. Equal perceived weight by construction,
	// which is what keeps the rail reading as a rail rather than as a warning,
	// and what gives the AA register one answer for all three.
	RailLiveBG  Token = "bc.rail.live.bg"  // LIVE — the app's red, at band weight
	RailNextBG  Token = "bc.rail.next.bg"  // UP NEXT — orange
	RailQueueBG Token = "bc.rail.queue.bg" // SCHEDULED / LINE UP — blue; GroupTodayBG's own value

	// A main-track card's ground, by ORIGIN (D-86). NARROW, BY RULING: one for
	// the cards the station proposed for itself and one for the cards the
	// OPERATOR asked for, rather than a ground per slot — most slots do not
	// exist yet (D-31), and the report type is already in the card's title.
	//
	// WARM MEANS YOURS. The operator's own card is the same lightness in a warm
	// cast, so it reads as "mine" without changing what the card IS — and
	// yellow is the one hue the rail does not use, so an operator's card can
	// never be mistaken for a rail state.
	CardBG         Token = "bc.card.bg"          // proposed by the station
	CardOperatorBG Token = "bc.card.operator.bg" // asked for by the operator

	// CardText is what a card's own words are painted in, and it exists so the
	// AA lift stays inside the console.
	//
	// MEASURED, NOT ASSUMED. `withAA` lifts a foreground until it reads on EVERY
	// ground it is registered against, so registering `TextBase` against these
	// grounds moved it in two themes — Observer's tables went with it. A colour
	// the operator's console introduced must not change the listener's. So the
	// cards carry a tone of their own, and lifting it reaches nothing else.
	CardText Token = "bc.card.text"

	// CardEmptyBG is the LIVE slot with nothing on the air (D-89).
	//
	// GREY BECAUSE GREY IS WHAT DORMANT LOOKS LIKE (HUM LEAD, 2026-09-11: "empty
	// state needs to be a grey box"). A slot painted in the card family would
	// read as a card the operator cannot make out; a slot painted in the rail's
	// red would say the station is live. It is neither a card nor a state, so it
	// wears the one colour that claims nothing.
	//
	// IT IS ITS OWN TOKEN AND NOT `GroupSectionBG`, which is the nearest grey in
	// the set: that token means "a section header on Observer", and borrowing it
	// would mean the console's empty slot moved whenever somebody retuned
	// Observer's bands. One meaning, one token.
	CardEmptyBG Token = "bc.card.empty.bg"

	AlertLabel  Token = "alert.label"  // watch/advisory yellow (panel tints)
	AlertDanger Token = "alert.danger" // warning red (panel tints, provider down)

	RadioFG      Token = "radio.fg"
	RadioBG      Token = "radio.bg"
	RadioAccent  Token = "radio.accent"        // title, VOL fill
	StateStopped Token = "radio.stopped"       // ■ STOPPED
	StatePlaying Token = "radio.playing"       // playing/paused (B4)
	RadioStation Token = "radio.station"       // location name in the player: bold bright yellow (UAT 40.4, CLIAmp)
	RepeatOn     Token = "radio.repeat.on"     // 'Repeat: On' label: yellow bold (UAT 52.1)
	VizOn        Token = "radio.viz.on"        // 'Viz: On' label: green bold (UAT 52.2)
	SpectrumLow  Token = "radio.spectrum.low"  // visualizer gradient: bottom third / quiet bands (UAT 92, CLIAmp)
	SpectrumMid  Token = "radio.spectrum.mid"  // middle third / mid bands
	SpectrumHigh Token = "radio.spectrum.high" // top third / loud bands

	FireMark Token = "fire.mark" // ▲ in the row marks and the FIRE section: orange (B5)

	SeismicMark Token = "seismic.mark" // ○●◉ in the SEISMIC section: violet — distinct from fire's orange (0.11.0)

	// The global event ticker's severity backgrounds (0.12.0): fixed Red /
	// Orange / Yellow that are theme-INDEPENDENT (the same across every theme
	// except monochrome, which renders them greyscale) — set on bare :root and
	// overridden only under Monochrome. TickerFG / TickerMutedFG go with them; the empty band matches GroupSectionBG.
	TickerDisasterBG Token = "ticker.disaster.bg"
	TickerWarningBG  Token = "ticker.warning.bg"
	TickerWatchBG    Token = "ticker.watch.bg"
	TickerMarineBG   Token = "ticker.marine.bg" // 0.12.0: the Tropical Cyclones lane (HUM LEAD colour pass)
	// 0.14.0: the two lanes that joined when every severe category gained one.
	// They are LANE tokens, not category tints borrowed from the [w] window: a
	// band is painted under bold white at full strength, a tint sits behind a
	// row, and the AA register checks the two differently (HUM LEAD, UAT
	// 2026-08-30).
	TickerAdvisoryBG  Token = "ticker.advisory.bg"  // Advisories — burnt orange, away from the Watch gold beside it
	TickerEmergencyBG Token = "ticker.emergency.bg" // Emergency Orders — THE RED (MVS-D-62)
	TickerStatementBG Token = "ticker.statement.bg" // Spec. Statements — teal, the one lane with no warm neighbour
	TickerFG          Token = "ticker.fg"
	TickerMutedFG     Token = "ticker.muted.fg"

	// The severe-events window's category tints (0.13.0, SAM-D-7): fixed,
	// pre-darkened hues keyed to the ticker lanes — Red disasters, Orange
	// warnings, Yellow watches/advisories/statements, Blue tropical — rendered
	// by CategoryTone onto the active modal substrate, so they read the same in
	// every theme (Monochrome greys them). Values: HUM LEAD's colour pass.
	EventCatDisasterBG  Token = "event.cat.disaster.bg"
	EventCatWarningBG   Token = "event.cat.warning.bg"
	EventCatAdvisoryBG  Token = "event.cat.advisory.bg"  // Advisories
	EventCatWatchBG     Token = "event.cat.watch.bg"     // Watches — a touch more yellow than Advisories
	EventCatStmtBG      Token = "event.cat.statement.bg" // Spec. Statements — a touch more green than Advisories
	EventCatEmergencyBG Token = "event.cat.emergency.bg" // Emergency Orders — THE RED (MVS-D-62); Disasters takes purple
	EventCatForecastBG  Token = "event.cat.forecast.bg"  // Forecasts and Outlooks — COLOUR IS THE HUM LEAD'S PASS
	EventCatMarineBG    Token = "event.cat.marine.bg"

	// The table's own palette (quality pass Q4a-004, L5-F4): before it the
	// kit painted these from its $TERM-gated palette, outside the theme.
	TableMuted Token = "table.muted" // row numbers and attribute cells (the kit's muted grey)
	TableName  Token = "table.name"  // an unselected, un-alerted NAME cell (the kit's bright white)

	ConfirmBG Token = "confirm.bg" // destructive-action confirmation tile (UAT 26.2)

	AlertModalWarnFG Token = "alert.modal.warning.fg"  // modal alert text, warning-grade (UAT 28.4)
	AlertModalAdvFG  Token = "alert.modal.advisory.fg" // modal alert text, advisory-grade (UAT 28.3)

	AlertModalText   Token = "alert.modal.text"       // [A] modal body text: white for contrast (UAT 55)
	AlertModalWarnBG Token = "alert.modal.warning.bg" // [A] details tile, warning-grade
	AlertModalAdvBG  Token = "alert.modal.advisory.bg"

	ModalTitle   Token = "modal.title" // a floating window's title: bold white against the tile
	ModalFG      Token = "modal.fg"
	ModalBGDark  Token = "modal.bg.dark"
	ModalBGLight Token = "modal.bg.light"

	WindowBGDark  Token = "window.bg.dark"  // hex
	WindowBGLight Token = "window.bg.light" // hex

	GradStart Token = "title.grad.start" // hex — gradient interpolation stops
	GradMid   Token = "title.grad.mid"
	GradEnd   Token = "title.grad.end"

	// TitleEdition is the EDITION word beside the wordmark — "Observer"
	// today, "Broadcaster" when that dashboard arrives (0.14.0). Bold, in the
	// theme's own light blue, so the edition reads as a companion to the
	// gradient rather than a second wordmark competing with it.
	//
	// Its own token, not FocusCell borrowed: the two happen to share a colour
	// in most themes and have nothing to do with each other, and the second
	// edition will want to differ.
	TitleEdition Token = "title.edition"
)

// defaultTheme is the HUM-LEAD-directed B3 palette, built fresh on each
// call: the theme registry (themes.go) owns the active table and every
// registered theme copies from this one (quality pass Q1, L3-F17 — no
// package-level map to guard).
func defaultTheme() map[Token]string {
	return withAA(map[Token]string{
		TextBase:   "250",
		TextBright: "97",

		TempHi:    "208",
		TempLo:    "51",
		TrendUp:   "137", // ~#A98D40
		TrendDown: "73",  // ~#409FA9

		FocusName:    "1;220",
		FocusCell:    "117", // light blue
		FocusPointer: "1;97",
		// A LIST's focus is its own pair of tokens, deliberately NOT the
		// table's. See render.ListMark for why the two differ.
		ListPointer:  "1;220",
		ListFocus:    "220",
		NameAdvisory: "186", // ~#D0CF89
		NameWarning:  "174", // ~#D08989

		ProviderOK:   "77",
		ProviderDown: "196",

		KeyChip:       "1;97;48;2;86;86;86",     // bg #565656 (UAT 18.4: contrast)
		KeyChipMuted:  "38;5;245;48;2;43;43;43", // half-tone text + bg #2b2b2b (UAT 21.1)
		ChipFlashUp:   "1;97;48;5;28",           // bold white on green
		ChipFlashDown: "1;97;48;5;124",          // bold white on red
		GroupText:     "1;97",

		GroupLocationBG: "48;2;97;97;97",
		GroupTodayBG:    "48;2;66;94;122",
		GroupTomorrowBG: "48;2;66;122;122",
		GroupExtendedBG: "48;2;94;94;122",

		// The console's grounds. The rail's three are GroupTodayBG's own channel
		// values permuted — see the token declarations for why they are a family
		// rather than three picks.
		RailLiveBG:     "48;2;122;66;66",
		RailNextBG:     "48;2;122;94;66",
		RailQueueBG:    "48;2;66;94;122",
		CardBG:         "48;2;36;36;42",
		CardOperatorBG: "48;2;54;48;36",
		CardText:       "250",
		CardEmptyBG:    "48;2;58;58;58", // a neutral grey, lighter than the card family
		GroupSectionBG: "48;2;34;34;34", // #222 (UAT 44.2)

		AlertLabel:  "220",
		AlertDanger: "196",

		RadioFG:      "38;5;250",
		RadioBG:      "49", // UAT 18.5: hidden for evaluation; re-enable value: 48;2;48;48;48 (#303030)
		RadioAccent:  "77",
		StateStopped: "1;245",
		StatePlaying: "1;77",
		RadioStation: "1;226",
		RepeatOn:     "1;220",
		VizOn:        "1;77",
		SpectrumLow:  "77",  // green
		SpectrumMid:  "220", // yellow
		SpectrumHigh: "196", // red
		FireMark:     "208", // orange
		SeismicMark:  "141", // violet (0.11.0): distinct from fire-orange 208, alert-red 196, advisory-yellow 220

		// The ticker severity backgrounds (0.12.0), fixed across themes (bold
		// white text on a saturated tile; the greyscale variants live under
		// Monochrome only).
		TickerDisasterBG: "48;2;140;20;110", // #8C146E — Disasters
		TickerWarningBG:  "48;2;160;85;15",  // amber
		TickerWatchBG:    "48;2;150;125;20", // dark gold
		TickerMarineBG:   "48;2;20;70;150",  // deep blue (Tropical Cyclones — HUM LEAD colour pass)

		TickerAdvisoryBG:  "48;2;129;60;14",  // #813C0E
		TickerStatementBG: "48;2;25;105;102", // #196966
		// PLACEHOLDER, pending the ruling: a new colour, or THE RED with every
		// other lane shifted down. Magenta only so it is unmistakably not final.
		TickerEmergencyBG: "48;2;150;20;20",     // #961414 — Emergency Orders: THE red
		TickerFG:          "1;38;2;255;255;255", // bold white
		TickerMutedFG:     "38;5;245",           // muted grey text

		// The [w] window's row tints MIRROR the lanes: the same hue and saturation
		// at 60 % of the lane's lightness (HUM LEAD, UAT 2026-08-30 — "so they
		// don't visually scream or bleed"). A band is glanced at across a dark
		// frame; a tint sits under a row somebody is reading, so it has to step
		// back without becoming a different colour.
		EventCatDisasterBG: "48;2;96;14;76", // #600E4C — Disasters
		EventCatWarningBG:  "48;2;94;43;10", // #5E2B0A — Warnings
		EventCatWatchBG:    "48;2;68;56;9",  // #443809 — Watches
		EventCatAdvisoryBG: "48;2;77;41;7",  // #4D2907 — Advisories
		EventCatStmtBG:     "48;2;13;53;51", // #0D3533 — Spec. Statements
		// Forecasts sit BELOW advisories, and the only tint in the set with no
		// lane above it — there is no Forecasts band on the ticker. So it is the
		// one that carries no hue: a neutral slate, a whisper of cool so it is
		// not mistaken for the table's own ground, and the least chroma in the
		// set because it is the least urgent thing the window shows.
		EventCatForecastBG:  "48;2;46;50;56",  // #2E3238 — Forecasts and Outlooks
		EventCatEmergencyBG: "48;2;114;15;15", // #720F0F — Emergency Orders
		EventCatMarineBG:    "48;2;16;55;118", // #103776 — Marine

		TableMuted: "245", // = tui.TableRowNumber / tui.TableAttribute
		TableName:  "97",  // = tui.TableLabel

		ConfirmBG: "48;2;79;12;12", // #4F0C0C — deep red under light text (UAT 109; was #AE7D7E, UAT 26.2)

		AlertModalWarnFG: "38;2;190;84;84",   // #BE5454 (UAT 28.4)
		AlertModalAdvFG:  "38;2;172;174;125", // #ACAE7D (UAT 28.3)

		AlertModalText:   "97",           // white (UAT 55)
		AlertModalWarnBG: "48;2;40;0;0",  // muted red tile (UAT 22)
		AlertModalAdvBG:  "48;2;32;28;0", // muted yellow tile (UAT 22)

		ModalTitle:   "1;97",
		ModalFG:      "38;5;250",
		ModalBGDark:  "48;2;29;40;48", // #1D2830
		ModalBGLight: "48;2;29;40;48", // a dark theme paints its own tile whatever the terminal reports (HUM LEAD 2026-08-29: light terminals are served by the Watchpost Light theme)

		WindowBGDark:  "#131313",
		WindowBGLight: "#131313", // as above: the ground is the theme's, not the terminal's

		GradStart: "#DD51D6", // the reference-CLI gradient
		GradMid:   "#378FE9",
		GradEnd:   "#7CE3B3",

		TitleEdition: "1;117", // bold light blue — 117 is what this palette already calls light blue (FocusCell)
	})
}

// Tok resolves a semantic token to its SGR params (or hex for window/
// gradient tokens). Resolution happens at render time, so a theme swap
// takes effect on the next frame.
func Tok(t Token) string { return activeTable()[t] }

// hexRGB parses a theme hex entry (#RRGGBB) into components.
func hexRGB(t Token) (r, g, b int) {
	h := activeTable()[t]
	if len(h) != 7 || h[0] != '#' {
		return 0, 0, 0
	}
	v := func(s string) int {
		n := 0
		for _, c := range s {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n += int(c - '0')
			case c >= 'a' && c <= 'f':
				n += int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				n += int(c-'A') + 10
			}
		}
		return n
	}
	return v(h[1:3]), v(h[3:5]), v(h[5:7])
}
