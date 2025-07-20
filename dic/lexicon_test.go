package dic

import (
	"testing"
)

// TestLexiconLookupCompareWithRust replicates the exact Rust lexicon test
func TestLexiconLookupCompareWithRust(t *testing.T) {
	// Load the exact same test dictionary as Rust
	dicPath := "../resources/system.dic.test"
	loader := NewDictionaryLoader()

	sysDict, err := loader.LoadSystemDictionary(dicPath)
	if err != nil {
		t.Skipf("Test dictionary not found: %v", err)
		return
	}

	// Test with the full Lexicon implementation
	lexicon := sysDict.LexiconSet()
	if lexicon == nil {
		t.Fatalf("Lexicon is nil")
	}

	t.Logf("Loaded test dictionary, lexicon size: %d", lexicon.Size())

	// Test 1: "東京都" - Rust expects 3 results
	testInput := "東京都"
	inputBytes := []byte(testInput)

	t.Logf("Testing input: '%s' (bytes: %v, hex: %x)", testInput, inputBytes, inputBytes)

	iter, err := lexicon.Lookup(inputBytes, 0)
	if err != nil {
		t.Errorf("Failed to create lexicon lookup: %v", err)
		return
	}

	var results []*LexiconEntry
	for {
		entry, err := iter.Next()
		if err != nil {
			t.Errorf("Iterator error: %v", err)
			break
		}
		if entry == nil {
			break
		}

		results = append(results, entry)

		// Calculate what text was matched
		matchedBytes := inputBytes[0:entry.End]
		t.Logf("Found lexicon entry: word_id=%s, end=%d, matched='%s' (bytes: %x)",
			entry.WordId.String(), entry.End, string(matchedBytes), matchedBytes)
	}

	t.Logf("Go found %d results, Rust expects 3", len(results))

	if len(results) != 3 {
		t.Errorf("Result count mismatch: Go found %d, Rust expects 3", len(results))

		// Debug: let's check what's in the test dictionary
		t.Logf("Debugging test dictionary structure...")

		// Check if any single bytes work with lexicon lookup
		singleByteMatches := 0
		for b := 0; b < 256; b++ {
			iter, err := lexicon.Lookup([]byte{byte(b)}, 0)
			if err != nil {
				continue
			}

			entry, err := iter.Next()
			if err != nil || entry == nil {
				continue
			}

			singleByteMatches++
			if singleByteMatches <= 5 { // Show first 5
				t.Logf("  Single byte 0x%02x works: word_id=%s", b, entry.WordId.String())
			}
		}

		t.Logf("Total single byte matches found: %d", singleByteMatches)

		if singleByteMatches == 0 {
			t.Logf("No single byte matches - the test dictionary may be empty or our implementation has issues")
		}
	}
}

// TestLexiconAllRustCases tests all the cases from Rust lexicon test
func TestLexiconAllRustCases(t *testing.T) {
	dicPath := "../resources/system.dic.test"
	loader := NewDictionaryLoader()

	sysDict, err := loader.LoadSystemDictionary(dicPath)
	if err != nil {
		t.Skipf("Test dictionary not found: %v", err)
		return
	}

	lexicon := sysDict.LexiconSet()
	if lexicon == nil {
		t.Fatalf("Lexicon is nil")
	}

	// All test cases from Rust lexicon test
	testCases := []struct {
		input    string
		offset   int
		expected int
		desc     string
	}{
		{"東京都", 0, 3, "東京都 should find 東, 東京, 東京都"},
		{"東京都に", 9, 2, "に at offset 9 should find に(接続助詞), に(格助詞)"},
		{"あれ", 0, 0, "あれ should find nothing"},
	}

	for i, tc := range testCases {
		t.Logf("\nTest case %d: %s", i+1, tc.desc)

		inputBytes := []byte(tc.input)
		t.Logf("Input: '%s', offset: %d", tc.input, tc.offset)
		t.Logf("Bytes: %v (hex: %x)", inputBytes, inputBytes)

		iter, err := lexicon.Lookup(inputBytes, tc.offset)
		if err != nil {
			t.Errorf("Failed to create lexicon lookup: %v", err)
			continue
		}

		var results []*LexiconEntry
		for {
			entry, err := iter.Next()
			if err != nil {
				t.Errorf("Iterator error: %v", err)
				break
			}
			if entry == nil {
				break
			}

			results = append(results, entry)

			// Calculate matched text
			// Note: entry.End is relative length from offset, not absolute position
			matchStart := tc.offset
			matchEnd := tc.offset + entry.End
			if matchEnd <= len(inputBytes) && matchStart < matchEnd {
				matchedBytes := inputBytes[matchStart:matchEnd]
				t.Logf("  Found lexicon entry: word_id=%s, end=%d, matched='%s'",
					entry.WordId.String(), entry.End, string(matchedBytes))
			} else {
				t.Logf("  Found lexicon entry: word_id=%s, end=%d (bounds: start=%d, end=%d, inputLen=%d)",
					entry.WordId.String(), entry.End, matchStart, matchEnd, len(inputBytes))
			}
		}

		t.Logf("Go found %d results, expected %d", len(results), tc.expected)

		if len(results) != tc.expected {
			t.Errorf("Test case %d failed: found %d results, expected %d", i+1, len(results), tc.expected)
		} else {
			t.Logf("✓ Test case %d passed", i+1)
		}
	}
}

// TestLexiconLookupHi tests the specific "贔" character that causes compatibility issues
func TestLexiconLookupHi(t *testing.T) {
	// Load system dictionary for testing
	loader := NewDictionaryLoader()
	dict, err := loader.LoadSystemDictionary("../resources/system.dic")
	if err != nil {
		t.Skipf("System dictionary not found: %v", err)
	}

	lexicon := dict.LexiconSet()

	// Test 1: "贔" lookup - investigating why this fails in Go but works in Rust
	input := []byte("贔")
	t.Logf("Testing character '贔' (bytes: %v, hex: %x)", input, input)

	iter, err := lexicon.Lookup(input, 0)
	if err != nil {
		t.Fatalf("Failed to lookup '贔': %v", err)
	}

	var entries []*LexiconEntry
	for {
		entry, err := iter.Next()
		if err != nil {
			t.Fatalf("Error during iteration: %v", err)
		}
		if entry == nil {
			break
		}
		entries = append(entries, entry)
		t.Logf("Found entry: WordId=%s, End=%d", entry.WordId.String(), entry.End)
	}

	t.Logf("Total entries found for '贔': %d", len(entries))

	// Test 2: "贔負" lookup - the compound word that causes compatibility issues
	input2 := []byte("贔負")
	t.Logf("Testing compound word '贔負' (bytes: %v, hex: %x)", input2, input2)

	iter2, err := lexicon.Lookup(input2, 0)
	if err != nil {
		t.Fatalf("Failed to lookup '贔負': %v", err)
	}

	var entries2 []*LexiconEntry
	for {
		entry, err := iter2.Next()
		if err != nil {
			t.Fatalf("Error during iteration: %v", err)
		}
		if entry == nil {
			break
		}
		entries2 = append(entries2, entry)
		t.Logf("Found entry: WordId=%s, End=%d", entry.WordId.String(), entry.End)
	}

	t.Logf("Total entries found for '贔負': %d", len(entries2))

	// Test 3: Manual trie traversal debug for "贔" - DISABLED for LexiconSet
	// (Internal implementation is not accessible via LexiconSet interface)
	t.Logf("Manual trie traversal test disabled for LexiconSet compatibility")
	return
}

// TestTrieBitManipulationForLexicon tests the trie bit manipulation functions
func TestTrieBitManipulationForLexicon(t *testing.T) {
	// Disabled for LexiconSet compatibility (trie not accessible)
	t.Skipf("Test disabled for LexiconSet compatibility")
}

// TestCompareDictionaryFiles compares our test dictionary with Rust's
func TestCompareDictionaryFiles(t *testing.T) {
	// Compare file sizes
	goPath := "../resources/system.dic.test"
	rustPath := "./sudachi.rs/sudachi/tests/resources/system.dic.test"

	// Check if files exist and compare sizes
	t.Logf("Comparing dictionary files:")
	t.Logf("Go path: %s", goPath)
	t.Logf("Rust path: %s", rustPath)

	// Load Go version
	loader := NewDictionaryLoader()

	goDict, err := loader.LoadSystemDictionary(goPath)
	if err != nil {
		t.Errorf("Failed to load Go dictionary: %v", err)
	} else {
		t.Logf("Go dictionary loaded successfully, trie size: %d", goDict.Trie().Size())
	}

	// Try to load Rust version with Go loader
	rustDict, err := loader.LoadSystemDictionary(rustPath)
	if err != nil {
		t.Logf("Cannot load Rust dictionary with Go loader: %v", err)
	} else {
		t.Logf("Rust dictionary loaded with Go loader, trie size: %d", rustDict.Trie().Size())

		if goDict != nil && rustDict != nil {
			if goDict.Trie().Size() != rustDict.Trie().Size() {
				t.Errorf("Dictionary sizes differ: Go=%d, Rust=%d",
					goDict.Trie().Size(), rustDict.Trie().Size())
			} else {
				t.Logf("✓ Dictionary sizes match: %d", goDict.Trie().Size())
			}
		}
	}
}
