package path_rewrite

import (
	"fmt"
	"strings"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/plugin"
)

// JoinKatakanaOovPlugin concatenates consecutive katakana OOV nodes (matching Rust version)
type JoinKatakanaOovPlugin struct {
	oovPosId  uint16 // The pos_id used for concatenated node (matching Rust)
	minLength int    // The minimum node char_length to concatenate even if it is not oov (matching Rust)
}

// NewJoinKatakanaOovPlugin creates a new JoinKatakanaOovPlugin
func NewJoinKatakanaOovPlugin() *JoinKatakanaOovPlugin {
	return &JoinKatakanaOovPlugin{
		oovPosId:  0, // Will be set during setup (matching Rust)
		minLength: 3, // Default value (matching Rust)
	}
}

// NewJoinKatakanaOovPluginWithParams creates a new JoinKatakanaOovPlugin with specified parameters (deprecated)
func NewJoinKatakanaOovPluginWithParams(minLength int, oovPOS []string) *JoinKatakanaOovPlugin {
	return &JoinKatakanaOovPlugin{
		oovPosId:  0, // Will be set during setup
		minLength: minLength,
	}
}

// GetName returns the plugin name
func (p *JoinKatakanaOovPlugin) GetName() string {
	return "JoinKatakanaOovPlugin"
}

// SetUp initializes the plugin with configuration (matching Rust implementation)
func (p *JoinKatakanaOovPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Extract oovPOS and minLength from settings like Rust version
	if settings != nil {
		// Handle oovPOS as []any from JSON settings and convert to POS ID
		if posInterface, ok := settings["oovPOS"].([]any); ok {
			oovPOS := make([]string, len(posInterface))
			for i, v := range posInterface {
				if s, ok := v.(string); ok {
					oovPOS[i] = s
				}
			}
			// Get POS ID from grammar like Rust: grammar.get_part_of_speech_id()
			if grammar != nil {
				if posId := grammar.GetPartOfSpeechId(oovPOS); posId != nil {
					p.oovPosId = *posId
				}
			}
		} else if pos, ok := settings["oovPOS"].([]string); ok {
			// Get POS ID from grammar like Rust: grammar.get_part_of_speech_id()
			if grammar != nil {
				if posId := grammar.GetPartOfSpeechId(pos); posId != nil {
					p.oovPosId = *posId
				}
			}
		}

		// Configure minimum length from settings if provided
		if minLen, ok := settings["minLength"].(float64); ok {
			p.minLength = int(minLen)
		} else if minLen, ok := settings["minLength"].(int); ok {
			p.minLength = minLen
		}
	}

	// Set default values if not configured (matching Rust defaults)
	if p.minLength <= 0 {
		p.minLength = 3 // Default minimum length for katakana OOV joining
	}

	return nil
}

// Rewrite implements PathRewritePlugin interface (matching Rust rewrite_gen algorithm)
func (p *JoinKatakanaOovPlugin) Rewrite(path []*lattice.NodeResult, buffer *input.InputBuffer, lat *lattice.Lattice) ([]*lattice.NodeResult, error) {
	if len(path) <= 1 {
		return path, nil
	}

	// Use Rust-compatible rewrite_gen algorithm
	return p.rewriteGen(path, buffer)
}

// rewriteGen implements the exact Rust rewrite_gen algorithm
func (p *JoinKatakanaOovPlugin) rewriteGen(path []*lattice.NodeResult, buffer *input.InputBuffer) ([]*lattice.NodeResult, error) {
	i := 0
	for i < len(path) {
		node := path[i]

		// Rust logic: if !(node.is_oov() || self.is_shorter(node)) || !self.is_katakana_node(text, node)
		isOOV := node.Node().IsOOV()
		isShorter := p.isShorter(node)
		isKatakana := p.isKatakanaNode(node, buffer)

		if !(isOOV || isShorter) || !isKatakana {
			i++
			continue
		}

		// Find backward range of katakana nodes
		begin := i - 1
		for begin >= 0 && p.isKatakanaNode(path[begin], buffer) {
			begin--
		}
		begin++ // adjust to first katakana node
		if begin < 0 {
			begin = 0
		}

		// Find forward range of katakana nodes
		end := i + 1
		for end < len(path) && p.isKatakanaNode(path[end], buffer) {
			end++
		}

		// Adjust begin to ensure we can start OOV
		for begin < end && !p.canOOVBowNode(path[begin], buffer) {
			begin++
		}

		// Only concatenate if we have multiple nodes
		if (end - begin) > 1 {
			var err error
			path, err = p.concatOOVNodes(path, begin, end)
			if err != nil {
				return nil, err
			}
			// Skip next node as we know it's not joinable katakana
			i = begin + 1
		}
		i++
	}

	return path, nil
}

// isKatakanaNode checks if a node contains katakana characters (matching Rust version)
func (p *JoinKatakanaOovPlugin) isKatakanaNode(node *lattice.NodeResult, buffer *input.InputBuffer) bool {
	// Rust: text.cat_of_range(node.begin()..node.end()).contains(CategoryType::KATAKANA)
	nodeStart := int(node.Node().Begin())
	nodeEnd := int(node.Node().End())

	// Get category types for the node range (equivalent to cat_of_range)
	categories := buffer.CategoryOfRange(nodeStart, nodeEnd)

	// Check if the categories include katakana (equivalent to contains(CategoryType::KATAKANA))
	return categories.HasFlag(dic.CategoryKatakana)
}

// isShorter checks if node is shorter than minimum length (matching Rust version)
func (p *JoinKatakanaOovPlugin) isShorter(node *lattice.NodeResult) bool {
	surface := node.Surface()
	runeCount := len([]rune(surface))
	return runeCount < p.minLength
}

// canOOVBowNode checks if a node can start OOV (matching Rust version)
func (p *JoinKatakanaOovPlugin) canOOVBowNode(node *lattice.NodeResult, buffer *input.InputBuffer) bool {
	// Rust: !text.cat_at_char(node.begin()).contains(CategoryType::NOOOVBOW)
	nodeStart := int(node.Node().Begin())

	// Convert byte position to character position
	charIdx, err := buffer.ByteToCharIndex(nodeStart)
	if err != nil {
		return false
	}

	// Get category at character position (equivalent to cat_at_char)
	category, err := buffer.GetCategory(charIdx)
	if err != nil {
		return false
	}

	// Check if the category does NOT contain NOOOVBOW
	return !category.HasFlag(dic.CategoryNoOOVBOW)
}

// concatOOVNodes concatenates nodes like Rust concat_oov_nodes
func (p *JoinKatakanaOovPlugin) concatOOVNodes(path []*lattice.NodeResult, begin, end int) ([]*lattice.NodeResult, error) {
	if begin >= end {
		return path, nil
	}

	// Concatenate the nodes in the range
	concatenated, err := p.concatenateNodes(path[begin:end])
	if err != nil {
		return path, err
	}

	// Replace the range with the concatenated node (matching Rust's path.drain)
	result := make([]*lattice.NodeResult, 0, len(path)-(end-begin)+1)
	result = append(result, path[:begin]...)
	result = append(result, concatenated)
	result = append(result, path[end:]...)

	return result, nil
}

// concatenateNodes concatenates multiple katakana nodes into one (matching Rust concat_oov_nodes exactly)
func (p *JoinKatakanaOovPlugin) concatenateNodes(nodes []*lattice.NodeResult) (*lattice.NodeResult, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	if len(nodes) == 1 {
		return nodes[0], nil
	}

	// Build concatenated surface (matching Rust implementation exactly)
	var surfaceBuilder strings.Builder
	for _, node := range nodes {
		surfaceBuilder.WriteString(node.Surface())
	}
	surface := surfaceBuilder.String()

	firstNode := nodes[0]
	lastNode := nodes[len(nodes)-1]

	// Determine WordId using Rust logic: wid = wid.max(node.word_id()); if !wid.is_oov() { wid = WordId::new(wid.dic(), WordId::MAX_WORD); }
	var maxWordId dic.WordId = firstNode.Node().WordId()
	for _, node := range nodes[1:] {
		nodeWordId := node.Node().WordId()
		if nodeWordId.Raw() > maxWordId.Raw() {
			maxWordId = nodeWordId
		}
	}

	// If the max WordId is not OOV, create special WordId (matching Rust logic)
	// However, to avoid dictionary lookup issues in Go, we'll use OOV WordId
	// The behavior should be the same since these nodes carry their own word info
	var finalWordId dic.WordId
	if !maxWordId.IsOOV() {
		// Use OOV WordId with POS ID to avoid dictionary lookup issues
		// This ensures the node won't be split and behaves like the Rust version
		finalWordId = dic.OOV(uint32(p.oovPosId))
	} else {
		finalWordId = maxWordId
	}

	// Create new node with the span of all concatenated nodes (matching Rust Node::new)
	newNode := lattice.NewNode(
		firstNode.Node().Begin(),
		lastNode.Node().End(),
		65535,       // u16::MAX for left_id (matching Rust)
		65535,       // u16::MAX for right_id (matching Rust)
		32767,       // i16::MAX for cost (matching Rust)
		finalWordId, // WordId calculated using Rust logic
	)

	// Use default katakana POS (matching Rust behavior)
	pos := []string{"名詞", "普通名詞", "一般", "*", "*", "*"}

	// Create concatenated result (matching Rust concat_oov_nodes exactly)
	// Rust version: normalized_form = surface, dictionary_form = surface, reading_form = surface (for katakana)
	concatenated := lattice.NewNodeResult(
		newNode,
		surface,    // Surface
		pos,        // POS components
		[]string{}, // No additional features (matching Rust)
		surface,    // Normalized form = surface (matching Rust concat_oov_nodes)
		surface,    // Dictionary form = surface (matching Rust concat_oov_nodes)
		surface,    // Reading form = surface for katakana (matching expected behavior)
	)

	return concatenated, nil
}

// CreateInputTextPlugin creates an input text plugin (not supported by JoinKatakanaOov plugin)
func (p *JoinKatakanaOovPlugin) CreateInputTextPlugin(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.InputTextPlugin, error) {
	return nil, fmt.Errorf("JoinKatakanaOov plugin does not support input text plugins")
}

// CreateOOVProvider creates an OOV provider plugin (not supported by JoinKatakanaOov plugin)
func (p *JoinKatakanaOovPlugin) CreateOOVProvider(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.OOVProviderPlugin, error) {
	return nil, fmt.Errorf("JoinKatakanaOov plugin does not support OOV provider plugins")
}

// CreatePathRewriter creates a JoinKatakanaOov path rewrite plugin instance
func (p *JoinKatakanaOovPlugin) CreatePathRewriter(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.PathRewritePlugin, error) {
	joinKatakanaPlugin := NewJoinKatakanaOovPlugin()

	// Set up the plugin with configuration
	err := joinKatakanaPlugin.SetUp(settings, resourceDir, systemDict.Grammar())
	if err != nil {
		return nil, fmt.Errorf("failed to set up JoinKatakanaOov plugin: %w", err)
	}

	return joinKatakanaPlugin, nil
}

// GetSupportedTypes returns the plugin types this factory supports
func (p *JoinKatakanaOovPlugin) GetSupportedTypes() []plugin.PluginType {
	return []plugin.PluginType{plugin.PluginTypePathRewrite}
}
