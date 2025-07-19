package input

import (
	"strings"
)

// ReplaceOp represents a single edit operation
// This matches Rust implementation: ReplaceOp<'a>
type ReplaceOp struct {
	What Range  // Range to replace
	With string // Replacement text
}

// NewReplaceOp creates a new replace operation
func NewReplaceOp(what Range, with string) ReplaceOp {
	return ReplaceOp{
		What: what,
		With: with,
	}
}

// InputEditor provides methods to modify input text with proper mapping tracking
// This matches Rust implementation: InputEditor<'a>
type InputEditor struct {
	replaces []ReplaceOp
}

// NewInputEditor creates a new input editor
func NewInputEditor() *InputEditor {
	return &InputEditor{
		replaces: make([]ReplaceOp, 0),
	}
}

// ReplaceRange replaces a range with the given text
func (ie *InputEditor) ReplaceRange(rangeToReplace Range, replacement string) {
	op := NewReplaceOp(rangeToReplace, replacement)
	ie.replaces = append(ie.replaces, op)
}

// ReplaceChar replaces a range with a single character
func (ie *InputEditor) ReplaceChar(rangeToReplace Range, replacement rune) {
	ie.ReplaceRange(rangeToReplace, string(replacement))
}

// GetReplaces returns the list of replace operations
func (ie *InputEditor) GetReplaces() []ReplaceOp {
	return ie.replaces
}

// ResolveEdits applies all edit operations and updates mappings
// This matches Rust implementation: resolve_edits() exactly
func ResolveEdits(
	source string,
	sourceMapping []int,
	edits []ReplaceOp,
) (string, []int, int) {
	start := 0
	curLen := len(source) // matching Rust: cur_len

	var target strings.Builder
	var targetMapping []int // matching Rust: target_mapping

	for _, edit := range edits {
		// Copy unchanged part before the edit (matching Rust: target.push_str(&source[start..edit.what.start]))
		if start < edit.What.Start {
			unchangedPart := source[start:edit.What.Start]
			target.WriteString(unchangedPart)

			// Copy the corresponding mapping (matching Rust: target_mapping.extend(source_mapping[start..edit.what.start].iter()))
			for i := start; i < edit.What.Start; i++ {
				targetMapping = append(targetMapping, sourceMapping[i])
			}
		}

		// Apply the replacement (matching Rust: start = edit.what.end)
		start = edit.What.End
		lengthChange := addReplace(sourceMapping, &target, &targetMapping, edit.What, edit.With)
		curLen += lengthChange
	}

	// Copy the remaining part after all edits (matching Rust: target.push_str(&source[start..]))
	if start < len(source) {
		remainingPart := source[start:]
		target.WriteString(remainingPart)
	}

	// Copy corresponding mapping (matching Rust: target_mapping.extend(source_mapping[start..].iter()))
	// This includes all remaining mapping entries, including the sentinel beyond len(source)
	for i := start; i < len(sourceMapping); i++ {
		targetMapping = append(targetMapping, sourceMapping[i])
	}

	// The First byte of mapping MUST be 0 (matching Rust: if let Some(v) = target_mapping.first_mut() { *v = 0; })
	if len(targetMapping) > 0 {
		(targetMapping)[0] = 0
	}

	return target.String(), targetMapping, curLen
}

// addReplace handles a single replacement operation with mapping update
// This exactly matches Rust implementation: fn add_replace()
func addReplace(
	sourceMapping []int,
	target *strings.Builder,
	targetMapping *[]int,
	what Range,
	with string,
) int {
	if with == "" { // matching Rust: if with.is_empty()
		return -(what.End - what.Start) // matching Rust: return -(what.len() as isize)
	}

	target.WriteString(with) // matching Rust: target.push_str(with)

	// The first char of replacing string will correspond with whole replaced string
	// (matching Rust comment exactly)
	*targetMapping = append(*targetMapping, sourceMapping[what.Start]) // matching Rust: target_mapping.push(source_mapping[what.start])
	pos := sourceMapping[what.End]                                     // matching Rust: let pos = source_mapping[what.end]
	for i := 1; i < len(with); i++ {                                   // matching Rust: for _ in 1..with.len()
		*targetMapping = append(*targetMapping, pos) // matching Rust: target_mapping.push(pos)
	}

	return len(with) - (what.End - what.Start) // matching Rust: with.len() as isize - what.len() as isize
}
