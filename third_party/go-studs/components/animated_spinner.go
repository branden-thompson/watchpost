package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// SpinnerState represents the current state of the spinner
type SpinnerState int

const (
	SpinnerQueued SpinnerState = iota
	SpinnerRunning
	SpinnerSuccess
	SpinnerFailure
	SpinnerAlert
)

// AnimatedSpinner provides an animated spinner component with configurable styling
type AnimatedSpinner struct {
	label             string
	spinnerChars      []string
	width             int
	currentFrame      int
	isRunning         bool
	spacingBefore     bool
	spacingAfter      bool
	state             SpinnerState
	completionMessage string
}

// StudsSpinnerType defines different spinner animation styles for STUDS components
type StudsSpinnerType int

const (
	StudsDotsSpinner StudsSpinnerType = iota
	StudsLineSpinner
	StudsArrowSpinner
	StudsCircleSpinner
	StudsBouncingBallSpinner
	StudsPulseSpinner
)

// NewAnimatedSpinner creates a new animated spinner with default settings
func NewAnimatedSpinner() *AnimatedSpinner {
	width, _ := rendering.GetTerminalSize()
	if width <= 0 {
		width = 80
	}

	return &AnimatedSpinner{
		label:             "Loading",
		spinnerChars:      []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		width:             width,
		currentFrame:      0,
		isRunning:         false,
		spacingBefore:     true,
		spacingAfter:      true,
		state:             SpinnerQueued,
		completionMessage: "",
	}
}

// WithLabel sets the spinner label
func (s *AnimatedSpinner) WithLabel(label string) *AnimatedSpinner {
	s.label = label
	return s
}

// WithType sets the spinner animation type
func (s *AnimatedSpinner) WithType(spinnerType StudsSpinnerType) *AnimatedSpinner {
	switch spinnerType {
	case StudsDotsSpinner:
		s.spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	case StudsLineSpinner:
		s.spinnerChars = []string{"|", "/", "-", "\\"}
	case StudsArrowSpinner:
		s.spinnerChars = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	case StudsCircleSpinner:
		s.spinnerChars = []string{"◐", "◓", "◑", "◒"}
	case StudsBouncingBallSpinner:
		s.spinnerChars = []string{"( ●    )", "(  ●   )", "(   ●  )", "(    ● )", "(     ●)", "(    ● )", "(   ●  )", "(  ●   )", "( ●    )", "(●     )"}
	case StudsPulseSpinner:
		s.spinnerChars = []string{"○", "○", "○", "○", "●", "●"}
	}
	return s
}

// WithWidth sets a custom terminal width
func (s *AnimatedSpinner) WithWidth(width int) *AnimatedSpinner {
	s.width = width
	return s
}

// WithHeaderSpacing configures for header use (no spacing before, spacing after)
func (s *AnimatedSpinner) WithHeaderSpacing() *AnimatedSpinner {
	s.spacingBefore = false
	s.spacingAfter = true
	return s
}

// WithFooterSpacing configures for footer use (spacing before, no spacing after)
func (s *AnimatedSpinner) WithFooterSpacing() *AnimatedSpinner {
	s.spacingBefore = true
	s.spacingAfter = false
	return s
}

// WithNoSpacing removes all automatic spacing
func (s *AnimatedSpinner) WithNoSpacing() *AnimatedSpinner {
	s.spacingBefore = false
	s.spacingAfter = false
	return s
}

// NextFrame advances the animation to the next frame
func (s *AnimatedSpinner) NextFrame() {
	s.currentFrame = (s.currentFrame + 1) % len(s.spinnerChars)
}

// CurrentFrame returns the current animation frame as a complete line
func (s *AnimatedSpinner) CurrentFrame() string {
	formatter := rendering.NewTextFormatter()

	// Build the spinner line
	var result strings.Builder

	// Add spacing before if configured
	if s.spacingBefore {
		result.WriteString("\n")
	}

	// Handle completion states vs running animation
	if !s.isRunning && s.state != SpinnerQueued {
		// Show completion state
		icon, iconColor := s.getStateIcon()
		message := s.completionMessage
		if message == "" {
			message = s.label + " - " + s.getStateText()
		}

		styledIcon := formatter.Style(icon, iconColor)
		styledMessage := formatter.Style(message, s.getStateTextColor())

		// Calculate spacing for completion message
		contentLength := len(icon) + 1 + len(message)
		paddingLength := s.width - contentLength
		if paddingLength < 0 {
			paddingLength = 0
		}
		padding := strings.Repeat(" ", paddingLength)

		result.WriteString(styledIcon)
		result.WriteString(" ")
		result.WriteString(styledMessage)
		result.WriteString(padding)
	} else {
		// Show running animation or queued state
		var spinnerChar string
		var spinnerColor string

		if s.state == SpinnerQueued {
			// Use a static indicator for queued state
			spinnerChar = "○"
			spinnerColor = "tui.TableAttribute" // White/default
		} else {
			// Use animated spinner for running state
			spinnerChar = s.spinnerChars[s.currentFrame]
			spinnerColor = "tui.InfoText" // Blue for running
		}

		styledSpinner := formatter.Style(spinnerChar, spinnerColor)
		labelColor := s.getLabelColor()
		styledLabel := formatter.Style(s.label, labelColor)

		// Calculate spacing (account for actual spinner character width)
		spinnerWidth := len(spinnerChar)
		contentLength := spinnerWidth + 1 + len(s.label) // spinner + space + label
		paddingLength := s.width - contentLength
		if paddingLength < 0 {
			paddingLength = 0
		}
		padding := strings.Repeat(" ", paddingLength)

		result.WriteString(styledSpinner)
		result.WriteString(" ")
		result.WriteString(styledLabel)
		result.WriteString(padding)
	}

	// Add spacing after if configured
	if s.spacingAfter {
		result.WriteString("\n")
	}

	return result.String()
}

// getStateIcon returns the icon and color for the current state
func (s *AnimatedSpinner) getStateIcon() (string, string) {
	switch s.state {
	case SpinnerSuccess:
		return "✓", "tui.SuccessText" // Green
	case SpinnerFailure:
		return "✗", "tui.ErrorText" // Red
	case SpinnerAlert:
		return "!", "tui.WarningText" // Yellow
	default:
		return "○", "tui.TableAttribute" // White/default
	}
}

// getStateText returns the default text for the current state
func (s *AnimatedSpinner) getStateText() string {
	switch s.state {
	case SpinnerSuccess:
		return "Complete"
	case SpinnerFailure:
		return "Failed"
	case SpinnerAlert:
		return "Warning"
	case SpinnerRunning:
		return "Running"
	default:
		return "Queued"
	}
}

// getStateTextColor returns the text color for the current state
func (s *AnimatedSpinner) getStateTextColor() string {
	switch s.state {
	case SpinnerSuccess:
		return "tui.SuccessText" // Green
	case SpinnerFailure:
		return "tui.ErrorText" // Red
	case SpinnerAlert:
		return "tui.WarningText" // Yellow
	case SpinnerRunning:
		return "tui.InfoText" // Blue
	default:
		return "tui.TableAttribute" // White/default
	}
}

// getLabelColor returns the label color for the current state
func (s *AnimatedSpinner) getLabelColor() string {
	switch s.state {
	case SpinnerRunning:
		return "tui.InfoText" // Blue for running
	default:
		return "tui.TableAttribute" // Default for queued
	}
}

// Start begins the animation loop (for actual usage, not demonstration)
func (s *AnimatedSpinner) Start() *AnimatedSpinner {
	s.isRunning = true
	s.state = SpinnerRunning
	return s
}

// Stop ends the animation
func (s *AnimatedSpinner) Stop() *AnimatedSpinner {
	s.isRunning = false
	return s
}

// CompleteWithState finishes the spinner with a specific state and message
func (s *AnimatedSpinner) CompleteWithState(state SpinnerState, message string) *AnimatedSpinner {
	s.isRunning = false
	s.state = state
	s.completionMessage = message
	return s
}

// CompleteSuccess finishes the spinner with success state
func (s *AnimatedSpinner) CompleteSuccess(message string) *AnimatedSpinner {
	return s.CompleteWithState(SpinnerSuccess, message)
}

// CompleteFailure finishes the spinner with failure state
func (s *AnimatedSpinner) CompleteFailure(message string) *AnimatedSpinner {
	return s.CompleteWithState(SpinnerFailure, message)
}

// CompleteAlert finishes the spinner with alert state
func (s *AnimatedSpinner) CompleteAlert(message string) *AnimatedSpinner {
	return s.CompleteWithState(SpinnerAlert, message)
}

// IsRunning returns the current animation state
func (s *AnimatedSpinner) IsRunning() bool {
	return s.isRunning
}

// AnimateFor runs the spinner animation for a specified duration (for demonstration)
func (s *AnimatedSpinner) AnimateFor(duration time.Duration, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	start := time.Now()
	s.isRunning = true
	s.state = SpinnerRunning // Set to running state for blue coloring

	for {
		select {
		case <-ticker.C:
			if time.Since(start) >= duration {
				s.isRunning = false
				return
			}

			// Clear the line and redraw
			fmt.Print("\r\033[K") // Clear current line
			fmt.Print(s.CurrentFrame())
			s.NextFrame()
		}
	}
}

// Render returns a static representation of the spinner (for documentation)
func (s *AnimatedSpinner) Render() string {
	return s.CurrentFrame()
}

// CreateAnimatedSpinner provides the single-function API for creating animated spinners
func CreateAnimatedSpinner(label string, spinnerType StudsSpinnerType, width ...int) string {
	spinner := NewAnimatedSpinner().WithLabel(label).WithType(spinnerType)

	if len(width) > 0 {
		spinner = spinner.WithWidth(width[0])
	}

	return spinner.Render()
}

// GetSpinnerClassification returns the component classification
func GetSpinnerClassification() ComponentClassification {
	return AnimatedSpinnerClassification
}
