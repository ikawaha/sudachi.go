package dic

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/*
var testdata embed.FS

func TestDictionaryLoader_LoadSystemDictionary(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		loader := NewDictionaryLoader()
		_, err := loader.LoadSystemDictionary("/nonexistent/path.dic")
		if err == nil {
			t.Error("LoadSystemDictionary() expected error for nonexistent file")
		}
	})

	t.Run("invalid extension", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		loader := NewDictionaryLoader()
		_, err = loader.LoadSystemDictionary(tmpFile.Name())
		if err == nil {
			t.Error("LoadSystemDictionary() expected error for non-.dic file")
		}
	})
}

func TestDictionaryLoader_LoadUserDictionary(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		loader := NewDictionaryLoader()
		_, err := loader.LoadUserDictionary("/nonexistent/path.dic")
		if err == nil {
			t.Error("LoadUserDictionary() expected error for nonexistent file")
		}
	})
}

func TestDictionaryLoader_LoadSystemDictionaryFromBytes(t *testing.T) {
	t.Run("insufficient data", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, 10) // Too small
		_, err := loader.LoadSystemDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadSystemDictionaryFromBytes() expected error for insufficient data")
		}
	})

	t.Run("invalid header", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize)
		// Write invalid version
		writeU64(data[0:8], 0x1234567890abcdef)
		_, err := loader.LoadSystemDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadSystemDictionaryFromBytes() expected error for invalid header")
		}
	})

	t.Run("user dict version in system dict", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize)
		// Write user dictionary version instead of system
		writeU64(data[0:8], UserDictVersion1)
		_, err := loader.LoadSystemDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadSystemDictionaryFromBytes() expected error for user dict version")
		}
	})
}

func TestDictionaryLoader_LoadUserDictionaryFromBytes(t *testing.T) {
	t.Run("insufficient data", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, 10) // Too small
		_, err := loader.LoadUserDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadUserDictionaryFromBytes() expected error for insufficient data")
		}
	})

	t.Run("invalid header", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize)
		// Write invalid version
		writeU64(data[0:8], 0x1234567890abcdef)
		_, err := loader.LoadUserDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadUserDictionaryFromBytes() expected error for invalid header")
		}
	})

	t.Run("system dict version in user dict", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize)
		// Write system dictionary version instead of user
		writeU64(data[0:8], SystemDictVersion1)
		_, err := loader.LoadUserDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadUserDictionaryFromBytes() expected error for system dict version")
		}
	})
}

func TestDictionaryLoader_parseDictionarySections(t *testing.T) {
	t.Run("insufficient data for grammar", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize+4) // Just enough for header, not grammar

		header := &Header{
			Version:     SystemDictVersionV2, // Has grammar
			CreateTime:  1234567890,
			Description: "Test",
		}

		_, err := loader.parseDictionarySections(data, header)
		if err == nil {
			t.Error("parseDictionarySections() expected error for insufficient grammar data")
		}
	})

	t.Run("zero trie size", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize+100) // Enough space

		// Create header without grammar to skip grammar parsing
		header := &Header{
			Version:     UserDictVersionV1, // No grammar
			CreateTime:  1234567890,
			Description: "Test",
		}

		// Write zero trie size at offset StorageSize
		writeU32(data[StorageSize:StorageSize+4], 0)

		_, err := loader.parseDictionarySections(data, header)
		if err == nil {
			t.Error("parseDictionarySections() expected error for zero trie size")
		}
	})

	t.Run("insufficient data for trie", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize+8) // Header + trie size, but no trie data

		header := &Header{
			Version:     UserDictVersionV1, // No grammar
			CreateTime:  1234567890,
			Description: "Test",
		}

		// Write trie size = 10, but don't provide enough data
		writeU32(data[StorageSize:StorageSize+4], 10)

		_, err := loader.parseDictionarySections(data, header)
		if err == nil {
			t.Error("parseDictionarySections() expected error for insufficient trie data")
		}
	})
}

func TestValidateDictionaryFile(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		err := ValidateDictionaryFile("/nonexistent/path.dic")
		if err == nil {
			t.Error("ValidateDictionaryFile() expected error for nonexistent file")
		}
	})

	t.Run("invalid extension", func(t *testing.T) {
		err := ValidateDictionaryFile("/some/path.txt")
		if err == nil {
			t.Error("ValidateDictionaryFile() expected error for invalid extension")
		}
	})

	t.Run("valid extension but file too small", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "loader_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "test.dic")
		data := make([]byte, 10) // Too small
		err = os.WriteFile(tmpFile, data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		err = ValidateDictionaryFile(tmpFile)
		if err == nil {
			t.Error("ValidateDictionaryFile() expected error for file too small")
		}
	})

	t.Run("invalid header", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "loader_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "test.dic")
		data := make([]byte, StorageSize+10)
		// Write invalid version
		writeU64(data[0:8], 0x1234567890abcdef)

		err = os.WriteFile(tmpFile, data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		err = ValidateDictionaryFile(tmpFile)
		if err == nil {
			t.Error("ValidateDictionaryFile() expected error for invalid header")
		}
	})
}

func TestGetDictionaryInfo(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := GetDictionaryInfo("/nonexistent/path.dic")
		if err == nil {
			t.Error("GetDictionaryInfo() expected error for nonexistent file")
		}
	})

	t.Run("invalid header", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "loader_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "test.dic")
		data := make([]byte, StorageSize)
		// Write invalid version
		writeU64(data[0:8], 0x1234567890abcdef)

		err = os.WriteFile(tmpFile, data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = GetDictionaryInfo(tmpFile)
		if err == nil {
			t.Error("GetDictionaryInfo() expected error for invalid header")
		}
	})

	t.Run("valid system dictionary info", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "loader_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "test.dic")
		data := make([]byte, StorageSize+100)

		// Create valid system dictionary header
		writeU64(data[0:8], SystemDictVersion2)
		writeU64(data[8:16], 1234567890)
		copy(data[16:], []byte("Test System Dictionary"))

		err = os.WriteFile(tmpFile, data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		info, err := GetDictionaryInfo(tmpFile)
		if err != nil {
			t.Fatalf("GetDictionaryInfo() unexpected error: %v", err)
		}

		if info.Version.ToU64() != SystemDictVersion2 {
			t.Errorf("GetDictionaryInfo().Version = %v, want SystemDictV2", info.Version)
		}
		if info.CreateTime != 1234567890 {
			t.Errorf("GetDictionaryInfo().CreateTime = %d, want 1234567890", info.CreateTime)
		}
		if info.Description != "Test System Dictionary" {
			t.Errorf("GetDictionaryInfo().Description = %q, want %q", info.Description, "Test System Dictionary")
		}
		if info.Size != len(data) {
			t.Errorf("GetDictionaryInfo().Size = %d, want %d", info.Size, len(data))
		}
		if !info.HasGrammar {
			t.Error("GetDictionaryInfo().HasGrammar = false, want true for system dict")
		}
		if !info.IsSystem {
			t.Error("GetDictionaryInfo().IsSystem = false, want true for system dict")
		}
	})

	t.Run("valid user dictionary info", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "loader_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "test.dic")
		data := make([]byte, StorageSize+50)

		// Create valid user dictionary header
		writeU64(data[0:8], UserDictVersion1)
		writeU64(data[8:16], 9876543210)
		copy(data[16:], []byte("Test User Dictionary"))

		err = os.WriteFile(tmpFile, data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		info, err := GetDictionaryInfo(tmpFile)
		if err != nil {
			t.Fatalf("GetDictionaryInfo() unexpected error: %v", err)
		}

		if info.Version.ToU64() != UserDictVersion1 {
			t.Errorf("GetDictionaryInfo().Version = %v, want UserDictV1", info.Version)
		}
		if info.CreateTime != 9876543210 {
			t.Errorf("GetDictionaryInfo().CreateTime = %d, want 9876543210", info.CreateTime)
		}
		if info.Description != "Test User Dictionary" {
			t.Errorf("GetDictionaryInfo().Description = %q, want %q", info.Description, "Test User Dictionary")
		}
		if info.HasGrammar {
			t.Error("GetDictionaryInfo().HasGrammar = true, want false for user dict v1")
		}
		if info.IsSystem {
			t.Error("GetDictionaryInfo().IsSystem = true, want false for user dict")
		}
	})
}

// Helper function to write uint32 in little-endian format
func writeU32(data []byte, value uint32) error {
	if len(data) < 4 {
		return fmt.Errorf("insufficient data: need at least %d bytes, have %d", 4, len(data))
	}
	data[0] = byte(value)
	data[1] = byte(value >> 8)
	data[2] = byte(value >> 16)
	data[3] = byte(value >> 24)
	return nil
}

func TestDictionaryLoader_StorageTypes(t *testing.T) {
	t.Run("file storage error handling", func(t *testing.T) {
		loader := NewDictionaryLoader()

		// Test with invalid path
		_, err := loader.LoadSystemDictionary("")
		if err == nil {
			t.Error("LoadSystemDictionary() expected error for empty path")
		}
	})

	t.Run("borrowed storage system dict", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize+100)

		// Create valid system dictionary header
		writeU64(data[0:8], SystemDictVersion1)
		writeU64(data[8:16], 1234567890)
		copy(data[16:], []byte("Test"))

		// This should fail because we don't have valid lexicon data
		_, err := loader.LoadSystemDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadSystemDictionaryFromBytes() should fail with incomplete data")
		}
	})

	t.Run("borrowed storage user dict", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize+100)

		// Create valid user dictionary header
		writeU64(data[0:8], UserDictVersion1)
		writeU64(data[8:16], 1234567890)
		copy(data[16:], []byte("Test"))

		// This should fail because we don't have valid lexicon data
		_, err := loader.LoadUserDictionaryFromBytes(data)
		if err == nil {
			t.Error("LoadUserDictionaryFromBytes() should fail with incomplete data")
		}
	})
}

func TestDictionaryLoader_LexiconSetCreation(t *testing.T) {
	t.Run("system dictionary creates lexicon set", func(t *testing.T) {
		loader := NewDictionaryLoader()
		data := make([]byte, StorageSize+100)

		// Create minimal valid system dictionary header
		writeU64(data[0:8], SystemDictVersion1)
		writeU64(data[8:16], 1234567890)
		copy(data[16:], []byte("Test"))

		// The loading will fail at lexicon parsing, but we test header validation
		_, err := loader.LoadSystemDictionaryFromBytes(data)
		if err == nil {
			t.Error("Expected error due to insufficient lexicon data")
		}

		// Error should be related to lexicon parsing, not header validation
		if !bytes.Contains([]byte(err.Error()), []byte("trie")) &&
			!bytes.Contains([]byte(err.Error()), []byte("lexicon")) {
			t.Errorf("Expected lexicon-related error, got: %v", err)
		}
	})
}

func BenchmarkDictionaryLoader_LoadSystemDictionaryFromBytes(b *testing.B) {
	data := make([]byte, StorageSize+100)

	// Create valid system dictionary header
	writeU64(data[0:8], SystemDictVersion2)
	writeU64(data[8:16], 1234567890)
	copy(data[16:], []byte("Benchmark Test"))

	loader := NewDictionaryLoader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will fail due to incomplete data, but tests header parsing performance
		_, _ = loader.LoadSystemDictionaryFromBytes(data)
	}
}

func BenchmarkValidateDictionaryFile(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "loader_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.dic")
	data := make([]byte, StorageSize+100)

	// Create valid header
	writeU64(data[0:8], SystemDictVersion2)
	writeU64(data[8:16], 1234567890)
	copy(data[16:], []byte("Benchmark Test"))

	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateDictionaryFile(tmpFile)
	}
}

// Helper functions for creating test dictionaries
func createMinimalSystemDictionary(t *testing.T) []byte {
	t.Helper()

	// Read test data
	matrixData, err := testdata.ReadFile("testdata/matrix_small.def")
	if err != nil {
		t.Fatalf("Failed to read test matrix: %v", err)
	}

	// Create minimal valid system dictionary structure
	// This is a simplified version - in practice you'd need a full dictionary builder
	data := make([]byte, StorageSize+1000) // Allocate enough space

	// Write system dictionary header
	writeU64(data[0:8], SystemDictVersion1)
	writeU64(data[8:16], 1234567890)
	copy(data[16:], []byte("Test System Dictionary"))

	// For testing purposes, we'll create a minimal structure
	// In a real implementation, this would use DictBuilder
	offset := StorageSize

	// Placeholder grammar section (simplified)
	grammarSize := 100
	copy(data[offset:offset+grammarSize], matrixData[:min(grammarSize, len(matrixData))])
	offset += grammarSize

	// Placeholder lexicon section
	// Write trie size
	writeU32(data[offset:offset+4], 1)
	offset += 4

	// Write minimal trie data (4 bytes)
	writeU32(data[offset:offset+4], 0)
	offset += 4

	// Write word id table size
	writeU32(data[offset:offset+4], 4)
	offset += 4

	// Write minimal word id table
	writeU32(data[offset:offset+4], 0)
	offset += 4

	// Write word params size
	writeU32(data[offset:offset+4], 12)
	offset += 4

	// Write minimal word params (3 uint32 values: left_id, right_id, cost)
	writeU32(data[offset:offset+4], 6)       // left_id
	writeU32(data[offset+4:offset+8], 6)     // right_id
	writeU32(data[offset+8:offset+12], 5293) // cost
	offset += 12

	return data[:offset]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Test functions using real dictionary data
func TestDictionaryLoader_RealDictionaryRoundTrip(t *testing.T) {
	t.Run("minimal system dictionary", func(t *testing.T) {
		dictData := createMinimalSystemDictionary(t)

		loader := NewDictionaryLoader()

		// This will likely fail due to incomplete dictionary structure,
		// but tests the header parsing and basic validation
		_, err := loader.LoadSystemDictionaryFromBytes(dictData)

		// We expect this to fail with lexicon parsing error, not header error
		if err == nil {
			t.Error("Expected error due to incomplete dictionary structure")
		}

		// Ensure it's not a header validation error
		if bytes.Contains([]byte(err.Error()), []byte("header")) {
			t.Errorf("Unexpected header error: %v", err)
		}
	})
}

func TestDictionaryLoader_TestDataValidation(t *testing.T) {
	t.Run("test lexicon data exists", func(t *testing.T) {
		data, err := testdata.ReadFile("testdata/minimal_1word.csv")
		if err != nil {
			t.Fatalf("Test lexicon data not found: %v", err)
		}

		expected := "京都,6,6,5293,京都,名詞,固有名詞,地名,一般,*,*,キョウト,京都,*,A,*,*,*,*"
		if string(data) != expected {
			t.Errorf("Unexpected lexicon content: got %q, want %q", string(data), expected)
		}
	})

	t.Run("test matrix data exists", func(t *testing.T) {
		data, err := testdata.ReadFile("testdata/matrix_small.def")
		if err != nil {
			t.Fatalf("Test matrix data not found: %v", err)
		}

		if len(data) == 0 {
			t.Error("Matrix data is empty")
		}

		// Check first line is "3 3"
		lines := bytes.Split(data, []byte("\n"))
		if len(lines) == 0 || string(lines[0]) != "3 3" {
			t.Errorf("Unexpected matrix header: got %q, want %q", string(lines[0]), "3 3")
		}
	})
}

func TestDictionaryLoader_HeaderValidationWithRealData(t *testing.T) {
	tests := []struct {
		name        string
		version     uint64
		description string
		expectError bool
	}{
		{"valid system v1", SystemDictVersion1, "Test System V1", false},
		{"valid system v2", SystemDictVersion2, "Test System V2", false},
		{"valid user v1", UserDictVersion1, "Test User V1", false},
		{"valid user v2", UserDictVersion2, "Test User V2", false},
		{"valid user v3", UserDictVersion3, "Test User V3", false},
		{"invalid version", 0x123456789abcdef0, "Invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, StorageSize+100)

			// Write header
			writeU64(data[0:8], tt.version)
			writeU64(data[8:16], 1234567890)
			copy(data[16:], []byte(tt.description))

			// Parse header only
			_, err := ParseHeader(data)

			if (err != nil) != tt.expectError {
				t.Errorf("ParseHeader() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestDictionaryLoader_StorageIntegration(t *testing.T) {
	t.Run("file storage with embedded test data", func(t *testing.T) {
		// Create temporary file with test data
		tmpDir, err := os.MkdirTemp("", "loader_integration")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "test.dic")

		// Create minimal dictionary with valid header
		dictData := make([]byte, StorageSize+200)
		writeU64(dictData[0:8], SystemDictVersion1)
		writeU64(dictData[8:16], 1234567890)
		copy(dictData[16:], []byte("Integration Test"))

		err = os.WriteFile(tmpFile, dictData, 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Test file loading
		loader := NewDictionaryLoader()
		_, err = loader.LoadSystemDictionary(tmpFile)

		// Should fail due to incomplete dictionary structure, but not due to file access
		if err == nil {
			t.Error("Expected error due to incomplete dictionary")
		}

		// Should not be a file access error
		if os.IsNotExist(err) {
			t.Errorf("Unexpected file access error: %v", err)
		}
	})

	t.Run("borrowed storage consistency", func(t *testing.T) {
		dictData := createMinimalSystemDictionary(t)

		loader := NewDictionaryLoader()

		// Test both loading methods give same error
		_, err1 := loader.LoadSystemDictionaryFromBytes(dictData)

		// Create temporary file with same data
		tmpDir, err := os.MkdirTemp("", "borrowed_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "test.dic")
		err = os.WriteFile(tmpFile, dictData, 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err2 := loader.LoadSystemDictionary(tmpFile)

		// Both should fail, but with similar error types
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("Inconsistent behavior: bytes=%v, file=%v", err1, err2)
		}
	})
}

// Benchmark with realistic data
func BenchmarkDictionaryLoader_RealDataParsing(b *testing.B) {
	dictData := createMinimalSystemDictionary(&testing.T{})
	loader := NewDictionaryLoader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loader.LoadSystemDictionaryFromBytes(dictData)
	}
}
