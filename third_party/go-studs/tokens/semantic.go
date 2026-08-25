package tokens

// SemanticTokens defines the semantic token layer mapping component usage to design tokens
// This is the top layer of the 3-layer design token architecture
var SemanticTokens = map[string]string{
	// Text semantic tokens
	"tui.HeaderLabel":    "color.gray.light", // Primary headers and labels
	"tui.SubheaderLabel": "color.gray.muted", // Secondary headers
	"tui.BodyText":       "color.gray.light", // Main body text
	"tui.MutedText":      "color.gray.muted", // Secondary, muted text

	// Status semantic tokens
	"tui.SuccessText": "color.green.success",  // Success messages and indicators
	"tui.ErrorText":   "color.red.error",      // Error messages and indicators
	"tui.WarningText": "color.yellow.warning", // Warning messages
	"tui.InfoText":    "color.blue.muted",     // Informational text

	// Component semantic tokens
	"tui.CommandText":    "color.orange.cmd",   // Command text in backticks
	"tui.TableHeader":    "color.purple.light", // Table column headers (light purple)
	"tui.TableRowNumber": "color.gray.muted",   // Table row numbers
	"tui.TableAttribute": "color.gray.muted",   // Table attribute text
	"tui.TableLabel":     "color.gray.light",   // Table content labels

	// On-call status colors (mapped to generic design tokens upstream — the
	// legacy color.red.oncall / color.orange.oncall aliases resolve to the
	// same raw values: ansi-196 / ansi-202)
	"tui.OnCallPrimary":   "color.red.bright", // Primary on-call (red)
	"tui.OnCallSecondary": "color.red.muted",  // Secondary on-call (orange)

	// Data table semantic tokens
	"tui.TableCell":      "color.gray.light", // Table cell content
	"tui.TableSeparator": "color.gray.dark",  // Table separator lines
	"tui.TableBorder":    "color.gray.dark",  // Table borders

	// Status indicator semantic tokens
	"tui.StatusQueued":   "color.gray.muted",     // Queued/pending status [ ]
	"tui.StatusRunning":  "color.blue.bright",    // Running status with spinner [⠋]
	"tui.StatusSuccess":  "color.green.success",  // Success status [✓]
	"tui.StatusFailed":   "color.red.error",      // Failed status [x]
	"tui.StatusWarning":  "color.yellow.warning", // Warning status [!]
	"tui.StatusBrackets": "color.gray.dark",      // Status indicator brackets

	// Progress bar semantic tokens
	"tui.ProgressLabel":    "color.gray.light",  // Progress bar labels
	"tui.ProgressFill":     "color.blue.bright", // Progress bar fill characters
	"tui.ProgressEmpty":    "color.gray.dark",   // Progress bar empty characters
	"tui.ProgressPercent":  "color.gray.light",  // Progress percentage text
	"tui.ProgressSteps":    "color.gray.muted",  // Progress step counter
	"tui.ProgressBrackets": "color.gray.dark",   // Progress bar brackets

	// Separator semantic tokens
	"tui.SeparatorLine":   "color.gray.dark",  // Separator line characters
	"tui.SeparatorDots":   "color.gray.dark",  // Separator dot characters
	"tui.SeparatorDash":   "color.gray.dark",  // Separator dash characters
	"tui.SeparatorEqual":  "color.gray.dark",  // Separator equal characters
	"tui.SeparatorWave":   "color.gray.dark",  // Separator wave characters
	"tui.SeparatorCustom": "color.gray.dark",  // Separator custom characters
	"tui.SeparatorLabel":  "color.gray.light", // Separator label text

	// CLI syntax highlighting semantic tokens (focused on terminal/command line content)
	"cli.Command":     "color.orange.cmd",     // CLI commands and tools
	"cli.Flag":        "color.blue.bright",    // Command line flags
	"cli.Path":        "color.gray.light",     // File and directory paths
	"cli.Variable":    "color.green.success",  // Environment variables
	"cli.String":      "color.yellow.warning", // Quoted strings in commands
	"cli.Comment":     "color.gray.muted",     // Shell comments
	"cli.URL":         "color.blue.bright",    // URLs
	"cli.Number":      "color.blue.bright",    // Numeric values (ports, versions)
	"cli.CommandLine": "color.gray.light",     // General command line styling
	"cli.Description": "color.gray.muted",     // Command descriptions

	// Legacy syntax highlighting semantic tokens (deprecated - use chroma for programming languages)
	"syntax.Command":     "color.orange.cmd",     // CLI commands and tools
	"syntax.Flag":        "color.blue.bright",    // Command line flags
	"syntax.Path":        "color.gray.light",     // File and directory paths
	"syntax.Variable":    "color.green.success",  // Variables and placeholders
	"syntax.String":      "color.yellow.warning", // Quoted strings
	"syntax.Keyword":     "color.blue.bright",    // Programming keywords
	"syntax.Comment":     "color.gray.muted",     // Comments
	"syntax.URL":         "color.blue.bright",    // URLs
	"syntax.Number":      "color.blue.bright",    // Numeric values
	"syntax.Operator":    "color.gray.light",     // Operators and symbols
	"syntax.CodeBlock":   "color.gray.light",     // Multi-line code blocks
	"syntax.InlineCode":  "color.orange.cmd",     // Inline code snippets
	"syntax.CommandLine": "color.orange.cmd",     // Command line formatting
	"syntax.Description": "color.gray.muted",     // Code descriptions

	// Go-specific syntax highlighting tokens
	"syntax.GoFunc":         "color.orange.cmd",    // Go 'func' keyword (orange)
	"syntax.GoFunctionName": "color.blue.muted",    // Go function names (muted blue)
	"syntax.GoType":         "color.magenta.lang",  // Go data types (magenta)
	"syntax.GoArgument":     "color.green.success", // Go function arguments (green)
	"syntax.GoPointer":      "color.red.error",     // Go pointers (*) (red)
	"syntax.GoComment":      "color.gray.muted",    // Go comments (gray - standard)

	// Spinner semantic tokens
	"tui.SpinnerFrame":   "color.blue.bright", // Spinner animation frames
	"tui.SpinnerLabel":   "color.gray.light",  // Spinner label text
	"tui.SpinnerElapsed": "color.gray.muted",  // Spinner elapsed time

	// Interactive semantic tokens
	"tui.PromptLabel": "color.blue.muted", // Prompt labels
	"tui.PromptInput": "color.gray.light", // User input text
	"tui.PromptHelp":  "color.gray.muted", // Help text in prompts

	// Special semantic tokens
	"tui.Highlight": "color.highlight", // Special highlights
	"tui.Primary":   "color.primary",   // Primary accent elements
	"tui.Secondary": "color.secondary", // Secondary accent elements

	// STUDS template header colors
	"tui.ComponentName":   "color.orange.cmd",     // Component names in headers (orange)
	"tui.CapabilityLabel": "color.magenta.header", // Capability adjectives (pink/magenta)

	// On-call and role specific tokens (from existing system)
	"tui.PrimaryOnCall":   "color.red.bright",    // Primary on-call indicators
	"tui.SecondaryOnCall": "color.red.muted",     // Secondary on-call indicators
	"tui.CurrentUser":     "color.green.success", // Current user highlighting
	"tui.OtherUser":       "color.blue.bright",   // Other user highlighting
}

// IsValidSemanticToken checks if a semantic token key exists
func IsValidSemanticToken(tokenKey string) bool {
	_, exists := SemanticTokens[tokenKey]
	return exists
}

// GetSemanticToken retrieves a semantic token value by key
func GetSemanticToken(tokenKey string) (string, bool) {
	token, exists := SemanticTokens[tokenKey]
	return token, exists
}

// UpdateSemanticToken allows runtime updates to semantic tokens
func UpdateSemanticToken(tokenKey, designTokenKey string) bool {
	if !IsValidDesignToken(designTokenKey) {
		return false
	}
	SemanticTokens[tokenKey] = designTokenKey
	return true
}

// GetAllSemanticTokens returns a copy of all semantic tokens for inspection
func GetAllSemanticTokens() map[string]string {
	tokens := make(map[string]string, len(SemanticTokens))
	for k, v := range SemanticTokens {
		tokens[k] = v
	}
	return tokens
}

// GetSemanticTokensByPrefix returns all semantic tokens matching a prefix
func GetSemanticTokensByPrefix(prefix string) map[string]string {
	matches := make(map[string]string)
	for k, v := range SemanticTokens {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			matches[k] = v
		}
	}
	return matches
}
