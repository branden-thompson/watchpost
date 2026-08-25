/*
Package components provides professional terminal UI building blocks following the GO-STUDS design pattern.

# GO-STUDS Design Philosophy

GO-STUDS = **GO Stylized Terminal UI Design System**

Core principles:
1. **Single-function entry points** - Primary API uses one function per component
2. **Go idioms** - Empty strings for optional params, variadics for settings
3. **Terminal-width aware** - Auto-detect terminal width by default
4. **Direct return values** - No chaining required for common cases
5. **Advanced API available** - Builder pattern for customization

# Example: SmartSeparator

	// Simple API (recommended)
	separator := CreateSmartSeparator("Left Label", "Right Label")
	fmt.Print(separator)

	// Advanced API (for customization)
	sep := NewSmartSeparator().
	    WithLeftLabel("Custom").
	    WithTerminalWidth(120).
	    WithColor(design.ColorPrimary)
	fmt.Print(sep.Render())

# Available Components

## SmartSeparator (smart_separator.go)

Intelligent separator lines with optional labels:

	CreateSmartSeparator("", "")              // Pure separator
	CreateSmartSeparator("Header", "")        // Left label only
	CreateSmartSeparator("", "Info")          // Right label only
	CreateSmartSeparator("Left", "Right")     // Both labels
	CreateSmartSeparator("L", "R", 80)        // Explicit width

## TextFormatter (text_formatter.go)

Terminal-width aware text utilities:

	formatter := NewTextFormatter()
	width := formatter.GetTerminalWidth()
	wrapped := formatter.WrapText(longText, width)

## CLISyntaxHighlighter (syntax_highlighter.go)

Syntax highlighting using Chroma library:

	highlighter := NewCLISyntaxHighlighter()
	colored := highlighter.HighlightGo(code)
	colored := highlighter.HighlightCommand(bashCmd)

Supports 300+ languages: Go, JavaScript, Python, Bash, etc.

## ProgressBar (progress.go)

Terminal-width aware progress bars. Default preset is "modern":

	pb := NewProgressBar(0.5).WithLabel("Installing")
	fmt.Println(pb.Render())
	// │███████████████░░░░░░░░░░░░░░░│ 50.00%

Other presets via the single-call factory:

	CreateProgressBar("Build", "npm", 75.0, 0, 0, ProgressRunning, 80)
	CreateProgressBar("", "squares", 50.0, 0, 0, ProgressRunning, 80)

Available styles: "modern" (default), "dos", "npm", "squares", "rounded",
"diamond". Empty string and "default" resolve to "modern".

## Spinner (spinner.go)

Animated loading spinners:

	spinner := NewSpinner()
	frame := spinner.Next()  // ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏

# Design System Integration

All components integrate with the built-in token system:

	import "github.com/branden-thompson/watchpost/third_party/go-studs/tokens"

	separator := CreateSmartSeparator("Title", "")
	// Uses semantic color tokens automatically

# Color Handling

Components support both basic ANSI and 256-color:

	// Basic ANSI (0-97)
	text := ApplyColor("Text", "91")  // Bright red

	// 256-color palette (0-255)
	text := ApplyColor("Text", "196")  // Deep red

The color system automatically detects and applies the correct ANSI format.

# Terminal Width Awareness

Components auto-detect terminal width:

	// Auto-detect (recommended)
	sep := CreateSmartSeparator("Title", "")

	// Override for testing
	sep := CreateSmartSeparator("Title", "", 80)

Test across widths:

	COLUMNS=60 ./app crews web
	COLUMNS=80 ./app crews web
	COLUMNS=120 ./app crews web

# Creating New Components

Follow the GO-STUDS pattern:

	// 1. Create simple entry point
	func CreateMyComponent(required string, optional string, settings ...int) string {
	    width := getTerminalWidth()
	    if len(settings) > 0 {
	        width = settings[0]
	    }

	    comp := NewMyComponent()
	    comp.SetRequiredField(required)
	    if optional != "" {
	        comp.SetOptionalField(optional)
	    }
	    comp.SetTerminalWidth(width)

	    return comp.Render()
	}

	// 2. Provide advanced builder API
	type MyComponent struct {
	    required      string
	    optional      string
	    terminalWidth int
	}

	func NewMyComponent() *MyComponent {
	    return &MyComponent{
	        terminalWidth: getTerminalWidth(),
	    }
	}

	func (c *MyComponent) WithOptional(val string) *MyComponent {
	    c.optional = val
	    return c
	}

	func (c *MyComponent) Render() string {
	    // Implementation
	}

# Related Packages

  - github.com/branden-thompson/watchpost/third_party/go-studs/tokens - Token system (colors, design tokens, semantic tokens)
  - github.com/branden-thompson/watchpost/third_party/go-studs/rendering - Text rendering utilities (formatters, color utils, animation)

# Testing Components

	func TestSmartSeparator(t *testing.T) {
	    widths := []int{60, 80, 100, 120}
	    for _, width := range widths {
	        sep := CreateSmartSeparator("Test", "Info", width)
	        assert.Equal(t, width, len(sep))
	    }
	}

# Common Pitfalls

## Using len() for Colored Text

DON'T use len() on ANSI-colored strings:

	// ❌ WRONG
	colored := ApplyColor("Test", "196")
	width := len(colored)  // Counts ANSI codes!

	// ✅ CORRECT
	width := getDisplayWidth(colored)  // Strips ANSI first

## Applying Colors Before Width Calculations

DON'T apply colors before calculating widths:

	// ❌ WRONG
	text := ApplyColor(label, color)
	padding := width - len(text)  // Wrong width!

	// ✅ CORRECT
	padding := width - len(label)  // Calculate first
	text := ApplyColor(label, color)  // Apply color last

For more examples, see:
  - docs/02_features/go-studs-refactoring/
  - Main README.md "GO-STUDS Library" section
*/
package components
