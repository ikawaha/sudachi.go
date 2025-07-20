package dic

import (
	"fmt"
)

// Dictionary represents a complete Sudachi dictionary system
type Dictionary struct {
	// System dictionary (required)
	system *SystemDictionary
	// User dictionaries (optional, multiple allowed)
	users []*UserDictionary
}

// SystemDictionary represents a system dictionary
type SystemDictionary struct {
	header     *Header
	data       []byte
	trie       *Trie
	lexiconSet *LexiconSet // Lexicon set for dictionary support (Rust compatible)
	grammar    *Grammar
}

// UserDictionary represents a user dictionary
type UserDictionary struct {
	header  *Header
	data    []byte
	trie    *Trie
	lexicon *Lexicon
	grammar *Grammar // User dictionaries may have grammar too
}

// NewDictionary creates a new dictionary from system and user dictionaries
func NewDictionary(system *SystemDictionary, users []*UserDictionary) *Dictionary {
	return &Dictionary{
		system: system,
		users:  users,
	}
}

// SystemDict returns the system dictionary
func (d *Dictionary) SystemDict() *SystemDictionary {
	return d.system
}

// UserDicts returns all user dictionaries
func (d *Dictionary) UserDicts() []*UserDictionary {
	return d.users
}

// UserDict returns a specific user dictionary by index
func (d *Dictionary) UserDict(index int) (*UserDictionary, error) {
	if index < 0 || index >= len(d.users) {
		return nil, fmt.Errorf("index out of bounds [0, %d): %d", len(d.users), index)
	}
	return d.users[index], nil
}

// SystemDictionary methods

// Header returns the header of the system dictionary
func (sd *SystemDictionary) Header() *Header {
	return sd.header
}

// Data returns the raw data of the system dictionary
func (sd *SystemDictionary) Data() []byte {
	return sd.data
}

// Trie returns the trie structure for prefix matching
func (sd *SystemDictionary) Trie() *Trie {
	return sd.trie
}

// LexiconSet returns the lexicon set for multiple dictionary support
func (sd *SystemDictionary) LexiconSet() *LexiconSet {
	return sd.lexiconSet
}

// Grammar returns the grammar for morphological analysis
func (sd *SystemDictionary) Grammar() *Grammar {
	return sd.grammar
}

// Size returns the size of the system dictionary data in bytes
func (sd *SystemDictionary) Size() int {
	return len(sd.data)
}

// Version returns the version of the system dictionary
func (sd *SystemDictionary) Version() HeaderVersion {
	return sd.header.Version
}

// UserDictionary methods

// Header returns the header of the user dictionary
func (ud *UserDictionary) Header() *Header {
	return ud.header
}

// Data returns the raw data of the user dictionary
func (ud *UserDictionary) Data() []byte {
	return ud.data
}

// Trie returns the trie structure for prefix matching
func (ud *UserDictionary) Trie() *Trie {
	return ud.trie
}

// Lexicon returns the complete lexicon for word lookup
func (ud *UserDictionary) Lexicon() *Lexicon {
	return ud.lexicon
}

// Grammar returns the grammar (may be nil for user dictionaries)
func (ud *UserDictionary) Grammar() *Grammar {
	return ud.grammar
}

// Size returns the size of the user dictionary data in bytes
func (ud *UserDictionary) Size() int {
	return len(ud.data)
}

// Version returns the version of the user dictionary
func (ud *UserDictionary) Version() HeaderVersion {
	return ud.header.Version
}

// LoadedDictionary represents a fully loaded dictionary with all components (Rust compatible)
type LoadedDictionary struct {
	grammar    *Grammar
	lexiconSet *LexiconSet
}

// NewLoadedDictionary creates a new loaded dictionary (Rust compatible)
func NewLoadedDictionary(grammar *Grammar, lexiconSet *LexiconSet) *LoadedDictionary {
	return &LoadedDictionary{
		grammar:    grammar,
		lexiconSet: lexiconSet,
	}
}

// Grammar returns the grammar
func (ld *LoadedDictionary) Grammar() *Grammar {
	return ld.grammar
}

// LexiconSet returns the lexicon set
func (ld *LoadedDictionary) LexiconSet() *LexiconSet {
	return ld.lexiconSet
}

// DictionaryInfo provides information about a dictionary
type DictionaryInfo struct {
	Version     HeaderVersion
	CreateTime  uint64
	Description string
	Size        int
	HasGrammar  bool
	IsSystem    bool
}

// Info returns information about the system dictionary
func (sd *SystemDictionary) Info() *DictionaryInfo {
	return &DictionaryInfo{
		Version:     sd.header.Version,
		CreateTime:  sd.header.CreateTime,
		Description: sd.header.Description,
		Size:        len(sd.data),
		HasGrammar:  sd.header.Version.HasGrammar(),
		IsSystem:    sd.header.Version.IsSystemDict(),
	}
}

// Info returns information about the user dictionary
func (ud *UserDictionary) Info() *DictionaryInfo {
	return &DictionaryInfo{
		Version:     ud.header.Version,
		CreateTime:  ud.header.CreateTime,
		Description: ud.header.Description,
		Size:        len(ud.data),
		HasGrammar:  ud.header.Version.HasGrammar(),
		IsSystem:    ud.header.Version.IsSystemDict(),
	}
}

// String returns a string representation of dictionary info
func (di *DictionaryInfo) String() string {
	dictType := "User"
	if di.IsSystem {
		dictType = "System"
	}
	return fmt.Sprintf("%s Dictionary: %s, Size: %d bytes, Grammar: %v, Description: %q",
		dictType, di.Version.String(), di.Size, di.HasGrammar, di.Description)
}
