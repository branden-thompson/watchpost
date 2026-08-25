package components

import (
	"fmt"
	"time"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// SpinnerType represents different predefined spinner animation styles
type SpinnerType int

const (
	SpinnerDots   SpinnerType = iota // ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
	SpinnerLines                     // |/-\
	SpinnerArrows                    // ←↖↑↗→↘↓↙
	SpinnerBounce                    // ⠁⠂⠄⠂
	SpinnerPulse                     // ●○◉○
	SpinnerClock                     // 🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛
	SpinnerBars                      // ▁▃▄▅▆▇█▇▆▅▄▃
	SpinnerCustom                    // User-defined frames
)

// Spinner represents an animated spinner component with configurable frames and timing
type Spinner struct {
	spinnerType         SpinnerType
	customFrames        []string
	speed               time.Duration
	spinnerID           string
	label               string
	showLabel           bool
	labelPosition       SpinnerLabelPosition
	running             bool
	animationController *rendering.AnimationController
	formatter           *rendering.TextFormatter
}

// SpinnerLabelPosition represents where to place the label relative to the spinner
type SpinnerLabelPosition int

const (
	LabelLeft  SpinnerLabelPosition = iota // "Loading... ⠋"
	LabelRight                             // "⠋ Loading..."
	LabelAbove                             // "Loading...\n⠋"
	LabelBelow                             // "⠋\nLoading..."
)

// NewSpinner creates a new spinner with default configuration
func NewSpinner(spinnerType SpinnerType, controller *rendering.AnimationController) *Spinner {
	return &Spinner{
		spinnerType:         spinnerType,
		speed:               rendering.DefaultSpinnerSpeed,
		spinnerID:           "",
		showLabel:           false,
		labelPosition:       LabelRight,
		running:             false,
		animationController: controller,
		formatter:           rendering.NewTextFormatter(),
	}
}

// WithLabel sets the spinner label text
func (s *Spinner) WithLabel(label string) *Spinner {
	s.label = label
	s.showLabel = len(label) > 0
	return s
}

// WithLabelPosition sets where the label appears relative to the spinner
func (s *Spinner) WithLabelPosition(position SpinnerLabelPosition) *Spinner {
	s.labelPosition = position
	return s
}

// WithSpeed sets the animation speed
func (s *Spinner) WithSpeed(speed time.Duration) *Spinner {
	s.speed = speed
	return s
}

// WithCustomFrames sets custom animation frames (automatically sets type to SpinnerCustom)
func (s *Spinner) WithCustomFrames(frames ...string) *Spinner {
	s.customFrames = frames
	s.spinnerType = SpinnerCustom
	return s
}

// Start begins the spinner animation
func (s *Spinner) Start() error {
	if s.running || s.animationController == nil {
		return fmt.Errorf("spinner already running or no animation controller")
	}

	// Generate unique spinner ID
	s.spinnerID = fmt.Sprintf("spinner-%p-%d", s, time.Now().UnixNano())

	// Get frames for the spinner type
	frames := s.getFrames()
	if len(frames) == 0 {
		return fmt.Errorf("no frames available for spinner type")
	}

	// Register with animation controller
	s.animationController.RegisterSpinner(s.spinnerID, frames, s.speed)
	s.running = true

	return nil
}

// Stop halts the spinner animation
func (s *Spinner) Stop() {
	if !s.running || s.animationController == nil || s.spinnerID == "" {
		return
	}

	s.animationController.DeregisterSpinner(s.spinnerID)
	s.running = false
	s.spinnerID = ""
}

// IsRunning returns true if the spinner is currently animating
func (s *Spinner) IsRunning() bool {
	return s.running
}

// Render generates the current spinner display
func (s *Spinner) Render() string {
	frame := s.getCurrentFrame()
	styledFrame := s.formatter.Style(frame, "tui.SpinnerFrame")

	if !s.showLabel || s.label == "" {
		return styledFrame
	}

	styledLabel := s.formatter.Style(s.label, "tui.SpinnerLabel")

	switch s.labelPosition {
	case LabelLeft:
		return styledLabel + " " + styledFrame
	case LabelRight:
		return styledFrame + " " + styledLabel
	case LabelAbove:
		return styledLabel + "\n" + styledFrame
	case LabelBelow:
		return styledFrame + "\n" + styledLabel
	default:
		return styledFrame + " " + styledLabel
	}
}

// RenderPlain generates the spinner display without styling (for width calculations)
func (s *Spinner) RenderPlain() string {
	frame := s.getCurrentPlainFrame()

	if !s.showLabel || s.label == "" {
		return frame
	}

	switch s.labelPosition {
	case LabelLeft:
		return s.label + " " + frame
	case LabelRight:
		return frame + " " + s.label
	case LabelAbove:
		return s.label + "\n" + frame
	case LabelBelow:
		return frame + "\n" + s.label
	default:
		return frame + " " + s.label
	}
}

// getCurrentFrame gets the current animation frame from the controller
func (s *Spinner) getCurrentFrame() string {
	if !s.running || s.animationController == nil || s.spinnerID == "" {
		// Return first frame as static fallback
		frames := s.getFrames()
		if len(frames) > 0 {
			return frames[0]
		}
		return "⠋" // Default fallback
	}

	frame := s.animationController.GetCurrentFrame(s.spinnerID)
	if frame == "" {
		// Fallback to first frame if controller returns empty
		frames := s.getFrames()
		if len(frames) > 0 {
			return frames[0]
		}
		return "⠋"
	}

	return frame
}

// getCurrentPlainFrame gets the current frame without animation (for width calculations)
func (s *Spinner) getCurrentPlainFrame() string {
	frames := s.getFrames()
	if len(frames) > 0 {
		return frames[0]
	}
	return "⠋"
}

// getFrames returns the animation frames for the current spinner type
func (s *Spinner) getFrames() []string {
	switch s.spinnerType {
	case SpinnerDots:
		return rendering.SpinnerDots
	case SpinnerLines:
		return []string{"|", "/", "-", "\\"}
	case SpinnerArrows:
		return []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	case SpinnerBounce:
		return []string{"⠁", "⠂", "⠄", "⠂"}
	case SpinnerPulse:
		return []string{"●", "○", "◉", "○"}
	case SpinnerClock:
		return []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
	case SpinnerBars:
		return []string{"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃"}
	case SpinnerCustom:
		if len(s.customFrames) > 0 {
			return s.customFrames
		}
		// Fallback to dots if custom frames are empty
		return rendering.SpinnerDots
	default:
		return rendering.SpinnerDots
	}
}

// GetFrameCount returns the number of frames in the animation
func (s *Spinner) GetFrameCount() int {
	return len(s.getFrames())
}

// GetSpeed returns the current animation speed
func (s *Spinner) GetSpeed() time.Duration {
	return s.speed
}

// GetVisualWidth returns the visual width of the spinner (accounting for label)
func (s *Spinner) GetVisualWidth() int {
	plainRender := s.RenderPlain()

	// For multi-line renders (above/below), return width of widest line
	if s.labelPosition == LabelAbove || s.labelPosition == LabelBelow {
		lines := []string{s.label, s.getCurrentPlainFrame()}
		maxWidth := 0
		for _, line := range lines {
			if len(line) > maxWidth {
				maxWidth = len(line)
			}
		}
		return maxWidth
	}

	return len(plainRender)
}

// Cleanup properly cleans up spinner resources
func (s *Spinner) Cleanup() {
	s.Stop()
}

// SpinnerGroup manages multiple related spinners
type SpinnerGroup struct {
	spinners   []*Spinner
	controller *rendering.AnimationController
}

// NewSpinnerGroup creates a new group of spinners
func NewSpinnerGroup(controller *rendering.AnimationController) *SpinnerGroup {
	return &SpinnerGroup{
		spinners:   make([]*Spinner, 0),
		controller: controller,
	}
}

// Add adds a spinner to the group
func (sg *SpinnerGroup) Add(spinner *Spinner) {
	if sg.controller != nil {
		spinner.animationController = sg.controller
	}
	sg.spinners = append(sg.spinners, spinner)
}

// StartAll starts all spinners in the group
func (sg *SpinnerGroup) StartAll() {
	for _, spinner := range sg.spinners {
		spinner.Start()
	}
}

// StopAll stops all spinners in the group
func (sg *SpinnerGroup) StopAll() {
	for _, spinner := range sg.spinners {
		spinner.Stop()
	}
}

// Cleanup cleans up all spinners in the group
func (sg *SpinnerGroup) Cleanup() {
	sg.StopAll()
}

// RenderAll renders all spinners in the group
func (sg *SpinnerGroup) RenderAll() []string {
	results := make([]string, len(sg.spinners))
	for i, spinner := range sg.spinners {
		results[i] = spinner.Render()
	}
	return results
}

// Count returns the number of spinners in the group
func (sg *SpinnerGroup) Count() int {
	return len(sg.spinners)
}

// Convenience constructors for common spinner configurations

// NewDotsSpinner creates a dots-style spinner with optional label
func NewDotsSpinner(controller *rendering.AnimationController, label string) *Spinner {
	spinner := NewSpinner(SpinnerDots, controller)
	if label != "" {
		spinner.WithLabel(label)
	}
	return spinner
}

// NewLinesSpinner creates a lines-style spinner with optional label
func NewLinesSpinner(controller *rendering.AnimationController, label string) *Spinner {
	spinner := NewSpinner(SpinnerLines, controller)
	if label != "" {
		spinner.WithLabel(label)
	}
	return spinner
}

// NewArrowSpinner creates an arrows-style spinner with optional label
func NewArrowSpinner(controller *rendering.AnimationController, label string) *Spinner {
	spinner := NewSpinner(SpinnerArrows, controller)
	if label != "" {
		spinner.WithLabel(label)
	}
	return spinner
}

// NewPulseSpinner creates a pulse-style spinner with optional label
func NewPulseSpinner(controller *rendering.AnimationController, label string) *Spinner {
	spinner := NewSpinner(SpinnerPulse, controller)
	if label != "" {
		spinner.WithLabel(label)
	}
	return spinner
}

// NewCustomSpinner creates a spinner with custom frames
func NewCustomSpinner(controller *rendering.AnimationController, label string, frames ...string) *Spinner {
	spinner := NewSpinner(SpinnerCustom, controller).
		WithCustomFrames(frames...)
	if label != "" {
		spinner.WithLabel(label)
	}
	return spinner
}

// NewFastSpinner creates a spinner with faster animation (75ms per frame)
func NewFastSpinner(spinnerType SpinnerType, controller *rendering.AnimationController, label string) *Spinner {
	spinner := NewSpinner(spinnerType, controller).
		WithSpeed(75 * time.Millisecond)
	if label != "" {
		spinner.WithLabel(label)
	}
	return spinner
}

// NewSlowSpinner creates a spinner with slower animation (200ms per frame)
func NewSlowSpinner(spinnerType SpinnerType, controller *rendering.AnimationController, label string) *Spinner {
	spinner := NewSpinner(spinnerType, controller).
		WithSpeed(200 * time.Millisecond)
	if label != "" {
		spinner.WithLabel(label)
	}
	return spinner
}

// LoadingSpinner provides a convenient loading indicator
type LoadingSpinner struct {
	spinner   *Spinner
	message   string
	startTime time.Time
}

// NewLoadingSpinner creates a new loading spinner with message
func NewLoadingSpinner(controller *rendering.AnimationController, message string) *LoadingSpinner {
	spinner := NewDotsSpinner(controller, message).
		WithLabelPosition(LabelLeft)

	return &LoadingSpinner{
		spinner: spinner,
		message: message,
	}
}

// Start begins the loading animation
func (ls *LoadingSpinner) Start() {
	ls.startTime = time.Now()
	ls.spinner.Start()
}

// Stop ends the loading animation
func (ls *LoadingSpinner) Stop() {
	ls.spinner.Stop()
}

// UpdateMessage changes the loading message while running
func (ls *LoadingSpinner) UpdateMessage(message string) {
	ls.message = message
	ls.spinner.WithLabel(message)
}

// GetElapsed returns the time since the spinner started
func (ls *LoadingSpinner) GetElapsed() time.Duration {
	if ls.startTime.IsZero() {
		return 0
	}
	return time.Since(ls.startTime)
}

// Render renders the loading spinner with elapsed time
func (ls *LoadingSpinner) Render() string {
	return ls.spinner.Render()
}

// RenderWithElapsed renders the loading spinner with elapsed time indicator
func (ls *LoadingSpinner) RenderWithElapsed() string {
	if ls.startTime.IsZero() {
		return ls.spinner.Render()
	}

	elapsed := time.Since(ls.startTime)
	elapsedText := fmt.Sprintf("(%.1fs)", elapsed.Seconds())
	styledElapsed := ls.spinner.formatter.Style(elapsedText, "tui.SpinnerElapsed")

	baseRender := ls.spinner.Render()
	return baseRender + " " + styledElapsed
}

// Cleanup cleans up the loading spinner
func (ls *LoadingSpinner) Cleanup() {
	ls.spinner.Cleanup()
}
