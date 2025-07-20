package dic

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Storage represents different types of dictionary data storage
type Storage interface {
	// Data returns the underlying byte slice
	Data() []byte
	// Close releases any resources (for memory-mapped files)
	Close() error
}

// FileStorage represents memory-mapped file storage
type FileStorage struct {
	file *os.File
	data []byte
}

// NewFileStorage creates a new memory-mapped file storage
func NewFileStorage(path string) (*FileStorage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open dictionary file: path: %s, error: %w", path, err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get file info: path: %s, error: %w", path, err)
	}

	size := int(stat.Size())
	if size == 0 {
		file.Close()
		return nil, fmt.Errorf("empty dictionary file: path: %s", path)
	}

	// Memory map the file
	data, err := syscall.Mmap(int(file.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to memory map file: path: %s, size: %d, error: %w", path, size, err)
	}

	return &FileStorage{
		file: file,
		data: data,
	}, nil
}

func (fs *FileStorage) Data() []byte {
	return fs.data
}

func (fs *FileStorage) Close() error {
	var err error
	if fs.data != nil {
		if unmapErr := syscall.Munmap(fs.data); unmapErr != nil {
			err = unmapErr
		}
		fs.data = nil
	}
	if fs.file != nil {
		if closeErr := fs.file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		fs.file = nil
	}
	return err
}

// OwnedStorage represents owned byte slice storage
type OwnedStorage struct {
	data []byte
}

// NewOwnedStorage creates a new owned storage from byte slice
func NewOwnedStorage(data []byte) *OwnedStorage {
	// Make a copy to ensure we own the data
	copied := make([]byte, len(data))
	copy(copied, data)
	return &OwnedStorage{data: copied}
}

func (os *OwnedStorage) Data() []byte {
	return os.data
}

func (os *OwnedStorage) Close() error {
	// No resources to release for owned storage
	return nil
}

// BorrowedStorage represents borrowed byte slice storage
type BorrowedStorage struct {
	data []byte
}

// NewBorrowedStorage creates a new borrowed storage from byte slice
// WARNING: The caller must ensure the data remains valid for the lifetime of this storage
func NewBorrowedStorage(data []byte) *BorrowedStorage {
	return &BorrowedStorage{data: data}
}

func (bs *BorrowedStorage) Data() []byte {
	return bs.data
}

func (bs *BorrowedStorage) Close() error {
	// No resources to release for borrowed storage
	return nil
}

// SudachiDicData manages multiple dictionary storages
type SudachiDicData struct {
	// System dictionary (required)
	system Storage
	// User dictionaries (optional, multiple allowed)
	user []Storage
}

// NewSudachiDicData creates a new dictionary data manager
func NewSudachiDicData(system Storage) *SudachiDicData {
	return &SudachiDicData{
		system: system,
		user:   make([]Storage, 0),
	}
}

// AddUser adds a user dictionary
func (sdd *SudachiDicData) AddUser(user Storage) {
	sdd.user = append(sdd.user, user)
}

// System returns the system dictionary data
func (sdd *SudachiDicData) System() []byte {
	return sdd.system.Data()
}

// User returns user dictionary data by index
func (sdd *SudachiDicData) User(index int) ([]byte, error) {
	if index < 0 || index >= len(sdd.user) {
		return nil, fmt.Errorf("index out of bouds [0, %d): %d", len(sdd.user), index)
	}
	return sdd.user[index].Data(), nil
}

// UserCount returns the number of user dictionaries
func (sdd *SudachiDicData) UserCount() int {
	return len(sdd.user)
}

// AllUsers returns all user dictionary data
func (sdd *SudachiDicData) AllUsers() [][]byte {
	result := make([][]byte, len(sdd.user))
	for i, storage := range sdd.user {
		result[i] = storage.Data()
	}
	return result
}

// Close releases all resources
func (sdd *SudachiDicData) Close() error {
	var errs error
	// Close system dictionary
	if err := sdd.system.Close(); err != nil {
		errs = errors.Join(errs, err)
	}
	// Close user dictionaries
	for _, user := range sdd.user {
		if err := user.Close(); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}
