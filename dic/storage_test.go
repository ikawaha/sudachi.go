package dic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOwnedStorage(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	storage := NewOwnedStorage(data)

	// Should return a copy, not the original
	result := storage.Data()
	if len(result) != len(data) {
		t.Errorf("Data() length = %d, want %d", len(result), len(data))
	}

	for i, b := range result {
		if b != data[i] {
			t.Errorf("Data()[%d] = %d, want %d", i, b, data[i])
		}
	}

	// Modify original data - should not affect storage
	data[0] = 99
	if storage.Data()[0] == 99 {
		t.Error("OwnedStorage should not be affected by modifications to original data")
	}

	// Close should not error
	if err := storage.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestBorrowedStorage(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	storage := NewBorrowedStorage(data)

	// Should return the same slice
	result := storage.Data()
	if len(result) != len(data) {
		t.Errorf("Data() length = %d, want %d", len(result), len(data))
	}

	// Modify original data - should affect storage
	data[0] = 99
	if storage.Data()[0] != 99 {
		t.Error("BorrowedStorage should be affected by modifications to original data")
	}

	// Close should not error
	if err := storage.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestFileStorage(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.dic")

	testData := []byte("Hello, World! This is test dictionary data.")
	err := os.WriteFile(tmpFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test successful loading
	storage, err := NewFileStorage(tmpFile)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v, want nil", err)
	}
	defer storage.Close()

	// Check data
	data := storage.Data()
	if len(data) != len(testData) {
		t.Errorf("Data() length = %d, want %d", len(data), len(testData))
	}

	for i, b := range data {
		if b != testData[i] {
			t.Errorf("Data()[%d] = %d, want %d", i, b, testData[i])
		}
	}

	// Test closing
	err = storage.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Test loading non-existent file
	_, err = NewFileStorage(filepath.Join(tmpDir, "nonexistent.dic"))
	if err == nil {
		t.Error("NewFileStorage() with non-existent file should return error")
	}

	// Test loading empty file
	emptyFile := filepath.Join(tmpDir, "empty.dic")
	err = os.WriteFile(emptyFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}

	_, err = NewFileStorage(emptyFile)
	if err == nil {
		t.Error("NewFileStorage() with empty file should return error")
	}
}

func TestSudachiDicData(t *testing.T) {
	systemData := []byte("system dictionary data")
	userData1 := []byte("user dictionary 1")
	userData2 := []byte("user dictionary 2")

	systemStorage := NewOwnedStorage(systemData)
	userStorage1 := NewOwnedStorage(userData1)
	userStorage2 := NewOwnedStorage(userData2)

	// Create SudachiDicData
	dicData := NewSudachiDicData(systemStorage)

	// Test system data
	result := dicData.System()
	if len(result) != len(systemData) {
		t.Errorf("System() length = %d, want %d", len(result), len(systemData))
	}

	// Test user count initially
	if dicData.UserCount() != 0 {
		t.Errorf("UserCount() = %d, want 0", dicData.UserCount())
	}

	// Add user dictionaries
	dicData.AddUser(userStorage1)
	dicData.AddUser(userStorage2)

	// Test user count after adding
	if dicData.UserCount() != 2 {
		t.Errorf("UserCount() = %d, want 2", dicData.UserCount())
	}

	// Test user data access
	user1Data, err := dicData.User(0)
	if err != nil {
		t.Errorf("User(0) error = %v, want nil", err)
	}
	if len(user1Data) != len(userData1) {
		t.Errorf("User(0) length = %d, want %d", len(user1Data), len(userData1))
	}

	user2Data, err := dicData.User(1)
	if err != nil {
		t.Errorf("User(1) error = %v, want nil", err)
	}
	if len(user2Data) != len(userData2) {
		t.Errorf("User(1) length = %d, want %d", len(user2Data), len(userData2))
	}

	// Test invalid user index
	_, err = dicData.User(-1)
	if err == nil {
		t.Error("User(-1) should return error")
	}

	_, err = dicData.User(2)
	if err == nil {
		t.Error("User(2) should return error")
	}

	// Test AllUsers
	allUsers := dicData.AllUsers()
	if len(allUsers) != 2 {
		t.Errorf("AllUsers() length = %d, want 2", len(allUsers))
	}

	// Test closing
	err = dicData.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}
