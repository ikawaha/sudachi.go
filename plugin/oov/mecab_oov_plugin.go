package oov

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/plugin"
)

// MeCabOovPlugin provides OOV processing using MeCab-style character categories and unknown word definitions
type MeCabOovPlugin struct {
	charCategory *dic.CharacterCategory
	unkDefs      *dic.UnknownWordDefinitions
	grammar      *dic.Grammar
}

// NewMeCabOovPlugin creates a new MeCabOovPlugin
func NewMeCabOovPlugin(charCategory *dic.CharacterCategory, unkDefs *dic.UnknownWordDefinitions, grammar *dic.Grammar) *MeCabOovPlugin {
	return &MeCabOovPlugin{
		charCategory: charCategory,
		unkDefs:      unkDefs,
		grammar:      grammar,
	}
}

// NewMeCabOovPluginFromResourceDir creates a MeCabOovPlugin from resource directory
// This matches Rust's set_up method behavior
func NewMeCabOovPluginFromResourceDir(resourceDir string, grammar *dic.Grammar) (*MeCabOovPlugin, error) {
	// Load character category system
	charCategory, err := dic.LoadCharacterCategoryFromResourceDir(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load character category: %w", err)
	}

	// Load unknown word definitions
	unkDefs, err := dic.LoadUnknownWordDefinitionsFromResourceDir(resourceDir, charCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to load unknown word definitions: %w", err)
	}

	return NewMeCabOovPlugin(charCategory, unkDefs, grammar), nil
}

// OOVCandidate represents a candidate for OOV processing
type OOVCandidate struct {
	Begin      int // Character start position
	End        int // Character end position
	Category   dic.CategoryType
	Definition *dic.UnknownWordDefinition
}

// ProvideOOVCandidates generates OOV candidates for the given input buffer
func (p *MeCabOovPlugin) ProvideOOVCandidates(buffer *input.InputBuffer, hasKnownWord []bool) ([]*OOVCandidate, error) {
	if !buffer.IsReadOnly() {
		return nil, fmt.Errorf("buffer must be in read-only state: call Build() on buffer before providing OOV")
	}

	// Set character category system in buffer for advanced categorization
	buffer.SetCharacterCategory(p.charCategory)

	candidates := make([]*OOVCandidate, 0)
	charCount := buffer.CharCount()

	for i := 0; i < charCount; {
		// Skip if there's already a known word at this position
		if hasKnownWord[i] {
			i++
			continue
		}

		char, err := buffer.GetChar(i)
		if err != nil {
			return nil, err
		}

		category := p.charCategory.GetCategory(char)
		categoryInfo := p.charCategory.GetCategoryInfo(category)

		// Skip if no category info or no unknown word definitions
		if categoryInfo == nil || !p.unkDefs.HasDefinitions(category) {
			i++
			continue
		}

		// Calculate the maximum length for this category
		maxLength := int(categoryInfo.Length)
		if maxLength == 0 {
			maxLength = charCount - i // No limit
		}

		// Generate candidates based on category behavior
		if categoryInfo.IsGroup {
			// Group consecutive same-category characters
			length := p.calculateGroupLength(buffer, i, category, maxLength)
			if length > 0 {
				candidates = append(candidates, p.createCandidates(i, i+length, category)...)
			}
			i += length
		} else {
			// Single character candidate
			if maxLength > 0 {
				candidates = append(candidates, p.createCandidates(i, i+1, category)...)
			}
			i++
		}
	}

	return candidates, nil
}

// calculateGroupLength calculates the length of consecutive same-category characters
func (p *MeCabOovPlugin) calculateGroupLength(buffer *input.InputBuffer, start int, baseCategory dic.CategoryType, maxLength int) int {
	charCount := buffer.CharCount()
	length := 0

	for i := start; i < charCount && length < maxLength; i++ {
		char, err := buffer.GetChar(i)
		if err != nil {
			break
		}

		category := p.charCategory.GetCategory(char)
		if category != baseCategory {
			break
		}

		length++
	}

	return length
}

// createCandidates creates OOV candidates for all unknown word definitions of the given category
func (p *MeCabOovPlugin) createCandidates(begin, end int, category dic.CategoryType) []*OOVCandidate {
	definitions := p.unkDefs.GetDefinitions(category)
	candidates := make([]*OOVCandidate, len(definitions))

	for i, definition := range definitions {
		candidates[i] = &OOVCandidate{
			Begin:      begin,
			End:        end,
			Category:   category,
			Definition: definition,
		}
	}

	return candidates
}

// GetCharacterCategory returns the character category system
func (p *MeCabOovPlugin) GetCharacterCategory() *dic.CharacterCategory {
	return p.charCategory
}

// GetUnknownWordDefinitions returns the unknown word definitions
func (p *MeCabOovPlugin) GetUnknownWordDefinitions() *dic.UnknownWordDefinitions {
	return p.unkDefs
}

// GetName returns the plugin name for identification
func (p *MeCabOovPlugin) GetName() string {
	return "MeCabOovPlugin"
}

// SetUp initializes the plugin with configuration (implements plugin.OOVProviderPlugin)
// This matches Rust Sudachi's set_up method
func (p *MeCabOovPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Load character category system
	charCategory, err := dic.LoadCharacterCategoryFromResourceDir(resourceDir)
	if err != nil {
		return fmt.Errorf("failed to load character category: %w", err)
	}

	// Load unknown word definitions
	unkDefs, err := dic.LoadUnknownWordDefinitionsFromResourceDir(resourceDir, charCategory)
	if err != nil {
		return fmt.Errorf("failed to load unknown word definitions: %w", err)
	}

	// Update p state
	p.charCategory = charCategory
	p.unkDefs = unkDefs
	p.grammar = grammar

	return nil
}

// ProvideOOV generates OOV nodes at the given character position
// This implements the plugin.OOVProviderPlugin interface using concrete lattice types
func (p *MeCabOovPlugin) ProvideOOV(charPos int, buffer *input.InputBuffer, lat *lattice.Lattice, createdWords plugin.CreatedWords) (plugin.CreatedWords, error) {
	// This is a simplified implementation that adds MeCab-style OOV nodes
	// to the lattice at the specified character position

	// Get the character at this position
	if charPos >= buffer.CharCount() {
		return createdWords, nil // Out of bounds
	}

	// Debug output removed - will use lattice dump instead

	char, err := buffer.GetChar(charPos)
	if err != nil {
		return createdWords, err
	}

	// Get character categories (matching Rust: for ctype in input.cat_at_char(offset).iter())
	categories := p.charCategory.GetCategory(char)

	// Iterate over all individual category flags (matching Rust behavior exactly)
	categoryIter := categories.Iter()
	current := createdWords

	for {
		category := categoryIter.Next()
		if category == 0 {
			break // No more categories
		}

		categoryInfo := p.charCategory.GetCategoryInfo(category)

		// Skip if no category info or no unknown word definitions
		if categoryInfo == nil || !p.unkDefs.HasDefinitions(category) {
			continue
		}

		// Generate OOV nodes for this category (matching Rust pattern)
		var err error
		current, err = p.generateOOVForCategory(charPos, buffer, lat, current, category, categoryInfo)
		if err != nil {
			return current, err
		}
	}

	return current, nil
}

// generateOOVForCategory generates OOV nodes for a specific category
// This matches the Rust inner loop logic exactly
func (p *MeCabOovPlugin) generateOOVForCategory(charPos int, buffer *input.InputBuffer, lat *lattice.Lattice, createdWords plugin.CreatedWords, category dic.CategoryType, categoryInfo *dic.CategoryInfo) (plugin.CreatedWords, error) {
	// Generate OOV nodes for this category
	definitions := p.unkDefs.GetDefinitions(category)
	if len(definitions) == 0 {
		return createdWords, nil
	}

	// Rust logic: let char_len = input.cat_continuous_len(offset);
	// Get continuous character length for this category
	charLen := p.getCategoryContinuousLength(buffer, charPos, category)
	if charLen == 0 {
		return createdWords, nil
	}

	// Rust logic: if !cinfo.is_invoke && other_words.not_empty() { continue; }
	if !categoryInfo.IsInvoke && createdWords.NotEmpty() {
		return createdWords, nil
	}

	// Rust logic: let oovs = match self.oov_list.get(&cinfo.category_type)
	// definitions already obtained above

	var numCreated int

	// Rust logic: if cinfo.is_group {
	if categoryInfo.IsGroup {
		// Group consecutive same-category characters
		for _, definition := range definitions {
			node := p.getOovNode(definition, charPos, charPos+charLen)
			err := p.insertNode(lat, node)
			if err != nil {
				return createdWords, err
			}
			numCreated++
		}
		charLen-- // Rust logic: llength -= 1;
	}

	// Rust logic: for i in 1..=cinfo.length {
	maxLength := int(categoryInfo.Length)
	for i := 1; i <= maxLength; i++ {
		// Rust logic: let sublength = input.char_distance(offset, i as usize);
		sublength := p.getCharDistance(buffer, charPos, i)
		if sublength > charLen {
			break
		}

		// Create OOV nodes for each definition
		for _, definition := range definitions {
			node := p.getOovNode(definition, charPos, charPos+sublength)
			err := p.insertNode(lat, node)
			if err != nil {
				return createdWords, err
			}
			numCreated++
		}
	}

	// Update CreatedWords with the created nodes (matching Rust behavior)
	// In Rust: other = other.add_word(node.char_range().len() as i64) for each node
	current := createdWords
	for i := 0; i < numCreated; i++ {
		// Assume each node is 1 character for simplicity (can be refined later)
		current = current.AddWord(1)
	}
	return current, nil
}

// getCategoryContinuousLength calculates continuous character length for the given category
// This matches Rust: input.cat_continuous_len(offset)
func (p *MeCabOovPlugin) getCategoryContinuousLength(buffer *input.InputBuffer, charPos int, baseCategory dic.CategoryType) int {
	charCount := buffer.CharCount()
	length := 0

	for i := charPos; i < charCount; i++ {
		char, err := buffer.GetChar(i)
		if err != nil {
			break
		}

		category := p.charCategory.GetCategory(char)
		if category != baseCategory {
			break
		}

		length++
	}

	return length
}

// getCharDistance calculates character distance from start position
// This matches Rust: input.char_distance(offset, i as usize)
func (p *MeCabOovPlugin) getCharDistance(buffer *input.InputBuffer, charPos int, distance int) int {
	charCount := buffer.CharCount()
	endPos := charPos + distance
	if endPos > charCount {
		endPos = charCount
	}
	return endPos - charPos
}

// getOovNode creates an OOV node from definition
// This matches Rust: fn get_oov_node(&self, oov: &Oov, start: usize, end: usize) -> Node
func (p *MeCabOovPlugin) getOovNode(definition *dic.UnknownWordDefinition, start, end int) *lattice.Node {
	// Extract POS ID from definition.POS array (matching Rust oov.pos_id)
	posId := uint16(0) // Default value
	if p.grammar != nil && len(definition.POS) == dic.POSDepth {
		// Use Grammar.GetPartOfSpeechId to get the POS ID
		// This matches Rust behavior where pos_id is resolved from POS components
		if id := p.grammar.GetPartOfSpeechId(definition.POS); id != nil {
			posId = *id
		}

		// Debug output removed - will use lattice dump instead
	}

	// Rust: Node::new(start as u16, end as u16, oov.left_id as u16, oov.right_id as u16, oov.cost, WordId::oov(oov.pos_id as u32))
	return lattice.NewNode(
		uint16(start),              // start as u16
		uint16(end),                // end as u16
		uint16(definition.LeftId),  // left_id as u16
		uint16(definition.RightId), // right_id as u16
		int16(definition.Cost),     // cost
		dic.OOV(uint32(posId)),     // WordId::oov(pos_id as u32)
	)
}

// insertNode inserts a node into the lattice
// This handles the connection matrix logic safely
func (p *MeCabOovPlugin) insertNode(lat *lattice.Lattice, node *lattice.Node) error {
	var connMatrix *dic.ConnectionMatrix
	if p.grammar != nil {
		connMatrix = p.grammar.ConnectionMatrix()
	}
	return lat.Insert(node, connMatrix)
}

// CreateInputTextPlugin creates an input text plugin (not supported by MeCab OOV plugin)
func (p *MeCabOovPlugin) CreateInputTextPlugin(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.InputTextPlugin, error) {
	return nil, fmt.Errorf("MeCab OOV plugin does not support input text plugins")
}

// CreateOOVProvider creates a MeCab OOV provider plugin instance
func (p *MeCabOovPlugin) CreateOOVProvider(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.OOVProviderPlugin, error) {
	mecabPlugin, err := NewMeCabOovPluginFromResourceDir(resourceDir, systemDict.Grammar())
	if err != nil {
		return nil, fmt.Errorf("failed to create MeCab OOV plugin: %w", err)
	}

	// Set up the plugin with configuration
	err = mecabPlugin.SetUp(settings, resourceDir, systemDict.Grammar())
	if err != nil {
		return nil, fmt.Errorf("failed to set up MeCab OOV plugin: %w", err)
	}

	return mecabPlugin, nil
}

// CreatePathRewriter creates a path rewrite plugin (not supported by MeCab OOV plugin)
func (p *MeCabOovPlugin) CreatePathRewriter(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.PathRewritePlugin, error) {
	return nil, fmt.Errorf("MeCab OOV plugin does not support path rewrite plugins")
}

// GetSupportedTypes returns the plugin types this factory supports
func (p *MeCabOovPlugin) GetSupportedTypes() []plugin.PluginType {
	return []plugin.PluginType{plugin.PluginTypeOOVProvider}
}
