package lattice

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// TestNode_StructSize tests that Node struct size is reasonable (matching Rust's lesser_than_32b test)
func TestNode_StructSize(t *testing.T) {
	// Rust test: assert_le!(core::mem::size_of::<Node>(), 32);
	nodeSize := unsafe.Sizeof(Node{})
	if nodeSize > 32 {
		t.Errorf("Node struct size %d bytes > 32 bytes (Rust compatibility)", nodeSize)
	}
	t.Logf("Node struct size: %d bytes (≤ 32 bytes, Rust compatible)", nodeSize)
}

// TestNodeIdx_Creation tests NodeIdx creation functionality (Rust: NodeIdx::new and NodeIdx::empty)
func TestNodeIdx_Creation(t *testing.T) {
	tests := []struct {
		name     string
		pos      uint16
		idx      uint16
		expected NodeIdx
	}{
		{"Basic creation", 5, 3, NodeIdx{pos: 5, idx: 3, valid: true}},
		{"Zero values", 0, 0, NodeIdx{pos: 0, idx: 0, valid: true}},
		{"Max values", 65535, 65535, NodeIdx{pos: 65535, idx: 65535, valid: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeIdx := NewNodeIdx(tt.pos, tt.idx)

			if nodeIdx.Position() != tt.pos {
				t.Errorf("NewNodeIdx(%d, %d).Position() = %d, want %d", tt.pos, tt.idx, nodeIdx.Position(), tt.pos)
			}

			if nodeIdx.Index() != tt.idx {
				t.Errorf("NewNodeIdx(%d, %d).Index() = %d, want %d", tt.pos, tt.idx, nodeIdx.Index(), tt.idx)
			}

			if !nodeIdx.IsValid() {
				t.Errorf("NewNodeIdx(%d, %d).IsValid() = false, want true", tt.pos, tt.idx)
			}
		})
	}
}

// TestNodeIdx_Empty tests empty NodeIdx creation (Rust: NodeIdx::empty)
func TestNodeIdx_Empty(t *testing.T) {
	emptyIdx := EmptyNodeIdx()

	if emptyIdx.IsValid() {
		t.Error("EmptyNodeIdx().IsValid() = true, want false")
	}

	// Empty index should have some deterministic values, but exact values are implementation detail
	t.Logf("EmptyNodeIdx: pos=%d, idx=%d, valid=%t", emptyIdx.Position(), emptyIdx.Index(), emptyIdx.IsValid())
}

// TestNode_Creation tests Node creation and basic methods
func TestNode_Creation(t *testing.T) {
	wordId := dic.FromRaw(12345)
	node := NewNode(1, 5, 100, 200, 300, wordId)

	if node.Begin() != 1 {
		t.Errorf("Node.Begin() = %d, want 1", node.Begin())
	}

	if node.End() != 5 {
		t.Errorf("Node.End() = %d, want 5", node.End())
	}

	if node.Length() != 4 {
		t.Errorf("Node.Length() = %d, want 4", node.Length())
	}

	if node.LeftId() != 100 {
		t.Errorf("Node.LeftId() = %d, want 100", node.LeftId())
	}

	if node.RightId() != 200 {
		t.Errorf("Node.RightId() = %d, want 200", node.RightId())
	}

	if node.Cost() != 300 {
		t.Errorf("Node.Cost() = %d, want 300", node.Cost())
	}

	if node.WordId() != wordId {
		t.Errorf("Node.WordId() = %v, want %v", node.WordId(), wordId)
	}
}

// TestNode_BOSCreation tests BOS node creation (Rust: connect_bos pattern)
func TestNode_BOSCreation(t *testing.T) {
	bosNode := BOSNode()

	if bosNode.Begin() != 0 {
		t.Errorf("BOSNode().Begin() = %d, want 0", bosNode.Begin())
	}

	if bosNode.End() != 0 {
		t.Errorf("BOSNode().End() = %d, want 0", bosNode.End())
	}

	if bosNode.LeftId() != 0 {
		t.Errorf("BOSNode().LeftId() = %d, want 0", bosNode.LeftId())
	}

	if bosNode.RightId() != 0 {
		t.Errorf("BOSNode().RightId() = %d, want 0", bosNode.RightId())
	}

	if bosNode.Cost() != 0 {
		t.Errorf("BOSNode().Cost() = %d, want 0", bosNode.Cost())
	}

	if bosNode.WordId() != BOSWordID {
		t.Errorf("BOSNode().WordId() = %v, want %v", bosNode.WordId(), BOSWordID)
	}

	// BOS should be special but not OOV in the regular sense
	if !bosNode.WordId().IsOOV() {
		t.Error("BOSNode().WordId().IsOOV() = false, want true (BOS uses special dictionary)")
	}
}

// TestNode_EOSCreation tests EOS node creation (Rust: connect_eos pattern)
func TestNode_EOSCreation(t *testing.T) {
	position := uint16(10)
	eosNode := EOSNode(position)

	if eosNode.Begin() != position {
		t.Errorf("EOSNode(%d).Begin() = %d, want %d", position, eosNode.Begin(), position)
	}

	if eosNode.End() != position {
		t.Errorf("EOSNode(%d).End() = %d, want %d", position, eosNode.End(), position)
	}

	if eosNode.LeftId() != 0 {
		t.Errorf("EOSNode(%d).LeftId() = %d, want 0", position, eosNode.LeftId())
	}

	if eosNode.RightId() != 0 {
		t.Errorf("EOSNode(%d).RightId() = %d, want 0", position, eosNode.RightId())
	}

	if eosNode.Cost() != 0 {
		t.Errorf("EOSNode(%d).Cost() = %d, want 0", position, eosNode.Cost())
	}

	if eosNode.WordId() != EOSWordID {
		t.Errorf("EOSNode(%d).WordId() = %v, want %v", position, eosNode.WordId(), EOSWordID)
	}

	// EOS should be special but not OOV in the regular sense
	if !eosNode.WordId().IsOOV() {
		t.Error("EOSNode().WordId().IsOOV() = false, want true (EOS uses special dictionary)")
	}
}

// TestVNode_Creation tests VNode creation and methods (Rust: VNode::new)
func TestVNode_Creation(t *testing.T) {
	prevIdx := NewNodeIdx(3, 1)
	vnode := NewVNode(1500, 250, prevIdx)

	if vnode.TotalCost() != 1500 {
		t.Errorf("VNode.TotalCost() = %d, want 1500", vnode.TotalCost())
	}

	if vnode.RightId() != 250 {
		t.Errorf("VNode.RightId() = %d, want 250", vnode.RightId())
	}

	if vnode.PrevIdx() != prevIdx {
		t.Errorf("VNode.PrevIdx() = %v, want %v", vnode.PrevIdx(), prevIdx)
	}
}

// TestVNode_BOSCreation tests BOS VNode creation (Rust: VNode::new(0, 0) pattern)
func TestVNode_BOSCreation(t *testing.T) {
	bosVNode := BOSVNode()

	if bosVNode.TotalCost() != 0 {
		t.Errorf("BOSVNode().TotalCost() = %d, want 0", bosVNode.TotalCost())
	}

	if bosVNode.RightId() != 0 {
		t.Errorf("BOSVNode().RightId() = %d, want 0", bosVNode.RightId())
	}

	if bosVNode.PrevIdx().IsValid() {
		t.Error("BOSVNode().PrevIdx().IsValid() = true, want false")
	}

	if !bosVNode.IsConnectedToBOS() {
		t.Error("BOSVNode().IsConnectedToBOS() = false, want true")
	}
}

// TestVNode_IsConnectedToBOS tests connection to BOS detection (Rust: is_connected_to_bos logic)
func TestVNode_IsConnectedToBOS(t *testing.T) {
	tests := []struct {
		name      string
		totalCost int32
		expected  bool
	}{
		{"Connected (zero cost)", 0, true},
		{"Connected (positive cost)", 1000, true},
		{"Connected (negative cost)", -500, true},
		{"Disconnected (MAX cost)", MaxInt32, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vnode := NewVNode(tt.totalCost, 0, EmptyNodeIdx())

			if vnode.IsConnectedToBOS() != tt.expected {
				t.Errorf("VNode{totalCost: %d}.IsConnectedToBOS() = %t, want %t",
					tt.totalCost, vnode.IsConnectedToBOS(), tt.expected)
			}
		})
	}
}

// TestWordId_SpecialNodes tests special word ID constants (Rust: WordId::BOS, WordId::EOS tests)
func TestWordId_SpecialNodes(t *testing.T) {
	// Test BOS WordId
	if BOSWordID.Dic() != 15 {
		t.Errorf("BOSWordID.Dic() = %d, want 15", BOSWordID.Dic())
	}

	if BOSWordID.Word() != 0 {
		t.Errorf("BOSWordID.Word() = %d, want 0", BOSWordID.Word())
	}

	if !BOSWordID.IsOOV() {
		t.Error("BOSWordID.IsOOV() = false, want true")
	}

	// Test EOS WordId
	if EOSWordID.Dic() != 15 {
		t.Errorf("EOSWordID.Dic() = %d, want 15", EOSWordID.Dic())
	}

	if EOSWordID.Word() != 1 {
		t.Errorf("EOSWordID.Word() = %d, want 1", EOSWordID.Word())
	}

	if !EOSWordID.IsOOV() {
		t.Error("EOSWordID.IsOOV() = false, want true")
	}
}

// TestNode_OOVDetection tests OOV node detection (Rust: is_oov logic)
func TestNode_OOVDetection(t *testing.T) {
	tests := []struct {
		name     string
		wordId   dic.WordId
		expected bool
	}{
		{"System dictionary word", dic.FromRaw(0x00001234), false},
		{"User dictionary word", dic.FromRaw(0x10001234), false},
		{"OOV word", dic.FromRaw(0xF0001234), true},
		{"BOS", BOSWordID, true},
		{"EOS", EOSWordID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewNode(0, 1, 0, 0, 0, tt.wordId)

			if node.IsOOV() != tt.expected {
				t.Errorf("Node{wordId: %v}.IsOOV() = %t, want %t",
					tt.wordId, node.IsOOV(), tt.expected)
			}
		})
	}
}

// TestNode_String tests string representation
func TestNode_String(t *testing.T) {
	wordId := dic.FromRaw(12345)
	node := NewNode(1, 5, 100, 200, 300, wordId)

	str := node.String()
	expected := fmt.Sprintf("Node{begin=1, end=5, leftId=100, rightId=200, cost=300, wordId=%s}", wordId.String())

	if str != expected {
		t.Errorf("Node.String() = %q, want %q", str, expected)
	}
}

// TestVNode_String tests VNode string representation
func TestVNode_String(t *testing.T) {
	prevIdx := NewNodeIdx(3, 1)
	vnode := NewVNode(1500, 250, prevIdx)

	str := vnode.String()
	expected := fmt.Sprintf("VNode{totalCost=1500, rightId=250, prevIdx=%v}", prevIdx)

	if str != expected {
		t.Errorf("VNode.String() = %q, want %q", str, expected)
	}
}

// TestLexiconEntry_Creation tests lexicon entry creation
func TestLexiconEntry_Creation(t *testing.T) {
	wordId := dic.FromRaw(98765)
	end := 7

	entry := NewLexiconEntry(wordId, end)

	if entry.WordId != wordId {
		t.Errorf("LexiconEntry.WordId = %v, want %v", entry.WordId, wordId)
	}

	if entry.End != end {
		t.Errorf("LexiconEntry.End = %d, want %d", entry.End, end)
	}
}

// TestConstants tests important constants
func TestConstants(t *testing.T) {
	// Test MaxInt32 constant
	expectedMaxInt32 := int32(^uint32(0) >> 1)
	if MaxInt32 != expectedMaxInt32 {
		t.Errorf("MaxInt32 = %d, want %d", MaxInt32, expectedMaxInt32)
	}

	// Test that MaxInt32 is indeed the maximum positive int32
	if MaxInt32 != 2147483647 {
		t.Errorf("MaxInt32 = %d, want 2147483647", MaxInt32)
	}

	t.Logf("MaxInt32 = %d (correct i32::MAX equivalent)", MaxInt32)
}

// MockInputBuffer is a simple mock for testing BytesRange functionality
type MockInputBuffer struct {
	charToByteMap map[int]int
}

func (m *MockInputBuffer) CharToByteIndex(charIdx int) (int, error) {
	if byteIdx, ok := m.charToByteMap[charIdx]; ok {
		return byteIdx, nil
	}
	return -1, fmt.Errorf("char index %d not found", charIdx)
}

func (m *MockInputBuffer) ByteToCharIndex(byteIdx int) (int, error) {
	// Reverse lookup (simplified)
	for charIdx, mappedByteIdx := range m.charToByteMap {
		if mappedByteIdx == byteIdx {
			return charIdx, nil
		}
	}
	return -1, fmt.Errorf("byte index %d not found", byteIdx)
}

func (m *MockInputBuffer) OrigSlice(modifiedRange input.Range) string {
	return "test"
}

// TestNode_BytesRange tests byte range calculation (Rust: bytes_range functionality)
func TestNode_BytesRange(t *testing.T) {
	// Create mock input buffer with known mappings
	mockBuffer := &MockInputBuffer{
		charToByteMap: map[int]int{
			0: 0,
			1: 1,
			2: 4, // Multi-byte character
			3: 7,
			4: 10,
			5: 11,
		},
	}

	node := NewNode(1, 3, 0, 0, 0, dic.FromRaw(123))

	bytesRange, err := node.BytesRange(mockBuffer)
	if err != nil {
		t.Fatalf("Node.BytesRange() failed: %v", err)
	}

	expectedRange := input.Range{Start: 1, End: 7}
	if !reflect.DeepEqual(bytesRange, expectedRange) {
		t.Errorf("Node.BytesRange() = %v, want %v", bytesRange, expectedRange)
	}
}

// TestNode_BytesRange_Error tests error handling in BytesRange
func TestNode_BytesRange_Error(t *testing.T) {
	mockBuffer := &MockInputBuffer{
		charToByteMap: map[int]int{0: 0, 1: 1}, // Missing mappings for 2, 3
	}

	node := NewNode(1, 3, 0, 0, 0, dic.FromRaw(123))

	_, err := node.BytesRange(mockBuffer)
	if err == nil {
		t.Error("Node.BytesRange() should fail with missing char index mapping")
	}
}
