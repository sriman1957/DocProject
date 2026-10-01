package documents

import (
	"bytes"
	"errors"
	"testing"
)

func TestValidateUploadPDF(t *testing.T) {
	t.Parallel()

	data := append(
		[]byte("%PDF-1.7\n"),
		[]byte("test document")...,
	)

	result, err := ValidateUpload(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MIMEType != MIMEPDF {
		t.Fatalf(
			"expected MIME type %s, got %s",
			MIMEPDF,
			result.MIMEType,
		)
	}

	if result.Size != int64(len(data)) {
		t.Fatalf(
			"expected size %d, got %d",
			len(data),
			result.Size,
		)
	}
}

func TestValidateUploadJPEG(t *testing.T) {
	t.Parallel()

	data := []byte{
		0xFF,
		0xD8,
		0xFF,
		0xE0,
		0x00,
		0x10,
		0x4A,
		0x46,
		0x49,
		0x46,
	}

	result, err := ValidateUpload(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MIMEType != MIMEJPEG {
		t.Fatalf(
			"expected MIME type %s, got %s",
			MIMEJPEG,
			result.MIMEType,
		)
	}
}

func TestValidateUploadPNG(t *testing.T) {
	t.Parallel()

	data := []byte{
		0x89,
		0x50,
		0x4E,
		0x47,
		0x0D,
		0x0A,
		0x1A,
		0x0A,
	}

	result, err := ValidateUpload(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MIMEType != MIMEPNG {
		t.Fatalf(
			"expected MIME type %s, got %s",
			MIMEPNG,
			result.MIMEType,
		)
	}
}

func TestValidateUploadRejectsEmptyFile(t *testing.T) {
	t.Parallel()

	_, err := ValidateUpload(nil)

	if !errors.Is(err, ErrEmptyFile) {
		t.Fatalf(
			"expected ErrEmptyFile, got %v",
			err,
		)
	}
}

func TestValidateUploadRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	data := make([]byte, MaxUploadSize+1)

	_, err := ValidateUpload(data)

	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf(
			"expected ErrFileTooLarge, got %v",
			err,
		)
	}
}

func TestValidateUploadRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	data := []byte("this is not a supported document")

	_, err := ValidateUpload(data)

	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf(
			"expected ErrUnsupportedType, got %v",
			err,
		)
	}
}

func TestValidateUploadRejectsFakePDF(t *testing.T) {
	t.Parallel()

	data := []byte("not really a pdf")

	_, err := ValidateUpload(data)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateUploadRejectsInvalidJPEG(t *testing.T) {
	t.Parallel()

	data := []byte{
		0xFF,
		0xD8,
		0x00,
	}

	_, err := ValidateUpload(data)

	if !errors.Is(err, ErrUnsupportedType) &&
		!errors.Is(err, ErrInvalidFile) {
		t.Fatalf(
			"expected file validation error, got %v",
			err,
		)
	}
}

func TestValidateUploadRejectsInvalidPNG(t *testing.T) {
	t.Parallel()

	data := bytes.Repeat([]byte{0x00}, 32)

	_, err := ValidateUpload(data)

	if !errors.Is(err, ErrUnsupportedType) &&
		!errors.Is(err, ErrInvalidFile) {
		t.Fatalf(
			"expected file validation error, got %v",
			err,
		)
	}
}

func TestReadAllLimited(t *testing.T) {
	t.Parallel()

	data := bytes.NewReader([]byte("hello"))

	result, err := readAllLimited(data, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "hello" {
		t.Fatalf(
			"expected hello, got %s",
			string(result),
		)
	}
}

func TestReadAllLimitedRejectsOversizedInput(t *testing.T) {
	t.Parallel()

	data := bytes.NewReader(
		bytes.Repeat([]byte("a"), 11),
	)

	_, err := readAllLimited(data, 10)

	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf(
			"expected ErrFileTooLarge, got %v",
			err,
		)
	}
}
