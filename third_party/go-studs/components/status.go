package components

import (
	"fmt"
	"time"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// StatusState represents the different states a status indicator can display
type StatusState int

const (
	StatusQueued  StatusState = iota // [ ] - Pending/queued state
	StatusRunning                    // [⠋] - Running with animated spinner
	StatusSuccess                    // [✓] - Successfully completed
	StatusFailed                     // [x] - Failed/error state
	StatusWarning                    // [!] - Warning/alert state
	StatusCustom                     // Custom icon state
)

// String returns string representation of StatusState
func (s StatusState) String() string {
	switch s {
	case StatusQueued:
		return "Queued"
	case StatusRunning:
		return "Running"
	case StatusSuccess:
		return "Success"
	case StatusFailed:
		return "Failed"
	case StatusWarning:
		return "Warning"
	case StatusCustom:
		return "Custom"
	default:
		return "Unknown"
	}
}

// StatusIndicator represents an animated status indicator component
type StatusIndicator struct {
	state               StatusState
	customIcon          string
	animationEnabled    bool
	spinnerFrames       []string
	spinnerSpeed        time.Duration
	spinnerID           string
	animationController *rendering.AnimationController
	formatter           *rendering.ANSIFormatter
}

// NewStatusIndicator creates a new status indicator with default configuration
func NewStatusIndicator(state StatusState) *StatusIndicator {
	return &StatusIndicator{
		state:            state,
		animationEnabled: true,
		spinnerFrames:    rendering.SpinnerDots, // Default to dots spinner
		spinnerSpeed:     rendering.DefaultSpinnerSpeed,
		formatter:        rendering.NewANSIFormatter(),
	}
}

// WithCustomIcon sets a custom icon for StatusCustom state
func (si *StatusIndicator) WithCustomIcon(icon string) *StatusIndicator {
	si.customIcon = icon
	return si
}

// WithAnimation enables or disables animation for running state
func (si *StatusIndicator) WithAnimation(enabled bool) *StatusIndicator {
	si.animationEnabled = enabled
	return si
}

// WithSpinner sets the spinner frames for animated running state
func (si *StatusIndicator) WithSpinner(frames []string) *StatusIndicator {
	si.spinnerFrames = frames
	return si
}

// WithSpinnerSpeed sets the animation speed for the spinner
func (si *StatusIndicator) WithSpinnerSpeed(speed time.Duration) *StatusIndicator {
	si.spinnerSpeed = speed
	return si
}

// WithAnimationController attaches an animation controller for managing animations
func (si *StatusIndicator) WithAnimationController(controller *rendering.AnimationController) *StatusIndicator {
	si.animationController = controller
	return si
}

// SetState updates the status indicator state
func (si *StatusIndicator) SetState(state StatusState) {
	// If changing from running state, clean up animation
	if si.state == StatusRunning && si.animationController != nil && si.spinnerID != "" {
		si.animationController.DeregisterSpinner(si.spinnerID)
		si.spinnerID = ""
	}

	si.state = state

	// If changing to running state, set up animation
	if state == StatusRunning && si.animationEnabled && si.animationController != nil {
		si.spinnerID = fmt.Sprintf("status-%p-%d", si, time.Now().UnixNano())
		si.animationController.RegisterSpinner(si.spinnerID, si.spinnerFrames, si.spinnerSpeed)
	}
}

// GetState returns the current status indicator state
func (si *StatusIndicator) GetState() StatusState {
	return si.state
}

// Render generates the status indicator display with semantic token styling
func (si *StatusIndicator) Render() string {
	icon := si.getIcon()
	styledIcon := si.styleIcon(icon)
	return fmt.Sprintf("[%s]", styledIcon)
}

// RenderPlain generates the status indicator without styling (for width calculations)
func (si *StatusIndicator) RenderPlain() string {
	icon := si.getPlainIcon()
	return fmt.Sprintf("[%s]", icon)
}

// GetWidth returns the visual width of the status indicator (always 3 characters: [X])
func (si *StatusIndicator) GetWidth() int {
	return 3
}

// getIcon returns the appropriate icon for the current state
func (si *StatusIndicator) getIcon() string {
	switch si.state {
	case StatusQueued:
		return " " // Space for queued state

	case StatusRunning:
		if si.animationEnabled && si.animationController != nil && si.spinnerID != "" {
			// Get current animated frame
			frame := si.animationController.GetCurrentFrame(si.spinnerID)
			if frame != "" {
				return frame
			}
		}
		// Fallback to static spinner frame
		if len(si.spinnerFrames) > 0 {
			return si.spinnerFrames[0]
		}
		return "⠋" // Default spinner character

	case StatusSuccess:
		return "✓"

	case StatusFailed:
		return "x"

	case StatusWarning:
		return "!"

	case StatusCustom:
		if si.customIcon != "" {
			return si.customIcon
		}
		return "?" // Default for custom without icon set

	default:
		return "?" // Unknown state fallback
	}
}

// getPlainIcon returns icon without any styling considerations
func (si *StatusIndicator) getPlainIcon() string {
	switch si.state {
	case StatusQueued:
		return " "
	case StatusRunning:
		return "⠋" // Always use first frame for plain rendering
	case StatusSuccess:
		return "✓"
	case StatusFailed:
		return "x"
	case StatusWarning:
		return "!"
	case StatusCustom:
		if si.customIcon != "" {
			return si.customIcon
		}
		return "?"
	default:
		return "?"
	}
}

// styleIcon applies semantic token styling to the icon
func (si *StatusIndicator) styleIcon(icon string) string {
	var semanticToken string

	switch si.state {
	case StatusQueued:
		semanticToken = "tui.StatusQueued"
	case StatusRunning:
		semanticToken = "tui.StatusRunning"
	case StatusSuccess:
		semanticToken = "tui.StatusSuccess"
	case StatusFailed:
		semanticToken = "tui.StatusFailed"
	case StatusWarning:
		semanticToken = "tui.StatusWarning"
	default:
		semanticToken = "tui.StatusQueued" // Default styling
	}

	return si.formatter.Style(icon, semanticToken)
}

// Cleanup properly cleans up animation resources
func (si *StatusIndicator) Cleanup() {
	if si.animationController != nil && si.spinnerID != "" {
		si.animationController.DeregisterSpinner(si.spinnerID)
		si.spinnerID = ""
	}
}

// IsAnimated returns true if the status indicator is currently animated
func (si *StatusIndicator) IsAnimated() bool {
	return si.state == StatusRunning && si.animationEnabled && si.animationController != nil && si.spinnerID != ""
}

// GetAnimationID returns the animation ID for debugging/monitoring
func (si *StatusIndicator) GetAnimationID() string {
	return si.spinnerID
}

// Convenience constructors for common status indicator configurations

// NewQueuedStatus creates a status indicator in queued state
func NewQueuedStatus() *StatusIndicator {
	return NewStatusIndicator(StatusQueued)
}

// NewRunningStatus creates an animated status indicator in running state
func NewRunningStatus(controller *rendering.AnimationController) *StatusIndicator {
	status := NewStatusIndicator(StatusRunning).
		WithAnimationController(controller).
		WithAnimation(true)

	// Automatically set up animation
	status.SetState(StatusRunning)
	return status
}

// NewSuccessStatus creates a status indicator in success state
func NewSuccessStatus() *StatusIndicator {
	return NewStatusIndicator(StatusSuccess)
}

// NewFailedStatus creates a status indicator in failed state
func NewFailedStatus() *StatusIndicator {
	return NewStatusIndicator(StatusFailed)
}

// NewWarningStatus creates a status indicator in warning state
func NewWarningStatus() *StatusIndicator {
	return NewStatusIndicator(StatusWarning)
}

// NewCustomStatus creates a status indicator with custom icon
func NewCustomStatus(icon string) *StatusIndicator {
	return NewStatusIndicator(StatusCustom).WithCustomIcon(icon)
}

// StatusIndicatorGroup manages multiple related status indicators
type StatusIndicatorGroup struct {
	indicators []StatusIndicator
	controller *rendering.AnimationController
}

// NewStatusIndicatorGroup creates a new group of status indicators
func NewStatusIndicatorGroup(controller *rendering.AnimationController) *StatusIndicatorGroup {
	return &StatusIndicatorGroup{
		indicators: make([]StatusIndicator, 0),
		controller: controller,
	}
}

// Add adds a status indicator to the group
func (sig *StatusIndicatorGroup) Add(indicator *StatusIndicator) {
	if sig.controller != nil {
		indicator.WithAnimationController(sig.controller)
	}
	sig.indicators = append(sig.indicators, *indicator)
}

// SetAllState sets all indicators in the group to the same state
func (sig *StatusIndicatorGroup) SetAllState(state StatusState) {
	for i := range sig.indicators {
		sig.indicators[i].SetState(state)
	}
}

// Cleanup cleans up all animations in the group
func (sig *StatusIndicatorGroup) Cleanup() {
	for i := range sig.indicators {
		sig.indicators[i].Cleanup()
	}
}

// RenderAll renders all status indicators in the group
func (sig *StatusIndicatorGroup) RenderAll() []string {
	results := make([]string, len(sig.indicators))
	for i, indicator := range sig.indicators {
		results[i] = indicator.Render()
	}
	return results
}
