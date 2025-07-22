package lattice

import (
	"sync"

	"github.com/ikawaha/sudachi.go/dic"
)

// NodePool provides efficient Node object pooling
type NodePool struct {
	pool sync.Pool
}

// newNodePool creates a new NodePool
func newNodePool() *NodePool {
	return &NodePool{
		pool: sync.Pool{
			New: func() any {
				return &Node{}
			},
		},
	}
}

// globalNodePool is the shared Node pool instance
var globalNodePool = newNodePool()

// Get retrieves a Node from the pool
func (p *NodePool) Get() *Node {
	return p.pool.Get().(*Node)
}

// Put returns a Node to the pool after resetting it
func (p *NodePool) Put(node *Node) {
	if node != nil {
		// Reset node fields to zero values
		*node = Node{}
		p.pool.Put(node)
	}
}

// NewNodeFromPool creates a new Node from the pool with the given parameters
func NewNodeFromPool(begin, end, leftId, rightId uint16, cost int16, wordId dic.WordId) *Node {
	node := globalNodePool.Get()
	node.begin = begin
	node.end = end
	node.leftId = leftId
	node.rightId = rightId
	node.cost = cost
	node.wordId = wordId
	return node
}

// ReturnToPool returns a Node to the pool for reuse
func (n *Node) ReturnToPool() {
	globalNodePool.Put(n)
}

// NodeResultPool provides efficient NodeResult object pooling
type NodeResultPool struct {
	pool sync.Pool
}

// newNodeResultPool creates a new NodeResultPool
func newNodeResultPool() *NodeResultPool {
	return &NodeResultPool{
		pool: sync.Pool{
			New: func() any {
				return &NodeResult{}
			},
		},
	}
}

// globalNodeResultPool is the shared NodeResult pool instance
var globalNodeResultPool = newNodeResultPool()

// Get retrieves a NodeResult from the pool
func (p *NodeResultPool) Get() *NodeResult {
	return p.pool.Get().(*NodeResult)
}

// Put returns a NodeResult to the pool after resetting it
func (p *NodeResultPool) Put(result *NodeResult) {
	if result != nil {
		// Reset NodeResult fields to zero values
		*result = NodeResult{}
		p.pool.Put(result)
	}
}

// NewNodeResultCompleteFromPool creates a complete NodeResult from the pool
func NewNodeResultCompleteFromPool(node *Node, surface string, pos []string, features []string,
	normalizedForm, dictionaryForm, readingForm string) *NodeResult {
	result := globalNodeResultPool.Get()
	result.node = node
	result.surface = surface
	result.normalizedForm = normalizedForm
	result.dictionaryForm = dictionaryForm
	result.readingForm = readingForm
	result.pos = pos
	result.features = features
	return result
}

// NewNodeResultFromPool creates a basic NodeResult from the pool
func NewNodeResultFromPool(node *Node, surface string, pos []string, features []string) *NodeResult {
	return NewNodeResultCompleteFromPool(node, surface, pos, features, surface, surface, surface)
}

// ReturnToPool returns a NodeResult to the pool for reuse
func (nr *NodeResult) ReturnToPool() {
	globalNodeResultPool.Put(nr)
}

// MorphemeListPool provides efficient MorphemeList object pooling
type MorphemeListPool struct {
	pool sync.Pool
}

// newMorphemeListPool creates a new MorphemeListPool
func newMorphemeListPool() *MorphemeListPool {
	return &MorphemeListPool{
		pool: sync.Pool{
			New: func() any {
				return &MorphemeList{
					results: make([]*NodeResult, 0, 8), // Start with capacity of 8
				}
			},
		},
	}
}

// globalMorphemeListPool is the shared MorphemeList pool instance
var globalMorphemeListPool = newMorphemeListPool()

// Get retrieves a MorphemeList from the pool
func (p *MorphemeListPool) Get() *MorphemeList {
	ml := p.pool.Get().(*MorphemeList)
	// Reset slice but keep capacity
	ml.results = ml.results[:0]
	return ml
}

// Put returns a MorphemeList to the pool after resetting it
func (p *MorphemeListPool) Put(ml *MorphemeList) {
	if ml != nil {
		// Clear the slice but preserve capacity if reasonable
		if cap(ml.results) <= 128 { // Prevent excessive memory retention
			ml.results = ml.results[:0]
		} else {
			ml.results = make([]*NodeResult, 0, 8)
		}
		p.pool.Put(ml)
	}
}

// NewMorphemeListFromPool creates a new MorphemeList from the pool
func NewMorphemeListFromPool() *MorphemeList {
	return globalMorphemeListPool.Get()
}

// ReturnToPool returns a MorphemeList to the pool for reuse
func (ml *MorphemeList) ReturnToPool() {
	globalMorphemeListPool.Put(ml)
}

// Pool statistics for monitoring and debugging
type PoolStats struct {
	NodePoolSize         int
	NodeResultPoolSize   int
	MorphemeListPoolSize int
}

// GetPoolStats returns current pool statistics (for debugging/monitoring)
func GetPoolStats() PoolStats {
	// Note: sync.Pool doesn't expose size directly, so these would need
	// custom implementation if detailed stats are required
	return PoolStats{
		NodePoolSize:         0, // Would need custom counter
		NodeResultPoolSize:   0, // Would need custom counter
		MorphemeListPoolSize: 0, // Would need custom counter
	}
}
