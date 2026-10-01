package documents

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	MaxUploadSize int64 = 20 * 1024 * 1024

	MIMEPDF  = "application/pdf"
	MIMEDOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MIMEJPEG = "image/jpeg"
	MIMEPNG  = "image/png"
)

var (
	ErrEmptyFile       = errors.New("empty file")
	ErrFileTooLarge    = errors.New("file too large")
	ErrUnsupportedType = errors.New("unsupported file type")
	ErrInvalidFile     = errors.New("invalid file")
	ErrInvalidDOCX     = errors.New("invalid DOCX file")
)

type ValidatedUpload struct {
	MIMEType string
	Size     int64
}

func ValidateUpload(data []byte) (ValidatedUpload, error) {
	if len(data) == 0 {
		return ValidatedUpload{}, ErrEmptyFile
	}

	if int64(len(data)) > MaxUploadSize {
		return ValidatedUpload{}, ErrFileTooLarge
	}

	mimeType := http.DetectContentType(data)

	switch mimeType {
	case MIMEPDF:
		if !isValidPDF(data) {
			return ValidatedUpload{}, ErrInvalidFile
		}

		return ValidatedUpload{
			MIMEType: MIMEPDF,
			Size:     int64(len(data)),
		}, nil

	case MIMEJPEG:
		if !isValidJPEG(data) {
			return ValidatedUpload{}, ErrInvalidFile
		}

		return ValidatedUpload{
			MIMEType: MIMEJPEG,
			Size:     int64(len(data)),
		}, nil

	case MIMEPNG:
		if !isValidPNG(data) {
			return ValidatedUpload{}, ErrInvalidFile
		}

		return ValidatedUpload{
			MIMEType: MIMEPNG,
			Size:     int64(len(data)),
		}, nil

	case "application/zip":
		if !isValidDOCX(data) {
			return ValidatedUpload{}, ErrInvalidDOCX
		}

		return ValidatedUpload{
			MIMEType: MIMEDOCX,
			Size:     int64(len(data)),
		}, nil

	default:
		return ValidatedUpload{}, fmt.Errorf(
			"%w: %s",
			ErrUnsupportedType,
			strings.TrimSpace(mimeType),
		)
	}
}

func isValidPDF(data []byte) bool {
	return bytes.HasPrefix(data, []byte("%PDF-"))
}

func isValidJPEG(data []byte) bool {
	if len(data) < 3 {
		return false
	}

	return data[0] == 0xFF &&
		data[1] == 0xD8 &&
		data[2] == 0xFF
}

func isValidPNG(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	signature := []byte{
		0x89,
		0x50,
		0x4E,
		0x47,
		0x0D,
		0x0A,
		0x1A,
		0x0A,
	}

	return bytes.Equal(data[:8], signature)
}

func isValidDOCX(data []byte) bool {
	reader := bytes.NewReader(data)

	zipReader, err := zip.NewReader(
		reader,
		int64(len(data)),
	)
	if err != nil {
		return false
	}

	requiredFiles := map[string]bool{
		"[Content_Types].xml": false,
		"_rels/.rels":         false,
		"word/document.xml":   false,
	}

	for _, file := range zipReader.File {
		if _, exists := requiredFiles[file.Name]; exists {
			requiredFiles[file.Name] = true
		}
	}

	for _, found := range requiredFiles {
		if !found {
			return false
		}
	}

	return true
}

func readAllLimited(
	reader io.Reader,
	maxSize int64,
) ([]byte, error) {
	if maxSize <= 0 {
		return nil, errors.New("invalid maximum size")
	}

	limited := io.LimitReader(reader, maxSize+1)

	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf(
			"read upload: %w",
			err,
		)
	}

	if int64(len(data)) > maxSize {
		return nil, ErrFileTooLarge
	}

	return data, nil
}
