package input_text

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/plugin"
)

// ProlongedSoundMarkPlugin replaces consecutive prolonged sound marks with a single symbol
// This is a faithful port of Rust Sudachi's ProlongedSoundMarkPlugin
type ProlongedSoundMarkPlugin struct {
	psmSet        map[rune]bool  // Set of prolonged sound mark characters
	replaceSymbol string         // Symbol to replace consecutive marks with
	regex         *regexp.Regexp // Compiled regex for matching consecutive marks
	debug         bool           // Debug flag for conditional output
}

// PluginSettings represents configuration settings for ProlongedSoundMarkPlugin
// This corresponds with raw config json file (matches Rust implementation)
type PluginSettings struct {
	ProlongedSoundMarks []string `json:"prolongedSoundMarks"` // Matches Rust: prolongedSoundMarks: Vec<char>
	ReplacementSymbol   string   `json:"replacementSymbol"`   // Matches Rust: replacementSymbol: Option<String>
}

// NewProlongedSoundMarkPlugin creates a new ProlongedSoundMarkPlugin with default values
// This matches Rust: #[derive(Default)]
func NewProlongedSoundMarkPlugin() *ProlongedSoundMarkPlugin {
	return &ProlongedSoundMarkPlugin{
		psmSet:        map[rune]bool{},
		replaceSymbol: "ー", // Default replacement symbol
		regex:         nil, // Will be set during setup
		debug:         false,
	}
}

// SetDebug sets the debug flag for the plugin
func (p *ProlongedSoundMarkPlugin) SetDebug(debug bool) {
	p.debug = debug
}

// GetName returns the plugin name for identification
func (p *ProlongedSoundMarkPlugin) GetName() string {
	return "ProlongedSoundMarkPlugin"
}

// SetUp initializes the plugin with configuration (implements plugin.InputTextPlugin)
// This matches Rust Sudachi's set_up method exactly
func (p *ProlongedSoundMarkPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Process settings if provided (matching Rust serde_json::from_value)
	if settings != nil {
		// Extract prolongedSoundMarks setting
		if psmInterface, ok := settings["prolongedSoundMarks"].([]any); ok {
			p.psmSet = make(map[rune]bool)
			for _, v := range psmInterface {
				if s, ok := v.(string); ok {
					// Convert string to runes (each should be a single character)
					runes := []rune(s)
					if len(runes) == 1 {
						p.psmSet[runes[0]] = true
					}
				}
			}
		}

		// Extract replacementSymbol setting (matching Rust: replacementSymbol.unwrap_or("ー".to_string()))
		if replaceSymbol, ok := settings["replacementSymbol"].(string); ok {
			p.replaceSymbol = replaceSymbol
		}
	}

	// Set default prolonged sound marks if none provided
	if len(p.psmSet) == 0 {
		// Default prolonged sound marks (matching common Japanese usage)
		defaultMarks := []rune{'ー', '〜', '〰'}
		p.psmSet = make(map[rune]bool)
		for _, mark := range defaultMarks {
			p.psmSet[mark] = true
		}
	}

	// Build regex pattern (matching Rust prolongs_as_regex function)
	regex, err := p.buildProlongedSoundRegex()
	if err != nil {
		return fmt.Errorf("failed to build prolonged sound regex: %w", err)
	}
	p.regex = regex

	return nil
}

// buildProlongedSoundRegex converts prolongation marks to a Regex which will match at least two patterns
// This matches Rust's prolongs_as_regex method exactly
func (p *ProlongedSoundMarkPlugin) buildProlongedSoundRegex() (*regexp.Regexp, error) {
	if len(p.psmSet) == 0 {
		return nil, fmt.Errorf("no prolonged sound marks configured")
	}

	var pattern strings.Builder
	pattern.WriteString("[")

	// Build character class (matching Rust regex escaping)
	for symbol := range p.psmSet {
		switch symbol {
		case '-', '[', ']', '^', '\\':
			// Escape special regex characters (matching Rust write!(pattern, "\\u{{{:X}}}", symbol as u32))
			pattern.WriteString(fmt.Sprintf("\\x{%X}", symbol))
		default:
			pattern.WriteRune(symbol)
		}
	}

	pattern.WriteString("]{2,}") // Match 2 or more consecutive marks

	regex, err := regexp.Compile(pattern.String())
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern '%s': %w", pattern.String(), err)
	}

	return regex, nil
}

// Rewrite implements InputTextPlugin interface
// This matches Rust's rewrite_impl method exactly
func (p *ProlongedSoundMarkPlugin) Rewrite(buffer *input.InputBuffer) error {
	// Get current state of buffer (which includes changes from previous plugins)
	current := buffer.Modified()

	if buffer.IsReadOnly() {
		return fmt.Errorf("buffer is read-only")
	}

	if p.regex == nil {
		return fmt.Errorf("plugin not properly initialized: regex is nil")
	}

	modified := current

	// Find all matches and replace them (matching Rust: for m in re.find_iter(data))
	matches := p.regex.FindAllStringIndex(current, -1)

	// Process matches in reverse order to maintain correct indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		start, end := match[0], match[1]

		// Replace consecutive prolonged sound marks with single replacement symbol
		// This matches Rust: edit.replace_ref(m.range(), &self.replace_symbol)
		modified = modified[:start] + p.replaceSymbol + modified[end:]
	}

	// Apply modification if any changes were made
	if modified != current {
		return buffer.SetModified(modified)
	}

	return nil
}

// RewriteImpl performs prolonged sound mark normalization (for compatibility with existing patterns)
func (p *ProlongedSoundMarkPlugin) RewriteImpl(input string) (string, bool) {
	if p.regex == nil {
		return input, false
	}

	if input == "" {
		return input, false
	}

	// Apply regex replacement
	result := p.regex.ReplaceAllString(input, p.replaceSymbol)
	changed := result != input

	return result, changed
}

// CreateInputTextPlugin creates a ProlongedSoundMarkPlugin instance
func (p *ProlongedSoundMarkPlugin) CreateInputTextPlugin(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.InputTextPlugin, error) {
	prolongedPlugin := NewProlongedSoundMarkPlugin()

	// Set up the plugin with configuration
	err := prolongedPlugin.SetUp(settings, resourceDir, grammar)
	if err != nil {
		return nil, fmt.Errorf("failed to set up ProlongedSoundMark plugin: %w", err)
	}

	return prolongedPlugin, nil
}

// CreateOOVProvider creates an OOV provider plugin (not supported by ProlongedSoundMark plugin)
func (p *ProlongedSoundMarkPlugin) CreateOOVProvider(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.OOVProviderPlugin, error) {
	return nil, fmt.Errorf("ProlongedSoundMark plugin does not support OOV provider plugins")
}

// CreatePathRewriter creates a path rewrite plugin (not supported by ProlongedSoundMark plugin)
func (p *ProlongedSoundMarkPlugin) CreatePathRewriter(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.PathRewritePlugin, error) {
	return nil, fmt.Errorf("ProlongedSoundMark plugin does not support path rewrite plugins")
}

// GetSupportedTypes returns the plugin types this factory supports
func (p *ProlongedSoundMarkPlugin) GetSupportedTypes() []plugin.PluginType {
	return []plugin.PluginType{plugin.PluginTypeInputText}
}
