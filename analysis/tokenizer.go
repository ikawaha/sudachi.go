package analysis

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/plugin"
	"github.com/ikawaha/sudachi.go/types"
)

// Tokenizer represents the main morphological analysis engine
type Tokenizer struct {
	// Dictionary components
	systemDict *dic.SystemDictionary
	userDicts  []*dic.UserDictionary

	// Analysis components
	lattice    *lattice.Lattice
	normalizer *input.Normalizer

	// Plugin system
	pluginManager *PluginManager

	// OOV handling
	// mecabOovPlugin removed - use plugin manager instead
	useSimpleOov   bool
	simpleOovPosId uint16 // POS ID for Simple OOV nodes (matches Rust implementation)

	// Character categorization system (for simple OOV mode)
	charCategory *dic.CharacterCategory

	// Current analysis state
	inputBuffer *input.InputBuffer
	mode        Mode

	// Debug mode flag
	debugMode bool
}

// NewTokenizer creates a new tokenizer with the given system dictionary
func NewTokenizer(systemDict *dic.SystemDictionary) (*Tokenizer, error) {
	// Initialize Simple OOV POS ID (matching Rust implementation)
	simpleOovPosId := uint16(2) // Default fallback
	if grammar := systemDict.Grammar(); grammar != nil {
		// Try to resolve the Simple OOV POS ["補助記号", "一般", "*", "*", "*", "*"]
		if posId, err := grammar.GetPOSId([]string{"補助記号", "一般", "*", "*", "*", "*"}); err == nil {
			simpleOovPosId = posId
		}
	}
	norm, err := createDefaultNormalizer()
	if err != nil {
		return nil, err
	}

	return &Tokenizer{
		systemDict:     systemDict,
		userDicts:      make([]*dic.UserDictionary, 0),
		lattice:        lattice.New(),
		normalizer:     norm,
		pluginManager:  NewPluginManager(),
		useSimpleOov:   true,                                   // Default to simple OOV
		simpleOovPosId: simpleOovPosId,                         // Dynamic POS ID resolution
		charCategory:   systemDict.Grammar().CharacterCategory, // For character categorization
		mode:           ModeA,
	}, nil
}

// AddUserDictionary adds a user dictionary
func (t *Tokenizer) AddUserDictionary(userDict *dic.UserDictionary) {
	t.userDicts = append(t.userDicts, userDict)
}

// SetMode sets the tokenization mode
func (t *Tokenizer) SetMode(mode Mode) {
	t.mode = mode
}

// Mode returns the current tokenization mode
func (t *Tokenizer) Mode() Mode {
	return t.mode
}

// GetPluginManager returns the plugin manager
func (t *Tokenizer) GetPluginManager() *PluginManager {
	return t.pluginManager
}

// SetDebugMode enables or disables debug mode
func (t *Tokenizer) SetDebugMode(debug bool) {
	t.debugMode = debug
	// Propagate debug flag to plugin manager
	t.pluginManager.SetDebug(debug)
}

// buildLattice constructs the word lattice for analysis
func (t *Tokenizer) buildLattice() error {
	charCount := t.inputBuffer.CharCount()
	t.lattice.Reset(charCount)

	modifiedBytes := t.inputBuffer.ModifiedBytes()

	// For each character position
	for charPos := 0; charPos < charCount; charPos++ {
		// Skip if no path to this position
		if !t.lattice.HasPreviousNode(charPos) {
			continue
		}

		// Get byte offset for this character position
		byteOffset, err := t.inputBuffer.CharToByteIndex(charPos)
		if err != nil {
			continue
		}

		// Initialize CreatedWords tracker (matching Rust: let mut created = CreatedWords::empty())
		createdWords := types.EmptyCreatedWords()

		// Dictionary lookup from all dictionaries (matching Rust pattern)
		createdWords, err = t.lookupDictionary(modifiedBytes, byteOffset, charPos, createdWords)
		if err != nil {
			return err
		}

		// Apply OOV provider plugins in order (matching Rust behavior)
		// Rust: created = self.provide_oovs(ch_off, created, provider.as_ref())?;
		if t.pluginManager.HasOOVProviders() {
			createdWords, err = t.pluginManager.ProvideOOV(charPos, t.inputBuffer, t.lattice, createdWords)
			if err != nil {
				return fmt.Errorf("failed to apply OOV provider plugins: %w", err)
			}
		}

		if createdWords.IsEmpty() {
			return fmt.Errorf("no words created at position %d", charPos)
		}
	}

	// Connect EOS
	connMatrix := t.systemDict.Grammar().ConnectionMatrix()
	err := t.lattice.ConnectEOS(connMatrix)
	if err != nil {
		return fmt.Errorf("failed to connect EOS: %w", err)
	}

	return nil
}

// lookupDictionary performs dictionary lookup and adds nodes to lattice
// Returns updated createdWords with the lengths of found words (matching Rust pattern)
func (t *Tokenizer) lookupDictionary(modifiedBytes []byte, byteOffset, charPos int, createdWords types.CreatedWords) (types.CreatedWords, error) {
	connMatrix := t.systemDict.Grammar().ConnectionMatrix()

	// Dictionary lookup at position

	// Use LexiconSet for unified multiple dictionary lookup (Rust compatible)
	return t.lookupInLexiconSet(t.systemDict.LexiconSet(), modifiedBytes, byteOffset, charPos, connMatrix, createdWords)
}

// lookupInLexiconSet performs lookup in a lexicon set
// Returns updated createdWords with found word lengths
func (t *Tokenizer) lookupInLexiconSet(lexiconSet *dic.LexiconSet, modifiedBytes []byte, byteOffset, charPos int, connMatrix *dic.ConnectionMatrix, createdWords types.CreatedWords) (types.CreatedWords, error) {
	// Create lexicon set iterator
	iter, err := lexiconSet.Lookup(modifiedBytes, byteOffset)
	if err != nil {
		return createdWords, err
	}

	// Count entries for debug
	entryCount := 0

	// Process all matching entries
	current := createdWords
	for {
		entry, err := iter.Next()
		if err != nil {
			break
		}
		if entry == nil {
			break
		}

		entryCount++

		// Convert byte end position to character position
		endBytePos := byteOffset + entry.End

		// Apply Rust boundary check (matching Rust implementation exactly)
		// if (e.end < input_bytes.len()) && !self.input.can_bow(e.end)
		if endBytePos < len(modifiedBytes) && !t.inputBuffer.CanBOW(endBytePos) {
			continue // Skip words that don't end at valid boundaries
		}

		endCharPos, err := t.inputBuffer.ByteToCharIndex(endBytePos)
		if err != nil {
			continue // Skip if byte position is invalid
		}

		// Calculate word length in characters (matching Rust implementation)
		wordLength := endCharPos - charPos
		current = current.AddWord(wordLength)

		// Get word parameters from lexicon set
		leftId, rightId, cost := lexiconSet.GetWordParam(entry.WordId)

		// Create node using pool
		node := lattice.NewNodeFromPool(
			uint16(charPos),
			uint16(endCharPos),
			uint16(leftId),
			uint16(rightId),
			cost,
			entry.WordId,
		)

		// Insert node into lattice
		err = t.lattice.Insert(node, connMatrix)
		if err != nil {
			return current, err
		}
	}

	// Return iterator to pool
	iter.ReturnToPool()

	return current, nil
}

// lookupInLexicon performs lookup in a specific lexicon
// Returns updated createdWords with found word lengths
func (t *Tokenizer) lookupInLexicon(lexicon *dic.Lexicon, modifiedBytes []byte, byteOffset, charPos int, connMatrix *dic.ConnectionMatrix, createdWords types.CreatedWords) (types.CreatedWords, error) {
	// Create lexicon iterator
	iter, err := lexicon.Lookup(modifiedBytes, byteOffset)
	if err != nil {
		return createdWords, err
	}

	// Process all matching entries
	current := createdWords
	for {
		entry, err := iter.Next()
		if err != nil {
			break
		}
		if entry == nil {
			break
		}

		// Convert byte end position to character position
		endBytePos := byteOffset + entry.End

		// Apply Rust boundary check (matching Rust implementation exactly)
		// if (e.end < input_bytes.len()) && !self.input.can_bow(e.end)
		if endBytePos < len(modifiedBytes) && !t.inputBuffer.CanBOW(endBytePos) {
			continue // Skip words that don't end at valid boundaries
		}

		endCharPos, err := t.inputBuffer.ByteToCharIndex(endBytePos)
		if err != nil {
			continue // Skip if byte position is invalid
		}

		// Calculate word length in characters (matching Rust implementation)
		wordLength := endCharPos - charPos
		current = current.AddWord(wordLength)

		// Get word parameters
		rawWordId := entry.WordId.Word()
		leftId, rightId, cost := lexicon.GetWordParam(rawWordId)

		// Debug disabled

		// Create node using pool
		node := lattice.NewNodeFromPool(
			uint16(charPos),
			uint16(endCharPos),
			uint16(leftId),
			uint16(rightId),
			cost,
			entry.WordId,
		)

		// Debug disabled

		// Insert node into lattice
		err = t.lattice.Insert(node, connMatrix)
		if err != nil {
			return current, err
		}
	}

	// Return iterator to pool
	iter.ReturnToPool()

	return current, nil
}

// generateOOV generates out-of-vocabulary words
func (t *Tokenizer) generateOOV(charPos int, createdWords *types.CreatedWords) error {
	if charPos >= t.inputBuffer.CharCount() {
		return nil
	}

	// Debug disabled

	_, err := t.inputBuffer.GetChar(charPos)
	if err != nil {
		return err
	}

	category, err := t.inputBuffer.GetCategory(charPos)
	if err != nil {
		return err
	}

	// Check for NOOOVBOW flags that block OOV generation
	if t.shouldSkipOOV(category) {
		return nil
	}

	// Use MeCab OOV plugin if available
	if !t.useSimpleOov {
		return t.generateMeCabOOV(charPos, createdWords)
	}
	return fmt.Errorf("failed to build lattice: no words created at position %d", charPos)
}

// shouldSkipOOV checks if OOV generation should be skipped for this category
func (t *Tokenizer) shouldSkipOOV(category dic.CategoryType) bool {
	// Skip OOV for NOOOVBOW categories (matches Rust behavior - uses intersects)
	return category.Intersects(dic.CategoryNoOOVBOW | dic.CategoryNoOOVBOW2)
}

// generateFallbackOOV generates OOV words as fallback using the last OOV provider (matching Rust behavior exactly)
func (t *Tokenizer) generateFallbackOOV(charPos int, createdWords *types.CreatedWords) error {
	if charPos >= t.inputBuffer.CharCount() {
		return nil
	}

	// Get the last OOV provider (matching Rust: let provider = self.oov_providers.last().unwrap())
	lastProvider := t.pluginManager.GetLastOOVProvider()
	if lastProvider == nil {
		return fmt.Errorf("no OOV providers available for fallback at position %d", charPos)
	}

	// Use the last provider for fallback (matching Rust: self.provide_oovs(ch_off, created, provider.as_ref())?)
	newCreatedWords, err := lastProvider.ProvideOOV(charPos, t.inputBuffer, t.lattice, *createdWords)
	if err != nil {
		return err
	}
	*createdWords = newCreatedWords

	return nil
}

// generateMeCabOOV generates OOV using MeCab plugin
func (t *Tokenizer) generateMeCabOOV(charPos int, createdWords *types.CreatedWords) error {
	// Set character category system in buffer
	// Character category is set in plugin manager

	// Create hasKnownWord slice for the entire buffer
	charCount := t.inputBuffer.CharCount()
	hasKnownWord := make([]bool, charCount)

	// Mark positions where we already have words (based on createdWords)
	if createdWords.NotEmpty() {
		hasKnownWord[charPos] = true
	}

	// Generate OOV candidates using Plugin Manager
	var err error
	*createdWords, err = t.pluginManager.ProvideOOV(charPos, t.inputBuffer, t.lattice, *createdWords)
	if err != nil {
		return err
	}

	return nil
}

// getMeCabOOVInfo gets part of speech information for MeCab OOV nodes
func (t *Tokenizer) getMeCabOOVInfo(node *lattice.Node) ([]string, []string, error) {
	// For OOV nodes, the WordId().Word() contains the POS ID set during node creation
	wordId := node.WordId()
	posId := uint16(wordId.Word()) // Extract POS ID (this is the posId from getOovNode)

	// Use Grammar to get POS from the resolved POS ID (matching Rust behavior)
	if t.systemDict != nil && t.systemDict.Grammar() != nil {
		if pos, err := t.systemDict.Grammar().GetPOS(posId); err == nil {
			return pos, []string{}, nil
		}
	}
	return nil, nil, fmt.Errorf("failed to resolve POS for OOV node with ID %d", posId)
}

// pathToMorphemes converts the optimal path to morpheme results with mode support
// pathToNodeResults converts a path of nodes to NodeResults with complete word information
// This enables PathRewritePlugins to access reading forms and other word information
func (t *Tokenizer) pathToNodeResults(path []*lattice.Node) ([]*lattice.NodeResult, error) {
	var results []*lattice.NodeResult

	for _, node := range path {
		// Skip BOS and EOS nodes specifically (not all OOV nodes)
		if node.WordId().Raw() == lattice.BOSWordID.Raw() || node.WordId().Raw() == lattice.EOSWordID.Raw() {
			continue
		}

		// Get surface form from original text
		surface, err := t.getSurfaceForm(node)
		if err != nil {
			return nil, err
		}

		// Get part of speech information
		pos, err := t.getPartOfSpeech(node)
		if err != nil {
			return nil, err
		}

		// Get normalized form (dictionary form)
		normalizedForm, err := t.getNormalizedForm(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get normalized form for node [%d,%d): %w", node.Begin(), node.End(), err)
		}

		// Get dictionary form (base form)
		dictionaryForm, err := t.getDictionaryForm(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get dictionary form for node [%d,%d): %w", node.Begin(), node.End(), err)
		}

		// Get reading form
		readingForm, err := t.getReadingForm(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get reading form for node [%d,%d): %w", node.Begin(), node.End(), err)
		}

		// Get synonym group IDs
		synonymGroupIds, err := t.getSynonymGroupIds(node)
		if err != nil {
			// If error, use empty features
			synonymGroupIds = []string{}
		}

		// Create NodeResult with complete word information using pool
		result := lattice.NewNodeResultCompleteFromPool(node, surface, pos, synonymGroupIds, normalizedForm, dictionaryForm, readingForm)
		results = append(results, result)
	}

	return results, nil
}

// nodeResultsToMorphemes converts NodeResults to MorphemeList
func (t *Tokenizer) nodeResultsToMorphemes(nodeResults []*lattice.NodeResult) (*lattice.MorphemeList, error) {
	results := lattice.NewMorphemeList()

	for _, nodeResult := range nodeResults {
		results.Add(nodeResult)
	}

	// Apply mode-based splitting if not Mode C
	if t.mode != ModeC {
		return t.applySplitting(results)
	}

	return results, nil
}

func (t *Tokenizer) pathToMorphemes(path []*lattice.Node) (*lattice.MorphemeList, error) {
	results := lattice.NewMorphemeList()

	// Debug disabled

	for _, node := range path {
		// Debug disabled

		// Skip BOS and EOS nodes specifically (not all OOV nodes)
		if node.WordId().Raw() == lattice.BOSWordID.Raw() || node.WordId().Raw() == lattice.EOSWordID.Raw() {
			continue
		}

		// Get surface form from original text
		surface, err := t.getSurfaceForm(node)
		if err != nil {
			return nil, err
		}

		// Get part of speech information
		pos, err := t.getPartOfSpeech(node)
		if err != nil {
			return nil, err
		}

		// Get normalized form (dictionary form)
		normalizedForm, err := t.getNormalizedForm(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get normalized form for node [%d,%d): %w", node.Begin(), node.End(), err)
		}

		// Get dictionary form (base form)
		dictionaryForm, err := t.getDictionaryForm(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get dictionary form for node [%d,%d): %w", node.Begin(), node.End(), err)
		}

		// Get reading form
		readingForm, err := t.getReadingForm(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get reading form for node [%d,%d): %w", node.Begin(), node.End(), err)
		}

		// Get synonym group IDs
		synonymGroupIds, err := t.getSynonymGroupIds(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get synonym group IDs for node [%d,%d): %w", node.Begin(), node.End(), err)
		}

		// Create result using pool
		result := lattice.NewNodeResultCompleteFromPool(node, surface, pos, synonymGroupIds, normalizedForm, dictionaryForm, readingForm)
		results.Add(result)
	}

	// Apply mode-based splitting if not Mode C
	if t.mode != ModeC {
		return t.applySplitting(results)
	}

	return results, nil
}

// applySplitting applies A/B mode splitting to the morpheme list
// This is a faithful port of Rust's mode-based splitting logic
func (t *Tokenizer) applySplitting(results *lattice.MorphemeList) (*lattice.MorphemeList, error) {
	// Rust implementation: Mode C never applies splitting
	if t.mode == ModeC {
		return results, nil
	}

	// Use the lattice package's Split implementation
	// which is a faithful port of Rust's MorphemeList.split_into()
	lexiconSet := t.systemDict.LexiconSet()

	return results.Split(lattice.Mode(t.mode), lexiconSet, t.inputBuffer, t.systemDict.Grammar())
}

// getSurfaceForm extracts the surface form for a node
// This matches Rust implementation exactly: morpheme.surface() -> orig_slice(bytes_range()) for ALL nodes
// Both dictionary and OOV nodes use original text (no normalization in surface)
func (t *Tokenizer) getSurfaceForm(node *lattice.Node) (string, error) {
	// Rust version uses orig_slice(bytes_range()) for ALL morphemes (dictionary and OOV)
	// This ensures surface always shows the original text, not normalized form

	// Get byte range in modified text (equivalent to node.bytes_range() in Rust)
	// This matches: morpheme.surface() -> node.bytes_range() -> orig_slice(bytes_range)
	modifiedByteRange, err := node.BytesRange(t.inputBuffer)
	if err != nil {
		return "", fmt.Errorf("failed to get byte range for node [%d,%d): %w", node.Begin(), node.End(), err)
	}

	// Extract from original text using OrigSlice (equivalent to orig_slice() in Rust)
	// This matches: orig_slice(bytes_range) -> to_orig(bytes_range) -> original[start..end]
	// Works for both dictionary and OOV nodes - surface always shows original text
	surfaceForm := t.inputBuffer.OrigSlice(modifiedByteRange)
	// Note: empty string is valid for normalized characters (like full-width -> half-width)

	return surfaceForm, nil
}

// getModifiedForm extracts the normalized form from modified text
func (t *Tokenizer) getModifiedForm(node *lattice.Node) (string, error) {
	// Get character range
	begin := int(node.Begin())
	end := int(node.End())

	if begin < 0 || end > t.inputBuffer.CharCount() || begin >= end {
		return "", fmt.Errorf("invalid character range: begin=%d, end=%d, charCount=%d", begin, end, t.inputBuffer.CharCount())
	}

	// Get byte offsets
	beginByteOffset, err := t.inputBuffer.CharToByteIndex(begin)
	if err != nil {
		return "", err
	}

	endByteOffset, err := t.inputBuffer.CharToByteIndex(end)
	if err != nil {
		return "", err
	}

	// Extract substring from modified (normalized) text
	modified := t.inputBuffer.Modified()

	if beginByteOffset < 0 || endByteOffset > len(modified) || beginByteOffset >= endByteOffset {
		return "", fmt.Errorf("invalid modified byte range: begin=%d, end=%d, modifiedLen=%d", beginByteOffset, endByteOffset, len(modified))
	}

	// Check UTF-8 character boundaries (same as OrigSlice)
	if !input.IsCharBoundary(modified, beginByteOffset) || !input.IsCharBoundary(modified, endByteOffset) {
		return "", fmt.Errorf("byte range not at UTF-8 character boundary: begin=%d, end=%d", beginByteOffset, endByteOffset)
	}

	return modified[beginByteOffset:endByteOffset], nil
}

// getNormalizedForm extracts the normalized (dictionary) form for a node
func (t *Tokenizer) getNormalizedForm(node *lattice.Node) (string, error) {
	wordId := node.WordId()

	// Check if this is an OOV node
	if wordId.IsOOV() {
		// Check if this is a joined numeric node (special case)
		if t.isJoinedNumericNode(node) {
			surface, err := t.getSurfaceForm(node)
			if err != nil {
				return surface, err
			}
		}

		// For other OOV nodes, use the normalized (modified) text as normalized form
		return t.getModifiedForm(node)
	}

	// Use LexiconSet for word info lookup (Rust compatible)
	wordInfo, err := t.systemDict.LexiconSet().GetWordInfo(wordId)
	if err != nil {
		return "", err
	}

	// Get the normalized form (headword) from the dictionary
	// Use GetNormalizedForm method (matching Rust behavior exactly)
	normalizedForm := wordInfo.GetNormalizedForm()

	// Return dictionary normalized form as-is (matching Rust behavior)
	// Numeric normalization is handled only by JoinNumericPlugin
	return normalizedForm, nil
}

// getDictionaryForm retrieves the dictionary form (base form) for a node
func (t *Tokenizer) getDictionaryForm(node *lattice.Node) (string, error) {
	wordId := node.WordId()

	// Check if this is an OOV node
	if wordId.IsOOV() {
		// For OOV nodes, use the normalized/modified form as dictionary form
		return t.getModifiedForm(node)
	}

	// Get word info from the dictionary
	var wordInfo *dic.WordInfo
	var err error

	// Use LexiconSet for word info lookup (Rust compatible)
	wordInfo, err = t.systemDict.LexiconSet().GetWordInfo(wordId)

	if err != nil {
		return "", err
	}

	// Get the dictionary form from the dictionary
	// Use GetDictionaryForm method (matching Rust behavior exactly)
	dictionaryForm := wordInfo.GetDictionaryForm()

	return dictionaryForm, nil
}

// getReadingForm retrieves the reading form (kana) for a node
func (t *Tokenizer) getReadingForm(node *lattice.Node) (string, error) {
	wordId := node.WordId()

	// Check if this is an OOV node
	if wordId.IsOOV() {
		// For OOV nodes, use the normalized/modified form as reading form (matches Rust behavior)
		// Rust version: For OOV words, the normalized form is used as reading
		return t.getModifiedForm(node)
	}

	// Use LexiconSet for word info lookup (Rust compatible)
	wordInfo, err := t.systemDict.LexiconSet().GetWordInfo(wordId)
	if err != nil {
		return "", err
	}
	return wordInfo.GetReadingForm(), nil
}

// getSynonymGroupIds retrieves the synonym group IDs for a node
func (t *Tokenizer) getSynonymGroupIds(node *lattice.Node) ([]string, error) {
	wordId := node.WordId()

	// Check if this is an OOV node
	if wordId.IsOOV() {
		// OOV nodes don't have synonym group IDs
		return []string{}, nil
	}

	// Get word info from the dictionary
	var wordInfo *dic.WordInfo
	var err error

	// Use LexiconSet for word info lookup (Rust compatible)
	wordInfo, err = t.systemDict.LexiconSet().GetWordInfo(wordId)

	if err != nil {
		return []string{}, err
	}

	// Convert synonym group IDs to strings
	synonymGroupIds := make([]string, len(wordInfo.SynonymGroupIds))
	for i, id := range wordInfo.SynonymGroupIds {
		synonymGroupIds[i] = fmt.Sprintf("%d", id)
	}

	return synonymGroupIds, nil
}

// isJoinedNumericNode checks if a node is a joined numeric node created by JoinNumericPlugin
func (t *Tokenizer) isJoinedNumericNode(node *lattice.Node) bool {
	if !node.WordId().IsOOV() {
		return false
	}

	// Check if this OOV node represents a numeric value by examining the surface
	surface, err := t.getSurfaceForm(node)
	if err != nil {
		return false
	}

	// Check if surface is purely numeric (digits, commas, periods)
	return t.isNumericSurface(surface)
}

// isNumericSurface checks if a surface contains only numeric characters and separators
func (t *Tokenizer) isNumericSurface(surface string) bool {
	if surface == "" {
		return false
	}

	hasDigit := false
	for _, r := range surface {
		if r >= '0' && r <= '9' {
			hasDigit = true
		} else if r == ',' || r == '.' {
			// Allow numeric separators
		} else if t.isKanjiNumeric(r) {
			// Allow Kanji numerics
			hasDigit = true
		} else {
			// Non-numeric character found
			return false
		}
	}

	return hasDigit
}

// isKanjiNumeric checks if character is a Kanji numeric
func (t *Tokenizer) isKanjiNumeric(r rune) bool {
	kanjiNumerics := "〇一二三四五六七八九十百千万億兆"
	for _, kn := range kanjiNumerics {
		if r == kn {
			return true
		}
	}
	return false
}

// getPartOfSpeech gets part of speech for a node
func (t *Tokenizer) getPartOfSpeech(node *lattice.Node) ([]string, error) {
	if node.IsOOV() {
		// Check if this is a joined numeric node by surface pattern
		if t.isJoinedNumericNode(node) {
			// Return numeric POS for joined numeric nodes
			return []string{"名詞", "数詞", "*", "*", "*", "*"}, nil
		}

		// Extract POS ID from the OOV node's WordId (matching Rust behavior exactly)
		// Rust: pos_id: inner.word_id().word() as u16
		posID := uint16(node.WordId().Word())

		// Get POS from grammar using the POS ID (matching Rust behavior)
		grammar := t.systemDict.Grammar()
		if grammar == nil {
			return nil, fmt.Errorf("grammar is not available in system dictionary")
		}

		pos, err := grammar.GetPOS(posID)
		if err != nil {
			return nil, fmt.Errorf("failed to get POS for OOV POS ID %d: %w", posID, err)
		}
		if len(pos) == 0 {
			return nil, fmt.Errorf("failed to resolve POS for OOV POS ID %d", posID)
		}

		return pos, nil
	}

	// Use LexiconSet for word info lookup (Rust compatible)
	wordInfo, err := t.systemDict.LexiconSet().GetWordInfo(node.WordId())
	if err != nil {
		return nil, fmt.Errorf("failed to get word info from lexicon set: %w", err)
	}

	// Get POS from grammar if available
	grammar := t.systemDict.Grammar()
	if grammar == nil {
		return nil, fmt.Errorf("grammar is not available in system dictionary")
	}
	pos, err := grammar.GetPOS(wordInfo.PosId)
	if err != nil {
		return nil, fmt.Errorf("failed to get POS for word ID %d: %w", node.WordId().Raw(), err)
	}
	if len(pos) == 0 {
		return nil, fmt.Errorf("failed to resolve POS for word ID %d", node.WordId().Raw())
	}

	return pos, nil
}

// GetLattice returns the current lattice (for debugging)
func (t *Tokenizer) GetLattice() *lattice.Lattice {
	return t.lattice
}

// GetInputBuffer returns the current input buffer (for debugging)
func (t *Tokenizer) GetInputBuffer() *input.InputBuffer {
	return t.inputBuffer
}

// GetSystemDict returns the system dictionary (for debugging)
func (t *Tokenizer) GetSystemDict() *dic.SystemDictionary {
	return t.systemDict
}

// Reset resets the tokenizer state
func (t *Tokenizer) Reset() {
	t.inputBuffer = nil
	if t.lattice != nil {
		t.lattice.Reset(0)
	}
}

// AddInputTextPlugin adds an input text plugin
func (t *Tokenizer) AddInputTextPlugin(plugin plugin.InputTextPlugin) {
	t.pluginManager.AddInputTextPlugin(plugin)
}

// AddOOVProvider adds an OOV provider plugin
func (t *Tokenizer) AddOOVProvider(plugin plugin.OOVProviderPlugin) {
	t.pluginManager.AddOOVProvider(plugin)
}

// AddPathRewriter adds a path rewrite plugin
func (t *Tokenizer) AddPathRewriter(plugin plugin.PathRewritePlugin) {
	t.pluginManager.AddPathRewriter(plugin)
}

// SetMeCabOovPlugin is deprecated - use plugin manager instead
func (t *Tokenizer) SetMeCabOovPlugin(mecabPlugin any) {
	// Deprecated: Use plugin manager instead
	// This method is kept for backward compatibility but does nothing
}

// SetSimpleOov switches to simple OOV mode
func (t *Tokenizer) SetSimpleOov() {
	t.useSimpleOov = true
	// mecabOovPlugin removed
}

// SetCharacterCategory sets the character category system (for simple OOV mode)
func (t *Tokenizer) SetCharacterCategory(charCategory *dic.CharacterCategory) {
	t.charCategory = charCategory
}

// createInputBufferWithCharacterCategory creates an input buffer with character category set before building
func (t *Tokenizer) createInputBufferWithCharacterCategory(text string) (*input.InputBuffer, *input.NormalizationInfo, error) {
	buffer := input.NewInputBufferFromPool()

	// Set character category system before build if available
	if t.charCategory != nil {
		buffer.SetCharacterCategory(t.charCategory)
	}

	// Start building the buffer (matching Rust: start_build())
	// Always use simple StartBuild, normalization will be done via input text plugins
	err := buffer.StartBuild(text)
	if err != nil {
		return nil, nil, err
	}

	// Apply input text plugins before building (while buffer is still writable)
	if t.pluginManager.HasInputTextPlugins() {
		err = t.pluginManager.ProcessInputText(buffer)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process input text plugins: %w", err)
		}
	}

	// Build the buffer (this will compute character categories using the set category system)
	err = buffer.Build(t.systemDict.Grammar())
	if err != nil {
		return nil, nil, err
	}

	// Create dummy normalization info for backward compatibility
	info := &input.NormalizationInfo{Applied: false}
	return buffer, info, nil
}

// dumpLattice outputs lattice information in Rust-compatible format
func (t *Tokenizer) dumpLattice() {
	// Define helper functions for lattice dump
	getSurface := func(node *lattice.Node) string {
		if t.inputBuffer == nil {
			return "UNKNOWN"
		}
		// Match Rust lattice dump logic exactly:
		// - Dictionary nodes (non-OOV): use original text (orig_slice_c)
		// - OOV nodes: use normalized text (curr_slice_c)
		var surface string
		var err error
		if node.WordId().IsOOV() {
			// OOV nodes: use normalized/modified form (like Rust curr_slice_c)
			surface, err = t.getModifiedForm(node)
		} else {
			// Dictionary nodes: use original form (like Rust orig_slice_c)
			surface, err = t.getSurfaceForm(node)
		}
		if err != nil {
			return "ERROR"
		}
		return surface
	}

	getPOS := func(node *lattice.Node) string {
		if node.WordId().IsOOV() {
			// For OOV nodes, try to get POS from grammar
			if t.systemDict != nil && t.systemDict.Grammar() != nil {
				if pos, err := t.systemDict.Grammar().GetPOS(uint16(node.WordId().Word())); err == nil {
					return fmt.Sprintf("%s, %s, %s, %s, %s, %s", pos[0], pos[1], pos[2], pos[3], pos[4], pos[5])
				}
			}
			return "OOV_POS_UNKNOWN"
		} else {
			// For dictionary nodes, get POS from word info
			if t.systemDict != nil {
				wordInfo, err := t.getPartOfSpeech(node)
				if err == nil && len(wordInfo) >= 6 {
					return fmt.Sprintf("%s, %s, %s, %s, %s, %s", wordInfo[0], wordInfo[1], wordInfo[2], wordInfo[3], wordInfo[4], wordInfo[5])
				}
			}
			return "DICT_POS_UNKNOWN"
		}
	}

	getConnectionCosts := func(node *lattice.Node) []int {
		// Calculate connection costs from previous nodes to this node (matching Rust exactly)
		// Rust: for l_node in &self.ends[r_node.begin()]
		var costs []int
		prevPos := int(node.Begin()) // This node's begin position

		// Get connection matrix
		connMatrix := t.systemDict.Grammar().ConnectionMatrix()
		if connMatrix == nil {
			return []int{-999} // Fallback if no connection matrix
		}

		// Check if previous position exists in lattice
		if prevPos < 0 || prevPos >= t.lattice.Size() || !t.lattice.HasNodes(prevPos) {
			return []int{} // No previous nodes
		}

		// Calculate costs from all previous nodes to this node (matching Rust exactly)
		// Rust: conn.cost(l_node.right_id(), r_node.left_id())
		prevNodes := t.lattice.GetNodesAt(prevPos)
		for _, prevNode := range prevNodes {
			if prevNode == nil {
				continue
			}
			// Calculate connection cost from previous node to this node
			connectCost := connMatrix.Cost(prevNode.RightId(), node.LeftId())
			costs = append(costs, int(connectCost))
		}

		return costs
	}

	// Pass lexicon set for accurate POS ID extraction (matching Rust behavior)
	var lexiconSet *dic.LexiconSet
	if t.systemDict != nil {
		lexiconSet = t.systemDict.LexiconSet()
	}
	t.lattice.DumpWithDetails(getSurface, getPOS, getConnectionCosts, lexiconSet)
}

// dumpPath outputs path information in Rust-compatible format
func (t *Tokenizer) dumpPath(header string, nodeResults []*lattice.NodeResult) {
	fmt.Println(header)
	for i, result := range nodeResults {
		if result == nil {
			continue
		}

		node := result.Node()
		if node == nil {
			continue
		}

		// Get surface form
		surface, err := t.getSurfaceForm(node)
		if err != nil {
			surface = "ERROR"
		}

		// Get dictionary ID and word ID
		var dictID int
		var wordID uint32
		if node.WordId().IsOOV() {
			dictID = -1
			wordID = node.WordId().Word()
		} else {
			dictID = 0
			wordID = node.WordId().Word()
		}

		// Get POS ID (matching Rust format exactly)
		var posID uint16 = 0
		if t.systemDict != nil && t.systemDict.LexiconSet() != nil {
			if node.WordId().IsOOV() {
				// OOV nodes store the POS ID directly in their WordId
				posID = uint16(node.WordId().Word())
			} else {
				// Dictionary nodes: get WordInfo from LexiconSet to access PosId
				if wordInfo, err := t.systemDict.LexiconSet().GetWordInfo(node.WordId()); err == nil {
					posID = wordInfo.PosId
				}
			}
		}

		// Format: path_index: begin_pos end_pos surface(dict_id, word_id) pos_id left_id right_id node_cost
		fmt.Printf("%d: %d %d %s(%d, %d) %d %d %d %d\n",
			i,
			node.Begin(),
			node.End(),
			surface,
			dictID,
			wordID,
			posID, // Changed from totalCost to posID (matching Rust)
			node.LeftId(),
			node.RightId(),
			node.Cost(),
		)
	}
}

// calculateCumulativeCost calculates the cumulative cost up to the given node index in the optimal path
func (t *Tokenizer) calculateCumulativeCost(nodeResults []*lattice.NodeResult, nodeIndex int) int32 {
	if nodeIndex < 0 || nodeIndex >= len(nodeResults) {
		return 0
	}

	// Get connection matrix for cost calculation
	connMatrix := t.systemDict.Grammar().ConnectionMatrix()
	if connMatrix == nil {
		return 0 // Fallback
	}

	var cumulativeCost int32 = 0

	// Calculate cumulative cost by summing node costs and connection costs
	for i := 0; i <= nodeIndex; i++ {
		if nodeResults[i] == nil || nodeResults[i].Node() == nil {
			continue
		}

		currentNode := nodeResults[i].Node()

		// Add the node's own cost
		cumulativeCost += int32(currentNode.Cost())

		// Add connection cost from previous node (if exists)
		if i > 0 && nodeResults[i-1] != nil && nodeResults[i-1].Node() != nil {
			prevNode := nodeResults[i-1].Node()
			connectionCost := connMatrix.Cost(prevNode.RightId(), currentNode.LeftId())
			cumulativeCost += int32(connectionCost)
		}
	}

	return cumulativeCost
}

// createDefaultNormalizer creates a normalizer with default Sudachi rewrite.def
// This replaces the removed CreateDefaultJapaneseNormalizer function
func createDefaultNormalizer() (*input.Normalizer, error) {
	normalizer, err := input.NewDefaultSudachiNormalizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create default Sudachi normalizer: %w", err)
	}

	return normalizer, nil
}
