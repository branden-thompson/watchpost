package components

import (
	"regexp"
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// CLISyntaxHighlighter provides intelligent CLI command syntax highlighting
// Focuses on terminal/command-line content that chroma doesn't handle well.
// For programming language syntax highlighting, use ChromaHighlighter instead.
type CLISyntaxHighlighter struct {
	formatter    *rendering.TextFormatter
	patterns     map[string]*regexp.Regexp
	tokens       map[string]string
	enabledTypes []CLISyntaxType
}

// CLISyntaxType represents different types of CLI-focused syntax highlighting
type CLISyntaxType int

const (
	CLICommand  CLISyntaxType = iota // Command names and CLI tools (git, npm, go)
	CLIFlag                          // Command line flags (--flag, -f)
	CLIPath                          // File and directory paths
	CLIVariable                      // Variables and placeholders ($HOME, ${VAR})
	CLIString                        // Quoted strings in commands
	CLIComment                       // Shell comments (#)
	CLIURL                           // URLs and web addresses
	CLINumber                        // Numeric values in CLI context
)

// NewCLISyntaxHighlighter creates a new CLI-focused syntax highlighter
func NewCLISyntaxHighlighter() *CLISyntaxHighlighter {
	sh := &CLISyntaxHighlighter{
		formatter: rendering.NewTextFormatter(),
		patterns:  make(map[string]*regexp.Regexp),
		tokens:    make(map[string]string),
		enabledTypes: []CLISyntaxType{
			CLICommand, CLIFlag, CLIPath, CLIVariable,
			CLIString, CLIComment, CLIURL, CLINumber,
		},
	}

	sh.initializePatterns()
	sh.initializeTokens()
	return sh
}

// WithEnabledTypes sets which CLI syntax types to highlight
func (sh *CLISyntaxHighlighter) WithEnabledTypes(types ...CLISyntaxType) *CLISyntaxHighlighter {
	sh.enabledTypes = types
	return sh
}

// Legacy compatibility - maintains the old API for existing code
func NewSyntaxHighlighter() *CLISyntaxHighlighter {
	return NewCLISyntaxHighlighter()
}

// initializePatterns sets up regex patterns for CLI-focused syntax elements
func (sh *CLISyntaxHighlighter) initializePatterns() {
	// CLI Command patterns (expanded list of common CLI tools)
	sh.patterns["command"] = regexp.MustCompile(`\b(npm|git|docker|kubectl|yarn|pnpm|go|cargo|pip|curl|wget|ssh|scp|rsync|grep|find|sed|awk|make|cmake|node|python|java|mvn|gradle|ls|cd|mkdir|rm|cp|mv|cat|head|tail|less|more|ps|kill|sudo|chmod|chown|tar|gzip|unzip|which|where|echo|printf|export|source|alias|history|top|htop|df|du|mount|umount|systemctl|service|brew|apt|yum|dnf)\b`)

	// CLI Flag patterns (--long-flag, -f, including common variations)
	sh.patterns["flag"] = regexp.MustCompile(`(^|\s)(--[\w-]+[=\w]*|-[a-zA-Z]+)(\s|$)`)

	// Path patterns (file and directory paths, expanded for CLI context)
	sh.patterns["path"] = regexp.MustCompile(`(\.{0,2}/[\w./~-]+|~/[\w./~-]+|/[\w./~-]+|\w+/[\w./~-]+|\w+\.\w{2,4}\b)`)

	// Variable patterns (${VAR}, $VAR, environment variables)
	sh.patterns["variable"] = regexp.MustCompile(`(\$\{[^}]+\}|\$[A-Z_][A-Z0-9_]*|\$\w+)`)

	// String patterns (quoted strings in CLI commands)
	sh.patterns["string"] = regexp.MustCompile(`("([^"\\]|\\.)*"|'([^'\\]|\\.)*')`)

	// URL patterns
	sh.patterns["url"] = regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)

	// Number patterns (ports, versions, counts in CLI context)
	sh.patterns["number"] = regexp.MustCompile(`\b\d+(\.\d+)*\b`)

	// Comment patterns (shell comments only)
	sh.patterns["comment"] = regexp.MustCompile(`#.*$`)
}

// initializeTokens maps pattern names to CLI-focused semantic tokens
func (sh *CLISyntaxHighlighter) initializeTokens() {
	sh.tokens["command"] = "cli.Command"   // CLI commands and tools
	sh.tokens["flag"] = "cli.Flag"         // Command line flags
	sh.tokens["path"] = "cli.Path"         // File and directory paths
	sh.tokens["variable"] = "cli.Variable" // Environment variables
	sh.tokens["string"] = "cli.String"     // Quoted strings in commands
	sh.tokens["comment"] = "cli.Comment"   // Shell comments
	sh.tokens["url"] = "cli.URL"           // URLs
	sh.tokens["number"] = "cli.Number"     // Numeric values
}

// ProcessBackticks processes text with backtick-delimited CLI command sections
func (sh *CLISyntaxHighlighter) ProcessBackticks(text string) string {
	// Pattern to match both single backticks (`code`) and triple backticks (```code```)
	backtickPattern := regexp.MustCompile("```([^`]+)```|`([^`]+)`")

	return backtickPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Determine if it's triple or single backticks
		if strings.HasPrefix(match, "```") && strings.HasSuffix(match, "```") {
			// Triple backticks - multi-line code block
			content := strings.TrimPrefix(strings.TrimSuffix(match, "```"), "```")
			return sh.highlightCodeBlock(content)
		} else if strings.HasPrefix(match, "`") && strings.HasSuffix(match, "`") {
			// Single backticks - inline code
			content := strings.TrimPrefix(strings.TrimSuffix(match, "`"), "`")
			return sh.highlightInlineCode(content)
		}
		return match
	})
}

// highlightCodeBlock processes multi-line CLI command blocks
func (sh *CLISyntaxHighlighter) highlightCodeBlock(content string) string {
	lines := strings.Split(content, "\n")
	highlightedLines := make([]string, len(lines))

	for i, line := range lines {
		highlightedLines[i] = sh.highlightLine(line)
	}

	// Wrap in code block styling
	styledContent := strings.Join(highlightedLines, "\n")
	return sh.formatter.Style("```\n"+styledContent+"\n```", "syntax.CodeBlock")
}

// highlightInlineCode processes single-line inline CLI commands
func (sh *CLISyntaxHighlighter) highlightInlineCode(content string) string {
	highlighted := sh.highlightLine(content)
	return sh.formatter.Style("`"+highlighted+"`", "syntax.InlineCode")
}

// highlightLine applies CLI syntax highlighting to a single line
func (sh *CLISyntaxHighlighter) highlightLine(line string) string {
	result := line

	// Apply CLI highlighting in order of precedence
	for _, syntaxType := range sh.enabledTypes {
		result = sh.applyPatternHighlighting(result, syntaxType)
	}

	return result
}

// applyPatternHighlighting applies highlighting for a specific CLI syntax type
func (sh *CLISyntaxHighlighter) applyPatternHighlighting(text string, syntaxType CLISyntaxType) string {
	var patternName string
	switch syntaxType {
	case CLICommand:
		patternName = "command"
	case CLIFlag:
		patternName = "flag"
	case CLIPath:
		patternName = "path"
	case CLIVariable:
		patternName = "variable"
	case CLIString:
		patternName = "string"
	case CLIComment:
		patternName = "comment"
	case CLIURL:
		patternName = "url"
	case CLINumber:
		patternName = "number"
	default:
		return text
	}

	pattern, exists := sh.patterns[patternName]
	if !exists {
		return text
	}

	token, exists := sh.tokens[patternName]
	if !exists {
		return text
	}

	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		return sh.formatter.Style(match, token)
	})
}

// HighlightCLIText applies CLI syntax highlighting to arbitrary text
func (sh *CLISyntaxHighlighter) HighlightCLIText(text string) string {
	return sh.ProcessBackticks(text)
}

// HighlightCommand specifically highlights command-line instructions
func (sh *CLISyntaxHighlighter) HighlightCommand(command string) string {
	// Enable only command-relevant syntax types
	originalTypes := sh.enabledTypes
	sh.enabledTypes = []CLISyntaxType{CLICommand, CLIFlag, CLIPath, CLIVariable, CLIString}

	result := sh.highlightLine(command)

	// Restore original types
	sh.enabledTypes = originalTypes

	return sh.formatter.Style(result, "cli.CommandLine")
}

// HighlightPath specifically highlights file system paths
func (sh *CLISyntaxHighlighter) HighlightPath(path string) string {
	return sh.formatter.Style(path, "cli.Path")
}

// HighlightURL specifically highlights web URLs
func (sh *CLISyntaxHighlighter) HighlightURL(url string) string {
	return sh.formatter.Style(url, "cli.URL")
}

// ProcessShellCode processes shell/bash code sections specifically
// For general programming language highlighting, use ChromaHighlighter instead
func (sh *CLISyntaxHighlighter) ProcessShellCode(text string) string {
	// Handle fenced code blocks with shell/bash language specifiers
	fencedPattern := regexp.MustCompile("```(bash|sh|shell|zsh|fish)?\\s*\\n([\\s\\S]*?)```")
	text = fencedPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := fencedPattern.FindStringSubmatch(match)
		if len(parts) >= 3 {
			content := parts[2]
			return sh.highlightShellBlock(content)
		}
		return match
	})

	// Handle inline code (assuming CLI commands)
	inlinePattern := regexp.MustCompile("`([^`]+)`")
	text = inlinePattern.ReplaceAllStringFunc(text, func(match string) string {
		content := strings.Trim(match, "`")
		return sh.highlightInlineCode(content)
	})

	return text
}

// highlightShellBlock applies CLI-focused highlighting for shell/bash blocks
func (sh *CLISyntaxHighlighter) highlightShellBlock(content string) string {
	// Focus on shell/CLI patterns only
	originalTypes := sh.enabledTypes
	sh.enabledTypes = []CLISyntaxType{CLICommand, CLIFlag, CLIPath, CLIVariable, CLIString, CLIComment}

	highlighted := sh.highlightCodeBlock(content)

	// Restore original types
	sh.enabledTypes = originalTypes

	return highlighted
}

// AddCustomPattern adds a custom CLI syntax pattern
func (sh *CLISyntaxHighlighter) AddCustomPattern(name string, pattern *regexp.Regexp, token string) {
	sh.patterns[name] = pattern
	sh.tokens[name] = token
}

// RemovePattern removes a CLI syntax pattern
func (sh *CLISyntaxHighlighter) RemovePattern(name string) {
	delete(sh.patterns, name)
	delete(sh.tokens, name)
}

// GetSupportedPatterns returns a list of all supported CLI pattern names
func (sh *CLISyntaxHighlighter) GetSupportedPatterns() []string {
	patterns := make([]string, 0, len(sh.patterns))
	for name := range sh.patterns {
		patterns = append(patterns, name)
	}
	return patterns
}

// CLICommandBlock represents a formatted CLI command block with description
type CLICommandBlock struct {
	command     string
	description string
	highlighter *CLISyntaxHighlighter
}

// NewCLICommandBlock creates a new CLI command block
func NewCLICommandBlock(command, description string) *CLICommandBlock {
	return &CLICommandBlock{
		command:     command,
		description: description,
		highlighter: NewCLISyntaxHighlighter(),
	}
}

// Render generates the formatted CLI command block
func (cb *CLICommandBlock) Render() string {
	var parts []string

	if cb.description != "" {
		desc := cb.highlighter.formatter.Style(cb.description, "cli.Description")
		parts = append(parts, desc)
	}

	highlightedCommand := cb.highlighter.HighlightCommand(cb.command)
	parts = append(parts, highlightedCommand)

	return strings.Join(parts, "\n")
}

// NOTE: For general code snippet highlighting, use ChromaHighlighter instead.
// CLISyntaxHighlighter is focused on CLI/terminal commands only.

// Convenience constructors for common syntax highlighting scenarios

// NewBashHighlighter creates a highlighter optimized for bash/shell commands
func NewBashHighlighter() *CLISyntaxHighlighter {
	highlighter := NewCLISyntaxHighlighter()
	highlighter.WithEnabledTypes(CLICommand, CLIFlag, CLIPath, CLIVariable, CLIString, CLIComment)
	return highlighter
}

// NOTE: For programming language highlighting, use ChromaHighlighter instead.
// This function is deprecated - use NewCLISyntaxHighlighter() for CLI commands.

// NewCLIMarkdownHighlighter creates a highlighter for CLI content within markdown
func NewCLIMarkdownHighlighter() *CLISyntaxHighlighter {
	highlighter := NewCLISyntaxHighlighter()
	// Enable CLI-focused types for markdown CLI content
	highlighter.WithEnabledTypes(
		CLICommand, CLIFlag, CLIPath, CLIVariable,
		CLIString, CLIComment, CLIURL, CLINumber,
	)
	return highlighter
}

// CLI Highlighting Helpers - convenience functions for common CLI highlighting tasks

// HighlightCLIBackticks is a convenience function for quick CLI backtick processing
func HighlightCLIBackticks(text string) string {
	highlighter := NewCLISyntaxHighlighter()
	return highlighter.ProcessBackticks(text)
}

// HighlightCLIMarkdown is a convenience function for CLI content in markdown
func HighlightCLIMarkdown(text string) string {
	highlighter := NewCLIMarkdownHighlighter()
	return highlighter.ProcessShellCode(text)
}

// HighlightBashCommand is a convenience function for bash command highlighting
func HighlightBashCommand(command string) string {
	highlighter := NewBashHighlighter()
	return highlighter.HighlightCommand(command)
}

// Legacy compatibility functions (maintain old API for existing code)

// HighlightBackticks - legacy function, use HighlightCLIBackticks for new code
func HighlightBackticks(text string) string {
	return HighlightCLIBackticks(text)
}

// Legacy CommandBlock type alias for backward compatibility
type CommandBlock = CLICommandBlock

// NewCommandBlock - legacy function, use NewCLICommandBlock for new code
func NewCommandBlock(command, description string) *CLICommandBlock {
	return NewCLICommandBlock(command, description)
}
