package path_rewrite

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/plugin"
)

// StringNumber represents a number as a string with scale and decimal point handling (Rust-compatible)
type StringNumber struct {
	significand string // The digits as a string
	scale       int    // Power of 10 multiplier (number of trailing zeros)
	point       int    // Decimal point position (-1 if none)
	isAllZero   bool   // Tracks if all digits processed so far are zero
}

// NewStringNumber creates a new StringNumber initialized to empty (matching Rust)
func NewStringNumber() *StringNumber {
	return &StringNumber{
		significand: "", // Empty like Rust version
		scale:       0,
		point:       -1,
		isAllZero:   true,
	}
}

// NewStringNumberFromString creates a StringNumber from a string representation
func NewStringNumberFromString(s string) *StringNumber {
	sn := NewStringNumber()
	sn.significand = s
	return sn
}

// clear resets the StringNumber to initial state (matching Rust implementation)
func (sn *StringNumber) clear() {
	sn.significand = ""
	sn.scale = 0
	sn.point = -1
	sn.isAllZero = true
}

// append adds a digit to the StringNumber (matching Rust implementation)
func (sn *StringNumber) append(i int) {
	if i != 0 {
		sn.isAllZero = false
	}
	sn.significand += strconv.Itoa(i)
}

// shiftScale multiplies the number by 10^i (matching Rust implementation)
func (sn *StringNumber) shiftScale(i int) {
	if sn.isZero() {
		sn.significand += "1"
	}
	sn.scale = sn.scale + i
}

// add combines this StringNumber with another (exact Rust implementation)
func (sn *StringNumber) add(other *StringNumber) bool {
	if other.isZero() {
		return true
	}

	if sn.isZero() {
		sn.significand += other.significand
		sn.scale = other.scale
		sn.point = other.point
		return true
	}

	// Exact Rust logic - this is where invalid number structures are rejected
	sn.normalizeScale()
	length := other.intLength()

	if sn.scale >= length {
		sn.fillZero(sn.scale - length)
		if other.point >= 0 {
			sn.point = len(sn.significand) + other.point
		}
		sn.significand += other.significand
		sn.scale = other.scale
		return true
	}

	// Critical: return false for invalid number structures (like "十三四")
	return false
}

// isZero checks if the number represents zero (exact Rust implementation)
func (sn *StringNumber) isZero() bool {
	return sn.significand == ""
}

// normalizeScale implements Rust normalize_scale method
func (sn *StringNumber) normalizeScale() {
	if sn.point >= 0 {
		nScale := len(sn.significand) - sn.point
		if nScale > sn.scale {
			sn.point += sn.scale
			sn.scale = 0
		} else {
			sn.scale -= nScale
			sn.point = -1
		}
	}
}

// intLength implements Rust int_length method
func (sn *StringNumber) intLength() int {
	sn.normalizeScale()
	if sn.point >= 0 {
		return sn.point
	}
	return len(sn.significand) + sn.scale
}

// fillZero implements Rust fill_zero method
func (sn *StringNumber) fillZero(length int) {
	for i := 0; i < length; i++ {
		sn.significand += "0"
	}
}

// ToString implements Rust StringNumber.to_string() method exactly
func (sn *StringNumber) ToString() string {
	if sn.isZero() {
		return "0"
	}

	sn.normalizeScale()
	if sn.scale > 0 {
		sn.fillZero(sn.scale)
	} else if sn.point >= 0 {
		// Insert decimal point
		if sn.point < len(sn.significand) {
			sn.significand = sn.significand[:sn.point] + "." + sn.significand[sn.point:]
		} else {
			sn.significand = sn.significand + "."
		}

		// Insert leading zero if needed
		if sn.point == 0 {
			sn.significand = "0" + sn.significand
		}

		// Remove trailing zeros (matching Rust implementation)
		for len(sn.significand) > 0 && sn.significand[len(sn.significand)-1] == '0' {
			sn.significand = sn.significand[:len(sn.significand)-1]
		}

		// Remove trailing decimal point
		if len(sn.significand) > 0 && sn.significand[len(sn.significand)-1] == '.' {
			sn.significand = sn.significand[:len(sn.significand)-1]
		}
	}

	return sn.significand
}

// NumericParserError represents parsing error states
type NumericParserError int

const (
	ErrorNone NumericParserError = iota
	ErrorPoint
	ErrorComma
)

// NumericParser parses numbers written in Arabic numerals or Kanji and provides normalization
type NumericParser struct {
	digitLength     int
	isFirstDigit    bool
	hasComma        bool
	hasHangingPoint bool
	errorState      NumericParserError
	total           *StringNumber
	subtotal        *StringNumber
	tmp             *StringNumber
	charToNum       map[rune]int
}

var charToNum = map[rune]int{
	// kanji numerals
	'〇': 0, '一': 1, '二': 2, '三': 3, '四': 4,
	'五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
	'十': -1,  // Unit marker, not digit value
	'百': -2,  // Unit marker, not digit value
	'千': -3,  // Unit marker, not digit value
	'万': -4,  // Unit marker, not digit value
	'億': -8,  // Unit marker, not digit value
	'兆': -12, // Unit marker, not digit value
	// arabic numerals
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4,
	'5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
}

// NewNumericParser creates a new NumericParser
func NewNumericParser() *NumericParser {
	return &NumericParser{
		digitLength:     0,
		isFirstDigit:    true,
		hasComma:        false,
		hasHangingPoint: false,
		errorState:      ErrorNone,
		total:           NewStringNumber(),
		subtotal:        NewStringNumber(),
		tmp:             NewStringNumber(),
		charToNum:       charToNum,
	}
}

// Clear resets the parser to initial state
func (p *NumericParser) Clear() {
	p.digitLength = 0
	p.isFirstDigit = true
	p.hasComma = false
	p.hasHangingPoint = false
	p.errorState = ErrorNone
	p.total = NewStringNumber()
	p.subtotal = NewStringNumber()
	p.tmp = NewStringNumber()
}

// Append processes a character and updates parser state (matching Rust implementation)
func (p *NumericParser) Append(c rune) bool {
	// Handle comma and period first (like Rust)
	if c == ',' {
		if p.hasComma || p.digitLength != 3 {
			p.errorState = ErrorComma
			return false
		}
		p.hasComma = true
		p.digitLength = 0
		return true
	}

	if c == '.' {
		if p.hasHangingPoint {
			p.errorState = ErrorPoint
			return false
		}
		p.hasHangingPoint = true
		p.total.point = len(p.total.significand)
		return true
	}

	// Get numeric value
	val, exists := p.charToNum[c]
	if !exists {
		return false
	}

	// Handle based on value type (matching Rust logic exactly)
	if p.isSmallUnit(val) {
		// Small units: 十(-1), 百(-2), 千(-3)
		p.tmp.shiftScale(-val)
		if !p.subtotal.add(p.tmp) {
			return false
		}
		p.tmp.clear()
		p.isFirstDigit = true
		p.digitLength = 0
		p.hasComma = false
	} else if p.isLargeUnit(val) {
		// Large units: 万(-4), 億(-8), 兆(-12), etc.
		if !p.subtotal.add(p.tmp) || p.subtotal.isZero() {
			return false
		}
		p.subtotal.shiftScale(-val)
		if !p.total.add(p.subtotal) {
			return false
		}
		p.subtotal.clear()
		p.tmp.clear()
		p.isFirstDigit = true
		p.digitLength = 0
		p.hasComma = false
	} else {
		// Regular digits (0-9)
		p.tmp.append(val)
		p.isFirstDigit = false
		p.digitLength++
		p.hasHangingPoint = false
	}

	return true
}

// isSmallUnit checks if value is a small unit (十, 百, 千)
func (p *NumericParser) isSmallUnit(n int) bool {
	return n >= -3 && n < 0
}

// isLargeUnit checks if value is a large unit (万, 億, 兆, etc.)
func (p *NumericParser) isLargeUnit(n int) bool {
	return n < -3
}

// Done returns whether parsing is complete and successful (matching Rust implementation)
func (p *NumericParser) Done() bool {
	// Finalize the parsing by adding remaining subtotals (like Rust)
	ret := p.subtotal.add(p.tmp) && p.total.add(p.subtotal)

	if p.hasHangingPoint {
		p.errorState = ErrorPoint
		return false
	}

	if p.hasComma && p.digitLength != 3 {
		p.errorState = ErrorComma
		return false
	}

	// Debug: check if we have any meaningful result
	finalResult := ret && p.errorState == ErrorNone && !p.total.isZero()

	return finalResult
}

// ErrorState returns the current error state
func (p *NumericParser) ErrorState() NumericParserError {
	return p.errorState
}

// GetNormalized returns the normalized string representation (matching Rust implementation)
func (p *NumericParser) GetNormalized() string {
	return p.total.ToString()
}

// JoinNumericPlugin concatenates consecutive numeric nodes (Rust-compatible)
// This is a PathRewritePlugin that operates on the final analysis path
type JoinNumericPlugin struct {
	numericPosId    uint16          // POS ID for "名詞,数詞,*,*,*,*"
	enableNormalize bool            // Whether to enable numeric normalization
	lexiconSet      *dic.LexiconSet // Dictionary access for reading forms
}

// NewJoinNumericPlugin creates a new Rust-compatible JoinNumericPlugin
func NewJoinNumericPlugin() *JoinNumericPlugin {
	return &JoinNumericPlugin{
		enableNormalize: true, // Default value (can be overridden by SetUp)
		numericPosId:    0,    // Will be set during setup
		lexiconSet:      nil,  // Will be set during setup
	}
}

// NewJoinNumericPluginWithNormalize creates a new JoinNumericPlugin with specified normalize setting (deprecated)
func NewJoinNumericPluginWithNormalize(enableNormalize bool) *JoinNumericPlugin {
	return &JoinNumericPlugin{
		enableNormalize: enableNormalize,
		numericPosId:    0,   // Will be set during setup
		lexiconSet:      nil, // Will be set during setup
	}
}

// GetName returns the plugin name
func (p *JoinNumericPlugin) GetName() string {
	return "JoinNumericPlugin"
}

// SetUp initializes the plugin with configuration (implements plugin.PathRewritePlugin)
// This matches Rust Sudachi's set_up method
func (p *JoinNumericPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Extract numeric POS ID from grammar if available
	if grammar != nil {
		// Standard numeric POS: 名詞,数詞,*,*,*,*
		numericPOS := []string{"名詞", "数詞", "*", "*", "*", "*"}
		if posId := grammar.GetPartOfSpeechId(numericPOS); posId != nil {
			p.numericPosId = *posId
		}
	}

	// Configure normalization from settings if provided
	if settings != nil {
		if enableNorm, ok := settings["enableNormalize"].(bool); ok {
			p.enableNormalize = enableNorm
		}
	}

	return nil
}

// SetNumericPosId sets the POS ID for numeric nodes (from grammar)
func (p *JoinNumericPlugin) SetNumericPosId(posId uint16) {
	p.numericPosId = posId
}

// SetLexiconSet sets the lexicon set for dictionary access
func (p *JoinNumericPlugin) SetLexiconSet(lexiconSet *dic.LexiconSet) {
	p.lexiconSet = lexiconSet
}

// Rewrite implements PathRewritePlugin interface - operates on final analysis results
func (p *JoinNumericPlugin) Rewrite(path []*lattice.NodeResult, buffer *input.InputBuffer, lat *lattice.Lattice) ([]*lattice.NodeResult, error) {
	if len(path) == 0 {
		return path, nil
	}

	// Process with Rust-like logic directly on NodeResult slice
	processedResults, err := p.rewriteResults(path, buffer)
	if err != nil {
		return path, err
	}

	return processedResults, nil
}

// rewriteResults processes NodeResult slice (matching Rust's rewrite_gen behavior)
func (p *JoinNumericPlugin) rewriteResults(results []*lattice.NodeResult, buffer *input.InputBuffer) ([]*lattice.NodeResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Call the exact Rust rewrite_gen equivalent
	return p.rewriteGen(results, buffer)
}

// CategoryTypes represents character category information
type CategoryTypes struct {
	IsNumeric      bool
	IsKanjiNumeric bool
}

// rewriteGen implements the exact Rust rewrite_gen logic
func (p *JoinNumericPlugin) rewriteGen(results []*lattice.NodeResult, buffer *input.InputBuffer) ([]*lattice.NodeResult, error) {
	var beginIdx int32 = -1
	commaAsDigit := true
	periodAsDigit := true
	parser := NewNumericParser()
	i := int32(-1)

	for i < int32(len(results))-1 {
		i++
		node := results[i]

		// Get character categories for this node (matching Rust ctypes)
		ctypes := p.getCategoryTypes(node, buffer)
		s := node.NormalizedForm()

		// Check if this is a numeric/comma/period character
		if ctypes.IsNumeric || ctypes.IsKanjiNumeric ||
			(commaAsDigit && s == ",") ||
			(periodAsDigit && s == ".") {

			if beginIdx < 0 {
				parser.Clear()
				beginIdx = i
			}

			// Process each character in the surface
			for _, c := range s {
				if !parser.Append(c) {
					if beginIdx >= 0 {
						if parser.ErrorState() == ErrorComma {
							commaAsDigit = false
							i = beginIdx - 1
						} else if parser.ErrorState() == ErrorPoint {
							periodAsDigit = false
							i = beginIdx - 1
						}
						beginIdx = -1
					}
					break
				}
			}
			continue
		}

		// Get single character for flag reset logic
		var c rune
		if len(s) == 1 {
			runes := []rune(s)
			if len(runes) == 1 {
				c = runes[0]
			} else {
				c = rune(0xFFFF) // char::MAX equivalent
			}
		} else {
			c = rune(0xFFFF) // char::MAX equivalent
		}

		// Process numeric sequence end
		if beginIdx >= 0 {
			// Call parser.Done() only once to avoid side effects
			isDone := parser.Done()

			if isDone {

				var err error
				results, err = p.concat(results, int(beginIdx), int(i), parser)
				if err != nil {
					return nil, err
				}
				i = beginIdx + 1
			} else {
				// Handle error state with previous node
				if i > 0 {
					ss := results[i-1].NormalizedForm()
					if (parser.ErrorState() == ErrorComma && ss == ",") ||
						(parser.ErrorState() == ErrorPoint && ss == ".") {
						var err error
						results, err = p.concat(results, int(beginIdx), int(i-1), parser)
						if err != nil {
							return nil, err
						}
						i = beginIdx + 2
					}
				}
			}
		}

		// Reset state
		beginIdx = -1
		if !commaAsDigit && c != ',' {
			commaAsDigit = true
		}
		if !periodAsDigit && c != '.' {
			periodAsDigit = true
		}
	}

	// Process final numeric sequence
	if beginIdx >= 0 {
		length := len(results)

		// Call parser.Done() only once to avoid side effects
		isDone := parser.Done()

		if isDone {

			var err error
			results, err = p.concat(results, int(beginIdx), length, parser)
			if err != nil {
				return nil, err
			}
		} else {
			// Rust版の動作に合わせて、失敗時は元のノードをそのまま保持
			// カンマ・ピリオドエラーの場合のみ部分的な結合を試行
			if length > 0 {
				ss := results[length-1].NormalizedForm()
				if (parser.ErrorState() == ErrorComma && ss == ",") ||
					(parser.ErrorState() == ErrorPoint && ss == ".") {
					var err error
					results, err = p.concat(results, int(beginIdx), length-1, parser)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return results, nil
}

// concat implements Rust's concat function logic exactly
func (p *JoinNumericPlugin) concat(path []*lattice.NodeResult, begin, end int, parser *NumericParser) ([]*lattice.NodeResult, error) {
	if begin >= end {
		return path, nil
	}

	// Critical fix: Check POS ID first like Rust version
	// Rust logic: if word_info.pos_id() != self.numeric_pos_id { return Ok(path); }
	firstNode := path[begin]

	if p.numericPosId != 0 && p.lexiconSet != nil && !firstNode.Node().IsOOV() {
		if wordInfo, err := p.lexiconSet.GetWordInfo(firstNode.Node().WordId()); err == nil {
			// If the dictionary word doesn't have numeric POS, skip processing entirely
			if wordInfo.PosId != p.numericPosId {
				return path, nil
			}
		}
	}

	if p.enableNormalize {
		normalizedForm := parser.GetNormalized()

		// Get dictionary normalized form like Rust version: word_info.normalized_form()
		var dictionaryNormalizedForm string
		if p.lexiconSet != nil && !firstNode.Node().IsOOV() {
			if wordInfo, err := p.lexiconSet.GetWordInfo(firstNode.Node().WordId()); err == nil {
				dictionaryNormalizedForm = wordInfo.NormalizedForm
			} else {
				// Fallback to NodeResult normalized form if dictionary access fails
				dictionaryNormalizedForm = path[begin].NormalizedForm()
			}
		} else {
			// Fallback to NodeResult normalized form if no lexicon access
			dictionaryNormalizedForm = path[begin].NormalizedForm()
		}

		// Rust behavior: concatenate if multiple nodes OR if normalized form differs from dictionary
		if end-begin > 1 || normalizedForm != dictionaryNormalizedForm {
			return p.concatNodes(path, begin, end, &normalizedForm)
		}
		// If single node and normalized form is same as dictionary, don't change (matching Rust logic)
		return path, nil
	}

	// Without normalization, only concatenate multiple nodes
	if end-begin > 1 {
		return p.concatNodes(path, begin, end, nil)
	}

	return path, nil
}

// getCategoryTypes gets category information for a node (matching Rust ctypes)
func (p *JoinNumericPlugin) getCategoryTypes(node *lattice.NodeResult, buffer *input.InputBuffer) CategoryTypes {
	result := CategoryTypes{}

	// Get node range
	nodeStart := int(node.Node().Begin())
	nodeEnd := int(node.Node().End())

	// Use CategoryOfRange to get common categories across the range (matching Rust cat_of_range)
	categories := buffer.CategoryOfRange(nodeStart, nodeEnd)

	// Check if the common categories include numeric types (matching Rust intersects logic)
	result.IsNumeric = categories.IsNumeric()
	result.IsKanjiNumeric = categories.IsKanjiNumeric()

	return result
}

// concatNodes implements Rust's concat_nodes exactly
func (p *JoinNumericPlugin) concatNodes(path []*lattice.NodeResult, begin, end int, normalizedForm *string) ([]*lattice.NodeResult, error) {
	if begin >= end {
		return nil, fmt.Errorf("invalid range: begin %d >= end %d", begin, end)
	}

	// Build concatenated forms (matching Rust's concat_nodes)
	var surfaceBuilder strings.Builder
	var readingBuilder strings.Builder
	var dictionaryBuilder strings.Builder
	var headWordLength uint16

	for i := begin; i < end; i++ {
		node := path[i]
		surface := node.Surface()
		surfaceBuilder.WriteString(surface)

		// Get reading form from NodeResult (matching Rust's data.reading_form)
		// This matches Rust: reading_form.push_str(&data.reading_form);
		reading := node.Reading()
		readingBuilder.WriteString(reading)

		// Get dictionary form from NodeResult (matching Rust's data.dictionary_form)
		dictionaryForm := node.DictionaryForm()
		dictionaryBuilder.WriteString(dictionaryForm)

		// Head word length - get from dictionary like Rust version
		if p.lexiconSet != nil && !node.Node().IsOOV() {
			if wordInfo, err := p.lexiconSet.GetWordInfo(node.Node().WordId()); err == nil {
				headWordLength += wordInfo.HeadWordLength
			} else {
				headWordLength += uint16(len(surface))
			}
		} else {
			headWordLength += uint16(len(surface))
		}
	}

	concatenatedSurface := surfaceBuilder.String()
	concatenatedReading := readingBuilder.String()
	concatenatedDictionary := dictionaryBuilder.String()

	// Handle normalized form (matching Rust logic)
	var finalNormalizedForm string
	if normalizedForm != nil {
		// Use provided normalized form when normalization is enabled
		finalNormalizedForm = *normalizedForm
	} else {
		// Default to surface form when no normalization provided
		finalNormalizedForm = concatenatedSurface
	}

	// Create new node (matching Rust's Node::new parameters)
	concatenatedNode := lattice.NewNode(
		path[begin].Node().Begin(),
		path[end-1].Node().End(),
		65535,       // u16::MAX for left_id
		65535,       // u16::MAX for right_id
		32767,       // i16::MAX for cost
		dic.Invalid, // WordId::INVALID
	)

	// Create new result (matching Rust's ResultNode::new)
	numericPOS := []string{"名詞", "数詞", "*", "*", "*", "*"}

	concatenatedResult := lattice.NewNodeResult(
		concatenatedNode,
		concatenatedSurface,
		numericPOS,
		[]string{}, // No additional features
		finalNormalizedForm,
		concatenatedDictionary,
		concatenatedReading,
	)

	// Replace the range with the concatenated node (matching Rust's path.drain)
	result := make([]*lattice.NodeResult, 0, len(path)-(end-begin)+1)
	result = append(result, path[:begin]...)
	result = append(result, concatenatedResult)
	result = append(result, path[end:]...)

	return result, nil
}

// CreateInputTextPlugin creates an input text plugin (not supported by JoinNumeric plugin)
func (p *JoinNumericPlugin) CreateInputTextPlugin(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.InputTextPlugin, error) {
	return nil, fmt.Errorf("JoinNumeric plugin does not support input text plugins")
}

// CreateOOVProvider creates an OOV provider plugin (not supported by JoinNumeric plugin)
func (p *JoinNumericPlugin) CreateOOVProvider(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.OOVProviderPlugin, error) {
	return nil, fmt.Errorf("JoinNumeric plugin does not support OOV provider plugins")
}

// CreatePathRewriter creates a JoinNumeric path rewrite plugin instance
func (p *JoinNumericPlugin) CreatePathRewriter(settings map[string]any, resourceDir string, systemDict *dic.SystemDictionary) (plugin.PathRewritePlugin, error) {
	joinNumericPlugin := NewJoinNumericPlugin()

	// Set up the plugin with configuration and set LexiconSet from SystemDictionary
	err := joinNumericPlugin.SetUp(settings, resourceDir, systemDict.Grammar())
	if err != nil {
		return nil, fmt.Errorf("failed to set up JoinNumeric plugin: %w", err)
	}

	// Set LexiconSet from SystemDictionary to enable POS checking
	joinNumericPlugin.SetLexiconSet(systemDict.LexiconSet())

	return joinNumericPlugin, nil
}

// GetSupportedTypes returns the plugin types this factory supports
func (p *JoinNumericPlugin) GetSupportedTypes() []plugin.PluginType {
	return []plugin.PluginType{plugin.PluginTypePathRewrite}
}
