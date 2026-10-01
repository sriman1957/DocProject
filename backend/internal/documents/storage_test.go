package documents

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileStorage(t *testing.T) {
	t.Parallel()

	root := filepath.Join(
		t.TempDir(),
		"documents",
	)

	storage, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if storage == nil {
		t.Fatal("expected storage, got nil")
	}

	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf(
			"expected storage root to exist: %v",
			err,
		)
	}

	if !info.IsDir() {
		t.Fatal("expected storage root to be a directory")
	}
}

func TestNewFileStorageRejectsEmptyRoot(t *testing.T) {
	t.Parallel()

	_, err := NewFileStorage("")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFileStorageSave(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	storage, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	key := "documents/test-file"

	data := []byte("certificate contents")

	if err := storage.Save(key, data); err != nil {
		t.Fatalf(
			"save file: %v",
			err,
		)
	}

	path := filepath.Join(
		root,
		"documents",
		"test-file",
	)

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read stored file: %v",
			err,
		)
	}

	if string(stored) != string(data) {
		t.Fatalf(
			"stored data mismatch: got %q, want %q",
			string(stored),
			string(data),
		)
	}
}

func TestFileStorageSaveCreatesNestedDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	storage, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	key := "documents/2026/09/certificate"

	data := []byte("certificate")

	if err := storage.Save(key, data); err != nil {
		t.Fatalf(
			"save file: %v",
			err,
		)
	}

	path := filepath.Join(
		root,
		"documents",
		"2026",
		"09",
		"certificate",
	)

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read stored file: %v",
			err,
		)
	}

	if string(stored) != string(data) {
		t.Fatalf(
			"stored data mismatch: got %q, want %q",
			string(stored),
			string(data),
		)
	}
}

func TestFileStorageSaveRejectsInvalidKey(t *testing.T) {
	t.Parallel()

	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	tests := []string{
		"",
		"/documents/file",
		"documents/../secret",
		"documents/user/../../secret",
		`documents\..\secret`,
		"uploads/file",
	}

	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			err := storage.Save(
				key,
				[]byte("test"),
			)

			if !errors.Is(
				err,
				ErrInvalidStorageKey,
			) {
				t.Fatalf(
					"expected ErrInvalidStorageKey, got %v",
					err,
				)
			}
		})
	}
}

func TestFileStorageSaveRejectsEmptyData(t *testing.T) {
	t.Parallel()

	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	err = storage.Save(
		"documents/test",
		nil,
	)

	if !errors.Is(err, ErrEmptyFile) {
		t.Fatalf(
			"expected ErrEmptyFile, got %v",
			err,
		)
	}
}

func TestFileStorageDelete(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	storage, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	key := "documents/delete-me"

	if err := storage.Save(
		key,
		[]byte("delete this"),
	); err != nil {
		t.Fatalf(
			"save file: %v",
			err,
		)
	}

	if err := storage.Delete(key); err != nil {
		t.Fatalf(
			"delete file: %v",
			err,
		)
	}

	path := filepath.Join(
		root,
		"documents",
		"delete-me",
	)

	if _, err := os.Stat(path); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"expected file to be deleted, stat error: %v",
			err,
		)
	}
}

func TestFileStorageDeleteNotFound(t *testing.T) {
	t.Parallel()

	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	err = storage.Delete(
		"documents/missing",
	)

	if !errors.Is(
		err,
		ErrStorageNotFound,
	) {
		t.Fatalf(
			"expected ErrStorageNotFound, got %v",
			err,
		)
	}
}

func TestFileStorageDeleteRejectsInvalidKey(t *testing.T) {
	t.Parallel()

	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	err = storage.Delete(
		"documents/../secret",
	)

	if !errors.Is(
		err,
		ErrInvalidStorageKey,
	) {
		t.Fatalf(
			"expected ErrInvalidStorageKey, got %v",
			err,
		)
	}
}

func TestFileStorageOpen(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	storage, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	key := "documents/open-me"
	data := []byte("stored document")

	if err := storage.Save(key, data); err != nil {
		t.Fatalf(
			"save file: %v",
			err,
		)
	}

	file, err := storage.Open(key)
	if err != nil {
		t.Fatalf(
			"open file: %v",
			err,
		)
	}
	defer file.Close()

	stored, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf(
			"read opened file: %v",
			err,
		)
	}

	if string(stored) != string(data) {
		t.Fatalf(
			"stored data mismatch: got %q, want %q",
			string(stored),
			string(data),
		)
	}
}

func TestFileStorageOpenNotFound(t *testing.T) {
	t.Parallel()

	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf(
			"create storage: %v",
			err,
		)
	}

	_, err = storage.Open(
		"documents/missing",
	)

	if !errors.Is(
		err,
		ErrStorageNotFound,
	) {
		t.Fatalf(
			"expected ErrStorageNotFound, got %v",
			err,
		)
	}
}
