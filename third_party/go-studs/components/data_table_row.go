package components

import (
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// ColumnDefinition defines how a column should be rendered in a data table row
type ColumnDefinition struct {
	Name              string // Column name/identifier
	Header            string // Column header text
	ShowName          bool   // Whether to show the column name
	Width             int    // Fixed width (0 = fill column)
	MinWidth          int    // Minimum width for fill columns
	MaxWidth          int    // Maximum width for fill columns (0 = no limit)
	SupportsWrap      bool   // Whether content can wrap within the column width
	Alignment         string // "left", "right", "center"
	Color             string // ANSI color code
	HeaderColor       string // ANSI color code for the header cell alone (Color otherwise; automatic token styling when both are empty)
	Fill              bool   // Whether this column should fill remaining space (alternative to Width=0)
	Truncatable       bool   // Whether this column can be truncated in narrow widths
	TruncatedMinWidth int    // Minimum width this column can be truncated to
	TruncationTail    string // Truncation indicator (default: "...")
	NoLeadingGutter   bool   // Suppress the gutter that would precede this column (pairs a prefix/badge column with its data column)
}

// DataTableDefinition defines the complete table structure with data
type DataTableDefinition struct {
	Columns     []ColumnDefinition
	Rows        []EnhancedTableRow // Row data with optional color overrides and badges
	GutterWidth int                // Space between columns (default: 1, recommended: 4)
	NoAutoStyle bool               // Skip the automatic semantic-token styling of headers and cells: only Color/HeaderColor/CellStyles/CellColors paint (a caller that owns its palette, and honours NO_COLOR through WrapSGR alone)
}

// EnhancedTableRow represents a single table row with presentation overrides
type EnhancedTableRow struct {
	Data       []string       // Cell values
	CellColors map[int]string // Column index -> semantic token override
	CellStyles map[int]string // Column index -> direct ANSI color override
	Badge      string         // Optional badge text (e.g., "(On-Call 1)")
	BadgeColor string         // Optional badge color semantic token
}

// DataTableRow handles rendering of table rows with automatic width calculation
// DataTable represents a complete terminal-width aware data table with contextual row operations
type DataTable struct {
	terminalWidth int
	columns       []ColumnDefinition
	rows          []EnhancedTableRow       // The table's data rows
	gutterWidth   int                      // Space between columns (configurable)
	prefix        string                   // Icon/indicator prefix (e.g., " ⏺ ")
	badge         string                   // Right-aligned badge (e.g., "(On-Call)")
	badgeGutter   int                      // Minimum spaces before badge
	renderBadge   bool                     // Whether to actually render the badge (default true)
	formatter     *rendering.TextFormatter // For automatic semantic token styling
	noAutoStyle   bool                     // DataTableDefinition.NoAutoStyle
	maxBadgeWidth int                      // Pre-calculated maximum badge width for consistent alignment across all rows
}

// Legacy alias for backward compatibility
type DataTableRow = DataTable

// gutterBefore reports whether a gutter precedes column i. With no column
// opting out via NoLeadingGutter this reproduces the legacy positional rule
// (a gutter precedes columns 3..last; none before the prefix/number/name
// zone) byte-identically. A column with NoLeadingGutter set hugs the column
// before it (e.g. a per-port badge slot and its port number).
func (dt *DataTable) gutterBefore(i int) bool {
	if i <= 0 || i >= len(dt.columns) {
		return false
	}
	if dt.columns[i].NoLeadingGutter {
		return false
	}
	return i >= 3
}

// gutterCount returns the number of inter-column gutters the current column
// set produces. Replaces the hardcoded len(columns)-3 in width math; equal
// to max(0, len(columns)-3) whenever no column opts out.
func (dt *DataTable) gutterCount() int {
	n := 0
	for i := range dt.columns {
		if dt.gutterBefore(i) {
			n++
		}
	}
	return n
}

// NewDataTable creates a new table-centric data table with contextual row operations
func NewDataTable(terminalWidth int, definition *DataTableDefinition) *DataTable {
	if terminalWidth < 20 {
		terminalWidth = 20
	}

	// Use definition's gutter width if specified, otherwise default to 1
	gutterWidth := 1
	if definition.GutterWidth > 0 {
		gutterWidth = definition.GutterWidth
	}

	table := &DataTable{
		terminalWidth: terminalWidth,
		columns:       definition.Columns,
		rows:          definition.Rows,
		gutterWidth:   gutterWidth,
		badgeGutter:   4,    // Default 4-space gutter
		renderBadge:   true, // Default to rendering badges
		formatter:     rendering.NewTextFormatter(),
		noAutoStyle:   definition.NoAutoStyle,
	}

	// Automatically calculate badge widths for proper alignment
	table.calculateMaxBadgeWidth()

	return table
}

// NewDataTableRow creates a new data table row renderer
func NewDataTableRow(terminalWidth int, columns []ColumnDefinition) *DataTableRow {
	if terminalWidth < 20 {
		terminalWidth = 20
	}
	return &DataTableRow{
		terminalWidth: terminalWidth,
		columns:       columns,
		gutterWidth:   1,    // Default 1-space gutter (backward compatible)
		badgeGutter:   4,    // Default 4-space gutter
		renderBadge:   true, // Default to rendering badges
		formatter:     rendering.NewTextFormatter(),
	}
}

// NewDataTableRowFromLayout creates a new data table row renderer from a definition
func NewDataTableRowFromLayout(terminalWidth int, definition *DataTableDefinition) *DataTableRow {
	if terminalWidth < 20 {
		terminalWidth = 20
	}

	// Use definition's gutter width if specified, otherwise default to 1
	gutterWidth := 1
	if definition.GutterWidth > 0 {
		gutterWidth = definition.GutterWidth
	}

	return &DataTableRow{
		terminalWidth: terminalWidth,
		columns:       definition.Columns,
		gutterWidth:   gutterWidth,
		badgeGutter:   4,    // Default 4-space gutter
		renderBadge:   true, // Default to rendering badges
		formatter:     rendering.NewTextFormatter(),
	}
}

// SetPrefix sets the prefix (icon/indicator) for the row
func (d *DataTableRow) SetPrefix(prefix string) *DataTableRow {
	d.prefix = prefix
	return d
}

// SetBadge sets the right-aligned badge and optional custom gutter
func (d *DataTableRow) SetBadge(badge string, gutter int) *DataTableRow {
	d.badge = badge
	if gutter > 0 {
		d.badgeGutter = gutter
	}
	return d
}

// SetBadgeRenderMode sets whether badges should be rendered or just reserved for space
func (d *DataTableRow) SetBadgeRenderMode(render bool) *DataTableRow {
	d.renderBadge = render
	return d
}

// RenderHeader renders the table header row
func (d *DataTableRow) RenderHeader() string {
	// Use the same width calculation as data rows, accounting for maximum badge width
	widths := d.calculateColumnWidthsWithBadge([]string{}, d.maxBadgeWidth)

	var headerParts []string
	for i, col := range d.columns {
		width := widths[i]
		header := col.Header

		// Apply alignment and padding FIRST (without colors)
		formatted := d.formatCell(header, width, col.Alignment, true)

		// Apply automatic semantic token colors if no explicit color is set
		if col.HeaderColor != "" {
			// The header's own color
			formatted = rendering.WrapSGR(formatted, col.HeaderColor)
		} else if col.Color != "" {
			// Use explicit color from column definition
			formatted = rendering.WrapSGR(formatted, col.Color)
		} else if header != "" && !d.noAutoStyle { // Only apply colors to non-empty headers
			// Apply automatic semantic token styling for table headers
			formatted = d.formatter.Style(formatted, "tui.TableHeader")
		}

		headerParts = append(headerParts, formatted)
	}

	// Join with configurable gutter width following pattern: [prefix][number][name][gutter][attr1][gutter]...
	var headerLine strings.Builder
	headerLine.WriteString(d.prefix)
	gutter := strings.Repeat(" ", d.gutterWidth)

	for i, part := range headerParts {
		if d.gutterBefore(i) {
			headerLine.WriteString(gutter)
		}
		headerLine.WriteString(part)
	}

	return headerLine.String()
}

// RenderRow renders a data row using the column definitions (legacy method for backward compatibility)
func (d *DataTableRow) RenderRow(data map[string]string) string {
	// Convert map data to slice format for consistency
	var rowData []string
	for _, col := range d.columns {
		value := data[col.Name]               // Use Name instead of Key
		if value == "" && col.Name == "Key" { // Fallback for old Key field
			// Try to find by header name for backward compatibility
			for k, v := range data {
				if k == col.Header {
					value = v
					break
				}
			}
		}
		rowData = append(rowData, value)
	}

	return d.FormatDataRow(rowData)
}

// FormatEnhancedTableRow renders an enhanced TableRow with color overrides and badges, supporting content wrapping
func (d *DataTableRow) FormatEnhancedTableRow(row EnhancedTableRow) string {
	if len(row.Data) != len(d.columns) {
		return "Error: Data length does not match column count"
	}

	// Use the pre-calculated maximum badge width for consistent alignment
	widths := d.calculateColumnWidthsWithBadge(row.Data, d.maxBadgeWidth)

	// Check if any column needs wrapping and prepare wrapped content
	var wrappedColumns [][]string
	maxLines := 1
	needsWrapping := false

	for i, col := range d.columns {
		width := widths[i]
		value := ""
		if i < len(row.Data) {
			value = row.Data[i]
		}

		// Handle wrapping if content is too wide AND column supports wrapping
		if d.getDisplayWidth(value) > width && col.SupportsWrap {
			lines := d.formatter.WrapText(value, width)
			wrappedColumns = append(wrappedColumns, lines)
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
			needsWrapping = true
		} else if col.SupportsWrap && strings.Contains(value, "\n") {
			// If column supports wrapping AND value contains newlines,
			// honor the pre-split lines instead of treating as overflow
			lines := strings.Split(value, "\n")
			wrappedColumns = append(wrappedColumns, lines)
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
			needsWrapping = true
		} else {
			// Handle truncation if content is too wide AND column is truncatable
			if d.getDisplayWidth(value) > width && col.Truncatable {
				truncationTail := col.TruncationTail
				if truncationTail == "" {
					truncationTail = "..." // Default
				}
				value = d.truncateText(value, width, truncationTail)
			}
			// Single line content
			wrappedColumns = append(wrappedColumns, []string{value})
		}
	}

	// If no wrapping is needed, use the original single-line logic
	if !needsWrapping {
		return d.formatSingleLineRow(row, widths)
	}

	// Multi-line rendering for wrapped content
	return d.formatMultiLineRow(row, widths, wrappedColumns, maxLines)
}

// formatSingleLineRow handles the original single-line row formatting logic
func (d *DataTableRow) formatSingleLineRow(row EnhancedTableRow, widths []int) string {
	var rowParts []string
	for i, col := range d.columns {
		width := widths[i]
		value := ""
		if i < len(row.Data) {
			value = row.Data[i]
		}

		// Handle truncation if content is too wide AND column is truncatable
		if d.getDisplayWidth(value) > width && col.Truncatable {
			truncationTail := col.TruncationTail
			if truncationTail == "" {
				truncationTail = "..." // Default
			}
			value = d.truncateText(value, width, truncationTail)
		}

		// Apply alignment and padding
		formatted := d.formatCell(value, width, col.Alignment, false)

		// Apply color styling with override priority: CellStyles > CellColors > Column.Color > Semantic Token
		if row.CellStyles != nil && row.CellStyles[i] != "" {
			// Highest priority: Direct ANSI color override
			formatted = rendering.WrapSGR(formatted, row.CellStyles[i])
		} else if row.CellColors != nil && row.CellColors[i] != "" {
			// Second priority: Semantic token override
			formatted = d.formatter.Style(formatted, row.CellColors[i])
		} else if col.Color != "" {
			// Third priority: Column definition color
			formatted = rendering.WrapSGR(formatted, col.Color)
		} else if value != "" && strings.TrimSpace(value) != "" {
			// Lowest priority: Automatic semantic token styling
			semanticToken := d.getSemanticTokenForColumn(i, col, value)
			if semanticToken != "" {
				formatted = d.formatter.Style(formatted, semanticToken)
			}
		}

		rowParts = append(rowParts, formatted)
	}

	// Join with configurable gutter width
	var rowLine strings.Builder
	gutter := strings.Repeat(" ", d.gutterWidth)

	for i, part := range rowParts {
		if d.gutterBefore(i) {
			rowLine.WriteString(gutter)
		}
		rowLine.WriteString(part)
	}

	finalRowLine := rowLine.String()

	// Add badge if specified
	if row.Badge != "" {
		badgeText := row.Badge
		if row.BadgeColor != "" {
			badgeText = d.formatter.Style(badgeText, row.BadgeColor)
		}
		finalRowLine = d.alignBadge(finalRowLine, badgeText)
	}

	return finalRowLine
}

// formatMultiLineRow handles rendering of rows with wrapped content across multiple terminal lines
func (d *DataTableRow) formatMultiLineRow(row EnhancedTableRow, widths []int, wrappedColumns [][]string, maxLines int) string {
	var resultLines []string

	// Render each line of the multi-line row
	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		var rowParts []string

		for i, col := range d.columns {
			width := widths[i]

			// Get the text for this line and column
			var value string
			if lineIdx < len(wrappedColumns[i]) {
				value = wrappedColumns[i][lineIdx]
			} else {
				value = "" // Empty for columns that don't extend to this line
			}

			// Apply alignment and padding
			formatted := d.formatCell(value, width, col.Alignment, false)

			// Apply color styling to ALL lines (not just the first line)
			// Note: For multi-line content, we apply semantic styling to non-empty content on every line
			if value != "" && strings.TrimSpace(value) != "" {
				// Apply color styling with override priority: CellStyles > CellColors > Column.Color > Semantic Token
				if row.CellStyles != nil && row.CellStyles[i] != "" {
					// Highest priority: Direct ANSI color override
					formatted = rendering.WrapSGR(formatted, row.CellStyles[i])
				} else if row.CellColors != nil && row.CellColors[i] != "" {
					// Second priority: Semantic token override
					formatted = d.formatter.Style(formatted, row.CellColors[i])
				} else if col.Color != "" {
					// Third priority: Column definition color
					formatted = rendering.WrapSGR(formatted, col.Color)
				} else {
					// Lowest priority: Automatic semantic token styling
					semanticToken := d.getSemanticTokenForColumn(i, col, value)
					if semanticToken != "" {
						formatted = d.formatter.Style(formatted, semanticToken)
					}
				}
			}

			rowParts = append(rowParts, formatted)
		}

		// Join with configurable gutter width
		var rowLine strings.Builder
		gutter := strings.Repeat(" ", d.gutterWidth)

		for i, part := range rowParts {
			if d.gutterBefore(i) {
				rowLine.WriteString(gutter)
			}
			rowLine.WriteString(part)
		}

		finalRowLine := rowLine.String()

		// Add badge only on the first line
		if lineIdx == 0 && row.Badge != "" {
			badgeText := row.Badge
			if row.BadgeColor != "" {
				badgeText = d.formatter.Style(badgeText, row.BadgeColor)
			}
			finalRowLine = d.alignBadge(finalRowLine, badgeText)
		}

		resultLines = append(resultLines, finalRowLine)
	}

	return strings.Join(resultLines, "\n")
}

// FormatDataRow renders a data row from a layout's data slice
func (d *DataTableRow) FormatDataRow(rowData []string) string {
	if len(rowData) != len(d.columns) {
		// Handle mismatched data gracefully
		return "Error: Data length does not match column count"
	}

	widths := d.calculateColumnWidthsFromData(rowData)

	var rowParts []string
	for i, col := range d.columns {
		width := widths[i]
		value := ""
		if i < len(rowData) {
			value = rowData[i]
		}

		// Handle truncation if content is too wide AND column is truncatable
		if d.getDisplayWidth(value) > width && col.Truncatable {
			// Use configurable truncation tail from column definition
			truncationTail := col.TruncationTail
			if truncationTail == "" {
				truncationTail = "..." // Default
			}
			value = d.truncateText(value, width, truncationTail)
		}

		// Apply alignment and padding FIRST (without colors)
		formatted := d.formatCell(value, width, col.Alignment, false)

		// Apply automatic semantic token colors based on column position/type
		if col.Color != "" {
			// Use explicit color from column definition
			formatted = rendering.WrapSGR(formatted, col.Color)
		} else if value != "" && strings.TrimSpace(value) != "" {
			// Apply automatic semantic token styling based on column position/name
			semanticToken := d.getSemanticTokenForColumn(i, col, value)
			if semanticToken != "" {
				formatted = d.formatter.Style(formatted, semanticToken)
			}
		}

		rowParts = append(rowParts, formatted)
	}

	// Join with configurable gutter width following pattern: [prefix][number][name][gutter][attr1][gutter]...
	var rowLine strings.Builder
	gutter := strings.Repeat(" ", d.gutterWidth)

	for i, part := range rowParts {
		if d.gutterBefore(i) {
			rowLine.WriteString(gutter)
		}
		rowLine.WriteString(part)
	}

	finalRowLine := rowLine.String()

	// Add badge if specified AND rendering is enabled
	if d.badge != "" && d.renderBadge {
		finalRowLine = d.alignBadge(finalRowLine, d.badge)
	}

	return finalRowLine
}

// calculateColumnWidths determines the width of each column based on terminal width
func (d *DataTableRow) calculateColumnWidths(data map[string]string) []int {
	var widths []int
	var fillColumns []int
	usedWidth := 0

	// Calculate prefix width (without ANSI codes)
	prefixWidth := d.getDisplayWidth(d.prefix)
	usedWidth += prefixWidth

	// Calculate gutter spaces between columns using configurable gutter width
	// Following your design: no gutter at beginning, between ICON/##, or end
	// So for N columns, we have (N-3) gutters assuming first 2 are prefix columns
	gutterCount := d.gutterCount()
	gutterSpace := d.gutterWidth * gutterCount
	usedWidth += gutterSpace

	// Calculate badge space if present
	badgeSpace := 0
	if d.badge != "" {
		badgeSpace = rendering.DisplayWidth(d.badge) + d.badgeGutter
		usedWidth += badgeSpace
	}

	// First pass: calculate fixed column widths and identify fill columns
	for i, col := range d.columns {
		if col.Fill || col.Width == 0 {
			// Fill column - will be calculated later
			widths = append(widths, 0)
			fillColumns = append(fillColumns, i)
		} else {
			// Fixed width column
			widths = append(widths, col.Width)
			usedWidth += col.Width
		}
	}

	// Second pass: distribute remaining width among fill columns
	if len(fillColumns) > 0 {
		remainingWidth := d.terminalWidth - usedWidth
		if remainingWidth > 0 {
			fillWidthPerColumn := remainingWidth / len(fillColumns)

			for _, colIndex := range fillColumns {
				col := d.columns[colIndex]
				width := fillWidthPerColumn

				// Apply minimum width constraint
				if col.MinWidth > 0 && width < col.MinWidth {
					width = col.MinWidth
				}

				// Apply maximum width constraint
				if col.MaxWidth > 0 && width > col.MaxWidth {
					width = col.MaxWidth
				}

				widths[colIndex] = width
			}
		} else {
			// Not enough space - use minimum widths
			for _, colIndex := range fillColumns {
				col := d.columns[colIndex]
				if col.MinWidth > 0 {
					widths[colIndex] = col.MinWidth
				} else {
					widths[colIndex] = 10 // Default minimum
				}
			}
		}
	}

	return widths
}

// calculateColumnWidthsFromData determines column widths from data slice instead of map
func (d *DataTableRow) calculateColumnWidthsFromData(rowData []string) []int {
	var widths []int
	var fillColumns []int
	usedWidth := 0

	// Calculate gutter spaces between columns using configurable gutter width
	// Following your design: no gutter at beginning, between ICON/##, or end
	// So for N columns, we have (N-3) gutters assuming first 2 are prefix columns
	gutterCount := d.gutterCount()
	gutterSpace := d.gutterWidth * gutterCount
	usedWidth += gutterSpace

	// Calculate badge space if present
	if d.badge != "" {
		badgeSpace := rendering.DisplayWidth(d.badge) + d.badgeGutter
		usedWidth += badgeSpace
	}

	// First pass: calculate fixed column widths and identify fill columns
	for i, col := range d.columns {
		if col.Fill || col.Width == 0 {
			// Fill column - will be calculated later
			widths = append(widths, 0)
			fillColumns = append(fillColumns, i)
		} else {
			// Fixed width column
			widths = append(widths, col.Width)
			usedWidth += col.Width
		}
	}

	// Second pass: distribute remaining width among fill columns
	if len(fillColumns) > 0 {
		remainingWidth := d.terminalWidth - usedWidth
		if remainingWidth > 0 {
			fillWidthPerColumn := remainingWidth / len(fillColumns)

			for _, colIndex := range fillColumns {
				col := d.columns[colIndex]
				width := fillWidthPerColumn

				// Apply minimum width constraint
				if col.MinWidth > 0 && width < col.MinWidth {
					width = col.MinWidth
				}

				// Apply maximum width constraint
				if col.MaxWidth > 0 && width > col.MaxWidth {
					width = col.MaxWidth
				}

				widths[colIndex] = width
			}
		} else {
			// Not enough space - use minimum widths
			for _, colIndex := range fillColumns {
				col := d.columns[colIndex]
				if col.MinWidth > 0 {
					widths[colIndex] = col.MinWidth
				} else {
					widths[colIndex] = 10 // Default minimum
				}
			}
		}
	}

	// Third pass: Apply intelligent truncation if table is still too wide
	d.applyIntelligentTruncation(widths, rowData)

	return widths
}

// applyIntelligentTruncation implements the intelligent truncation algorithm
func (d *DataTableRow) applyIntelligentTruncation(widths []int, rowData []string) {
	// Calculate total used width including gutters
	totalUsedWidth := 0
	for _, width := range widths {
		totalUsedWidth += width
	}

	// Add gutter space calculation
	gutterCount := d.gutterCount()
	totalUsedWidth += d.gutterWidth * gutterCount

	// Add badge space if present
	if d.badge != "" {
		totalUsedWidth += rendering.DisplayWidth(d.badge) + d.badgeGutter
	}

	// Check if we need truncation
	if totalUsedWidth <= d.terminalWidth {
		return // No truncation needed
	}

	// Find truncatable columns
	var truncatableColumns []int
	for i, col := range d.columns {
		if col.Truncatable {
			truncatableColumns = append(truncatableColumns, i)
		}
	}

	if len(truncatableColumns) == 0 {
		return // No columns can be truncated
	}

	// Calculate excess width that needs to be trimmed
	excessWidth := totalUsedWidth - d.terminalWidth

	// Distribute truncation across truncatable columns. Every pass takes at
	// least one cell off the excess or drops a column from the list, so the
	// passes are bounded by the excess itself; the explicit cap keeps the
	// loop provably finite (P10-02).
	for pass := 0; pass <= maxTruncationPasses && excessWidth > 0 && len(truncatableColumns) > 0; pass++ {
		truncationPerColumn := maxInt(1, excessWidth/len(truncatableColumns))
		var remainingTruncatable []int

		for _, colIndex := range truncatableColumns {
			col := d.columns[colIndex]
			currentWidth := widths[colIndex]

			// Determine minimum width for this column
			minWidth := col.TruncatedMinWidth
			if minWidth == 0 {
				minWidth = 5 // Default minimum to fit "..." + 2 chars
			}

			// Calculate how much we can truncate this column
			canTruncate := currentWidth - minWidth
			if canTruncate <= 0 {
				continue // This column is already at minimum
			}

			// Truncate this column
			toTruncate := minInt(truncationPerColumn, canTruncate)
			widths[colIndex] -= toTruncate
			excessWidth -= toTruncate

			// Keep this column in the list if it can be truncated further
			if widths[colIndex] > minWidth {
				remainingTruncatable = append(remainingTruncatable, colIndex)
			}
		}

		truncatableColumns = remainingTruncatable
		if len(remainingTruncatable) == 0 {
			break // No more columns can be truncated
		}
	}
}

// minInt returns the smaller of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the larger of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// truncateText truncates text to fit within the specified width with configurable tail
func (d *DataTableRow) truncateText(text string, width int, truncationTail string) string {
	if truncationTail == "" {
		truncationTail = "..." // Default truncation tail
	}

	tailWidth := len(truncationTail)
	if width <= tailWidth {
		return strings.Repeat(".", width)
	}
	if d.getDisplayWidth(text) <= width {
		return text
	}

	// Truncate and add configurable tail
	for i := len(text) - 1; i > 0; i-- {
		truncated := text[:i]
		if d.getDisplayWidth(truncated)+tailWidth <= width {
			return truncated + truncationTail
		}
	}
	return strings.Repeat(".", width)
}

// formatCell formats a cell value with proper alignment and padding
func (d *DataTableRow) formatCell(value string, width int, alignment string, isHeader bool) string {
	displayWidth := d.getDisplayWidth(value)

	if displayWidth >= width {
		// Value is too long - return as-is for basic formatting
		// NOTE: Smart truncation is handled by EnhancedTableLayout's Truncatable/TruncationTail properties
		return value
	}

	padding := width - displayWidth

	switch alignment {
	case "right":
		return strings.Repeat(" ", padding) + value
	case "center":
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + value + strings.Repeat(" ", rightPad)
	default: // "left" or unspecified
		return value + strings.Repeat(" ", padding)
	}
}

// alignBadge aligns the badge to the right edge of the terminal using header/footer logic
func (d *DataTableRow) alignBadge(content, badge string) string {
	// Use the exact same calculation as HeaderFooterComponent
	contentDisplayWidth := d.getDisplayWidth(content)
	badgeDisplayWidth := rendering.DisplayWidth(badge)

	// Calculate total used width: content + badge (no lead/tail like header/footer)
	totalUsedWidth := contentDisplayWidth + badgeDisplayWidth

	// Calculate padding needed to fill terminal width
	paddingNeeded := d.terminalWidth - totalUsedWidth

	// Ensure minimum gutter
	if paddingNeeded < d.badgeGutter {
		paddingNeeded = d.badgeGutter
	}

	return content + strings.Repeat(" ", paddingNeeded) + badge
}

// getDisplayWidth returns the display width of a string, stripping ANSI color codes
// getSemanticTokenForColumn determines the appropriate semantic token for a column value
func (d *DataTableRow) getSemanticTokenForColumn(columnIndex int, col ColumnDefinition, value string) string {
	if d.noAutoStyle {
		return "" // the caller paints every cell it wants painted (DataTableDefinition.NoAutoStyle)
	}
	// Determine semantic token based on column position and name (presentational logic only)
	switch columnIndex {
	case 0:
		// First column (prefix/status icons) - no automatic coloring
		return ""
	case 1:
		// Second column (row numbers like "01.", "02.")
		if strings.Contains(value, ".") {
			return "tui.TableRowNumber"
		}
		return ""
	case 2:
		// Third column (main names/labels)
		return "tui.TableLabel"
	default:
		// Remaining columns (attributes) - determine based on column name and content
		colName := strings.ToLower(col.Name)
		colHeader := strings.ToLower(col.Header)

		// Special handling for specific column types
		if colName == "ldap" || colHeader == "ldap" || strings.HasPrefix(value, "@") {
			return "tui.TableAttribute"
		}
		if colName == "level" || colHeader == "level" || strings.HasPrefix(value, "IC") || strings.HasPrefix(value, "MR") {
			return "tui.TableAttribute"
		}
		if colName == "role" || colHeader == "role" {
			// All roles use the muted grey (245) in the muted style - only on-call gets special colors
			return "tui.TableAttribute"
		}
		if colName == "status" || colHeader == "status" {
			// More muted status coloring like the original - most things stay gray
			statusLower := strings.ToLower(value)
			switch statusLower {
			case "failed", "error":
				return "tui.ErrorText" // Only failures get red
			default:
				return "tui.TableAttribute" // Everything else stays muted grey
			}
		}

		// Default to attribute styling
		return "tui.TableAttribute"
	}
}

// maxTruncationPasses bounds applyIntelligentTruncation: a pass trims at
// least one cell, and no terminal is this wide.
const maxTruncationPasses = 1 << 16

func (d *DataTableRow) getDisplayWidth(text string) int {
	return rendering.DisplayWidth(text)
}

// CalculateColumnWidthsFromData exposes the column width calculation method
func (d *DataTableRow) CalculateColumnWidthsFromData(rowData []string) []int {
	return d.calculateColumnWidthsFromData(rowData)
}

// FormatCell exposes the cell formatting method
func (d *DataTableRow) FormatCell(value string, width int, alignment string, isHeader bool) string {
	return d.formatCell(value, width, alignment, isHeader)
}

// SetMaxBadgeWidthFromRows analyzes all enhanced rows to find the maximum badge width
func (d *DataTableRow) SetMaxBadgeWidthFromRows(rows []EnhancedTableRow) {
	maxWidth := 0
	for _, row := range rows {
		if row.Badge != "" {
			badgeWidth := d.getDisplayWidth(row.Badge) + d.badgeGutter
			if badgeWidth > maxWidth {
				maxWidth = badgeWidth
			}
		}
	}
	d.maxBadgeWidth = maxWidth
}

// calculateColumnWidthsWithBadge determines column widths with explicit badge width parameter
func (d *DataTableRow) calculateColumnWidthsWithBadge(rowData []string, badgeWidth int) []int {
	var widths []int
	var fillColumns []int
	usedWidth := 0

	// Calculate gutter spaces between columns
	gutterCount := d.gutterCount()
	gutterSpace := d.gutterWidth * gutterCount
	usedWidth += gutterSpace

	// Use the provided badge width
	usedWidth += badgeWidth

	// First pass: calculate fixed column widths and identify fill columns
	for i, col := range d.columns {
		if col.Fill || col.Width == 0 {
			// This is a fill column - we'll calculate its width later
			fillColumns = append(fillColumns, i)
			widths = append(widths, 0) // Placeholder
		} else {
			// Fixed width column
			widths = append(widths, col.Width)
			usedWidth += col.Width
		}
	}

	// Second pass: distribute remaining width among fill columns
	remainingWidth := d.terminalWidth - usedWidth
	if remainingWidth > 0 && len(fillColumns) > 0 {
		// Check for minimum width constraints on fill columns
		totalMinWidth := 0
		for _, colIndex := range fillColumns {
			col := d.columns[colIndex]
			minWidth := col.MinWidth
			if minWidth == 0 {
				minWidth = 10 // Default minimum
			}
			totalMinWidth += minWidth
		}

		if remainingWidth >= totalMinWidth {
			// Distribute remaining width proportionally
			fillWidth := remainingWidth / len(fillColumns)
			remainder := remainingWidth % len(fillColumns)

			for i, colIndex := range fillColumns {
				assignedWidth := fillWidth
				if i == 0 {
					assignedWidth += remainder // Give remainder to first fill column
				}

				// Ensure minimum width is respected
				col := d.columns[colIndex]
				minWidth := col.MinWidth
				if minWidth == 0 {
					minWidth = 10
				}
				if assignedWidth < minWidth {
					assignedWidth = minWidth
				}

				widths[colIndex] = assignedWidth
			}
		} else {
			// Not enough space - use minimum widths
			for _, colIndex := range fillColumns {
				col := d.columns[colIndex]
				minWidth := col.MinWidth
				if minWidth == 0 {
					minWidth = 10
				}
				widths[colIndex] = minWidth
			}
		}
	} else {
		// No remaining width or no fill columns - use minimum widths for fill columns
		for _, colIndex := range fillColumns {
			col := d.columns[colIndex]
			minWidth := col.MinWidth
			if minWidth == 0 {
				minWidth = 10
			}
			widths[colIndex] = minWidth
		}
	}

	return widths
}

// NewEnhancedTableLayout creates a new table definition with configurable gutters
func NewEnhancedTableLayout(gutterWidth int, columns []ColumnDefinition) *DataTableDefinition {
	return &DataTableDefinition{
		Columns:     columns,
		GutterWidth: gutterWidth,
	}
}

// calculateMaxBadgeWidth automatically calculates the maximum badge width for proper alignment
func (dt *DataTable) calculateMaxBadgeWidth() {
	maxWidth := 0
	for _, row := range dt.rows {
		if row.Badge != "" {
			// Include the badge gutter space in the calculation for proper alignment
			badgeWidth := len(row.Badge) + dt.badgeGutter
			if badgeWidth > maxWidth {
				maxWidth = badgeWidth
			}
		}
	}
	dt.maxBadgeWidth = maxWidth
}

// Header returns the formatted table header
func (dt *DataTable) Header() string {
	return dt.RenderHeader()
}

// Row returns a specific formatted row by index with full table context
func (dt *DataTable) Row(index int) string {
	if index < 0 || index >= len(dt.rows) {
		return ""
	}
	return dt.FormatEnhancedTableRow(dt.rows[index])
}

// Rows returns all formatted data rows
func (dt *DataTable) Rows() []string {
	var rows []string
	for _, row := range dt.rows {
		rows = append(rows, dt.FormatEnhancedTableRow(row))
	}
	return rows
}

// Render returns the complete formatted table (header + all rows)
func (dt *DataTable) Render() string {
	var result strings.Builder

	// Render header
	result.WriteString(dt.Header())
	result.WriteString("\n")

	// Blank separators follow ONLY wrapped (multi-line) rows — uniform
	// every-row separation hurts dense-table readability (D-29).
	rows := dt.Rows()
	for i, row := range rows {
		result.WriteString(row)
		result.WriteString("\n")

		// Add extra blank line after rows with wrapped content (multi-line rows)
		// but not after the last row to avoid trailing space
		if i < len(rows)-1 && strings.Contains(row, "\n") {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// CreateDataTable creates and renders a complete terminal-width aware data table in one call
//
// 🎯 GO-STUDS API DESIGN PATTERN: This function exemplifies the idiomatic Go API pattern
// that ALL STUDS components should follow - single function handles all variations.
//
// Handles complete table rendering including:
//   - Header row with proper styling
//   - All data rows with badge alignment
//   - Enhanced row features (colors, badges)
//   - Terminal-width awareness
//   - Automatic badge width calculation
//
// Usage:
//
//	definition := &DataTableDefinition{...}
//	output := CreateDataTable(terminalWidth, definition)
//	fmt.Print(output)
func CreateDataTable(width int, definition *DataTableDefinition) string {
	// Use the new table-centric API internally
	table := NewDataTable(width, definition)
	return table.Render()
}
