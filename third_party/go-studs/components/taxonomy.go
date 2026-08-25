package components

// ComponentTaxonomy defines the classification system for GO-STUDS components
// Format: [Capability Adjective][Component Type]
// Example: "Terminal-width Aware Layout"
//
// ## Component Selection Guide
//
// GO-STUDS provides multiple components for similar purposes. This guide helps you choose correctly.
//
// ### SmartSeparator vs HeaderFooterComponent
//
// Both components render terminal-width aware headers/footers/separators, but serve different purposes:
//
// **Use HeaderFooterComponent when:**
//   - Implementing BaseRenderer pattern (standardized footers)
//   - Want consistent styling across views
//   - Don't need spacing or layout control
//   - Simple left/right label pattern
//   - Code simplicity is priority
//
// **Use SmartSeparator when:**
//   - Building complex nested layouts with insets
//   - Need spacing control (WithHeaderSpacing, WithFooterSpacing, WithNoSpacing)
//   - Variable lead/tail dash lengths (for custom insets)
//   - Responsive rendering for narrow terminals (ResponsiveRender)
//   - Label truncation needed
//   - Layout context awareness important
//
// For detailed examples and migration guides, see:
// docs/03_developer_guides/05-go-studs-component-selection.md

// CapabilityAdjective defines what special capabilities a component has
type CapabilityAdjective string

const (
	// Terminal behavior capabilities
	TerminalWidthAware CapabilityAdjective = "Terminal-width Aware"
	Static             CapabilityAdjective = "Static"

	// Animation capabilities
	Animated CapabilityAdjective = "Animated"

	// Interactive capabilities (future expansion)
	Interactive CapabilityAdjective = "Interactive"
	Responsive  CapabilityAdjective = "Responsive"
)

// ComponentType defines the primary function/category of the component
type ComponentType string

const (
	// Structural types
	Layout      ComponentType = "Layout"
	DataDisplay ComponentType = "Data Display"
	Interface   ComponentType = "Interface"
	Indicator   ComponentType = "Indicator"

	// Future expansion types
	Control    ComponentType = "Control"
	Navigation ComponentType = "Navigation"
	Feedback   ComponentType = "Feedback"
)

// ComponentClassification combines capability and type for full component description
type ComponentClassification struct {
	Capability CapabilityAdjective
	Type       ComponentType
}

// String returns the formatted classification string
func (cc ComponentClassification) String() string {
	return string(cc.Capability) + " " + string(cc.Type)
}

// NewComponentClassification creates a new component classification
func NewComponentClassification(capability CapabilityAdjective, componentType ComponentType) ComponentClassification {
	return ComponentClassification{
		Capability: capability,
		Type:       componentType,
	}
}

// Predefined component classifications for common GO-STUDS components
var (
	// SmartSeparator: Highly configurable terminal-width aware separator for complex layouts
	// Use when: Need spacing control, insets, responsive rendering, or label truncation
	// Example: Nested subdividers in app-status plugin
	SmartSeparatorClassification = NewComponentClassification(TerminalWidthAware, Layout)

	// Progress bars: terminal-width aware, show data/status
	ProgressBarClassification = NewComponentClassification(TerminalWidthAware, DataDisplay)

	// Spinners: animated status indicators
	SpinnerClassification         = NewComponentClassification(Animated, Indicator)
	AnimatedSpinnerClassification = NewComponentClassification(Animated, Indicator)

	// Data tables: terminal-width aware data display
	DataTableClassification = NewComponentClassification(TerminalWidthAware, DataDisplay)

	// Status badges: show state information
	StatusBadgeClassification = NewComponentClassification(Static, Indicator)

	// BadgeAligner: terminal-width aware badge right-alignment
	BadgeAlignerClassification = NewComponentClassification(TerminalWidthAware, Layout)

	// HeaderFooter: Simple, opinionated component for standardized headers/footers
	// Use when: Implementing BaseRenderer pattern, want consistent styling, simple use case
	// Example: Standard footer in crews plugin BaseRenderer
	HeaderFooterClassification = NewComponentClassification(TerminalWidthAware, Layout)

	// SyntaxHighlighting: static syntax highlighting for code and commands
	SyntaxHighlightingClassification = NewComponentClassification(Static, Layout)

	// Interactive forms: user input interfaces (future)
	FormClassification = NewComponentClassification(Interactive, Interface)
)
