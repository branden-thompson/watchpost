package render

import "fmt"

// quattro.go — additional built-in themes mapped from the Omarchy Quattro
// palettes (github.com/basecamp/omarchy, quattro branch). Each Quattro theme
// ships one uniform, semantic colors.toml (accent, background/foreground
// variants, and the named hues); this maps that palette onto Watchpost's
// tokens systematically, in truecolor, so a whole family of themes shares one
// derivation. HUM LEAD directs the final colours (the palettes are faithful to
// upstream; per-token nudges are a colour pass). Dark palettes only for now —
// the light Quattro themes need the (still-placeholder) light window path.

// quattroPalette is the subset of a Quattro colors.toml this mapping reads.
type quattroPalette struct {
	accent, selection, muted   string
	bg, darkBg, lightBg        string
	fg, darkFg, brightFg       string
	red, yellow, orange, green string
	cyan, blue, magenta        string
	brightRed, brightYellow    string
}

// qhex parses "#rrggbb" into components; a malformed value reads as black.
func qhex(h string) (r, g, b int) {
	if len(h) != 7 || h[0] != '#' {
		return 0, 0, 0
	}
	nib := func(c byte) int {
		switch {
		case c >= '0' && c <= '9':
			return int(c - '0')
		case c >= 'a' && c <= 'f':
			return int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			return int(c-'A') + 10
		}
		return 0
	}
	return nib(h[1])<<4 | nib(h[2]), nib(h[3])<<4 | nib(h[4]), nib(h[5])<<4 | nib(h[6])
}

// fgTruecolor / bgTruecolor emit a fully-qualified truecolor SGR list (the raw
// path drops bare palette indices — TestCompositeThemeTokensAreLegalRawSGR).
func fgTruecolor(h string) string { r, g, b := qhex(h); return fmt.Sprintf("38;2;%d;%d;%d", r, g, b) }
func boldFg(h string) string      { return "1;" + fgTruecolor(h) }
func bgTruecolor(h string) string { r, g, b := qhex(h); return fmt.Sprintf("48;2;%d;%d;%d", r, g, b) }

// chip is a composite key-cap style: an optionally-bold truecolor foreground on
// a truecolor background.
func chip(fgHex, bgHex string, bold bool) string {
	pre := ""
	if bold {
		pre = "1;"
	}
	return pre + fgTruecolor(fgHex) + ";" + bgTruecolor(bgHex)
}

// blend mixes a toward b (t of a, 1-t of b) → the interpolated components.
func blend(a, b string, t float64) (int, int, int) {
	ar, ag, ab := qhex(a)
	br, bg, bb := qhex(b)
	c := func(x, y int) int { return int(float64(x)*t + float64(y)*(1-t)) }
	return c(ar, br), c(ag, bg), c(ab, bb)
}

// mix is blend as a background tile (a hue toward the dark base) — the group and
// alert backgrounds; fgMix is blend as a foreground (lifting a hue toward the
// bright text so a header stays AA on lighter theme backgrounds).
func mix(hue, base string, t float64) string {
	r, g, b := blend(hue, base, t)
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

func fgMix(a, toward string, t float64) string {
	r, g, b := blend(a, toward, t)
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

// quattroOverrides maps a Quattro palette onto Watchpost's tokens. Foregrounds
// ride the palette's own colours; group/alert tiles are the hues mixed toward
// the dark background; the ticker severity backgrounds are theme-independent
// (left to inherit the fixed Red/Orange/Yellow/Blue). The six AA-checked tokens
// (TextBase/Bright, TableHeader/Muted/Name, ModalTitle) stay pure foregrounds
// on the palette's readable colours.
func quattroOverrides(p quattroPalette) map[Token]string {
	m := quattroForegrounds(p)
	for k, v := range quattroTiles(p) {
		m[k] = v // the composite/background/hex tokens
	}
	return m
}

// quattroForegrounds are the palette's plain-foreground tokens (the AA-checked
// text and the semantic hues ride the palette's own colours).
func quattroForegrounds(p quattroPalette) map[Token]string {
	return map[Token]string{
		TextBase:         fgTruecolor(p.fg),
		TextBright:       fgTruecolor(p.brightFg),
		TempHi:           fgTruecolor(p.orange),
		TempLo:           fgTruecolor(p.cyan),
		TrendUp:          fgTruecolor(p.orange),
		TrendDown:        fgTruecolor(p.cyan),
		FocusName:        boldFg(p.accent),
		FocusCell:        fgTruecolor(p.blue),
		FocusPointer:     boldFg(p.brightFg),
		NameAdvisory:     fgTruecolor(p.yellow),
		NameWarning:      fgTruecolor(p.red),
		ProviderOK:       fgTruecolor(p.green),
		ProviderDown:     fgTruecolor(p.brightRed),
		GroupText:        boldFg(p.brightFg),
		AlertWarnFG:      fgTruecolor(p.red),
		AlertAdvFG:       fgTruecolor(p.yellow),
		AlertLabel:       fgTruecolor(p.yellow),
		AlertDanger:      fgTruecolor(p.red),
		RadioAccent:      fgTruecolor(p.accent),
		StateStopped:     boldFg(p.muted),
		StatePlaying:     boldFg(p.green),
		RadioStation:     boldFg(p.brightYellow),
		RepeatOn:         boldFg(p.yellow),
		VizOn:            boldFg(p.green),
		SpectrumLow:      fgTruecolor(p.green),
		SpectrumMid:      fgTruecolor(p.yellow),
		SpectrumHigh:     fgTruecolor(p.red),
		FireMark:         fgTruecolor(p.orange),
		SeismicMark:      fgTruecolor(p.magenta),
		TableHeader:      fgMix(p.magenta, p.brightFg, 0.7), // a lifted magenta — the light-purple header, kept AA on lighter backgrounds (e.g. Nord)
		TableMuted:       fgTruecolor(p.fg),
		TableName:        fgTruecolor(p.brightFg),
		AlertModalWarnFG: fgTruecolor(p.red),
		AlertModalAdvFG:  fgTruecolor(p.yellow),
		AlertModalText:   fgTruecolor(p.brightFg),
		ModalTitle:       boldFg(p.brightFg),
		ModalFG:          fgTruecolor(p.fg),
	}
}

// quattroTiles are the composite chips, the tinted-dark group/alert tiles, the
// modal/window backgrounds, and the title gradient (hex).
func quattroTiles(p quattroPalette) map[Token]string {
	return map[Token]string{
		KeyChip:          chip(p.brightFg, p.muted, true),
		KeyChipMuted:     chip(p.darkFg, p.selection, false),
		ChipFlashUp:      chip(p.bg, p.green, true),
		ChipFlashDown:    chip(p.bg, p.red, true),
		GroupLocationBG:  bgTruecolor(p.selection),
		GroupTodayBG:     mix(p.blue, p.darkBg, 0.45),
		GroupTomorrowBG:  mix(p.cyan, p.darkBg, 0.45),
		GroupExtendedBG:  mix(p.magenta, p.darkBg, 0.45),
		GroupSectionBG:   bgTruecolor(p.darkBg),
		ConfirmBG:        mix(p.red, p.darkBg, 0.40),
		AlertModalWarnBG: mix(p.red, p.darkBg, 0.30),
		AlertModalAdvBG:  mix(p.yellow, p.darkBg, 0.25),
		ModalBGDark:      bgTruecolor(p.darkBg),
		ModalBGLight:     bgTruecolor(p.lightBg),
		WindowBGDark:     p.bg,      // hex — the theme's own background
		GradStart:        p.magenta, // the W A T C H P O S T wordmark, on-palette
		GradMid:          p.accent,
		GradEnd:          p.green,
	}
}

// quattroThemes are the dark Omarchy Quattro palettes shipped as built-ins
// (a function, not a global — P10-06). Names become the [t] chooser's entries.
func quattroThemes() map[string]quattroPalette {
	return map[string]quattroPalette{
		"Tokyo Night": {
			accent: "#7aa2f7", selection: "#292e42", muted: "#414868",
			bg: "#1a1b26", darkBg: "#13141c", lightBg: "#24283b",
			fg: "#a9b1d6", darkFg: "#565f89", brightFg: "#c0caf5",
			red: "#f7768e", yellow: "#e0af68", orange: "#eb927b", green: "#9ece6a",
			cyan: "#449dab", blue: "#7aa2f7", magenta: "#ad8ee6",
			brightRed: "#ff7a93", brightYellow: "#ff9e64",
		},
		"Gruvbox": {
			accent: "#7daea3", selection: "#504945", muted: "#665c54",
			bg: "#282828", darkBg: "#1e1e1e", lightBg: "#3c3836",
			fg: "#d4be98", darkFg: "#7c6f64", brightFg: "#d4be98",
			red: "#ea6962", yellow: "#d8a657", orange: "#e1875c", green: "#a9b665",
			cyan: "#89b482", blue: "#7daea3", magenta: "#d3869b",
			brightRed: "#ea6962", brightYellow: "#d8a657",
		},
		"Nord": {
			accent: "#81a1c1", selection: "#434c5e", muted: "#4c566a",
			bg: "#2e3440", darkBg: "#222730", lightBg: "#3b4252",
			fg: "#d8dee9", darkFg: "#667080", brightFg: "#eceff4",
			red: "#bf616a", yellow: "#ebcb8b", orange: "#d5967a", green: "#a3be8c",
			cyan: "#88c0d0", blue: "#81a1c1", magenta: "#b48ead",
			brightRed: "#bf616a", brightYellow: "#ebcb8b",
		},
		"Catppuccin": {
			accent: "#89b4fa", selection: "#45475a", muted: "#585b70",
			bg: "#1e1e2e", darkBg: "#161622", lightBg: "#313244",
			fg: "#cdd6f4", darkFg: "#6c7086", brightFg: "#cdd6f4",
			red: "#f38ba8", yellow: "#f9e2af", orange: "#f6b6ab", green: "#a6e3a1",
			cyan: "#94e2d5", blue: "#89b4fa", magenta: "#f5c2e7",
			brightRed: "#f38ba8", brightYellow: "#f9e2af",
		},
		"Everforest": {
			accent: "#7fbbb3", selection: "#3d484d", muted: "#475258",
			bg: "#2d353b", darkBg: "#21272c", lightBg: "#343f44",
			fg: "#d3c6aa", darkFg: "#859289", brightFg: "#d3c6aa",
			red: "#e67e80", yellow: "#dbbc7f", orange: "#e09d7f", green: "#a7c080",
			cyan: "#83c092", blue: "#7fbbb3", magenta: "#d699b6",
			brightRed: "#e67e80", brightYellow: "#dbbc7f",
		},
		"Kanagawa": {
			accent: "#7e9cd8", selection: "#363646", muted: "#54546d",
			bg: "#1f1f28", darkBg: "#17171e", lightBg: "#223249",
			fg: "#dcd7ba", darkFg: "#727169", brightFg: "#dcd7ba",
			red: "#c34043", yellow: "#c0a36e", orange: "#c17158", green: "#76946a",
			cyan: "#6a9589", blue: "#7e9cd8", magenta: "#957fb8",
			brightRed: "#e82424", brightYellow: "#e6c384",
		},
		"Osaka Jade": {
			accent: "#509475", selection: "#32473b", muted: "#53685b",
			bg: "#111c18", darkBg: "#0c1512", lightBg: "#23372b",
			fg: "#c1c497", darkFg: "#81b8a8", brightFg: "#f7e8b2",
			red: "#ff5345", yellow: "#e5c736", orange: "#a2734b", green: "#63b07a",
			cyan: "#2dd5b7", blue: "#8cd3cb", magenta: "#d2689c",
			brightRed: "#ff5345", brightYellow: "#e5c736",
		},
	}
}
