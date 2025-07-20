package dic

import (
	"testing"
)

func TestWordIdCreation(t *testing.T) {
	tests := []struct {
		name     string
		dic      uint8
		word     uint32
		expected WordId
	}{
		{"system dict word 0", 0, 0, WordId{raw: 0}},
		{"system dict word 1", 0, 1, WordId{raw: 1}},
		{"system dict max word", 0, 0x0fffffff, WordId{raw: 0x0fffffff}},
		{"user dict 1", 1, 0, WordId{raw: 0x10000000}},
		{"user dict 14", 14, 0x0fffffff, WordId{raw: 0xefffffff}},
		{"oov dict", 15, 3121, WordId{raw: 0xf0000c31}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := New(tt.dic, tt.word)
			if id.raw != tt.expected.raw {
				t.Errorf("New(%d, %d) = %x, want %x", tt.dic, tt.word, id.raw, tt.expected.raw)
			}
			if id.Dic() != tt.dic {
				t.Errorf("Dic() = %d, want %d", id.Dic(), tt.dic)
			}
			if id.Word() != tt.word {
				t.Errorf("Word() = %d, want %d", id.Word(), tt.word)
			}
		})
	}
}

func TestFromRaw(t *testing.T) {
	tests := []struct {
		raw      uint32
		expected WordId
	}{
		{0, WordId{raw: 0}},
		{0x12345678, WordId{raw: 0x12345678}},
		{0xffffffff, WordId{raw: 0xffffffff}},
	}

	for _, tt := range tests {
		id := FromRaw(tt.raw)
		if id.raw != tt.expected.raw {
			t.Errorf("FromRaw(%x) = %x, want %x", tt.raw, id.raw, tt.expected.raw)
		}
	}
}

func TestChecked(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		validTests := []struct {
			dic  uint8
			word uint32
		}{
			{0, 0},
			{15, 0x0fffffff},
			{7, 12345},
		}

		for _, tt := range validTests {
			id, err := Checked(tt.dic, tt.word)
			if err != nil {
				t.Errorf("Checked(%d, %d) unexpected error: %v", tt.dic, tt.word, err)
			}
			if id.Dic() != tt.dic {
				t.Errorf("Checked(%d, %d) dic = %d, want %d", tt.dic, tt.word, id.Dic(), tt.dic)
			}
			if id.Word() != tt.word {
				t.Errorf("Checked(%d, %d) word = %d, want %d", tt.dic, tt.word, id.Word(), tt.word)
			}
		}
	})

	t.Run("invalid dictionary ID", func(t *testing.T) {
		_, err := Checked(16, 0)
		if err == nil {
			t.Error("Checked(16, 0) expected error for invalid dictionary ID")
		}
	})

	t.Run("invalid word ID", func(t *testing.T) {
		_, err := Checked(0, 0x10000000) // exceeds 28 bits
		if err == nil {
			t.Error("Checked(0, 0x10000000) expected error for invalid word ID")
		}
	})
}

func TestOOV(t *testing.T) {
	tests := []uint32{0, 1, 12345, 0x0fffffff}

	for _, posId := range tests {
		id := OOV(posId)
		if !id.IsOOV() {
			t.Errorf("OOV(%d).IsOOV() = false, want true", posId)
		}
		if id.Dic() != 15 {
			t.Errorf("OOV(%d).Dic() = %d, want 15", posId, id.Dic())
		}
		if id.Word() != posId {
			t.Errorf("OOV(%d).Word() = %d, want %d", posId, id.Word(), posId)
		}
	}
}

func TestIsSystem(t *testing.T) {
	tests := []struct {
		id       WordId
		expected bool
	}{
		{New(0, 0), true},
		{New(0, 12345), true},
		{New(1, 0), false},
		{New(14, 0), false},
		{New(15, 0), false},
	}

	for _, tt := range tests {
		if result := tt.id.IsSystem(); result != tt.expected {
			t.Errorf("WordId(%d, %d).IsSystem() = %v, want %v",
				tt.id.Dic(), tt.id.Word(), result, tt.expected)
		}
	}
}

func TestIsUser(t *testing.T) {
	tests := []struct {
		id       WordId
		expected bool
	}{
		{New(0, 0), false},
		{New(1, 0), true},
		{New(14, 0), true},
		{New(15, 0), false},
	}

	for _, tt := range tests {
		if result := tt.id.IsUser(); result != tt.expected {
			t.Errorf("WordId(%d, %d).IsUser() = %v, want %v",
				tt.id.Dic(), tt.id.Word(), result, tt.expected)
		}
	}
}

func TestIsOOV(t *testing.T) {
	tests := []struct {
		id       WordId
		expected bool
	}{
		{New(0, 0), false},
		{New(1, 0), false},
		{New(14, 0), false},
		{New(15, 0), true},
		{OOV(12345), true},
	}

	for _, tt := range tests {
		if result := tt.id.IsOOV(); result != tt.expected {
			t.Errorf("WordId(%d, %d).IsOOV() = %v, want %v",
				tt.id.Dic(), tt.id.Word(), result, tt.expected)
		}
	}
}

func TestIsSpecial(t *testing.T) {
	tests := []struct {
		id       WordId
		expected bool
	}{
		{EOS, true},
		{BOS, true},
		{Invalid, false},
		{New(0, 0), false},
		{New(15, 12345), false},
	}

	for _, tt := range tests {
		if result := tt.id.IsSpecial(); result != tt.expected {
			t.Errorf("WordId{%x}.IsSpecial() = %v, want %v",
				tt.id.raw, result, tt.expected)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		id       WordId
		expected string
	}{
		{New(0, 521321), "(0, 521321)"},
		{New(1, 12345), "(1, 12345)"},
		{OOV(42), "(-1, 42)"},
		{BOS, "(-1, 268435454)"}, // BOS.Word() is masked to 28 bits
	}

	for _, tt := range tests {
		if result := tt.id.String(); result != tt.expected {
			t.Errorf("WordId{%x}.String() = %q, want %q",
				tt.id.raw, result, tt.expected)
		}
	}
}

func TestSpecialConstants(t *testing.T) {
	t.Run("special constants values", func(t *testing.T) {
		if Invalid.raw != 0xffffffff {
			t.Errorf("Invalid.raw = %x, want 0xffffffff", Invalid.raw)
		}
		if BOS.raw != 0xfffffffe {
			t.Errorf("BOS.raw = %x, want 0xfffffffe", BOS.raw)
		}
		if EOS.raw != 0xfffffffd {
			t.Errorf("EOS.raw = %x, want 0xfffffffd", EOS.raw)
		}
	})

	t.Run("special constants properties", func(t *testing.T) {
		if !EOS.IsSpecial() {
			t.Error("EOS.IsSpecial() = false, want true")
		}
		if !BOS.IsSpecial() {
			t.Error("BOS.IsSpecial() = false, want true")
		}
		if Invalid.IsSpecial() {
			t.Error("Invalid.IsSpecial() = true, want false")
		}
	})
}

func TestEqual(t *testing.T) {
	id1 := New(0, 12345)
	id2 := New(0, 12345)
	id3 := New(1, 12345)

	if !id1.Equal(id2) {
		t.Error("Equal WordIds should be equal")
	}
	if id1.Equal(id3) {
		t.Error("Different WordIds should not be equal")
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		id1      WordId
		id2      WordId
		expected int
	}{
		{New(0, 100), New(0, 200), -1},
		{New(0, 200), New(0, 100), 1},
		{New(0, 100), New(0, 100), 0},
		{New(0, 100), New(1, 50), -1}, // lower dic ID
		{New(2, 50), New(1, 100), 1},  // higher dic ID
	}

	for _, tt := range tests {
		result := tt.id1.Compare(tt.id2)
		if result != tt.expected {
			t.Errorf("WordId(%d,%d).Compare(WordId(%d,%d)) = %d, want %d",
				tt.id1.Dic(), tt.id1.Word(), tt.id2.Dic(), tt.id2.Word(),
				result, tt.expected)
		}
	}
}

func BenchmarkWordIdCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New(uint8(i%16), uint32(i%1000000))
	}
}

func BenchmarkWordIdAccess(b *testing.B) {
	id := New(5, 123456)

	b.Run("Dic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.Dic()
		}
	})

	b.Run("Word", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.Word()
		}
	})

	b.Run("IsSystem", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.IsSystem()
		}
	})
}
