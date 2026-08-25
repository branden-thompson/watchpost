package components

import (
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// TableAlignment represents column alignment options
type TableAlignment int

const (
	AlignLeft TableAlignment = iota
	AlignRight
	AlignCenter
)

// TableColumn represents a table column configuration
type TableColumn struct {
	Header    string
	Width     int
	Alignment TableAlignment
	MinWidth  int
	MaxWidth  int
	Truncate  bool
}

// TableRow represents a row of data in the table
type TableRow struct {
	Cells []string
	Style string // Optional semantic token for row styling
}

// AdvancedDataTable represents a terminal-width aware data table component with advanced features
type AdvancedDataTable struct {
	columns          []TableColumn
	rows             []TableRow
	headers          []string
	width            int // Terminal width (auto-detected if 0)
	minTableWidth    int // Minimum total table width
	showHeaders      bool
	headerSeparator  bool
	borderStyle      TableBorderStyle
	formatter        *rendering.TextFormatter
	truncationSuffix string
	cellPadding      int
}

// TableBorderStyle represents different table border styles
type TableBorderStyle int

const (
	BorderNone TableBorderStyle = iota
	BorderLight
	BorderHeavy
	BorderRounded
	BorderCustom
)

// NewAdvancedDataTable creates a new data table with default configuration
func NewAdvancedDataTable() *AdvancedDataTable {
	return &AdvancedDataTable{
		columns:          make([]TableColumn, 0),
		rows:             make([]TableRow, 0),
		headers:          make([]string, 0),
		minTableWidth:    20,
		showHeaders:      true,
		headerSeparator:  true,
		borderStyle:      BorderLight,
		formatter:        rendering.NewTextFormatter(),
		truncationSuffix: "...",
		cellPadding:      1,
	}
}

// WithColumns sets the table columns configuration
func (dt *AdvancedDataTable) WithColumns(columns []TableColumn) *AdvancedDataTable {
	dt.columns = columns
	dt.headers = make([]string, len(columns))
	for i, col := range columns {
		dt.headers[i] = col.Header
	}
	return dt
}

// WithHeaders sets simple string headers (creates default columns)
func (dt *AdvancedDataTable) WithHeaders(headers ...string) *AdvancedDataTable {
	dt.headers = headers
	dt.columns = make([]TableColumn, len(headers))
	for i, header := range headers {
		dt.columns[i] = TableColumn{
			Header:    header,
			Width:     0, // Auto-calculated
			Alignment: AlignLeft,
			Truncate:  true,
		}
	}
	return dt
}

// WithRows sets the table data rows
func (dt *AdvancedDataTable) WithRows(rows [][]string) *AdvancedDataTable {
	dt.rows = make([]TableRow, len(rows))
	for i, row := range rows {
		dt.rows[i] = TableRow{
			Cells: row,
			Style: "", // Default styling
		}
	}
	return dt
}

// AddRow adds a single data row to the table
func (dt *AdvancedDataTable) AddRow(cells ...string) *AdvancedDataTable {
	dt.rows = append(dt.rows, TableRow{
		Cells: cells,
		Style: "",
	})
	return dt
}

// AddStyledRow adds a data row with custom styling
func (dt *AdvancedDataTable) AddStyledRow(style string, cells ...string) *AdvancedDataTable {
	dt.rows = append(dt.rows, TableRow{
		Cells: cells,
		Style: style,
	})
	return dt
}

// WithTerminalWidth sets the terminal width for responsive design
func (dt *AdvancedDataTable) WithTerminalWidth(width int) *AdvancedDataTable {
	dt.width = width
	return dt
}

// WithBorderStyle sets the table border style
func (dt *AdvancedDataTable) WithBorderStyle(style TableBorderStyle) *AdvancedDataTable {
	dt.borderStyle = style
	return dt
}

// WithHeaderSeparator enables or disables the header separator line
func (dt *AdvancedDataTable) WithHeaderSeparator(show bool) *AdvancedDataTable {
	dt.headerSeparator = show
	return dt
}

// WithCellPadding sets the internal cell padding
func (dt *AdvancedDataTable) WithCellPadding(padding int) *AdvancedDataTable {
	dt.cellPadding = padding
	return dt
}

// Render generates the complete table display
func (dt *AdvancedDataTable) Render() string {
	if len(dt.columns) == 0 && len(dt.headers) == 0 {
		return ""
	}

	terminalWidth := dt.getTerminalWidth()
	layout := dt.calculateLayout(terminalWidth)

	var lines []string

	// Render headers if enabled
	if dt.showHeaders {
		headerLine := dt.renderHeaderRow(layout)
		lines = append(lines, headerLine)

		// Add separator if enabled
		if dt.headerSeparator {
			separatorLine := dt.renderSeparatorRow(layout)
			lines = append(lines, separatorLine)
		}
	}

	// Render data rows
	for _, row := range dt.rows {
		rowLine := dt.renderDataRow(row, layout)
		lines = append(lines, rowLine)
	}

	return strings.Join(lines, "\n")
}

// RenderCompact generates a minimal table for narrow terminals
func (dt *AdvancedDataTable) RenderCompact(width int) string {
	if width < 20 {
		width = 20
	}

	// For compact mode, show only essential columns
	compactColumns := dt.selectCompactColumns(width)
	if len(compactColumns) == 0 {
		return dt.renderVerticalList(width)
	}

	// Create temporary table with compact columns
	compactTable := NewAdvancedDataTable().
		WithColumns(compactColumns).
		WithTerminalWidth(width).
		WithBorderStyle(BorderNone).
		WithCellPadding(1)

	// Add rows with only essential data
	for _, row := range dt.rows {
		compactCells := make([]string, len(compactColumns))
		for i, colIndex := range dt.getCompactColumnIndices(compactColumns) {
			if colIndex < len(row.Cells) {
				compactCells[i] = row.Cells[colIndex]
			}
		}
		compactTable.AddRow(compactCells...)
	}

	return compactTable.Render()
}

// TableLayout represents the calculated layout for table columns
type TableLayout struct {
	ColumnWidths []int
	TotalWidth   int
	AvailWidth   int
	BorderWidth  int
	PaddingWidth int
	RequiresWrap bool
}

// calculateLayout determines optimal column widths for the given terminal width
func (dt *AdvancedDataTable) calculateLayout(terminalWidth int) TableLayout {
	layout := TableLayout{
		AvailWidth:   terminalWidth,
		BorderWidth:  dt.getBorderWidth(),
		PaddingWidth: dt.cellPadding * 2 * len(dt.columns),
	}

	// Calculate available width for content
	contentWidth := terminalWidth - layout.BorderWidth - layout.PaddingWidth
	if contentWidth < dt.minTableWidth {
		contentWidth = dt.minTableWidth
		layout.RequiresWrap = true
	}

	// Calculate column widths
	layout.ColumnWidths = dt.calculateColumnWidths(contentWidth)
	layout.TotalWidth = dt.sumWidths(layout.ColumnWidths) + layout.BorderWidth + layout.PaddingWidth

	return layout
}

// calculateColumnWidths determines optimal width for each column
func (dt *AdvancedDataTable) calculateColumnWidths(availableWidth int) []int {
	if len(dt.columns) == 0 {
		return []int{}
	}

	widths := make([]int, len(dt.columns))

	// First pass: set minimum widths
	totalMinWidth := 0
	for i, col := range dt.columns {
		minWidth := len(col.Header)
		if col.MinWidth > minWidth {
			minWidth = col.MinWidth
		}
		widths[i] = minWidth
		totalMinWidth += minWidth
	}

	// If we have extra space, distribute it proportionally
	extraSpace := availableWidth - totalMinWidth
	if extraSpace > 0 {
		dt.distributeExtraSpace(widths, extraSpace)
	}

	// Apply maximum width constraints
	for i, col := range dt.columns {
		if col.MaxWidth > 0 && widths[i] > col.MaxWidth {
			widths[i] = col.MaxWidth
		}
	}

	return widths
}

// distributeExtraSpace allocates extra width across columns
func (dt *AdvancedDataTable) distributeExtraSpace(widths []int, extraSpace int) {
	if len(widths) == 0 {
		return
	}

	spacePerColumn := extraSpace / len(widths)
	remainder := extraSpace % len(widths)

	for i := range widths {
		widths[i] += spacePerColumn
		if i < remainder {
			widths[i]++
		}
	}
}

// renderHeaderRow generates the table header row
func (dt *AdvancedDataTable) renderHeaderRow(layout TableLayout) string {
	if len(dt.headers) == 0 {
		return ""
	}

	cells := make([]string, len(dt.headers))
	for i, header := range dt.headers {
		width := layout.ColumnWidths[i]
		cells[i] = dt.formatCell(header, width, AlignCenter, "tui.TableHeader")
	}

	return dt.joinCells(cells)
}

// renderSeparatorRow generates the header separator line
func (dt *AdvancedDataTable) renderSeparatorRow(layout TableLayout) string {
	cells := make([]string, len(layout.ColumnWidths))
	for i, width := range layout.ColumnWidths {
		cells[i] = dt.formatter.Style(strings.Repeat("-", width), "tui.TableSeparator")
	}
	return dt.joinCells(cells)
}

// renderDataRow generates a data row
func (dt *AdvancedDataTable) renderDataRow(row TableRow, layout TableLayout) string {
	cells := make([]string, len(layout.ColumnWidths))

	for i, width := range layout.ColumnWidths {
		cellValue := ""
		if i < len(row.Cells) {
			cellValue = row.Cells[i]
		}

		alignment := AlignLeft
		if i < len(dt.columns) {
			alignment = dt.columns[i].Alignment
		}

		semanticToken := "tui.TableCell"
		if row.Style != "" {
			semanticToken = row.Style
		}

		cells[i] = dt.formatCell(cellValue, width, alignment, semanticToken)
	}

	return dt.joinCells(cells)
}

// formatCell formats a single cell with alignment and truncation
func (dt *AdvancedDataTable) formatCell(content string, width int, alignment TableAlignment, semanticToken string) string {
	if width <= 0 {
		return ""
	}

	// Truncate if necessary
	if len(content) > width {
		if width > len(dt.truncationSuffix) {
			content = content[:width-len(dt.truncationSuffix)] + dt.truncationSuffix
		} else {
			content = content[:width]
		}
	}

	// Apply alignment using the ANSI formatter
	ansiFormatter := dt.formatter.GetANSIFormatter()
	var formatted string
	switch alignment {
	case AlignLeft:
		formatted = ansiFormatter.PadRight(content, width)
	case AlignRight:
		formatted = ansiFormatter.PadLeft(content, width)
	case AlignCenter:
		formatted = ansiFormatter.PadCenter(content, width)
	default:
		formatted = ansiFormatter.PadRight(content, width)
	}

	// Apply semantic token styling
	return dt.formatter.Style(formatted, semanticToken)
}

// joinCells combines cells into a row with borders
func (dt *AdvancedDataTable) joinCells(cells []string) string {
	if dt.borderStyle == BorderNone {
		return strings.Join(cells, strings.Repeat(" ", dt.cellPadding*2))
	}

	padding := strings.Repeat(" ", dt.cellPadding)
	border := dt.getBorderCharacter()
	styledBorder := dt.formatter.Style(border, "tui.TableBorder")

	return styledBorder + padding + strings.Join(cells, padding+styledBorder+padding) + padding + styledBorder
}

// getBorderCharacter returns the border character for the current style
func (dt *AdvancedDataTable) getBorderCharacter() string {
	switch dt.borderStyle {
	case BorderLight:
		return "|"
	case BorderHeavy:
		return "║"
	case BorderRounded:
		return "│"
	default:
		return "|"
	}
}

// getBorderWidth calculates the total width consumed by borders
func (dt *AdvancedDataTable) getBorderWidth() int {
	if dt.borderStyle == BorderNone {
		return 0
	}
	// Left border + separators between columns + right border
	return len(dt.columns) + 1
}

// selectCompactColumns chooses essential columns for compact display
func (dt *AdvancedDataTable) selectCompactColumns(availableWidth int) []TableColumn {
	if len(dt.columns) == 0 {
		return []TableColumn{}
	}

	// For compact mode, prioritize shorter headers and essential data
	compactColumns := make([]TableColumn, 0)
	usedWidth := 0

	for _, col := range dt.columns {
		minCellWidth := len(col.Header) + 4 // Header + padding + borders
		if usedWidth+minCellWidth <= availableWidth {
			compactColumns = append(compactColumns, col)
			usedWidth += minCellWidth
		}
	}

	return compactColumns
}

// getCompactColumnIndices returns the original column indices for compact columns
func (dt *AdvancedDataTable) getCompactColumnIndices(compactColumns []TableColumn) []int {
	indices := make([]int, 0)
	for _, compactCol := range compactColumns {
		for i, originalCol := range dt.columns {
			if compactCol.Header == originalCol.Header {
				indices = append(indices, i)
				break
			}
		}
	}
	return indices
}

// renderVerticalList renders data as a vertical list for very narrow terminals
func (dt *AdvancedDataTable) renderVerticalList(width int) string {
	var lines []string

	for rowIndex, row := range dt.rows {
		if rowIndex > 0 {
			lines = append(lines, dt.formatter.Style(strings.Repeat("-", width-2), "tui.TableSeparator"))
		}

		for colIndex, cell := range row.Cells {
			if colIndex < len(dt.headers) {
				header := dt.formatter.Style(dt.headers[colIndex]+":", "tui.TableHeader")
				value := dt.formatter.Style(cell, "tui.TableCell")
				lines = append(lines, fmt.Sprintf("%s %s", header, value))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// sumWidths calculates the total width of all columns
func (dt *AdvancedDataTable) sumWidths(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}
	return total
}

// getTerminalWidth returns the terminal width, detecting automatically if not set
func (dt *AdvancedDataTable) getTerminalWidth() int {
	if dt.width > 0 {
		return dt.width
	}

	// Auto-detect terminal width
	caps := dt.formatter.GetCapabilities()
	if caps.Width > 0 {
		return caps.Width
	}

	// Fallback to reasonable default
	return 80
}

// GetVisualWidth returns the total width the table will consume
func (dt *AdvancedDataTable) GetVisualWidth() int {
	terminalWidth := dt.getTerminalWidth()
	layout := dt.calculateLayout(terminalWidth)
	return layout.TotalWidth
}

// ResponsiveRender automatically adjusts table layout for different terminal widths
func (dt *AdvancedDataTable) ResponsiveRender() string {
	terminalWidth := dt.getTerminalWidth()

	// For very narrow terminals, use vertical list format
	if terminalWidth < 40 {
		return dt.renderVerticalList(terminalWidth)
	}

	// For narrow terminals, use compact mode
	if terminalWidth < 80 {
		return dt.RenderCompact(terminalWidth)
	}

	// Full render for wide terminals
	return dt.Render()
}

// Clear removes all data from the table while preserving configuration
func (dt *AdvancedDataTable) Clear() *AdvancedDataTable {
	dt.rows = make([]TableRow, 0)
	return dt
}

// RowCount returns the number of data rows in the table
func (dt *AdvancedDataTable) RowCount() int {
	return len(dt.rows)
}

// ColumnCount returns the number of columns in the table
func (dt *AdvancedDataTable) ColumnCount() int {
	return len(dt.columns)
}

// Convenience constructors for common table configurations

// NewSimpleTable creates a table with basic string headers
func NewSimpleTable(headers ...string) *AdvancedDataTable {
	return NewAdvancedDataTable().WithHeaders(headers...)
}

// NewBorderedTable creates a table with borders and headers
func NewBorderedTable(headers ...string) *AdvancedDataTable {
	return NewAdvancedDataTable().
		WithHeaders(headers...).
		WithBorderStyle(BorderLight).
		WithHeaderSeparator(true)
}

// NewCompactTable creates a minimal table for space-constrained displays
func NewCompactTable(headers ...string) *AdvancedDataTable {
	return NewAdvancedDataTable().
		WithHeaders(headers...).
		WithBorderStyle(BorderNone).
		WithCellPadding(1).
		WithHeaderSeparator(false)
}
