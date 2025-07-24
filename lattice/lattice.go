package lattice

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
)

// EOSInfo represents the EOS connection information (matching Rust implementation)
// Rust: Option<(NodeIdx, i32)> where the tuple contains (previous_node_index, total_cost)
type EOSInfo struct {
	prevIdx NodeIdx // Previous node index that EOS connects to
	cost    int32   // Total cost of the path to EOS
}

// Lattice represents the word lattice for Viterbi search
type Lattice struct {
	// Viterbi nodes by end position (optimized for search)
	ends [][]VNode
	// Full nodes by end position (complete information)
	endsFull [][]*Node
	// Node indices for path reconstruction
	indices [][]NodeIdx
	// End of sentence node information (matching Rust implementation exactly)
	// Rust: eos: Option<(NodeIdx, i32)>
	eos *EOSInfo
	// Lattice size (number of character positions)
	size int
}

// New creates a new lattice
func New() *Lattice {
	return &Lattice{}
}

// Reset resets the lattice for a new analysis with the given size
// This is a faithful port of Rust's reset method
func (l *Lattice) Reset(size int) {
	targetSize := size + 1 // +1 for EOS position

	// Rust: Self::reset_vec(&mut self.ends, length + 1);
	l.resetVec(&l.ends, targetSize)
	l.resetVecNodePtr(&l.endsFull, targetSize)
	l.resetVecNodeIdx(&l.indices, targetSize)

	// Rust: self.eos = None;
	l.eos = nil
	// Rust: self.size = length + 1;
	l.size = targetSize
	// Rust: self.connect_bos();
	l.addBOS()
}

// resetVec is a faithful port of Rust's reset_vec function
// Rust: fn reset_vec<T>(data: &mut Vec<Vec<T>>, target: usize)
func (l *Lattice) resetVec(data *[][]VNode, target int) {
	// Rust: for v in data.iter_mut() { v.clear(); }
	for i := range *data {
		(*data)[i] = (*data)[i][:0]
	}
	// Rust: let cur_len = data.len();
	curLen := len(*data)
	// Rust: if cur_len <= target {
	if curLen <= target {
		// Rust: data.reserve(target - cur_len);
		// Rust: for _ in cur_len..target { data.push(Vec::with_capacity(16)) }
		newSlice := make([][]VNode, target)
		copy(newSlice, *data)
		for i := curLen; i < target; i++ {
			newSlice[i] = make([]VNode, 0, 16) // capacity 16 like Rust
		}
		*data = newSlice
	} else {
		*data = (*data)[:target]
	}
}

// resetVecNodePtr resets [][]*Node slice (Go specific due to pointer type)
func (l *Lattice) resetVecNodePtr(data *[][]*Node, target int) {
	for i := range *data {
		(*data)[i] = (*data)[i][:0]
	}
	curLen := len(*data)
	if curLen <= target {
		newSlice := make([][]*Node, target)
		copy(newSlice, *data)
		for i := curLen; i < target; i++ {
			newSlice[i] = make([]*Node, 0, 16) // capacity 16 like Rust
		}
		*data = newSlice
	} else {
		*data = (*data)[:target]
	}
}

// resetVecNodeIdx resets [][]NodeIdx slice
func (l *Lattice) resetVecNodeIdx(data *[][]NodeIdx, target int) {
	for i := range *data {
		(*data)[i] = (*data)[i][:0]
	}
	curLen := len(*data)
	if curLen <= target {
		newSlice := make([][]NodeIdx, target)
		copy(newSlice, *data)
		for i := curLen; i < target; i++ {
			newSlice[i] = make([]NodeIdx, 0, 16) // capacity 16 like Rust
		}
		*data = newSlice
	} else {
		*data = (*data)[:target]
	}
}

// addBOS adds a beginning-of-sentence node at position 0 (matching Rust implementation)
func (l *Lattice) addBOS() {
	// Rust: fn connect_bos(&mut self) {
	//     self.ends[0].push(VNode::new(0, 0));
	// }
	// Note: Rust only adds VNode, but Go needs Node and indices for GetBestPath to work
	bosNode := BOSNode()
	bosVNode := BOSVNode()

	l.ends[0] = append(l.ends[0], *bosVNode)
	l.endsFull[0] = append(l.endsFull[0], bosNode)
	l.indices[0] = append(l.indices[0], EmptyNodeIdx())
}

// Insert inserts a node into the lattice and connects it optimally
func (l *Lattice) Insert(node *Node, connMatrix *dic.ConnectionMatrix) error {
	if node.End() >= uint16(l.size) {
		return fmt.Errorf("node end position out of bounds: end=%d, size=%d", node.End(), l.size)
	}
	// Find optimal connection from previous nodes
	prevIdx, totalCost := l.connectNode(node, connMatrix)

	// Create VNode for Viterbi search
	vnode := NewVNode(totalCost, node.RightId(), prevIdx)

	// Insert into lattice
	endPos := int(node.End())
	l.ends[endPos] = append(l.ends[endPos], *vnode)
	l.endsFull[endPos] = append(l.endsFull[endPos], node)

	// Record previous node index for path reconstruction (matching Rust implementation)
	l.indices[endPos] = append(l.indices[endPos], prevIdx)

	return nil
}

// connectNode finds the optimal connection for a node using Viterbi algorithm
func (l *Lattice) connectNode(rNode *Node, connMatrix *dic.ConnectionMatrix) (NodeIdx, int32) {
	begin := int(rNode.Begin())
	nodeCost := int32(rNode.Cost())

	if begin >= len(l.ends) || len(l.ends[begin]) == 0 {
		return EmptyNodeIdx(), MaxInt32
	}

	minCost := MaxInt32
	var prevIdx NodeIdx

	// Find a minimum cost path from previous nodes
	for i, lVNode := range l.ends[begin] {
		if !lVNode.IsConnectedToBOS() {
			continue
		}

		// Calculate connection cost
		connectCost := int32(connMatrix.Cost(lVNode.RightId(), rNode.LeftId()))
		newCost := lVNode.TotalCost() + connectCost + nodeCost

		if newCost < minCost {
			minCost = newCost
			prevIdx = NewNodeIdx(uint16(begin), uint16(i))
		}
	}

	if minCost == MaxInt32 {
		prevIdx = EmptyNodeIdx()
	}

	return prevIdx, minCost
}

// ConnectEOS connects the end-of-sentence node (matching Rust implementation exactly)
func (l *Lattice) ConnectEOS(connMatrix *dic.ConnectionMatrix) error {
	eosPos := l.size - 1
	eosNode := EOSNode(uint16(eosPos))

	// Find optimal connection to EOS (matching Rust: self.connect_node(&node, conn))
	prevIdx, totalCost := l.connectNode(eosNode, connMatrix)
	if totalCost == MaxInt32 {
		// Rust: Err(SudachiError::EosBosDisconnect)
		return fmt.Errorf("failed to connect EOS: no valid path to end of sentence")
	}

	// Store EOS connection info (matching Rust implementation exactly)
	// Rust: self.eos = Some((idx, cost))
	l.eos = &EOSInfo{
		prevIdx: prevIdx,
		cost:    totalCost,
	}

	return nil
}

// HasPreviousNode returns true if there are nodes ending at the given position
func (l *Lattice) HasPreviousNode(pos int) bool {
	if pos >= len(l.ends) {
		return false
	}
	return len(l.ends[pos]) > 0
}

// GetBestPath returns the optimal path from BOS to EOS (faithful Rust implementation)
// This is a direct port of Rust's fill_top_path algorithm
func (l *Lattice) GetBestPath() ([]*Node, error) {
	// Rust: if self.eos.is_none() { return; }
	if l.eos == nil {
		return nil, fmt.Errorf("EOS not connected: call ConnectEOS before getting best path")
	}

	// Rust: let (mut idx, _) = self.eos.unwrap();
	// Start with the NodeIdx that EOS points to (the last actual morpheme node)
	var indices []NodeIdx
	currentIdx := l.eos.prevIdx
	indices = append(indices, currentIdx)

	// Rust: loop {
	for {
		// Rust: let prev_idx = self.indices[idx.end() as usize][idx.index() as usize];
		prevIdx := l.indices[currentIdx.Position()][currentIdx.Index()]

		// Rust: if prev_idx.end() != 0 {
		if prevIdx.Position() != 0 {
			// Rust: result.push(prev_idx);
			indices = append(indices, prevIdx)
			// Rust: idx = prev_idx;
			currentIdx = prevIdx
		} else {
			// Rust: break; (finish if BOS)
			break
		}
	}

	// Convert indices to nodes (in reverse order to get BOS->EOS)
	// Note: Rust's fill_top_path fills in reverse order, so we reverse here
	var result []*Node
	for i := len(indices) - 1; i >= 0; i-- {
		nodeIdx := indices[i]
		pos := int(nodeIdx.Position())
		idx := int(nodeIdx.Index())

		if pos >= len(l.endsFull) || idx >= len(l.endsFull[pos]) {
			return nil, fmt.Errorf("invalid node in path: pos=%d, idx=%d", pos, idx)
		}

		node := l.endsFull[pos][idx]
		result = append(result, node)
	}

	return result, nil
}

// Size returns the lattice size
func (l *Lattice) Size() int {
	return l.size
}

// GetNodes returns all nodes ending at the given position
func (l *Lattice) GetNodes(pos int) []*Node {
	if pos >= len(l.endsFull) {
		return nil
	}
	return l.endsFull[pos]
}

// DumpWithDetails outputs the lattice in Rust-compatible debug format with full details
// Format: node_id: begin end surface(dict_id, word_id) POS left_id right_id node_cost: connection_costs...
func (l *Lattice) DumpWithDetails(getSurface func(*Node) string, getPOS func(*Node) string, getConnectionCosts func(*Node) []int) {
	fmt.Println("=== Lattice dump:")

	// Collect all nodes from all positions (excluding BOS node at position 0)
	var allNodes []*Node
	for pos := 1; pos < len(l.endsFull); pos++ { // Start from 1 to skip BOS node like Rust
		nodes := l.endsFull[pos]
		for _, node := range nodes {
			if node != nil {
				allNodes = append(allNodes, node)
			}
		}
	}

	// Sort nodes in Rust-compatible order: end position (descending) → start position (ascending)
	// This matches Rust's lattice dump ordering exactly
	for i := 0; i < len(allNodes)-1; i++ {
		for j := i + 1; j < len(allNodes); j++ {
			// Primary sort: end position descending
			if allNodes[i].End() < allNodes[j].End() {
				allNodes[i], allNodes[j] = allNodes[j], allNodes[i]
			} else if allNodes[i].End() == allNodes[j].End() {
				// Secondary sort: start position ascending
				if allNodes[i].Begin() > allNodes[j].Begin() {
					allNodes[i], allNodes[j] = allNodes[j], allNodes[i]
				}
			}
		}
	}

	// Output sorted nodes
	for nodeID, node := range allNodes {
		// Get surface form
		surface := "UNKNOWN"
		if getSurface != nil {
			surface = getSurface(node)
		}

		// Get dictionary ID and word ID from the node's WordId
		var dictID int
		var wordID uint32

		if node.WordId().IsOOV() {
			dictID = -1
			wordID = node.WordId().Word()
		} else {
			dictID = 0 // System dictionary (assuming 0 for now)
			wordID = node.WordId().Word()
		}

		// Get POS string
		posStr := "UNKNOWN_POS"
		if getPOS != nil {
			posStr = getPOS(node)
		}

		// Format: node_id: begin end surface(dict_id, word_id) POS left_id right_id node_cost: connection_costs...
		fmt.Printf("%d: %d %d %s(%d, %d) %s %d %d %d:",
			nodeID,
			node.Begin(),
			node.End(),
			surface,
			dictID,
			wordID,
			posStr,
			node.LeftId(),
			node.RightId(),
			node.Cost(),
		)

		// Add connection costs
		if getConnectionCosts != nil {
			costs := getConnectionCosts(node)
			for _, cost := range costs {
				fmt.Printf(" %d", cost)
			}
		} else {
			fmt.Printf(" COSTS_UNKNOWN")
		}
		fmt.Println()
	}
}

// Dump outputs a basic lattice dump without detailed information
func (l *Lattice) Dump() {
	l.DumpWithDetails(nil, nil, nil)
}

// HasNodes returns true if there are nodes at the given position
func (l *Lattice) HasNodes(pos int) bool {
	if pos < 0 || pos >= len(l.endsFull) {
		return false
	}
	return len(l.endsFull[pos]) > 0
}

// GetNodesAt returns all nodes at the given position
func (l *Lattice) GetNodesAt(pos int) []*Node {
	if pos < 0 || pos >= len(l.endsFull) {
		return nil
	}
	return l.endsFull[pos]
}
