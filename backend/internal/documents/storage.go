package documents

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrInvalidStorageKey = errors.New("invalid storage key")
	ErrStorageNotFound   = errors.New("storage object not found")
)

type Storage interface {
	Save(key string, data []byte) error
	Delete(key string) error
}

type FileStorage struct {
	root string
}

func NewFileStorage(root string) (*FileStorage, error) {
	if root == "" {
		return nil, errors.New("storage root is required")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve storage root: %w",
			err,
		)
	}

	if err := os.MkdirAll(absoluteRoot, 0o750); err != nil {
		return nil, fmt.Errorf(
			"create storage root: %w",
			err,
		)
	}

	return &FileStorage{
		root: absoluteRoot,
	}, nil
}

func (s *FileStorage) Save(key string, data []byte) error {
	if !IsSafeStorageKey(key) {
		return ErrInvalidStorageKey
	}

	if len(data) == 0 {
		return ErrEmptyFile
	}

	destination, err := s.resolvePath(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(
		filepath.Dir(destination),
		0o750,
	); err != nil {
		return fmt.Errorf(
			"create storage directory: %w",
			err,
		)
	}

	tempFile, err := os.CreateTemp(
		filepath.Dir(destination),
		".upload-*",
	)
	if err != nil {
		return fmt.Errorf(
			"create temporary storage file: %w",
			err,
		)
	}

	tempPath := tempFile.Name()

	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(data); err != nil {
		cleanup()

		return fmt.Errorf(
			"write temporary storage file: %w",
			err,
		)
	}

	if err := tempFile.Sync(); err != nil {
		cleanup()

		return fmt.Errorf(
			"sync temporary storage file: %w",
			err,
		)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"close temporary storage file: %w",
			err,
		)
	}

	if err := os.Rename(tempPath, destination); err != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"move storage file into place: %w",
			err,
		)
	}

	return nil
}

func (s *FileStorage) Delete(key string) error {
	if !IsSafeStorageKey(key) {
		return ErrInvalidStorageKey
	}

	path, err := s.resolvePath(key)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrStorageNotFound
		}

		return fmt.Errorf(
			"delete storage file: %w",
			err,
		)
	}

	return nil
}

func (s *FileStorage) Open(key string) (io.ReadCloser, error) {
	if !IsSafeStorageKey(key) {
		return nil, ErrInvalidStorageKey
	}

	path, err := s.resolvePath(key)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrStorageNotFound
		}

		return nil, fmt.Errorf(
			"open storage file: %w",
			err,
		)
	}

	return file, nil
}

func (s *FileStorage) resolvePath(key string) (string, error) {
	if !IsSafeStorageKey(key) {
		return "", ErrInvalidStorageKey
	}

	candidate := filepath.Join(
		s.root,
		filepath.FromSlash(key),
	)

	relative, err := filepath.Rel(s.root, candidate)
	if err != nil {
		return "", fmt.Errorf(
			"resolve storage path: %w",
			err,
		)
	}

	if relative == ".." ||
		len(relative) >= 3 &&
			relative[:3] == ".."+string(os.PathSeparator) {
		return "", ErrInvalidStorageKey
	}

	return candidate, nil
}
