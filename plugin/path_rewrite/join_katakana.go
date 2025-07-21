package path_rewrite

import (
	"strings"
	"unicode"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
)

// JoinKatakanaOovPlugin concatenates consecutive katakana OOV nodes
type JoinKatakanaOovPlugin struct {
	minLength int
	oovPOS    []string
}

// NewJoinKatakanaOovPlugin creates a new JoinKatakanaOovPlugin
func NewJoinKatakanaOovPlugin() *JoinKatakanaOovPlugin {
	return &JoinKatakanaOovPlugin{
		minLength: 3,                                           // Default value
		oovPOS:    []string{"名詞", "普通名詞", "一般", "*", "*", "*"}, // Default POS
	}
}

// NewJoinKatakanaOovPluginWithParams creates a new JoinKatakanaOovPlugin with specified parameters (deprecated)
func NewJoinKatakanaOovPluginWithParams(minLength int, oovPOS []string) *JoinKatakanaOovPlugin {
	return &JoinKatakanaOovPlugin{
		minLength: minLength,
		oovPOS:    oovPOS,
	}
}

// GetName returns the plugin name
func (p *JoinKatakanaOovPlugin) GetName() string {
	return "JoinKatakanaOovPlugin"
}

// SetUp initializes the plugin with configuration (implements plugin.PathRewritePlugin)
func (p *JoinKatakanaOovPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Configure minimum length from settings if provided
	if settings != nil {
		if minLen, ok := settings["minLength"].(float64); ok {
			p.minLength = int(minLen)
		} else if minLen, ok := settings["minLength"].(int); ok {
			p.minLength = minLen
		}
		// Handle oovPOS as []any from JSON settings
		if posInterface, ok := settings["oovPOS"].([]any); ok {
			oovPOS := make([]string, len(posInterface))
			for i, v := range posInterface {
				if s, ok := v.(string); ok {
					oovPOS[i] = s
				}
			}
			p.oovPOS = oovPOS
		} else if pos, ok := settings["oovPOS"].([]string); ok {
			p.oovPOS = pos
		}
	}

	// Set default values if not configured
	if p.minLength <= 0 {
		p.minLength = 3 // Default minimum length for katakana OOV joining
	}

	return nil
}

// Rewrite implements PathRewritePlugin interface
func (p *JoinKatakanaOovPlugin) Rewrite(path []*lattice.NodeResult, buffer *input.InputBuffer, lat *lattice.Lattice) ([]*lattice.NodeResult, error) {
	if len(path) <= 1 {
		return path, nil
	}

	result := make([]*lattice.NodeResult, 0, len(path))
	i := 0

	for i < len(path) {
		current := path[i]

		// Check if current node is katakana OOV
		if !p.isKatakanaOOV(current, buffer) {
			result = append(result, current)
			i++
			continue
		}

		// Find consecutive katakana OOV nodes
		start := i
		for i < len(path) && p.isKatakanaOOV(path[i], buffer) {
			i++
		}
		end := i

		// Calculate total length
		totalLength := 0
		for j := start; j < end; j++ {
			surface := path[j].Surface()
			totalLength += len([]rune(surface))
		}

		// Only join if we have multiple nodes or meet minimum length requirement
		if end-start > 1 && totalLength >= p.minLength {
			// Concatenate katakana nodes
			concatenated, err := p.concatenateNodes(path[start:end])
			if err != nil {
				// If concatenation fails, add nodes individually
				for j := start; j < end; j++ {
					result = append(result, path[j])
				}
			} else {
				result = append(result, concatenated)
			}
		} else {
			// Add nodes individually if they don't meet criteria
			for j := start; j < end; j++ {
				result = append(result, path[j])
			}
		}
	}

	return result, nil
}

// isKatakanaOOV checks if a node is a katakana OOV node
func (p *JoinKatakanaOovPlugin) isKatakanaOOV(node *lattice.NodeResult, buffer *input.InputBuffer) bool {
	// Check if node is OOV
	if !node.Node().IsOOV() {
		return false
	}

	// Check if surface is katakana
	surface := node.Surface()
	if surface == "" {
		return false
	}

	// All characters must be katakana
	for _, r := range surface {
		if !unicode.In(r, unicode.Katakana) {
			return false
		}
	}

	return true
}

// concatenateNodes concatenates multiple katakana nodes into one
func (p *JoinKatakanaOovPlugin) concatenateNodes(nodes []*lattice.NodeResult) (*lattice.NodeResult, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	if len(nodes) == 1 {
		return nodes[0], nil
	}

	// Build concatenated surface, reading, and dictionary forms
	var surfaceBuilder strings.Builder
	var readingBuilder strings.Builder
	var dictionaryBuilder strings.Builder

	firstNode := nodes[0]
	lastNode := nodes[len(nodes)-1]

	for _, node := range nodes {
		surfaceBuilder.WriteString(node.Surface())
		readingBuilder.WriteString(node.Reading())
		dictionaryBuilder.WriteString(node.DictionaryForm())
	}

	surface := surfaceBuilder.String()
	reading := readingBuilder.String()
	dictionary := dictionaryBuilder.String()

	// Create new node with the span of all concatenated nodes
	newNode := lattice.NewNode(
		firstNode.Node().Begin(),
		lastNode.Node().End(),
		65535,       // u16::MAX for left_id
		65535,       // u16::MAX for right_id
		32767,       // i16::MAX for cost
		dic.Invalid, // WordId::INVALID for OOV
	)

	// Use provided OOV POS or default katakana POS
	pos := p.oovPOS
	if len(pos) == 0 {
		pos = []string{"名詞", "普通名詞", "一般", "*", "*", "*"}
	}

	// Create concatenated result
	concatenated := lattice.NewNodeResult(
		newNode,
		surface,
		pos,
		[]string{}, // No additional features
		surface,    // Normalized form is same as surface for katakana
		dictionary,
		reading,
	)

	return concatenated, nil
}
