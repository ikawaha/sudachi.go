package config

import (
	"os"
	"strings"
	"testing"
)

// TestResolvePaths_Exe tests $exe path resolution (Rust version: resolve_exe)
func TestResolvePaths_Exe(t *testing.T) {
	// Create a minimal config directly for testing
	cfg := &Config{
		resolver: NewPathResolver(1),
	}

	paths := cfg.ResolvePaths("$exe/data")
	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	// Get current executable directory for comparison
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}
	execDir := execPath[:strings.LastIndex(execPath, "/")]

	// First path should start with executable directory + "/deps"
	if !strings.Contains(paths[0], execDir) {
		t.Errorf("First path should contain exec dir %s, got %s", execDir, paths[0])
	}
	if !strings.Contains(paths[0], "/deps") {
		t.Errorf("First path should contain '/deps', got %s", paths[0])
	}

	// Second path should start with executable directory
	if !strings.HasPrefix(paths[1], execDir) {
		t.Errorf("Second path should start with exec dir %s, got %s", execDir, paths[1])
	}
}

// TestResolvePaths_Cfg tests $cfg path resolution (Rust version: resolve_cfg)
func TestResolvePaths_Cfg(t *testing.T) {
	// Create a config with a resource directory in the resolver
	resolver := NewPathResolver(1)
	resolver.Add("resources")
	cfg := &Config{
		resolver: resolver,
	}

	paths := cfg.ResolvePaths("$cfg/data")
	if len(paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(paths))
	}

	// Path should end with "resources/data" (default resource directory)
	if !strings.HasSuffix(paths[0], "resources/data") {
		t.Errorf("Path should end with 'resources/data', got %s", paths[0])
	}
}

// TestBuilder_Fallback tests configuration fallback (Rust version: config_builder_fallback)
func TestBuilder_Fallback(t *testing.T) {
	// Create first builder with a path set
	testPath := "test"
	builder1 := &ConfigBuilder{
		PathField: &testPath,
	}

	// Create empty second builder
	builder2 := &ConfigBuilder{}

	// Apply fallback
	result := builder2.Fallback(builder1)

	// The result should have the path from builder1
	if result.PathField == nil || *result.PathField != "test" {
		t.Errorf("Expected path 'test', got '%v'", result.PathField)
	}
}

// TestParseSurfaceProjection tests surface projection parsing (Rust version: surface_projection_tryfrom)
func TestParseSurfaceProjection(t *testing.T) {
	tests := []struct {
		input    string
		expected SurfaceProjection
		hasError bool
	}{
		{"surface", Surface, false},
		{"normalized", Normalized, false},
		{"reading", Reading, false},
		{"dictionary", Dictionary, false},
		{"dictionary_and_surface", DictionaryAndSurface, false},
		{"normalized_and_surface", NormalizedAndSurface, false},
		{"normalized_nouns", NormalizedNouns, false},
		{"invalid", Surface, true}, // Should return error for unknown projection
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := ParseSurfaceProjection(test.input)

			if test.hasError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", test.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input '%s': %v", test.input, err)
				}
				if result != test.expected {
					t.Errorf("Expected %v for input '%s', got %v", test.expected, test.input, result)
				}
			}
		})
	}
}

// TestResolvePaths_RegularPath tests regular path resolution (no special prefixes)
func TestResolvePaths_RegularPath(t *testing.T) {
	// Create a minimal config for testing
	cfg := &Config{
		resolver: NewPathResolver(1),
	}

	paths := cfg.ResolvePaths("regular/path")
	if len(paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(paths))
	}
	if paths[0] != "regular/path" {
		t.Errorf("Expected 'regular/path', got '%s'", paths[0])
	}
}

// TestSurfaceProjection_String tests String() method for SurfaceProjection
func TestSurfaceProjection_String(t *testing.T) {
	tests := []struct {
		projection SurfaceProjection
		expected   string
	}{
		{Surface, "surface"},
		{Normalized, "normalized"},
		{Reading, "reading"},
		{Dictionary, "dictionary"},
		{DictionaryAndSurface, "dictionary_and_surface"},
		{NormalizedAndSurface, "normalized_and_surface"},
		{NormalizedNouns, "normalized_nouns"},
		{SurfaceProjection(99), "unknown"}, // Invalid value
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.projection.String()
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}
