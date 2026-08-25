package rendering

import (
	"sync"
	"time"
)

// AnimationController manages global animation state for spinners and progress indicators
type AnimationController struct {
	spinners    map[string]*SpinnerState
	mutex       sync.RWMutex
	ticker      *time.Ticker
	running     bool
	framerate   time.Duration
	stopChannel chan bool
}

// SpinnerState represents the state of an individual animated spinner
type SpinnerState struct {
	ID           string
	Frames       []string
	CurrentFrame int
	Speed        time.Duration // Duration between frame updates
	LastUpdate   time.Time
	Active       bool
	LoopCount    int // Number of complete cycles (for debugging/monitoring)
}

// NewAnimationController creates a new global animation controller
func NewAnimationController() *AnimationController {
	return &AnimationController{
		spinners:    make(map[string]*SpinnerState),
		framerate:   50 * time.Millisecond, // 20fps default
		stopChannel: make(chan bool, 1),
	}
}

// Start begins the global animation loop
func (ac *AnimationController) Start() {
	ac.mutex.Lock()
	if ac.running {
		ac.mutex.Unlock()
		return
	}

	ac.running = true
	ac.ticker = time.NewTicker(ac.framerate)
	ac.mutex.Unlock()

	go ac.animationLoop()
}

// Stop halts the global animation loop and cleans up resources
func (ac *AnimationController) Stop() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if !ac.running {
		return
	}

	ac.running = false
	if ac.ticker != nil {
		ac.ticker.Stop()
	}

	select {
	case ac.stopChannel <- true:
	default:
	}

	// Clean up all spinners
	ac.spinners = make(map[string]*SpinnerState)
}

// RegisterSpinner adds a new spinner to the animation system
func (ac *AnimationController) RegisterSpinner(id string, frames []string, speed time.Duration) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	ac.spinners[id] = &SpinnerState{
		ID:           id,
		Frames:       frames,
		CurrentFrame: 0,
		Speed:        speed,
		LastUpdate:   time.Now(),
		Active:       true,
		LoopCount:    0,
	}
}

// DeregisterSpinner removes a spinner from the animation system
func (ac *AnimationController) DeregisterSpinner(id string) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	delete(ac.spinners, id)
}

// GetCurrentFrame returns the current frame for a spinner
func (ac *AnimationController) GetCurrentFrame(id string) string {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	spinner, exists := ac.spinners[id]
	if !exists || !spinner.Active || len(spinner.Frames) == 0 {
		return ""
	}

	return spinner.Frames[spinner.CurrentFrame]
}

// SetSpinnerActive enables or disables a specific spinner
func (ac *AnimationController) SetSpinnerActive(id string, active bool) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if spinner, exists := ac.spinners[id]; exists {
		spinner.Active = active
	}
}

// GetSpinnerState returns the current state of a spinner (for debugging)
func (ac *AnimationController) GetSpinnerState(id string) *SpinnerState {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	if spinner, exists := ac.spinners[id]; exists {
		// Return a copy to prevent external modification
		return &SpinnerState{
			ID:           spinner.ID,
			Frames:       spinner.Frames,
			CurrentFrame: spinner.CurrentFrame,
			Speed:        spinner.Speed,
			LastUpdate:   spinner.LastUpdate,
			Active:       spinner.Active,
			LoopCount:    spinner.LoopCount,
		}
	}
	return nil
}

// GetStats returns animation system statistics
func (ac *AnimationController) GetStats() map[string]interface{} {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	activeCount := 0
	for _, spinner := range ac.spinners {
		if spinner.Active {
			activeCount++
		}
	}

	return map[string]interface{}{
		"total_spinners":  len(ac.spinners),
		"active_spinners": activeCount,
		"framerate_ms":    ac.framerate.Milliseconds(),
		"running":         ac.running,
	}
}

// animationLoop runs the main animation update loop
func (ac *AnimationController) animationLoop() {
	defer func() {
		ac.mutex.Lock()
		ac.running = false
		ac.mutex.Unlock()
	}()

	for {
		select {
		case <-ac.ticker.C:
			ac.updateSpinners()

		case <-ac.stopChannel:
			return
		}
	}
}

// updateSpinners advances animation frames for all active spinners
func (ac *AnimationController) updateSpinners() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	now := time.Now()

	for _, spinner := range ac.spinners {
		if !spinner.Active || len(spinner.Frames) == 0 {
			continue
		}

		// Check if enough time has passed for this spinner's update
		if now.Sub(spinner.LastUpdate) >= spinner.Speed {
			spinner.CurrentFrame = (spinner.CurrentFrame + 1) % len(spinner.Frames)
			spinner.LastUpdate = now

			// Track complete cycles for monitoring
			if spinner.CurrentFrame == 0 {
				spinner.LoopCount++
			}
		}
	}
}

// Predefined spinner frame sets for common use cases
var (
	SpinnerDots     = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	SpinnerLine     = []string{"|", "/", "-", "\\"}
	SpinnerArrow    = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	SpinnerPulse    = []string{"●", "○", "●", "○"}
	SpinnerBounce   = []string{"⠁", "⠂", "⠄", "⠂"}
	SpinnerClock    = []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
	SpinnerProgress = []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}
)

// DefaultSpinnerSpeed is the recommended update interval for smooth animation
const DefaultSpinnerSpeed = 100 * time.Millisecond

// Performance constants
const (
	MaxFramerate         = 50 * time.Millisecond  // 20fps maximum to prevent excessive CPU usage
	MinFramerate         = 500 * time.Millisecond // 2fps minimum for very slow animations
	RecommendedFramerate = 100 * time.Millisecond // 10fps recommended for good balance
)

// SetFramerate updates the global animation framerate with bounds checking
func (ac *AnimationController) SetFramerate(framerate time.Duration) {
	if framerate < MaxFramerate {
		framerate = MaxFramerate
	}
	if framerate > MinFramerate {
		framerate = MinFramerate
	}

	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	ac.framerate = framerate
	if ac.running && ac.ticker != nil {
		ac.ticker.Stop()
		ac.ticker = time.NewTicker(framerate)
	}
}
