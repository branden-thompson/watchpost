/*
Package rendering provides terminal rendering utilities for GO-STUDS components.

# Overview

The rendering package consolidates color handling and terminal rendering logic
from multiple locations across the codebase. It provides optimized, battle-tested
utilities for ANSI color application and display width calculations.

This package eliminates ~400 lines of duplicated color handling code from:
  - plugins/app-status/renderers
  - internal/tui/components
  - internal/setup (wizard)
  - plugins/crews/renderers (4+ duplicate implementations)

# Key Features

  - ANSI color code application with 256-color support
  - Display width calculation for colored text (ANSI-aware)
  - Optimized string building to reduce memory allocations
  - Support for both basic ANSI and extended 256-color palette
  - Single source of truth for color handling

# Usage Examples

Basic usage:

	import "github.com/branden-thompson/watchpost/third_party/go-studs/rendering"

	colorUtils := rendering.NewColorUtils()
	colored := colorUtils.ApplyColor("Text", "92")

Simple one-off coloring:

	text := rendering.ApplyColorSimple("Warning", "93")

# Related Packages

  - internal/tui/design - Color constant definitions
  - pkg/go-studs/components - GO-STUDS component library
*/
package rendering
