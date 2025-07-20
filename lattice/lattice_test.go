package lattice

import (
	"encoding/binary"
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
)

func TestLattice_NewLattice(t *testing.T) {
	lattice := New()

	if lattice == nil {
		t.Fatal("New() returned nil")
	}

	if lattice.size != 0 {
		t.Errorf("New().size = %d, want 0", lattice.size)
	}

	if lattice.eos != nil {
		t.Errorf("New().eos = %v, want nil", lattice.eos)
	}
}

func TestLattice_Reset(t *testing.T) {
	lattice := New()
	size := 10

	lattice.Reset(size)

	expectedSize := size + 1 // +1 for EOS position
	if lattice.size != expectedSize {
		t.Errorf("Reset(%d): size = %d, want %d", size, lattice.size, expectedSize)
	}

	if len(lattice.ends) != expectedSize {
		t.Errorf("Reset(%d): len(ends) = %d, want %d", size, len(lattice.ends), expectedSize)
	}

	if len(lattice.endsFull) != expectedSize {
		t.Errorf("Reset(%d): len(endsFull) = %d, want %d", size, len(lattice.endsFull), expectedSize)
	}

	if len(lattice.indices) != expectedSize {
		t.Errorf("Reset(%d): len(indices) = %d, want %d", size, len(lattice.indices), expectedSize)
	}

	// Check that BOS was added at position 0
	if len(lattice.ends[0]) != 1 {
		t.Errorf("Reset(%d): BOS not added, len(ends[0]) = %d, want 1", size, len(lattice.ends[0]))
	}

	// Verify initial capacity is 16 (matching Rust implementation)
	if cap(lattice.ends[1]) != 16 {
		t.Errorf("Reset(%d): cap(ends[1]) = %d, want 16 (Rust compatible)", size, cap(lattice.ends[1]))
	}
}

func TestEOSInfo(t *testing.T) {
	prevIdx := NewNodeIdx(5, 3)
	cost := int32(1234)

	eosInfo := &EOSInfo{
		prevIdx: prevIdx,
		cost:    cost,
	}

	if eosInfo.prevIdx.Position() != 5 {
		t.Errorf("EOSInfo.prevIdx.Position() = %d, want 5", eosInfo.prevIdx.Position())
	}

	if eosInfo.prevIdx.Index() != 3 {
		t.Errorf("EOSInfo.prevIdx.Index() = %d, want 3", eosInfo.prevIdx.Index())
	}

	if eosInfo.cost != cost {
		t.Errorf("EOSInfo.cost = %d, want %d", eosInfo.cost, cost)
	}
}

func TestLattice_ConnectEOS(t *testing.T) {
	lattice := New()
	lattice.Reset(2) // Small lattice

	// Test that ConnectEOS fails when no valid path exists (expected behavior)
	connMatrix := &dic.ConnectionMatrix{}
	err := lattice.ConnectEOS(connMatrix)
	if err == nil {
		t.Error("ConnectEOS() should fail when no valid path exists")
	}

	// Test that EOS remains nil when connection fails
	if lattice.eos != nil {
		t.Error("ConnectEOS() failed but eos is not nil")
	}
}

func TestLattice_GetBestPath_RustCompatibility(t *testing.T) {
	lattice := New()
	lattice.Reset(2)

	// Test GetBestPath without EOS connection (should fail)
	path, err := lattice.GetBestPath()
	if err == nil {
		t.Error("GetBestPath() should fail when EOS not connected")
	}
	if path != nil {
		t.Error("GetBestPath() should return nil path when EOS not connected")
	}
}

func TestLattice_HasPreviousNode(t *testing.T) {
	lattice := New()
	lattice.Reset(5)

	// Position 0 should have BOS node
	if !lattice.HasPreviousNode(0) {
		t.Error("HasPreviousNode(0) = false, want true (BOS should be present)")
	}

	// Other positions should be empty initially
	for i := 1; i < 5; i++ {
		if lattice.HasPreviousNode(i) {
			t.Errorf("HasPreviousNode(%d) = true, want false (should be empty initially)", i)
		}
	}

	// Out of bounds should return false
	if lattice.HasPreviousNode(10) {
		t.Error("HasPreviousNode(10) = true, want false (out of bounds)")
	}
}

func TestLattice_Insert_RustCompatibility(t *testing.T) {
	lattice := New()
	lattice.Reset(5)

	// Create a minimal connection matrix with zero costs
	data := make([]byte, 4)                     // 1x1 matrix with 2 bytes per entry (int16)
	binary.LittleEndian.PutUint16(data[0:2], 0) // cost = 0
	binary.LittleEndian.PutUint16(data[2:4], 0) // cost = 0

	connMatrix, err := dic.NewConnectionMatrix(data, 0, 1, 1)
	if err != nil {
		t.Fatalf("Failed to create connection matrix: %v", err)
	}

	// Test node insertion (matching Rust's insert behavior)
	node := NewNode(1, 3, 0, 0, 500, dic.FromRaw(789)) // Use leftId=0, rightId=0 for 1x1 matrix
	err = lattice.Insert(node, connMatrix)
	if err != nil {
		t.Fatalf("Insert() failed: %v", err)
	}

	// Verify node was inserted at correct end position
	endPos := int(node.End())
	if len(lattice.endsFull[endPos]) != 1 {
		t.Errorf("Insert(): len(endsFull[%d]) = %d, want 1", endPos, len(lattice.endsFull[endPos]))
	}

	if len(lattice.ends[endPos]) != 1 {
		t.Errorf("Insert(): len(ends[%d]) = %d, want 1", endPos, len(lattice.ends[endPos]))
	}

	if len(lattice.indices[endPos]) != 1 {
		t.Errorf("Insert(): len(indices[%d]) = %d, want 1", endPos, len(lattice.indices[endPos]))
	}

	// Verify inserted node
	insertedNode := lattice.endsFull[endPos][0]
	if insertedNode.Begin() != node.Begin() || insertedNode.End() != node.End() {
		t.Errorf("Insert(): inserted node range [%d,%d], want [%d,%d]",
			insertedNode.Begin(), insertedNode.End(), node.Begin(), node.End())
	}
}

func TestLattice_ResetVec_RustCompatibility(t *testing.T) {
	lattice := New()

	// Test reset behavior with different sizes
	sizes := []int{5, 10, 3, 15}

	for _, size := range sizes {
		lattice.Reset(size)

		// Verify each inner slice has capacity 16 (matching Rust)
		for i := 1; i < lattice.size; i++ { // Skip position 0 (BOS)
			if cap(lattice.ends[i]) != 16 {
				t.Errorf("Reset(%d): cap(ends[%d]) = %d, want 16 (Rust compatible)",
					size, i, cap(lattice.ends[i]))
			}
			if cap(lattice.endsFull[i]) != 16 {
				t.Errorf("Reset(%d): cap(endsFull[%d]) = %d, want 16 (Rust compatible)",
					size, i, cap(lattice.endsFull[i]))
			}
			if cap(lattice.indices[i]) != 16 {
				t.Errorf("Reset(%d): cap(indices[%d]) = %d, want 16 (Rust compatible)",
					size, i, cap(lattice.indices[i]))
			}
		}
	}
}

func TestLattice_BOS_RustCompatibility(t *testing.T) {
	lattice := New()
	lattice.Reset(5)

	// Verify BOS was added correctly (matching Rust's connect_bos)
	if len(lattice.ends[0]) != 1 {
		t.Errorf("BOS: len(ends[0]) = %d, want 1", len(lattice.ends[0]))
	}

	// Rust only adds to ends[0], not endsFull[0] or indices[0]
	// But our Go implementation needs to track nodes for GetBestPath
	bosVNode := lattice.ends[0][0]
	if bosVNode.TotalCost() != 0 {
		t.Errorf("BOS: TotalCost() = %d, want 0", bosVNode.TotalCost())
	}

	if bosVNode.RightId() != 0 {
		t.Errorf("BOS: RightId() = %d, want 0", bosVNode.RightId())
	}

	if bosVNode.PrevIdx().IsValid() {
		t.Error("BOS: PrevIdx().IsValid() = true, want false")
	}
}
