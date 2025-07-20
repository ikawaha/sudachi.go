package lattice

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// InputBufferInterface defines the interface for input buffer operations needed by Split
type InputBufferInterface interface {
	CharToByteIndex(charIdx int) (int, error)
	ByteToCharIndex(byteIdx int) (int, error)
	// OrigSlice returns the original text slice for the given byte range
	// This matches Rust's InputBuffer.orig_slice(bytes_range) functionality
	OrigSlice(modifiedRange input.Range) string
}

// NodeIdx represents an index to a node in the lattice
type NodeIdx struct {
	pos   uint16 // Position in lattice
	idx   uint16 // Index within position
	valid bool   // Whether this index is valid
}

// NewNodeIdx creates a new valid node index
func NewNodeIdx(pos, idx uint16) NodeIdx {
	return NodeIdx{
		pos:   pos,
		idx:   idx,
		valid: true,
	}
}

// EmptyNodeIdx creates an invalid node index
func EmptyNodeIdx() NodeIdx {
	return NodeIdx{
		valid: false,
	}
}

// IsValid returns true if this node index is valid
func (ni NodeIdx) IsValid() bool {
	return ni.valid
}

// Position returns the position in the lattice
func (ni NodeIdx) Position() uint16 {
	return ni.pos
}

// Index returns the index within the position
func (ni NodeIdx) Index() uint16 {
	return ni.idx
}

// Node represents a morpheme candidate in the lattice
type Node struct {
	begin   uint16     // Start position (characters)
	end     uint16     // End position (characters)
	leftId  uint16     // Left connection ID
	rightId uint16     // Right connection ID
	cost    int16      // Word cost
	wordId  dic.WordId // Dictionary word ID
}

// NewNode creates a new node
func NewNode(begin, end, leftId, rightId uint16, cost int16, wordId dic.WordId) *Node {
	return &Node{
		begin:   begin,
		end:     end,
		leftId:  leftId,
		rightId: rightId,
		cost:    cost,
		wordId:  wordId,
	}
}

// Begin returns the start position in characters
func (n *Node) Begin() uint16 {
	return n.begin
}

// End returns the end position in characters
func (n *Node) End() uint16 {
	return n.end
}

// Length returns the character length of this node
func (n *Node) Length() uint16 {
	return n.end - n.begin
}

// LeftId returns the left connection ID
func (n *Node) LeftId() uint16 {
	return n.leftId
}

// RightId returns the right connection ID
func (n *Node) RightId() uint16 {
	return n.rightId
}

// Cost returns the word cost
func (n *Node) Cost() int16 {
	return n.cost
}

// WordId returns the dictionary word ID
func (n *Node) WordId() dic.WordId {
	return n.wordId
}

// IsOOV returns true if this is an out-of-vocabulary word
func (n *Node) IsOOV() bool {
	return n.wordId.IsOOV()
}

// BytesRange returns the byte range for this node in modified text
// This requires the InputBuffer to convert character positions to byte positions
// Matches Rust implementation: ResultNode.bytes_range()
func (n *Node) BytesRange(inputBuffer InputBufferInterface) (input.Range, error) {
	beginByte, err := inputBuffer.CharToByteIndex(int(n.begin))
	if err != nil {
		return input.Range{}, err
	}

	endByte, err := inputBuffer.CharToByteIndex(int(n.end))
	if err != nil {
		return input.Range{}, err
	}

	return input.Range{Start: beginByte, End: endByte}, nil
}

// String returns a string representation of the node
func (n *Node) String() string {
	return fmt.Sprintf("Node{begin=%d, end=%d, leftId=%d, rightId=%d, cost=%d, wordId=%s}",
		n.begin, n.end, n.leftId, n.rightId, n.cost, n.wordId.String())
}

// VNode represents an optimized node for Viterbi search
type VNode struct {
	totalCost int32   // Cumulative path cost
	rightId   uint16  // Right connection ID
	prevIdx   NodeIdx // Previous node index for path reconstruction
}

// NewVNode creates a new Viterbi node
func NewVNode(totalCost int32, rightId uint16, prevIdx NodeIdx) *VNode {
	return &VNode{
		totalCost: totalCost,
		rightId:   rightId,
		prevIdx:   prevIdx,
	}
}

// TotalCost returns the cumulative path cost
func (vn *VNode) TotalCost() int32 {
	return vn.totalCost
}

// RightId returns the right connection ID
func (vn *VNode) RightId() uint16 {
	return vn.rightId
}

// PrevIdx returns the previous node index
func (vn *VNode) PrevIdx() NodeIdx {
	return vn.prevIdx
}

// IsConnectedToBOS returns true if this node has a valid path from beginning of sentence
func (vn *VNode) IsConnectedToBOS() bool {
	return vn.totalCost != int32(^uint32(0)>>1) // Not i32::MAX
}

// String returns a string representation of the VNode
func (vn *VNode) String() string {
	return fmt.Sprintf("VNode{totalCost=%d, rightId=%d, prevIdx=%v}",
		vn.totalCost, vn.rightId, vn.prevIdx)
}

// LexiconEntry represents an entry from dictionary lookup
type LexiconEntry struct {
	WordId dic.WordId
	End    int
}

// NewLexiconEntry creates a new lexicon entry
func NewLexiconEntry(wordId dic.WordId, end int) *LexiconEntry {
	return &LexiconEntry{
		WordId: wordId,
		End:    end,
	}
}

const (
	MaxInt32 = int32(^uint32(0) >> 1) // i32::MAX equivalent
)

// BOS/EOS constants - special dictionary ID 15, word IDs 0 and 1
var (
	BOSWordID = dic.FromRaw(uint32(15)<<28 | 0) // BOS at dic 15, word 0
	EOSWordID = dic.FromRaw(uint32(15)<<28 | 1) // EOS at dic 15, word 1
)

// BOSNode creates a beginning-of-sentence node
func BOSNode() *Node {
	return &Node{
		begin:   0,
		end:     0,
		leftId:  0,
		rightId: 0,
		cost:    0,
		wordId:  BOSWordID,
	}
}

// EOSNode creates an end-of-sentence node
func EOSNode(position uint16) *Node {
	return &Node{
		begin:   position,
		end:     position,
		leftId:  0,
		rightId: 0,
		cost:    0,
		wordId:  EOSWordID,
	}
}

// BOSVNode creates a beginning-of-sentence VNode
func BOSVNode() *VNode {
	return &VNode{
		totalCost: 0,
		rightId:   0,
		prevIdx:   EmptyNodeIdx(),
	}
}
