package lattice

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
)

// Morpheme represents a morpheme interface
type Morpheme interface {
	// Surface returns the surface form
	Surface() string

	// Begin returns the start position in characters
	Begin() uint16

	// End returns the end position in characters
	End() uint16

	// Length returns the character length
	Length() uint16

	// POS returns the part of speech
	POS() []string

	// Features returns additional features
	Features() []string
}

// NodeResult represents a final analysis result
type NodeResult struct {
	node           *Node
	surface        string   // Surface form
	normalizedForm string   // Normalized form (headword)
	dictionaryForm string   // Dictionary form (base form)
	readingForm    string   // Reading form (kana)
	pos            []string // Part of speech
	features       []string // Additional features
}

// NewNodeResult creates a new node result with all fields specified (Rust-compatible single constructor)
func NewNodeResult(node *Node, surface string, pos []string, features []string, normalizedForm string, dictionaryForm string, readingForm string) *NodeResult {
	return &NodeResult{
		node:           node,
		surface:        surface,
		normalizedForm: normalizedForm,
		dictionaryForm: dictionaryForm,
		readingForm:    readingForm,
		pos:            pos,
		features:       features,
	}
}

// Node returns the underlying node
func (nr *NodeResult) Node() *Node {
	return nr.node
}

// Surface returns the surface form
func (nr *NodeResult) Surface() string {
	return nr.surface
}

// NormalizedForm returns the normalized form (dictionary form)
func (nr *NodeResult) NormalizedForm() string {
	return nr.normalizedForm
}

// Begin returns the start position in characters
func (nr *NodeResult) Begin() uint16 {
	return nr.node.Begin()
}

// End returns the end position in characters
func (nr *NodeResult) End() uint16 {
	return nr.node.End()
}

// Length returns the character length
func (nr *NodeResult) Length() uint16 {
	return nr.node.Length()
}

// POS returns the part of speech
func (nr *NodeResult) POS() []string {
	return nr.pos
}

// Features returns additional features
func (nr *NodeResult) Features() []string {
	return nr.features
}

// Reading returns the reading form (kana)
func (nr *NodeResult) Reading() string {
	return nr.readingForm
}

// DictionaryForm returns the dictionary form (base form)
func (nr *NodeResult) DictionaryForm() string {
	return nr.dictionaryForm
}

// IsOOV returns true if this node represents an OOV (out-of-vocabulary) word
func (nr *NodeResult) IsOOV() bool {
	return nr.node.WordId().IsOOV()
}

// DictionaryId returns the dictionary ID (matching Rust's dictionary_id() method)
func (nr *NodeResult) DictionaryId() int {
	wordId := nr.node.WordId()
	if wordId.IsOOV() || wordId.Raw() == dic.Invalid.Raw() {
		return -1 // OOV nodes and INVALID nodes return -1 like Rust version
	}
	return int(wordId.Dic())
}

// String returns a string representation of the result
func (nr *NodeResult) String() string {
	return fmt.Sprintf("NodeResult{surface='%s', pos=%v, begin=%d, end=%d}",
		nr.surface, nr.pos, nr.node.Begin(), nr.node.End())
}

// MorphemeList represents a list of morpheme analysis results
type MorphemeList struct {
	results []*NodeResult
}

// NewMorphemeList creates a new morpheme list
func NewMorphemeList() *MorphemeList {
	return &MorphemeList{
		results: make([]*NodeResult, 0),
	}
}

// Add adds a result to the list
func (ml *MorphemeList) Add(result *NodeResult) {
	ml.results = append(ml.results, result)
}

// Get returns the result at the given index
func (ml *MorphemeList) Get(index int) *NodeResult {
	if index < 0 || index >= len(ml.results) {
		return nil
	}
	return ml.results[index]
}

// Size returns the number of results
func (ml *MorphemeList) Size() int {
	return len(ml.results)
}

// IsEmpty returns true if the list is empty
func (ml *MorphemeList) IsEmpty() bool {
	return len(ml.results) == 0
}

// Clear removes all results
func (ml *MorphemeList) Clear() {
	ml.results = ml.results[:0]
}

// Results returns all results
func (ml *MorphemeList) Results() []*NodeResult {
	return ml.results
}

// String returns a string representation of the morpheme list
func (ml *MorphemeList) String() string {
	if len(ml.results) == 0 {
		return "MorphemeList{empty}"
	}

	result := "MorphemeList{\n"
	for i, morpheme := range ml.results {
		result += fmt.Sprintf("  [%d]: %s\n", i, morpheme.String())
	}
	result += "}"
	return result
}
