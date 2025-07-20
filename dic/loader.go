package dic

import (
	"errors"
	"fmt"
	"path/filepath"
)

// DictionaryLoader handles loading of dictionary files
type DictionaryLoader struct{}

// NewDictionaryLoader creates a new dictionary loader
func NewDictionaryLoader() *DictionaryLoader {
	return &DictionaryLoader{}
}

// LoadSystemDictionary loads a system dictionary from the file
func (dl *DictionaryLoader) LoadSystemDictionary(path string) (*SystemDictionary, error) {
	storage, err := NewFileStorage(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load system dictionary: path: %s, error: %w", path, err)
	}

	return dl.LoadSystemDictionaryFromStorage(storage)
}

// LoadSystemDictionaryFromStorage loads a system dictionary from storage
func (dl *DictionaryLoader) LoadSystemDictionaryFromStorage(storage Storage) (*SystemDictionary, error) {
	data := storage.Data()

	// Parse header
	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system dictionary header: %w", err)
	}

	// Validate that this is a system dictionary
	if !header.Version.IsSystemDict() {
		return nil, fmt.Errorf("invalid dictionary type: expected system dictionary, got %s", header.Version.String())
	}

	// Parse dictionary sections
	sections, err := dl.parseDictionarySections(data, header)
	if err != nil {
		return nil, err
	}

	// Create LexiconSet with the system lexicon
	numSystemPOS := 0
	if sections.grammar != nil {
		numSystemPOS = sections.grammar.POSListSize()
	}
	lexiconSet := NewLexiconSet(sections.lexicon, numSystemPOS)

	return &SystemDictionary{
		header:     header,
		data:       data,
		trie:       sections.trie,
		lexiconSet: lexiconSet,
		grammar:    sections.grammar,
	}, nil
}

// LoadUserDictionary loads a user dictionary from the file
func (dl *DictionaryLoader) LoadUserDictionary(path string) (*UserDictionary, error) {
	storage, err := NewFileStorage(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load user dictionary: path %s", path)
	}

	return dl.LoadUserDictionaryFromStorage(storage)
}

// LoadUserDictionaryFromStorage loads a user dictionary from storage
func (dl *DictionaryLoader) LoadUserDictionaryFromStorage(storage Storage) (*UserDictionary, error) {
	data := storage.Data()

	// Parse header
	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user dictionary header: %w", err)
	}

	// Validate that this is a user dictionary
	if !header.Version.IsUserDict() {
		return nil, fmt.Errorf("invalid dictionary type: expected user dictionary, got %s", header.Version)
	}

	// Parse dictionary sections
	sections, err := dl.parseDictionarySections(data, header)
	if err != nil {
		return nil, err
	}

	return &UserDictionary{
		header:  header,
		data:    data,
		trie:    sections.trie,
		lexicon: sections.lexicon,
		grammar: sections.grammar,
	}, nil
}

// LoadSystemDictionaryFromBytes loads a system dictionary from byte slice
func (dl *DictionaryLoader) LoadSystemDictionaryFromBytes(data []byte) (*SystemDictionary, error) {
	storage := NewBorrowedStorage(data)
	return dl.LoadSystemDictionaryFromStorage(storage)
}

// LoadUserDictionaryFromBytes loads a user dictionary from byte slice
func (dl *DictionaryLoader) LoadUserDictionaryFromBytes(data []byte) (*UserDictionary, error) {
	storage := NewBorrowedStorage(data)
	return dl.LoadUserDictionaryFromStorage(storage)
}

// dictionarySections represents the parsed sections of a dictionary
type dictionarySections struct {
	trie        *Trie
	wordIdTable *WordIdTable
	wordParams  *WordParams
	wordInfos   *WordInfos
	lexicon     *Lexicon
	grammar     *Grammar
}

// parseDictionarySections parses the sections of a dictionary after the header
func (dl *DictionaryLoader) parseDictionarySections(data []byte, header *Header) (*dictionarySections, error) {
	// Skip the header
	offset := StorageSize

	var grammar *Grammar

	// Parse a grammar section if present (for system dictionaries)
	if header.Version.HasGrammar() {
		var err error
		grammar, err = NewGrammar(data, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grammar: %w", err)
		}
		offset += grammar.StorageSize()
	}

	// Now parse lexicon section
	reader, err := NewReaderAt(data, offset)
	if err != nil {
		return nil, err
	}

	// Parse trie section (first part of lexicon)
	trieSize, err := reader.ReadU32()
	if err != nil {
		return nil, fmt.Errorf("failed to read trie size: %w", err)
	}

	// Validate trie size
	if trieSize == 0 {
		return nil, errors.New("invalid trie size: tire size cannot be zero")
	}

	// Read trie data
	trieData, err := reader.ReadSlice(int(trieSize) * 4) // u32 elements
	if err != nil {
		return nil, fmt.Errorf("failed to read trie data: %w", err)
	}

	// Create trie
	trie, err := NewTrie(trieData, 0, int(trieSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create trie: %w", err)
	}

	// Parse WordIdTable
	wordIdTableSize, err := reader.ReadU32()
	if err != nil {
		return nil, fmt.Errorf("failed to read word id table size: %w", err)
	}

	wordIdTable := NewWordIdTable(data, wordIdTableSize, reader.Offset())
	err = reader.Skip(int(wordIdTableSize))
	if err != nil {
		return nil, fmt.Errorf("failed to skip word id table: %w", err)
	}

	// Parse WordParams
	wordParamsSize, err := reader.ReadU32()
	if err != nil {
		return nil, fmt.Errorf("failed to read word params size: %w", err)
	}

	wordParams := NewWordParams(data, wordParamsSize, reader.Offset())
	err = reader.Skip(wordParams.StorageSize() - 4) // Skip data, but size was already read
	if err != nil {
		return nil, fmt.Errorf("failed to skip word params: %w", err)
	}

	// Parse WordInfos (rest of the data)
	wordInfos := NewWordInfos(data, reader.Offset(), wordParams.Size(), header.Version.HasSynonymGroupIds())

	// Create complete lexicon
	lex := NewLexicon(trie, wordIdTable, wordParams, wordInfos)

	// Set dictionary ID (0 for system dictionary)
	err = lex.SetDicID(0)
	if err != nil {
		return nil, fmt.Errorf("failed to set lexicon dictionary ID: %w", err)
	}

	return &dictionarySections{
		trie:        trie,
		wordIdTable: wordIdTable,
		wordParams:  wordParams,
		wordInfos:   wordInfos,
		lexicon:     lex,
		grammar:     grammar,
	}, nil
}

// getGrammarSize calculates the size of the grammar section
// This method is deprecated - use NewGrammar instead
func (dl *DictionaryLoader) getGrammarSize(data []byte, offset int) (int, error) {
	// Create a temporary grammar to get its size
	grammar, err := NewGrammar(data, offset)
	if err != nil {
		return 0, err
	}
	return grammar.StorageSize(), nil
}

// ValidateDictionaryFile validates a dictionary file without fully loading it
func ValidateDictionaryFile(path string) error {
	// Check file extension
	ext := filepath.Ext(path)
	if ext != ".dic" {
		return fmt.Errorf("invalid file extension: expected .dic, got %s", ext)
	}

	// Try to load just the header
	storage, err := NewFileStorage(path)
	if err != nil {
		return err
	}
	defer storage.Close()

	data := storage.Data()
	if len(data) < StorageSize {
		return fmt.Errorf("insufficient data: need %d bytes, have %d", StorageSize, len(data))
	}

	// Parse and validate header
	_, err = ParseHeader(data)
	if err != nil {
		return fmt.Errorf("invalid dictionary header: %w", err)
	}

	// Basic validation - check if we have enough data for at least a trie size
	if len(data) < StorageSize+4 {
		return fmt.Errorf("dictionary file too small: file size %d, minimum required %d", len(data), StorageSize+4)
	}

	// For validation, we don't need to parse everything - just check the basic structure
	// The detailed parsing will be done by the loader
	return nil
}

// GetDictionaryInfo returns information about a dictionary file without fully loading it
func GetDictionaryInfo(path string) (*DictionaryInfo, error) {
	storage, err := NewFileStorage(path)
	if err != nil {
		return nil, err
	}
	defer storage.Close()

	data := storage.Data()
	header, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}

	return &DictionaryInfo{
		Version:     header.Version,
		CreateTime:  header.CreateTime,
		Description: header.Description,
		Size:        len(data),
		HasGrammar:  header.Version.HasGrammar(),
		IsSystem:    header.Version.IsSystemDict(),
	}, nil
}
