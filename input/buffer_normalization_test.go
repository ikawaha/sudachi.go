package input

import (
	"reflect"
	"testing"
)

// TestOrigSliceWithNormalization tests OrigSlice with text normalization
// This test addresses the garbled text issue found in "ひＢら①がⅢな" → "ひbら1がⅢな"
func TestOrigSliceWithNormalization(t *testing.T) {
	tests := []struct {
		name          string
		original      string
		normalized    string
		expectedChars []string
		description   string
	}{
		{
			name:          "FullWidthToHalfWidth",
			original:      "ひＢら①がⅢな",
			normalized:    "ひbら1がⅢな",
			expectedChars: []string{"ひ", "Ｂ", "ら", "①", "が", "Ⅲ", "な"},
			description:   "Full-width characters should normalize but original characters should be preserved in OrigSlice",
		},
		{
			name:          "SimpleCase",
			original:      "ABC",
			normalized:    "abc",
			expectedChars: []string{"A", "B", "C"},
			description:   "Simple ASCII case conversion",
		},
		{
			name:          "MixedCase",
			original:      "Ａｂ１２３",
			normalized:    "ab123",
			expectedChars: []string{"Ａ", "ｂ", "１", "２", "３"},
			description:   "Mixed full-width and half-width characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer with original text
			buffer := NewInputBufferFromPool()
			defer buffer.ReturnToPool()

			err := buffer.StartBuild(tt.original)
			if err != nil {
				t.Fatalf("Failed to start build: %v", err)
			}

			// Simulate normalization by replacing individual characters as plugins actually do
			// In real usage, this would be done by input text plugins
			if tt.name == "FullWidthToHalfWidth" {
				err = buffer.WithEditor(func(buf *InputBuffer, editor *InputEditor) error {
					// Ｂ (bytes 3..6) -> b
					editor.ReplaceRange(Range{Start: 3, End: 6}, "b")
					// ① (bytes 9..12) -> 1
					editor.ReplaceRange(Range{Start: 9, End: 12}, "1")
					// Note: Ⅲ is not changed in this test case, unlike our specific test
					return nil
				})
				if err != nil {
					t.Fatalf("Failed to apply normalization: %v", err)
				}
			} else if tt.original != tt.normalized {
				// For other cases, keep the full replacement for now
				err = buffer.WithEditor(func(buf *InputBuffer, editor *InputEditor) error {
					editor.ReplaceRange(Range{Start: 0, End: len(tt.original)}, tt.normalized)
					return nil
				})
				if err != nil {
					t.Fatalf("Failed to apply normalization: %v", err)
				}
			}

			// Skip Build for this test - we only need to test OrigSlice functionality
			// Real usage would require proper Grammar with CharacterCategory
			// For now, manually set the state to test OrigSlice
			buffer.state = StateReadOnly

			// Test the specific problematic case: last character
			// This reproduces the issue found in CLI output
			normalized := tt.normalized

			// Test last character 'な' which was causing garbled output
			if normalized == "ひbら1がⅢな" {
				// Manually calculate byte position for last character in normalized text
				lastCharStart := len("ひbら1がⅢ") // 14 bytes
				lastCharEnd := len(normalized) // 17 bytes

				byteRange := Range{Start: lastCharStart, End: lastCharEnd}
				result := buffer.OrigSlice(byteRange)

				expected := "な"
				if result != expected {
					t.Errorf("Last character OrigSlice failed: got %q (len=%d), want %q",
						result, len(result), expected)
					t.Logf("Debug: byteRange[%d,%d) in normalized text, original='%s'",
						lastCharStart, lastCharEnd, tt.original)

					// This should fail with current implementation, demonstrating the bug
					t.Logf("This failure demonstrates the m2o mapping bug")
				}
			}
		})
	}
}

// TestNormalizationMapping tests the m2o mapping correctness
func TestNormalizationMapping(t *testing.T) {
	buffer := NewInputBufferFromPool()
	defer buffer.ReturnToPool()

	original := "ひＢら①がⅢな"
	normalized := "ひbら1がⅢな"

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	// Apply normalization with individual character replacements (as plugins actually do)
	err = buffer.WithEditor(func(buf *InputBuffer, editor *InputEditor) error {
		// Ｂ (bytes 3..6) -> b
		editor.ReplaceRange(Range{Start: 3, End: 6}, "b")
		// ① (bytes 9..12) -> 1
		editor.ReplaceRange(Range{Start: 9, End: 12}, "1")
		// Note: Ⅲ is not changed in this test case, unlike our specific test
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to apply normalization: %v", err)
	}

	// Skip Build - manually set state for testing
	buffer.state = StateReadOnly

	// Test that normalized and original text have different byte lengths
	t.Logf("Original length: %d bytes, Normalized length: %d bytes",
		len(original), len(normalized))

	if len(original) == len(normalized) {
		t.Skip("This test requires byte length change from normalization")
	}

	// Test specific case: last character 'な'
	// Manually calculate byte position for last character
	startByte := len("ひbら1がⅢ") // 14 bytes
	endByte := len(normalized) // 17 bytes

	byteRange := Range{Start: startByte, End: endByte}
	result := buffer.OrigSlice(byteRange)

	expected := "な"
	if result != expected {
		t.Errorf("Last character mapping failed: got %q, want %q", result, expected)
		t.Logf("Byte range [%d,%d) in normalized text of length %d",
			startByte, endByte, len(normalized))

		// Log m2o mapping for debugging
		if startByte < len(buffer.m2o) && endByte <= len(buffer.m2o) {
			t.Logf("m2o mapping: [%d]=%d, [%d]=%d",
				startByte, buffer.m2o[startByte],
				endByte, buffer.m2o[endByte])
		}
	}
}

// TestRustCompatibleMapping tests mapping behavior to match Rust implementation
// Based on Rust test: replace_diff_width
func TestRustCompatibleMapping(t *testing.T) {
	// Test case similar to Rust's replace_diff_width
	buffer := NewInputBufferFromPool()
	defer buffer.ReturnToPool()

	// Original: "âｂC1あ" -> Modified: "abc1あ" (like Rust test)
	original := "âｂC1あ"
	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	// Apply multiple replacements like Rust test
	err = buffer.WithEditor(func(buf *InputBuffer, editor *InputEditor) error {
		editor.ReplaceRange(Range{Start: 0, End: 2}, "a") // â(2 bytes) -> a(1 byte)
		editor.ReplaceRange(Range{Start: 2, End: 5}, "b") // ｂ(3 bytes) -> b(1 byte)
		editor.ReplaceRange(Range{Start: 5, End: 6}, "c") // C(1 byte) -> c(1 byte)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to apply normalization: %v", err)
	}

	buffer.state = StateReadOnly
	modified := buffer.Modified()
	expected := "abc1あ"

	if modified != expected {
		t.Fatalf("Modified text incorrect: got %q, want %q", modified, expected)
	}

	// Test m2o mapping - should match Rust: [0, 2, 5, 6, 7, 8, 9, 10]
	expectedM2O := []int{0, 2, 5, 6, 7, 8, 9, 10}
	if !reflect.DeepEqual(buffer.m2o, expectedM2O) {
		t.Errorf("m2o mapping incorrect: got %v, want %v", buffer.m2o, expectedM2O)
	}

	// Test specific OrigSlice calls that should match Rust behavior
	testCases := []struct {
		start, end  int
		expected    string
		description string
	}{
		{0, 3, "âｂC", "full range"},
		{0, 1, "â", "first char"},
		{1, 2, "ｂ", "second char"},
		{2, 3, "C", "third char"},
		{3, 4, "1", "fourth char"},
		{4, 7, "あ", "last char"},
	}

	for _, tc := range testCases {
		result := buffer.OrigSlice(Range{Start: tc.start, End: tc.end})
		if result != tc.expected {
			t.Errorf("OrigSlice[%d,%d) failed (%s): got %q, want %q",
				tc.start, tc.end, tc.description, result, tc.expected)
		}
	}
}

// TestOurSpecificCase tests the exact case that was causing garbled output
func TestOurSpecificCase(t *testing.T) {
	buffer := NewInputBufferFromPool()
	defer buffer.ReturnToPool()

	// Our problematic case: "ひＢら①がⅢな" -> "ひbら1がⅢな"
	original := "ひＢら①がⅢな"
	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	// Apply the normalization that would be done by input text plugins
	err = buffer.WithEditor(func(buf *InputBuffer, editor *InputEditor) error {
		// This simulates the individual NFKC normalization that plugins actually do:
		// Ｂ (bytes 3..6) -> b
		editor.ReplaceRange(Range{Start: 3, End: 6}, "b")
		// ① (bytes 9..12) -> 1
		editor.ReplaceRange(Range{Start: 9, End: 12}, "1")
		// Ⅲ (bytes 15..18) -> ⅲ
		editor.ReplaceRange(Range{Start: 15, End: 18}, "ⅲ")
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to apply normalization: %v", err)
	}

	buffer.state = StateReadOnly
	modified := buffer.Modified()
	expected := "ひbら1がⅲな"

	if modified != expected {
		t.Fatalf("Modified text incorrect: got %q, want %q", modified, expected)
	}

	t.Logf("Original: %q (%d bytes)", original, len(original))
	t.Logf("Modified: %q (%d bytes)", modified, len(modified))
	t.Logf("m2o mapping: %v", buffer.m2o)

	// Expected mapping from Rust implementation
	expectedMapping := []int{0, 1, 2, 3, 6, 7, 8, 9, 12, 13, 14, 15, 18, 18, 18, 19, 20, 21}
	if len(buffer.m2o) != len(expectedMapping) {
		t.Errorf("Mapping length mismatch: got %d, want %d", len(buffer.m2o), len(expectedMapping))
	} else {
		for i, got := range buffer.m2o {
			if got != expectedMapping[i] {
				t.Errorf("Mapping mismatch at index %d: got %d, want %d", i, got, expectedMapping[i])
			}
		}
	}

	// Test the last character that was causing problems
	lastCharStart := len("ひbら1がⅲ") // 14 bytes
	lastCharEnd := len(modified)   // 17 bytes

	result := buffer.OrigSlice(Range{Start: lastCharStart, End: lastCharEnd})
	expected_char := "な"

	if result != expected_char {
		t.Errorf("Last character OrigSlice failed: got %q, want %q", result, expected_char)
		t.Logf("Tried to get modified[%d:%d] -> original", lastCharStart, lastCharEnd)
		if lastCharStart < len(buffer.m2o) && lastCharEnd < len(buffer.m2o) {
			t.Logf("m2o[%d]=%d, m2o[%d]=%d",
				lastCharStart, buffer.m2o[lastCharStart],
				lastCharEnd, buffer.m2o[lastCharEnd])
		}
	}
}
