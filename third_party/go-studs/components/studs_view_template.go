package components

import (
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// StudsViewTemplate provides a centralized layout system for all STUDS component demonstrations
// Uses smart placeholder rendering - blank sections are automatically skipped
type StudsViewTemplate struct {
	ComponentName     string                  // e.g., "SmartSeparator", "Progress Bars"
	Classification    ComponentClassification // Component taxonomy classification
	CodeSignatures    []string                // Function signatures for Code Invocation section
	SpecimenContent   func() string           // Dynamic content for Specimens section (nil = skip)
	BreakpointContent func() string           // Dynamic content for Breakpoints section (nil = skip)
	FooterTitle       string                  // Footer left label (e.g., "GO-STUDS Progress Bars")
	FooterSubtitle    string                  // Footer right label (e.g., "Terminal-width Aware Data Display")
}

// NewStudsViewTemplate creates a new STUDS view template with the given component information
func NewStudsViewTemplate(componentName string, classification ComponentClassification) *StudsViewTemplate {
	return &StudsViewTemplate{
		ComponentName:  componentName,
		Classification: classification,
		FooterTitle:    fmt.Sprintf("GO-STUDS %s", componentName),
		FooterSubtitle: classification.String(),
	}
}

// WithCodeSignatures adds function signatures to the Code Invocation section
func (t *StudsViewTemplate) WithCodeSignatures(signatures ...string) *StudsViewTemplate {
	t.CodeSignatures = signatures
	return t
}

// WithSpecimens adds dynamic specimen content
func (t *StudsViewTemplate) WithSpecimens(contentFunc func() string) *StudsViewTemplate {
	t.SpecimenContent = contentFunc
	return t
}

// WithBreakpoints adds dynamic breakpoint content
func (t *StudsViewTemplate) WithBreakpoints(contentFunc func() string) *StudsViewTemplate {
	t.BreakpointContent = contentFunc
	return t
}

// WithFooter customizes the footer labels
func (t *StudsViewTemplate) WithFooter(title, subtitle string) *StudsViewTemplate {
	t.FooterTitle = title
	t.FooterSubtitle = subtitle
	return t
}

// Render generates the complete STUDS component view using the template
func (t *StudsViewTemplate) Render() {
	t.renderHeader()
	t.renderCodeInvocation()
	t.renderSpecimens()
	t.renderBreakpoints()
	t.renderFooter()
}

// RenderWithoutFooter generates the STUDS component view but skips the footer
// Use this when the component needs custom interactive elements after the template
func (t *StudsViewTemplate) RenderWithoutFooter() {
	t.renderHeader()
	t.renderCodeInvocation()
	t.renderSpecimens()
	t.renderBreakpoints()
}

// renderHeader creates the component header with classification
func (t *StudsViewTemplate) renderHeader() {
	formatter := rendering.NewTextFormatter()
	componentPart := formatter.Style(fmt.Sprintf("STUDS COMPONENT: %s", t.ComponentName), "tui.ComponentName")
	capabilityPart := formatter.Style(t.Classification.String(), "tui.CapabilityLabel")

	header := NewSmartSeparator().
		WithLeftLabel(componentPart).
		WithRightLabel(capabilityPart).
		WithHeaderSpacing()
	fmt.Print(header.Render())
}

// renderCodeInvocation renders the Code Invocation section if signatures are provided
func (t *StudsViewTemplate) renderCodeInvocation() {
	if len(t.CodeSignatures) == 0 {
		return // Skip section if no signatures
	}

	fmt.Println("Code Invocation:")
	fmt.Println()

	// Syntax highlight the function signatures with chroma
	chromaHighlighter := NewChromaHighlighter()
	for _, sig := range t.CodeSignatures {
		highlighted := chromaHighlighter.HighlightFunctionSignatureCompat(sig)
		fmt.Printf("   %s\n", highlighted)
	}
	fmt.Println()
}

// renderSpecimens renders the Specimens section if content function is provided
func (t *StudsViewTemplate) renderSpecimens() {
	if t.SpecimenContent == nil {
		return // Skip section if no content function
	}

	content := t.SpecimenContent()
	if strings.TrimSpace(content) == "" {
		return // Skip section if content is empty
	}

	fmt.Println("Specimens:")
	fmt.Println()
	fmt.Print(content)
	fmt.Println()
}

// renderBreakpoints renders the Breakpoints section if content function is provided
func (t *StudsViewTemplate) renderBreakpoints() {
	if t.BreakpointContent == nil {
		return // Skip section if no content function
	}

	content := t.BreakpointContent()
	if strings.TrimSpace(content) == "" {
		return // Skip section if content is empty
	}

	fmt.Println("Supported Breakpoints:")
	fmt.Println()
	fmt.Print(content)
	fmt.Println()
}

// renderFooter creates the component footer
func (t *StudsViewTemplate) renderFooter() {
	footer := NewSmartSeparator().
		WithLeftLabel(t.FooterTitle).
		WithRightLabel(t.FooterSubtitle).
		WithFooterSpacing()
	fmt.Print(footer.Render())
}
