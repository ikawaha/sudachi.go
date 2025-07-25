package input

import (
	"sync"

	"github.com/ikawaha/sudachi.go/dic"
)

// InputBufferPool provides a pool of InputBuffer objects to reduce allocations
type InputBufferPool struct {
	pool sync.Pool
}

// Global pool instance for InputBuffer objects
var globalInputBufferPool = &InputBufferPool{
	pool: sync.Pool{
		New: func() any {
			return &InputBuffer{
				state: StateClean,
				// Pre-allocate slices with reasonable capacity
				m2o:              make([]int, 0, 256),
				modChars:         make([]rune, 0, 128),
				modC2B:           make([]int, 0, 128),
				modB2C:           make([]int, 0, 256),
				modBOW:           make([]bool, 0, 256),
				modCat:           make([]dic.CategoryType, 0, 128),
				modCatContinuity: make([]int, 0, 128),
			}
		},
	},
}

// Get retrieves an InputBuffer from the pool
func (p *InputBufferPool) Get() *InputBuffer {
	return p.pool.Get().(*InputBuffer)
}

// Put returns an InputBuffer to the pool after resetting it
func (p *InputBufferPool) Put(buffer *InputBuffer) {
	if buffer != nil {
		buffer.ResetForPool()
		p.pool.Put(buffer)
	}
}

// ResetForPool clears all fields of an InputBuffer for reuse in pool
func (ib *InputBuffer) ResetForPool() {
	ib.trueOriginal = ""
	ib.original = ""
	ib.modified = ""
	ib.normalizationInfo = nil
	ib.charCategory = nil
	ib.state = StateClean
	// Reset slices but keep capacity for efficiency
	ib.m2o = ib.m2o[:0]
	ib.modChars = ib.modChars[:0]
	ib.modC2B = ib.modC2B[:0]
	ib.modB2C = ib.modB2C[:0]
	ib.modBOW = ib.modBOW[:0]
	ib.modCat = ib.modCat[:0]
	ib.modCatContinuity = ib.modCatContinuity[:0]
}

// NewInputBufferFromPool creates a new InputBuffer using the global pool
func NewInputBufferFromPool() *InputBuffer {
	return globalInputBufferPool.Get()
}

// ReturnToPool returns an InputBuffer to the global pool
func (ib *InputBuffer) ReturnToPool() {
	globalInputBufferPool.Put(ib)
}
