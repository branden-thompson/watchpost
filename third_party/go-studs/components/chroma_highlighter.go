package components

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// ChromaHighlighter provides professional syntax highlighting using the chroma library
// with integration to GO-STUDS design tokens
type ChromaHighlighter struct {
	formatter *rendering.TextFormatter
}

// NewChromaHighlighter creates a new chroma-based syntax highlighter
func NewChromaHighlighter() *ChromaHighlighter {
	return &ChromaHighlighter{
		formatter: rendering.NewTextFormatter(),
	}
}

// HighlightCode highlights code in the specified language using chroma
// Supported languages: "go", "bash", "shell", "javascript", "python", etc.
func (ch *ChromaHighlighter) HighlightCode(code, language string) (string, error) {
	// Get lexer for the specified language
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Create a custom style that maps to our GO-STUDS tokens
	style := ch.createGOStudsStyle()

	// Create formatter for terminal output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return code, nil // Fallback to original code
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

	// Format with our custom style
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code, err
	}

	return buf.String(), nil
}

// HighlightGoCode highlights Go code using chroma with GO-STUDS styling
func (ch *ChromaHighlighter) HighlightGoCode(code string) (string, error) {
	return ch.HighlightCode(code, "go")
}

// HighlightBashCommand highlights bash/shell commands using chroma
func (ch *ChromaHighlighter) HighlightBashCommand(command string) (string, error) {
	return ch.HighlightCode(command, "bash")
}

// HighlightFunctionSignature highlights a Go function signature (compatibility with existing API)
func (ch *ChromaHighlighter) HighlightFunctionSignature(signature string) (string, error) {
	return ch.HighlightGoCode(signature)
}

// createGOStudsStyle creates a chroma style that maps to our GO-STUDS design tokens
func (ch *ChromaHighlighter) createGOStudsStyle() *chroma.Style {
	// Use a built-in style and customize it, or use default style
	// For now, let's use the default terminal style and let chroma handle it
	return styles.Get("native")
}

// Legacy compatibility - replace the old GoSyntaxHighlighter
func (ch *ChromaHighlighter) HighlightFunctionSignatureCompat(signature string) string {
	result, err := ch.HighlightFunctionSignature(signature)
	if err != nil {
		// Fallback to original text if highlighting fails
		return signature
	}
	return strings.TrimSpace(result)
}

// HighlightCommand highlights terminal/bash commands with syntax highlighting
// Usage: HighlightCommand("git commit -m 'Initial commit'")
func (ch *ChromaHighlighter) HighlightCommand(command string) string {
	result, err := ch.HighlightBashCommand(command)
	if err != nil {
		// Fallback to original text if highlighting fails
		return command
	}
	return strings.TrimSpace(result)
}

// HighlightMultiLanguage highlights code with automatic language detection fallback
func (ch *ChromaHighlighter) HighlightMultiLanguage(code, language string) string {
	result, err := ch.HighlightCode(code, language)
	if err != nil {
		// Try common fallbacks
		fallbacks := []string{"text", ""}
		for _, fallback := range fallbacks {
			if result, err := ch.HighlightCode(code, fallback); err == nil {
				return strings.TrimSpace(result)
			}
		}
		// Final fallback to original text
		return code
	}
	return strings.TrimSpace(result)
}
