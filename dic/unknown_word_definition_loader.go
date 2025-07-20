package dic

import (
	"os"
	"path/filepath"

	error2 "github.com/ikawaha/sudachi.go/error"
)

// LoadUnknownWordDefinitionsFromFile loads unknown word definitions from unk.def file
func LoadUnknownWordDefinitionsFromFile(unkDefPath string, charCategory *CharacterCategory) (*UnknownWordDefinitions, error) {
	file, err := os.Open(unkDefPath)
	if err != nil {
		return nil, error2.NewErrWithContext(
			"failed to open unk.def file",
			err.Error())
	}
	defer file.Close()

	uwd := NewUnknownWordDefinitions()
	if err := uwd.LoadFromReader(file, charCategory); err != nil {
		return nil, error2.NewErrWithContext(
			"failed to parse unk.def file",
			err.Error())
	}

	return uwd, nil
}

// LoadUnknownWordDefinitionsFromResourceDir loads unknown word definitions from resource directory
func LoadUnknownWordDefinitionsFromResourceDir(resourceDir string, charCategory *CharacterCategory) (*UnknownWordDefinitions, error) {
	unkDefPath := filepath.Join(resourceDir, "unk.def")
	return LoadUnknownWordDefinitionsFromFile(unkDefPath, charCategory)
}
