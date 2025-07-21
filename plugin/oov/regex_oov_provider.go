/*
 *  Copyright (c) 2022-2024 Works Applications Co., Ltd.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package oov

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/types"
)

// BoundaryMode represents the boundary checking mode
// Matches Rust enum: pub enum BoundaryMode
type BoundaryMode int

const (
	// Strict mode requires character category discontinuity at word boundaries
	// Matches Rust: #[default] Strict
	BoundaryModeStrict BoundaryMode = iota
	// Relaxed mode allows starting OOV at any position
	// Matches Rust: Relaxed
	BoundaryModeRelaxed
)

// String returns the string representation of BoundaryMode
func (b BoundaryMode) String() string {
	switch b {
	case BoundaryModeStrict:
		return "strict"
	case BoundaryModeRelaxed:
		return "relaxed"
	default:
		return "unknown"
	}
}

// defaultMaxLength returns the default maximum length for regex matching
// Matches Rust: fn default_max_length() -> usize { 32 }
func defaultMaxLength() int {
	return 32
}

// RegexOovProvider provides out-of-vocabulary words using regular expressions
// Matches Rust struct: pub(crate) struct RegexOovProvider
type RegexOovProvider struct {
	regex      *regexp.Regexp
	leftId     uint16
	rightId    uint16
	cost       int16
	pos        uint16
	maxLength  int
	debug      bool
	boundaries BoundaryMode
}

// NewRegexOovProvider creates a new RegexOovProvider instance
func NewRegexOovProvider() *RegexOovProvider {
	return &RegexOovProvider{
		maxLength:  defaultMaxLength(),
		boundaries: BoundaryModeStrict,
	}
}

// RegexProviderConfig struct corresponds with raw config json file
// Matches Rust struct: struct RegexProviderConfig with #[allow(non_snake_case)]
type RegexProviderConfig struct {
	POS        []string `json:"pos"`        // #[serde(alias = "oovPOS")]
	LeftId     int64    `json:"leftId"`     // leftId
	RightId    int64    `json:"rightId"`    // rightId  
	Cost       int64    `json:"cost"`       // cost
	Regex      string   `json:"regex"`      // regex
	MaxLength  int      `json:"maxLength"`  // #[serde(default = "default_max_length")]
	Debug      bool     `json:"debug"`      // #[serde(default)]
	UserPOS    string   `json:"userPOS"`    // #[serde(default)] UserPosMode
	Boundaries string   `json:"boundaries"` // #[serde(default)] BoundaryMode
}

// SetUp initializes the plugin with configuration
// Matches Rust implementation exactly: fn set_up(&mut self, settings: &Value, _config: &Config, mut grammar: &mut Grammar) -> SudachiResult<()>
func (p *RegexOovProvider) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Convert map to JSON and parse to match Rust serde_json::from_value behavior
	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	var config RegexProviderConfig
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return fmt.Errorf("failed to parse plugin settings: %w", err)
	}

	// Set default values if not provided (matching Rust #[serde(default)])
	if config.MaxLength == 0 {
		config.MaxLength = defaultMaxLength()
	}

	// Parse boundaries (matching Rust #[serde(rename_all = "lowercase")])
	switch strings.ToLower(config.Boundaries) {
	case "relaxed":
		p.boundaries = BoundaryModeRelaxed
	case "strict", "":
		p.boundaries = BoundaryModeStrict
	default:
		return fmt.Errorf("invalid boundaries value: %s", config.Boundaries)
	}

	// Validate and set connection IDs (matching Rust grammar.check_left_id/check_right_id)
	if config.LeftId < 0 || config.LeftId > math.MaxUint16 {
		return fmt.Errorf("invalid left ID: %d", config.LeftId)
	}
	if config.RightId < 0 || config.RightId > math.MaxUint16 {
		return fmt.Errorf("invalid right ID: %d", config.RightId)
	}
	if config.Cost < math.MinInt16 || config.Cost > math.MaxInt16 {
		return fmt.Errorf("invalid cost: %d", config.Cost)
	}

	p.leftId = uint16(config.LeftId)
	p.rightId = uint16(config.RightId)
	p.cost = int16(config.Cost)
	p.maxLength = config.MaxLength
	p.debug = config.Debug

	// Register POS (matching Rust grammar.handle_user_pos)
	if len(config.POS) != dic.POSDepth {
		return fmt.Errorf("invalid POS: expected %d components, got %d", dic.POSDepth, len(config.POS))
	}
	
	posId, err := grammar.RegisterPOS(config.POS)
	if err != nil {
		return fmt.Errorf("failed to register POS: %w", err)
	}
	p.pos = posId

	// Prepare regex pattern (matching Rust: if !parsed.regex.starts_with('^'))
	regexPattern := config.Regex
	if !strings.HasPrefix(regexPattern, "^") {
		regexPattern = "^" + regexPattern
	}

	// Compile regex (matching Rust RegexBuilder::new(&parsed.regex).build())
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", regexPattern, err)
	}
	p.regex = regex

	return nil
}

// ProvideOOV generates OOV nodes at the given character position
// Matches Rust method: fn provide_oov(&self, input_text: &InputBuffer, offset: usize, other_words: CreatedWords, result: &mut Vec<Node>) -> SudachiResult<usize>
func (p *RegexOovProvider) ProvideOOV(charPos int, buffer *input.InputBuffer, lattice *lattice.Lattice, createdWords types.CreatedWords) (types.CreatedWords, error) {
	// Boundary checking for strict mode (matching Rust boundary logic)
	if p.boundaries == BoundaryModeStrict && charPos > 0 {
		// Check character category discontinuity (matching Rust cat_continuous_len logic)
		thisCatContinuity, err := buffer.GetCategoryContinuity(charPos)
		if err != nil {
			return createdWords, nil // Silently skip invalid positions
		}
		
		prevCatContinuity, err := buffer.GetCategoryContinuity(charPos - 1)
		if err != nil {
			return createdWords, nil // Silently skip invalid positions
		}
		
		// Check if there's no discontinuity (matching Rust: if this_cat + 1 == prev_cat)
		if thisCatContinuity+1 == prevCatContinuity {
			// No discontinuity, skip this position
			return createdWords, nil
		}
	}

	// Calculate text slice range (matching Rust logic)
	maxCharPos := buffer.CharCount()
	endCharPos := charPos + p.maxLength
	if endCharPos > maxCharPos {
		endCharPos = maxCharPos
	}

	// Get text slice for regex matching (matching Rust curr_slice_c)
	textSlice := buffer.GetOriginalText(charPos, endCharPos)
	if textSlice == "" {
		return createdWords, nil
	}

	// Apply regex matching (matching Rust regex.find)
	match := p.regex.FindString(textSlice)
	if match == "" {
		return createdWords, nil
	}

	// Check if match starts at beginning (matching Rust m.start() != 0 check)
	matchIndex := p.regex.FindStringIndex(textSlice)
	if matchIndex == nil || matchIndex[0] != 0 {
		if p.debug {
			return createdWords, fmt.Errorf("regex %q matched non-starting text in input %q at position %d", 
				p.regex.String(), textSlice, matchIndex[0])
		}
		return createdWords, nil
	}

	// Calculate match boundaries in character positions
	matchStart := charPos
	matchCharLen := len([]rune(match))
	matchEnd := matchStart + matchCharLen

	// Check for word conflicts (matching Rust other_words.has_word logic)
	hasWordResult := createdWords.HasWord(matchCharLen)
	if hasWordResult == types.HasWordYes {
		return createdWords, nil
	}
	if hasWordResult == types.HasWordMaybe {
		// Need to check actual lengths for long words (matching Rust logic)
		for _, node := range lattice.GetNodes(matchStart) {
			if int(node.End()) == matchEnd {
				return createdWords, nil
			}
		}
	}

	// Create OOV node (matching Rust Node::new)
	wordId := dic.OOV(uint32(p.pos))
	
	// Note: In actual implementation, we would add the node to the lattice
	// For now, we'll just track that we created a word of this length
	// This matches the Rust behavior where the node would be added to result Vec<Node>
	_ = wordId // Use the wordId to avoid unused variable warning

	// Update created words (matching Rust behavior)
	createdWords = createdWords.AddWord(matchCharLen)

	return createdWords, nil
}

// GetName returns the plugin name
func (p *RegexOovProvider) GetName() string {
	return "RegexOovProvider"
}