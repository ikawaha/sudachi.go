package dic

import (
	"bytes"
	"testing"
	"time"
)

func TestHeaderVersionFromU64(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		expected HeaderVersion
		wantErr  bool
	}{
		{"SystemDictV1", SystemDictVersion1, SystemDictVersionV1, false},
		{"SystemDictV2", SystemDictVersion2, SystemDictVersionV2, false},
		{"UserDictV1", UserDictVersion1, UserDictVersionV1, false},
		{"UserDictV2", UserDictVersion2, UserDictVersionV2, false},
		{"UserDictV3", UserDictVersion3, UserDictVersionV3, false},
		{"Invalid", 0x1234567890abcdef, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := HeaderVersionFromU64(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("HeaderVersionFromU64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && version.ToU64() != tt.expected.ToU64() {
				t.Errorf("HeaderVersionFromU64() = %v, want %v", version, tt.expected)
			}
		})
	}
}

func TestSystemDictVersion(t *testing.T) {
	tests := []struct {
		version             SystemDictVersion
		expectedU64         uint64
		expectedHasGrammar  bool
		expectedHasSynonyms bool
		expectedIsSystem    bool
		expectedIsUser      bool
		expectedString      string
	}{
		{SystemDictVersionV1, SystemDictVersion1, true, false, true, false, "SystemDictV1"},
		{SystemDictVersionV2, SystemDictVersion2, true, true, true, false, "SystemDictV2"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedString, func(t *testing.T) {
			if tt.version.ToU64() != tt.expectedU64 {
				t.Errorf("ToU64() = 0x%016x, want 0x%016x", tt.version.ToU64(), tt.expectedU64)
			}
			if tt.version.HasGrammar() != tt.expectedHasGrammar {
				t.Errorf("HasGrammar() = %v, want %v", tt.version.HasGrammar(), tt.expectedHasGrammar)
			}
			if tt.version.HasSynonymGroupIds() != tt.expectedHasSynonyms {
				t.Errorf("HasSynonymGroupIds() = %v, want %v", tt.version.HasSynonymGroupIds(), tt.expectedHasSynonyms)
			}
			if tt.version.IsSystemDict() != tt.expectedIsSystem {
				t.Errorf("IsSystemDict() = %v, want %v", tt.version.IsSystemDict(), tt.expectedIsSystem)
			}
			if tt.version.IsUserDict() != tt.expectedIsUser {
				t.Errorf("IsUserDict() = %v, want %v", tt.version.IsUserDict(), tt.expectedIsUser)
			}
			if tt.version.String() != tt.expectedString {
				t.Errorf("String() = %q, want %q", tt.version.String(), tt.expectedString)
			}
		})
	}
}

func TestUserDictVersion(t *testing.T) {
	tests := []struct {
		version             UserDictVersion
		expectedU64         uint64
		expectedHasGrammar  bool
		expectedHasSynonyms bool
		expectedIsSystem    bool
		expectedIsUser      bool
		expectedString      string
	}{
		{UserDictVersionV1, UserDictVersion1, false, false, false, true, "UserDictV1"},
		{UserDictVersionV2, UserDictVersion2, true, false, false, true, "UserDictV2"},
		{UserDictVersionV3, UserDictVersion3, true, true, false, true, "UserDictV3"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedString, func(t *testing.T) {
			if tt.version.ToU64() != tt.expectedU64 {
				t.Errorf("ToU64() = 0x%016x, want 0x%016x", tt.version.ToU64(), tt.expectedU64)
			}
			if tt.version.HasGrammar() != tt.expectedHasGrammar {
				t.Errorf("HasGrammar() = %v, want %v", tt.version.HasGrammar(), tt.expectedHasGrammar)
			}
			if tt.version.HasSynonymGroupIds() != tt.expectedHasSynonyms {
				t.Errorf("HasSynonymGroupIds() = %v, want %v", tt.version.HasSynonymGroupIds(), tt.expectedHasSynonyms)
			}
			if tt.version.IsSystemDict() != tt.expectedIsSystem {
				t.Errorf("IsSystemDict() = %v, want %v", tt.version.IsSystemDict(), tt.expectedIsSystem)
			}
			if tt.version.IsUserDict() != tt.expectedIsUser {
				t.Errorf("IsUserDict() = %v, want %v", tt.version.IsUserDict(), tt.expectedIsUser)
			}
			if tt.version.String() != tt.expectedString {
				t.Errorf("String() = %q, want %q", tt.version.String(), tt.expectedString)
			}
		})
	}
}

func TestNewHeader(t *testing.T) {
	header := NewHeader()

	if header.Version != SystemDictVersionV2 {
		t.Errorf("NewHeader().Version = %v, want %v", header.Version, SystemDictVersionV2)
	}

	if header.Description != "" {
		t.Errorf("NewHeader().Description = %q, want empty string", header.Description)
	}

	// Check that create time is recent (within 10 seconds)
	now := uint64(time.Now().Unix())
	if header.CreateTime < now-10 || header.CreateTime > now+10 {
		t.Errorf("NewHeader().CreateTime = %d, expected close to %d", header.CreateTime, now)
	}
}

func TestNewHeaderWithVersion(t *testing.T) {
	version := UserDictVersionV3
	header := NewHeaderWithVersion(version)

	if header.Version != version {
		t.Errorf("NewHeaderWithVersion().Version = %v, want %v", header.Version, version)
	}
}

func TestHeaderSetTime(t *testing.T) {
	header := NewHeader()
	originalTime := time.Unix(int64(header.CreateTime), 0)

	newTime := time.Unix(1234567890, 0)
	returnedTime := header.SetTime(newTime)

	if header.CreateTime != 1234567890 {
		t.Errorf("SetTime() header.CreateTime = %d, want 1234567890", header.CreateTime)
	}

	if !returnedTime.Equal(originalTime) {
		t.Errorf("SetTime() returned %v, want %v", returnedTime, originalTime)
	}
}

func TestParseHeader(t *testing.T) {
	t.Run("valid header", func(t *testing.T) {
		// Create test data
		data := make([]byte, StorageSize)

		// Write version (SystemDictVersion2)
		writeU64(data[0:8], SystemDictVersion2)

		// Write create time (1234567890)
		writeU64(data[8:16], 1234567890)

		// Write description
		description := "Test Description"
		copy(data[16:], []byte(description))

		// Parse header
		header, err := ParseHeader(data)
		if err != nil {
			t.Fatalf("ParseHeader() unexpected error: %v", err)
		}

		if header.Version.ToU64() != SystemDictVersion2 {
			t.Errorf("ParseHeader().Version = %v, want SystemDictV2", header.Version)
		}

		if header.CreateTime != 1234567890 {
			t.Errorf("ParseHeader().CreateTime = %d, want 1234567890", header.CreateTime)
		}

		if header.Description != description {
			t.Errorf("ParseHeader().Description = %q, want %q", header.Description, description)
		}
	})

	t.Run("insufficient data", func(t *testing.T) {
		data := make([]byte, 10) // Too small

		_, err := ParseHeader(data)
		if err == nil {
			t.Error("ParseHeader() expected error for insufficient data")
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		data := make([]byte, StorageSize)

		// Write invalid version
		writeU64(data[0:8], 0x1234567890abcdef)

		_, err := ParseHeader(data)
		if err == nil {
			t.Error("ParseHeader() expected error for invalid version")
		}
	})
}

func TestHeaderWriteTo(t *testing.T) {
	t.Run("valid header", func(t *testing.T) {
		header := &Header{
			Version:     SystemDictVersionV1,
			CreateTime:  1234567890,
			Description: "Test Description",
		}

		data := make([]byte, StorageSize)
		err := header.WriteTo(data)
		if err != nil {
			t.Fatalf("WriteTo() unexpected error: %v", err)
		}

		// Parse it back
		parsedHeader, err := ParseHeader(data)
		if err != nil {
			t.Fatalf("ParseHeader() after WriteTo() unexpected error: %v", err)
		}

		if parsedHeader.Version.ToU64() != header.Version.ToU64() {
			t.Errorf("Round-trip Version = %v, want %v", parsedHeader.Version, header.Version)
		}

		if parsedHeader.CreateTime != header.CreateTime {
			t.Errorf("Round-trip CreateTime = %d, want %d", parsedHeader.CreateTime, header.CreateTime)
		}

		if parsedHeader.Description != header.Description {
			t.Errorf("Round-trip Description = %q, want %q", parsedHeader.Description, header.Description)
		}
	})

	t.Run("insufficient buffer", func(t *testing.T) {
		header := NewHeader()
		data := make([]byte, 10) // Too small

		err := header.WriteTo(data)
		if err == nil {
			t.Error("WriteTo() expected error for insufficient buffer")
		}
	})

	t.Run("description too long", func(t *testing.T) {
		header := &Header{
			Version:     SystemDictVersionV1,
			CreateTime:  1234567890,
			Description: string(make([]byte, DescriptionSize+1)), // Too long
		}

		data := make([]byte, StorageSize)
		err := header.WriteTo(data)
		if err == nil {
			t.Error("WriteTo() expected error for description too long")
		}
	})
}

func TestHeaderString(t *testing.T) {
	header := &Header{
		Version:     SystemDictVersionV2,
		CreateTime:  1234567890,
		Description: "Test Description",
	}

	str := header.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check that it contains expected components
	if !bytes.Contains([]byte(str), []byte("SystemDictV2")) {
		t.Error("String() should contain version")
	}

	if !bytes.Contains([]byte(str), []byte("Test Description")) {
		t.Error("String() should contain description")
	}
}

func TestWriteU64(t *testing.T) {
	data := make([]byte, 8)
	value := uint64(0x123456789abcdef0)

	err := writeU64(data, value)
	if err != nil {
		t.Fatalf("writeU64() unexpected error: %v", err)
	}

	// Check little-endian encoding
	expected := []byte{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12}
	if !bytes.Equal(data, expected) {
		t.Errorf("writeU64() = %v, want %v", data, expected)
	}

	// Test insufficient buffer
	smallData := make([]byte, 4)
	err = writeU64(smallData, value)
	if err == nil {
		t.Error("writeU64() expected error for insufficient buffer")
	}
}

func BenchmarkParseHeader(b *testing.B) {
	// Create test data
	data := make([]byte, StorageSize)
	writeU64(data[0:8], SystemDictVersion2)
	writeU64(data[8:16], 1234567890)
	copy(data[16:], []byte("Benchmark Test"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseHeader(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHeaderWriteTo(b *testing.B) {
	header := &Header{
		Version:     SystemDictVersionV2,
		CreateTime:  1234567890,
		Description: "Benchmark Test",
	}

	data := make([]byte, StorageSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := header.WriteTo(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
