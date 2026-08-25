package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// ProgressState represents the current state of the progress bar
type ProgressState int

const (
	ProgressQueued  ProgressState = iota // White/default - waiting to start
	ProgressRunning                      // Blue - actively progressing
	ProgressSuccess                      // Green - completed successfully
	ProgressFailure                      // Red - failed/errored
	ProgressAlert                        // Yellow - warning/attention needed
)

// ProgressBar represents a terminal-width aware progress bar component
type ProgressBar struct {
	progress     float64       // 0.0 to 1.0
	label        string        // "Installing Dependencies"
	step         int           // Current step number
	totalSteps   int           // Total steps count
	width        int           // Terminal width (auto-detected if 0)
	minBarWidth  int           // Minimum progress bar width (default: 10)
	fillChar     string        // Fill character (default: "█")
	emptyChar    string        // Empty character (default: "░")
	leftBracket  string        // Left bracket character (default: "[")
	rightBracket string        // Right bracket character (default: "]")
	showPercent  bool          // Show percentage (default: true)
	showSteps    bool          // Show step counter (default: true)
	showLabel    bool          // Show label (default: true)
	state        ProgressState // Current progress state
	formatter    *rendering.TextFormatter
}

// NewProgressBar creates a new progress bar with default configuration.
// Defaults to the "modern" preset: block fill (█/░) framed by U+2502
// box-drawing brackets (│). To opt back into the legacy ■/□ no-bracket
// visual, call CreateProgressBar with style="squares".
func NewProgressBar(progress float64) *ProgressBar {
	return &ProgressBar{
		progress:     clampProgress(progress),
		minBarWidth:  10,
		fillChar:     "█",
		emptyChar:    "░",
		leftBracket:  "│",
		rightBracket: "│",
		showPercent:  true,
		showSteps:    false,           // Default to false, enable when steps are set
		showLabel:    false,           // Default to false, enable when label is set
		state:        ProgressRunning, // Default to running state
		formatter:    rendering.NewTextFormatter(),
	}
}

// WithLabel sets the progress bar label
func (pb *ProgressBar) WithLabel(label string) *ProgressBar {
	pb.label = label
	pb.showLabel = len(label) > 0
	return pb
}

// WithSteps sets the current and total step counts
func (pb *ProgressBar) WithSteps(current, total int) *ProgressBar {
	pb.step = current
	pb.totalSteps = total
	pb.showSteps = total > 0
	return pb
}

// WithTerminalWidth sets the terminal width for responsive design
func (pb *ProgressBar) WithTerminalWidth(width int) *ProgressBar {
	pb.width = width
	return pb
}

// WithMinBarWidth sets the minimum width for the progress bar portion
func (pb *ProgressBar) WithMinBarWidth(minWidth int) *ProgressBar {
	pb.minBarWidth = minWidth
	return pb
}

// WithFillChar sets the character used for filled progress
func (pb *ProgressBar) WithFillChar(char string) *ProgressBar {
	pb.fillChar = char
	return pb
}

// WithEmptyChar sets the character used for empty progress
func (pb *ProgressBar) WithEmptyChar(char string) *ProgressBar {
	pb.emptyChar = char
	return pb
}

// WithNPMStyle sets the progress bar to use npm-style characters (# and -)
func (pb *ProgressBar) WithNPMStyle() *ProgressBar {
	pb.fillChar = "#"
	pb.emptyChar = "-"
	return pb
}

// WithModernStyle sets the progress bar to the modern preset:
// fill=█, empty=░, brackets=│ (U+2502 BOX DRAWINGS LIGHT VERTICAL).
// This is the same configuration NewProgressBar starts with; the method
// is provided for explicit opt-in from a non-default starting point.
func (pb *ProgressBar) WithModernStyle() *ProgressBar {
	pb.fillChar = "█"
	pb.emptyChar = "░"
	pb.leftBracket = "│"
	pb.rightBracket = "│"
	return pb
}

// WithPercentage enables or disables percentage display
func (pb *ProgressBar) WithPercentage(show bool) *ProgressBar {
	pb.showPercent = show
	return pb
}

// SetProgress updates the progress value (0.0 to 1.0)
func (pb *ProgressBar) SetProgress(progress float64) {
	pb.progress = clampProgress(progress)
}

// SetSteps updates the step counts
func (pb *ProgressBar) SetSteps(current, total int) {
	pb.step = current
	pb.totalSteps = total
	pb.showSteps = total > 0
}

// GetProgress returns the current progress value
func (pb *ProgressBar) GetProgress() float64 {
	return pb.progress
}

// Render generates the complete progress bar display
func (pb *ProgressBar) Render() string {
	terminalWidth := pb.getTerminalWidth()
	layout := pb.calculateLayout(terminalWidth)

	var parts []string

	// Add label if enabled
	if pb.showLabel && pb.label != "" {
		labelText := pb.formatter.Style(pb.label, "tui.ProgressLabel")
		parts = append(parts, labelText+":")
	}

	// Generate progress bar
	bar := pb.renderProgressBar(layout.BarWidth)
	parts = append(parts, bar)

	// Add percentage if enabled
	if pb.showPercent {
		percentage := pb.formatter.Style(fmt.Sprintf("%.2f%%", pb.progress*100), "tui.ProgressPercent")
		parts = append(parts, percentage)
	}

	// Add step counter if enabled
	if pb.showSteps && pb.totalSteps > 0 {
		stepText := pb.formatter.Style(fmt.Sprintf("(%d/%d)", pb.step, pb.totalSteps), "tui.ProgressSteps")
		parts = append(parts, stepText)
	}

	return strings.Join(parts, " ")
}

// RenderCompact generates a compact progress bar without label
func (pb *ProgressBar) RenderCompact(width int) string {
	// Calculate bracket overhead
	bracketOverhead := 0
	if pb.leftBracket != "" {
		bracketOverhead += rendering.DisplayWidth(pb.leftBracket)
	}
	if pb.rightBracket != "" {
		bracketOverhead += rendering.DisplayWidth(pb.rightBracket)
	}

	if width < pb.minBarWidth+bracketOverhead {
		width = pb.minBarWidth + bracketOverhead
	}

	bar := pb.renderProgressBar(width - bracketOverhead)

	// Add brackets if they exist
	result := ""
	if pb.leftBracket != "" {
		result += pb.formatter.Style(pb.leftBracket, "tui.ProgressBrackets")
	}
	result += bar
	if pb.rightBracket != "" {
		result += pb.formatter.Style(pb.rightBracket, "tui.ProgressBrackets")
	}

	return result
}

// ProgressLayout represents the calculated layout for a progress bar
type ProgressLayout struct {
	BarWidth   int
	LabelWidth int
	TotalUsed  int
	Available  int
}

// calculateLayout determines the optimal layout for the given terminal width
func (pb *ProgressBar) calculateLayout(terminalWidth int) ProgressLayout {
	layout := ProgressLayout{
		Available: terminalWidth,
	}

	// Calculate fixed elements width
	fixedWidth := 0
	numParts := 0 // Count of parts that will be joined with spaces

	// Label: "Label: " (no leading space, but has trailing space from join)
	if pb.showLabel && pb.label != "" {
		layout.LabelWidth = rendering.DisplayWidth(pb.label) + 1 // "Label:"
		fixedWidth += layout.LabelWidth
		numParts++
	}

	// Progress bar will be counted as one part
	numParts++

	// Percentage: " nnn.nn%"
	if pb.showPercent {
		fixedWidth += 7 // "100.00%" (no leading space, handled by join)
		numParts++
	}

	// Steps: " (nn/nn)"
	if pb.showSteps && pb.totalSteps > 0 {
		stepText := fmt.Sprintf("(%d/%d)", pb.step, pb.totalSteps)
		fixedWidth += rendering.DisplayWidth(stepText)
		numParts++
	}

	// Calculate bracket width (part of progress bar, not separate)
	bracketWidth := 0
	if pb.leftBracket != "" {
		bracketWidth += rendering.DisplayWidth(pb.leftBracket)
	}
	if pb.rightBracket != "" {
		bracketWidth += rendering.DisplayWidth(pb.rightBracket)
	}

	// Account for spaces between parts (numParts - 1 spaces)
	spacesWidth := numParts - 1

	// Total fixed width = text elements + spaces between parts + brackets
	totalFixedWidth := fixedWidth + spacesWidth + bracketWidth

	// Calculate available width for progress bar content (excluding brackets)
	availableWidth := terminalWidth - totalFixedWidth
	if availableWidth < pb.minBarWidth {
		availableWidth = pb.minBarWidth
	}

	layout.BarWidth = availableWidth
	layout.TotalUsed = totalFixedWidth + availableWidth

	return layout
}

// renderProgressBar generates the visual progress bar portion
func (pb *ProgressBar) renderProgressBar(width int) string {
	if width < 1 {
		width = 1
	}

	filled := int(math.Round(pb.progress * float64(width)))
	if filled > width {
		filled = width
	}
	empty := width - filled

	// Create fill and empty sections with state-aware semantic token styling
	fillSection := ""
	if filled > 0 {
		fillColor := pb.getStateFillColor()
		fillSection = pb.formatter.Style(strings.Repeat(pb.fillChar, filled), fillColor)
	}

	emptySection := ""
	if empty > 0 {
		emptySection = pb.formatter.Style(strings.Repeat(pb.emptyChar, empty), "tui.ProgressEmpty")
	}

	// Wrap with brackets (style-dependent)
	leftBracket := ""
	rightBracket := ""
	if pb.leftBracket != "" {
		leftBracket = pb.formatter.Style(pb.leftBracket, "tui.ProgressBrackets")
	}
	if pb.rightBracket != "" {
		rightBracket = pb.formatter.Style(pb.rightBracket, "tui.ProgressBrackets")
	}

	return leftBracket + fillSection + emptySection + rightBracket
}

// getStateFillColor returns the appropriate fill color based on progress state
func (pb *ProgressBar) getStateFillColor() string {
	switch pb.state {
	case ProgressQueued:
		return "tui.TableAttribute" // White/default
	case ProgressRunning:
		return "tui.InfoText" // Blue
	case ProgressSuccess:
		return "tui.SuccessText" // Green
	case ProgressFailure:
		return "tui.ErrorText" // Red
	case ProgressAlert:
		return "tui.WarningText" // Yellow
	default:
		return "tui.ProgressFill" // Default progress fill color
	}
}

// getTerminalWidth returns the safe usable width accounting for potential inline positioning
func (pb *ProgressBar) getTerminalWidth() int {
	if pb.width > 0 {
		return pb.width
	}

	// Auto-detect terminal width using the same method as other components
	terminalWidth, _ := rendering.GetTerminalSize()
	if terminalWidth <= 0 {
		terminalWidth = 80 // Fallback default
	}

	// Implement safe minimum strategy: assume reasonable indentation scenarios
	// The component should utilize available space effectively while maintaining
	// safety for common indentation patterns.
	//
	// Strategy: Reserve conservative but reasonable space for indentation
	// - Small terminals (≤60): Reserve 8 chars for indentation (lists, code blocks)
	// - Medium terminals (61-100): Reserve 12 chars for indentation
	// - Large terminals (>100): Reserve 16 chars for indentation
	// - Ensure minimum progress bar width of 30 chars for readability

	var maxIndentation int
	if terminalWidth <= 60 {
		maxIndentation = 8 // Small terminals: assume modest indentation
	} else if terminalWidth <= 100 {
		maxIndentation = 12 // Medium terminals: assume moderate indentation
	} else {
		maxIndentation = 16 // Large terminals: assume deeper nesting possible
	}

	safeWidth := terminalWidth - maxIndentation

	// Ensure minimum readable progress bar width
	minUsableWidth := 30
	if safeWidth < minUsableWidth {
		// If we can't fit minimum width with safety margin, use terminal width directly
		// This handles very narrow terminals where being conservative would be unusable
		if terminalWidth >= minUsableWidth {
			safeWidth = terminalWidth - 5 // Just 5 chars safety margin
		} else {
			safeWidth = terminalWidth // Use full width on very narrow terminals
		}
	}

	return safeWidth
}

// clampProgress ensures progress value is within valid range
func clampProgress(progress float64) float64 {
	if progress < 0.0 {
		return 0.0
	}
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// GetVisualWidth returns the visual width of the progress bar
func (pb *ProgressBar) GetVisualWidth() int {
	terminalWidth := pb.getTerminalWidth()
	layout := pb.calculateLayout(terminalWidth)
	return layout.TotalUsed
}

// ResponsiveRender automatically adjusts to different terminal widths
func (pb *ProgressBar) ResponsiveRender() string {
	terminalWidth := pb.getTerminalWidth()

	// For very narrow terminals, use compact mode
	if terminalWidth < 40 {
		return pb.renderNarrowMode(terminalWidth)
	}

	// For medium terminals, may need to abbreviate label
	if terminalWidth < 60 && pb.showLabel && rendering.DisplayWidth(pb.label) > 15 {
		originalLabel := pb.label
		pb.label = pb.abbreviateLabel(pb.label, 12)
		result := pb.Render()
		pb.label = originalLabel // Restore original
		return result
	}

	// Full render for wide terminals
	return pb.Render()
}

// renderNarrowMode renders a minimal progress bar for narrow terminals
func (pb *ProgressBar) renderNarrowMode(width int) string {
	// Minimal format: [████████] 65% (13/20)
	parts := make([]string, 0, 3)

	// Always show progress bar
	barWidth := pb.minBarWidth
	if width > barWidth+10 { // Leave room for percentage
		barWidth = width - 10
	}
	bar := pb.renderProgressBar(barWidth)
	parts = append(parts, bar)

	// Add percentage if there's room
	if width > barWidth+6 { // Room for " 100%"
		percentage := pb.formatter.Style(fmt.Sprintf("%.0f%%", pb.progress*100), "tui.ProgressPercent")
		parts = append(parts, percentage)
	}

	// Add steps if there's room and they're enabled
	if pb.showSteps && pb.totalSteps > 0 && width > barWidth+15 {
		stepText := pb.formatter.Style(fmt.Sprintf("(%d/%d)", pb.step, pb.totalSteps), "tui.ProgressSteps")
		parts = append(parts, stepText)
	}

	return strings.Join(parts, " ")
}

// abbreviateLabel shortens a label to fit in limited space
func (pb *ProgressBar) abbreviateLabel(label string, maxLength int) string {
	if rendering.DisplayWidth(label) <= maxLength {
		return label
	}

	return rendering.TruncateString(label, maxLength, "...")
}

// MultiProgressBar manages multiple progress bars for complex operations.
// When bars in the same multi-display use different presets, their inner
// fill widths can differ by the bracket overhead (0 cells for "dos" and
// "squares", 2 cells for bracketed presets like "modern", "npm", "rounded",
// "diamond"). The outer terminal width remains consistent.
type MultiProgressBar struct {
	bars      []*ProgressBar
	formatter *rendering.TextFormatter
}

// NewMultiProgressBar creates a new multi-progress bar manager
func NewMultiProgressBar() *MultiProgressBar {
	return &MultiProgressBar{
		bars:      make([]*ProgressBar, 0),
		formatter: rendering.NewTextFormatter(),
	}
}

// AddProgressBar adds a progress bar to the multi-progress display
func (mpb *MultiProgressBar) AddProgressBar(bar *ProgressBar) {
	mpb.bars = append(mpb.bars, bar)
}

// SetOverallProgress sets the progress of the first (main) progress bar
func (mpb *MultiProgressBar) SetOverallProgress(progress float64) {
	if len(mpb.bars) > 0 {
		mpb.bars[0].SetProgress(progress)
	}
}

// RenderAll renders all progress bars with consistent width
func (mpb *MultiProgressBar) RenderAll() []string {
	if len(mpb.bars) == 0 {
		return []string{}
	}

	// Use the first bar's terminal width for consistency
	terminalWidth := mpb.bars[0].getTerminalWidth()

	results := make([]string, len(mpb.bars))
	for i, bar := range mpb.bars {
		bar.WithTerminalWidth(terminalWidth)
		results[i] = bar.Render()
	}

	return results
}

// GetOverallProgress returns the average progress across all bars
func (mpb *MultiProgressBar) GetOverallProgress() float64 {
	if len(mpb.bars) == 0 {
		return 0.0
	}

	total := 0.0
	for _, bar := range mpb.bars {
		total += bar.GetProgress()
	}

	return total / float64(len(mpb.bars))
}

// CreateProgressBar creates a progress bar with clean, single-function API.
//
// Parameters:
//
//	label:       progress bar label (empty string hides label)
//	style:       one of "modern" (default), "dos", "npm", "squares",
//	             "rounded", "diamond". Empty string and "default" both
//	             resolve to "modern". Unknown style strings fall back
//	             to "modern".
//	percentage:  progress as percentage (0.0 to 100.0)
//	currentStep: current step number (0 to hide steps)
//	totalSteps:  total step count (0 to hide steps)
//	state:       ProgressQueued/Running/Success/Failure/Alert
//	width:       optional terminal width (omit for auto-detection)
//
// Preset visuals:
//
//	modern:  │███░░░│    (default; block fill + U+2502 brackets)
//	dos:     ███░░░      (block fill, no brackets)
//	npm:     [###---]    (hash/dash + ASCII brackets)
//	squares: ■■■□□□      (preserves the pre-v0.4.0 default visual)
//	rounded: (***---)    (asterisk/dash + parens)
//	diamond: [◆◆◆◇◇◇]    (diamond fill + ASCII brackets)
func CreateProgressBar(label string, style string, percentage float64, currentStep int, totalSteps int, state ProgressState, width ...int) string {
	// Convert percentage to progress (0.0 to 1.0)
	progress := percentage / 100.0

	pb := &ProgressBar{
		progress:    clampProgress(progress),
		label:       label,
		step:        currentStep,
		totalSteps:  totalSteps,
		minBarWidth: 10,
		showPercent: true,
		showSteps:   totalSteps > 0 && currentStep > 0,
		showLabel:   len(label) > 0,
		state:       state,
		formatter:   rendering.NewTextFormatter(),
	}

	// Set style-specific characters and brackets.
	switch style {
	case "modern", "default", "":
		pb.fillChar = "█"
		pb.emptyChar = "░"
		pb.leftBracket = "│"
		pb.rightBracket = "│"
	case "dos":
		pb.fillChar = "█"
		pb.emptyChar = "░"
		pb.leftBracket = ""
		pb.rightBracket = ""
	case "npm":
		pb.fillChar = "#"
		pb.emptyChar = "-"
		pb.leftBracket = "["
		pb.rightBracket = "]"
	case "squares":
		pb.fillChar = "■"
		pb.emptyChar = "□"
		pb.leftBracket = ""
		pb.rightBracket = ""
	case "rounded":
		pb.fillChar = "*"
		pb.emptyChar = "-"
		pb.leftBracket = "("
		pb.rightBracket = ")"
	case "diamond":
		pb.fillChar = "◆"
		pb.emptyChar = "◇"
		pb.leftBracket = "["
		pb.rightBracket = "]"
	default: // unknown style falls back to modern
		pb.fillChar = "█"
		pb.emptyChar = "░"
		pb.leftBracket = "│"
		pb.rightBracket = "│"
	}

	// Set terminal width if provided
	if len(width) > 0 && width[0] > 0 {
		pb.width = width[0]
	}

	return pb.Render()
}

// WithState adds state management to existing builder pattern
func (pb *ProgressBar) WithState(state ProgressState) *ProgressBar {
	pb.state = state
	return pb
}

// SetState updates the progress bar state
func (pb *ProgressBar) SetState(state ProgressState) {
	pb.state = state
}

// GetState returns the current progress bar state
func (pb *ProgressBar) GetState() ProgressState {
	return pb.state
}

// GetProgressBarClassification returns the component classification for progress bars
func GetProgressBarClassification() ComponentClassification {
	return ProgressBarClassification
}
