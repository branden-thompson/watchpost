package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// EventStream represents a type of monitored resource
type EventStream struct {
	Category string // service, build, deploy, container, process
	Target   string // database, frontend, production, webapp, etc.
	Context  string // optional additional context
}

// ParseEventStream parses an event stream string into structured components
// Format: "category.target" or "category.target.context"
// Examples: "service.database", "build.frontend", "deploy.production.canary"
func ParseEventStream(eventStream string) EventStream {
	parts := strings.Split(eventStream, ".")

	stream := EventStream{}
	if len(parts) >= 1 {
		stream.Category = parts[0]
	}
	if len(parts) >= 2 {
		stream.Target = parts[1]
	}
	if len(parts) >= 3 {
		stream.Context = parts[2]
	}

	return stream
}

// CreateStatusIndicator creates a self-managing status indicator that monitors the specified event stream
// This is the primary API - just tell it what to watch, and it handles everything internally
func CreateStatusIndicator(eventStream string) string {
	stream := ParseEventStream(eventStream)
	monitor := newStatusMonitor(stream)
	return monitor.getCurrentStatus()
}

// statusMonitor handles all the internal state management and monitoring logic
type statusMonitor struct {
	stream       EventStream
	currentState StatusState
	startTime    time.Time
	formatter    *rendering.ANSIFormatter
	spinnerFrame int
}

// newStatusMonitor creates a new internal status monitor
func newStatusMonitor(stream EventStream) *statusMonitor {
	monitor := &statusMonitor{
		stream:       stream,
		currentState: StatusQueued, // Always start in queued state
		startTime:    time.Now(),
		formatter:    rendering.NewANSIFormatter(),
		spinnerFrame: 0,
	}

	// Simulate the monitoring process
	monitor.updateState()
	return monitor
}

// updateState simulates monitoring the event stream and updates state accordingly
// Uses deterministic mock data instead of timing for consistent demo behavior
func (sm *statusMonitor) updateState() {
	// Simulate realistic states based on event stream type and target
	switch sm.stream.Category {
	case "service":
		sm.updateServiceStatus()
	case "build":
		sm.updateBuildStatus()
	case "deploy":
		sm.updateDeployStatus()
	case "container":
		sm.updateContainerStatus()
	case "process":
		sm.updateProcessStatus()
	default:
		sm.updateGenericStatus()
	}
}

// updateServiceStatus simulates service monitoring with deterministic mock states
func (sm *statusMonitor) updateServiceStatus() {
	// Simulate service health check results based on service type
	switch sm.stream.Target {
	case "database":
		sm.currentState = StatusSuccess // Database is healthy
	case "external-api":
		sm.currentState = StatusWarning // External API has degraded performance
	case "cache":
		sm.currentState = StatusFailed // Cache service is down
	case "queued":
		sm.currentState = StatusQueued // Service starting up
	default:
		sm.currentState = StatusSuccess // Most services are healthy
	}
}

// updateBuildStatus simulates build process monitoring with deterministic states
func (sm *statusMonitor) updateBuildStatus() {
	// Simulate build results based on build target and context
	switch sm.stream.Target {
	case "frontend":
		sm.currentState = StatusRunning // Frontend build in progress (animated)
	case "backend":
		if sm.stream.Context == "tests" {
			sm.currentState = StatusFailed // Backend tests failed
		} else {
			sm.currentState = StatusSuccess // Backend build successful
		}
	case "queued":
		sm.currentState = StatusQueued // Build queued
	default:
		sm.currentState = StatusSuccess // Most builds succeed
	}
}

// updateDeployStatus simulates deployment monitoring with deterministic states
func (sm *statusMonitor) updateDeployStatus() {
	// Simulate deployment results based on environment and context
	switch sm.stream.Target {
	case "production":
		if sm.stream.Context == "canary" {
			sm.currentState = StatusWarning // Canary deployment has issues
		} else {
			sm.currentState = StatusSuccess // Production deployment successful
		}
	case "staging":
		if sm.stream.Context == "canary" {
			sm.currentState = StatusWarning // Staging canary has warnings
		} else {
			sm.currentState = StatusSuccess // Staging deployments usually succeed
		}
	case "queued":
		sm.currentState = StatusQueued // Deployment queued
	default:
		sm.currentState = StatusSuccess // Most deployments succeed
	}
}

// updateContainerStatus simulates container monitoring with deterministic states
func (sm *statusMonitor) updateContainerStatus() {
	// Simulate container health based on container type
	switch sm.stream.Target {
	case "webapp":
		sm.currentState = StatusSuccess // Web application container is running
	case "database":
		sm.currentState = StatusSuccess // Database container is running
	case "cache":
		sm.currentState = StatusFailed // Cache container failed to start
	case "queued":
		sm.currentState = StatusQueued // Container starting
	default:
		sm.currentState = StatusSuccess // Most containers start successfully
	}
}

// updateProcessStatus simulates process monitoring with deterministic states
func (sm *statusMonitor) updateProcessStatus() {
	// Simulate process health based on process type
	switch sm.stream.Target {
	case "background-sync":
		sm.currentState = StatusFailed // Background sync process failed
	case "queued":
		sm.currentState = StatusQueued // Process starting
	default:
		sm.currentState = StatusSuccess // Most processes run successfully
	}
}

// updateGenericStatus simulates generic monitoring for unknown categories
func (sm *statusMonitor) updateGenericStatus() {
	// Default to success for unknown types
	sm.currentState = StatusSuccess
}

// getCurrentStatus returns the current formatted status indicator
func (sm *statusMonitor) getCurrentStatus() string {
	icon := sm.getStatusIcon()
	styledIcon := sm.styleIcon(icon)
	return fmt.Sprintf("[%s]", styledIcon)
}

// getStatusIcon returns the appropriate icon for the current state
func (sm *statusMonitor) getStatusIcon() string {
	switch sm.currentState {
	case StatusQueued:
		return " " // Empty space for queued
	case StatusRunning:
		// Animate the spinner for running state
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frameIndex := int(time.Since(sm.startTime)/150/time.Millisecond) % len(spinnerFrames)
		return spinnerFrames[frameIndex]
	case StatusSuccess:
		return "✓"
	case StatusFailed:
		return "x"
	case StatusWarning:
		return "!"
	default:
		return "?"
	}
}

// styleIcon applies semantic token styling to the icon
func (sm *statusMonitor) styleIcon(icon string) string {
	var semanticToken string

	switch sm.currentState {
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
		semanticToken = "tui.StatusQueued"
	}

	return sm.formatter.Style(icon, semanticToken)
}

// Convenience functions for common event streams

// CreateServiceStatus creates a status indicator for service monitoring
func CreateServiceStatus(serviceName string) string {
	return CreateStatusIndicator(fmt.Sprintf("service.%s", serviceName))
}

// CreateBuildStatus creates a status indicator for build monitoring
func CreateBuildStatus(buildTarget string) string {
	return CreateStatusIndicator(fmt.Sprintf("build.%s", buildTarget))
}

// CreateDeployStatus creates a status indicator for deployment monitoring
func CreateDeployStatus(environment string) string {
	return CreateStatusIndicator(fmt.Sprintf("deploy.%s", environment))
}

// CreateContainerStatus creates a status indicator for container monitoring
func CreateContainerStatus(containerName string) string {
	return CreateStatusIndicator(fmt.Sprintf("container.%s", containerName))
}

// Examples of the declarative API in action:
//
// CreateStatusIndicator("service.database")           // [✓] - Database service is healthy
// CreateStatusIndicator("build.frontend")            // [⠋] - Frontend build in progress
// CreateStatusIndicator("deploy.production.canary")  // [!] - Canary deployment has warnings
// CreateStatusIndicator("container.webapp")          // [ ] - Webapp container starting
// CreateStatusIndicator("process.background-sync")   // [x] - Background sync process failed
//
// The human engineer just specifies what to watch, everything else is automatic!
