package analysis

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/lattice"
)

// Tokenize performs morphological analysis on the input text
func (t *Tokenizer) Tokenize(text string) (*lattice.MorphemeList, error) {
	// Create normalized input buffer with proper character category setup
	buffer, _, err := t.createInputBufferWithCharacterCategory(text)
	if err != nil {
		return nil, fmt.Errorf("failed to create input buffer: %w", err)
	}

	// Ensure buffer is returned to pool when function exits
	defer buffer.ReturnToPool()

	t.inputBuffer = buffer

	// Debug: Input dump (show normalized text like Rust version)
	if t.debugMode {
		fmt.Println("=== Input dump:")
		fmt.Println(buffer.Modified())
	}

	// Build lattice
	if err := t.buildLattice(); err != nil {
		return nil, fmt.Errorf("failed to build lattice: %w", err)
	}

	// Debug: Lattice dump
	if t.debugMode {
		t.dumpLattice()
	}

	// Get the best path
	path, err := t.lattice.GetBestPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get best path: %w", err)
	}

	if len(path) == 0 {
		return nil, fmt.Errorf("empty path returned: no morphemes in optimal path")
	}

	// Convert a path to NodeResults with complete word information
	nodeResults, err := t.pathToNodeResults(path)
	if err != nil {
		return nil, fmt.Errorf("failed to convert path to node results: %w", err)
	}

	// Debug: Before Rewriting
	if t.debugMode {
		t.dumpPath("=== Before Rewriting:", nodeResults)
	}

	// Apply path rewrite plugins (matching Rust behavior)
	if t.pluginManager.HasPathRewriters() {
		rewrittenNodeResults, err := t.pluginManager.RewritePath(nodeResults, t.inputBuffer, t.lattice)
		if err != nil {
			return nil, fmt.Errorf("failed to apply path rewrite plugins: %w", err)
		}
		nodeResults = rewrittenNodeResults
	}

	// Debug: After Rewriting
	if t.debugMode {
		t.dumpPath("=== After Rewriting:", nodeResults)
	}

	// Convert NodeResults to morpheme results
	results, err := t.nodeResultsToMorphemes(nodeResults)
	if err != nil {
		return nil, fmt.Errorf("failed to convert node results to morphemes: %w", err)
	}

	return results, nil
}
