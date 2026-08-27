// Package render is the D-9 pivot seam: the ONLY package allowed to import
// go-studs. The location table rides go-studs' DataTable (spec struct =
// DataTableDefinition/ColumnDefinition, data struct = EnhancedTableRow —
// the reference-CLI pattern, HUM LEAD directive UAT session 2): auto-sizing,
// truncation, and gutters are the component's job; this seam only assembles
// the column spec for the current width (UAT-2D responsive drops + the
// ultra-wide EXTENDED FORECAST columns) and formats watchpost values.
// Rules baked in (architecture §4): severity and health are NEVER color-only
// (text glyphs/labels always present — R-12a); --ascii swaps glyphs without
// dropping signals (RS-14); widths are always explicit; units convert here
// and only here (D-19: f/c live toggle; SI internal).
package render

// The package is split by responsibility (quality pass Q2): units.go,
// table.go (the go-studs seam), sgr.go, panel.go, text.go, theme.go and
// themes.go; this file carries the package documentation.
