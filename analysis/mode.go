package analysis

import (
	"fmt"
	"strings"

	"github.com/ikawaha/sudachi.go/lattice"
)

// Mode represents the unit to split text
//
// Some examples:
//
//	A：選挙/管理/委員/会
//	B：選挙/管理/委員会
//	C：選挙管理委員会
//
//	A：客室/乗務/員
//	B：客室/乗務員
//	C：客室乗務員
//
//	A：労働/者/協同/組合
//	B：労働者/協同/組合
//	C：労働者協同組合
//
//	A：機能/性/食品
//	B：機能性/食品
//	C：機能性食品
//
// See Sudachi documentation for more details
type Mode int

const (
	// ModeA represents short unit tokenization
	ModeA Mode = Mode(lattice.ModeA)

	// ModeB represents middle unit tokenization (word-like)
	ModeB = Mode(lattice.ModeB)

	// ModeC represents long unit tokenization (named entity)
	ModeC = Mode(lattice.ModeC)
)

// String returns the string representation of the mode
func (m Mode) String() string {
	switch m {
	case ModeA:
		return "A"
	case ModeB:
		return "B"
	case ModeC:
		return "C"
	default:
		return "Unknown"
	}
}

// ParseMode parses a string into a Mode
func ParseMode(s string) (Mode, error) {
	switch strings.ToUpper(s) {
	case "A":
		return ModeA, nil
	case "B":
		return ModeB, nil
	case "C":
		return ModeC, nil
	default:
		return Mode(0), fmt.Errorf(`mode must be one of "A", "B", or "C" (in lower or upper case), got: %s`, s)
	}
}

// MarshalText implements the text marshaling interface
func (m Mode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements the text unmarshalling interface
func (m *Mode) UnmarshalText(text []byte) error {
	mode, err := ParseMode(string(text))
	if err != nil {
		return err
	}
	*m = mode
	return nil
}
