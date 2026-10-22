import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('VaultDocument model', () {
    test('parses a valid document metadata response using actual schema', () {
      final json = {
        'id': 101,
        'original_filename': 'semester_marksheet.pdf',
        'mime_type': 'application/pdf',
        'file_size_bytes': 245678,
        'sha256':
            'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
        'uploaded_at': '2026-09-29T10:00:00.000Z',
      };

      final doc = VaultDocument.fromJson(json);

      expect(doc.id, 101);
      expect(doc.originalFilename, 'semester_marksheet.pdf');
      expect(doc.mimeType, 'application/pdf');
      expect(doc.fileSizeBytes, 245678);
      expect(
        doc.sha256,
        'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
      );
      expect(doc.uploadedAt, isNotNull);
      expect(doc.fileExtension, 'PDF');
      expect(doc.fileIcon, Icons.picture_as_pdf_rounded);
    });

    test('handles missing or null optional metadata safely', () {
      final json = <String, dynamic>{'id': 102};

      final doc = VaultDocument.fromJson(json);

      expect(doc.id, 102);
      expect(doc.originalFilename, 'Untitled Document');
      expect(doc.mimeType, 'application/octet-stream');
      expect(doc.fileSizeBytes, 0);
      expect(doc.sha256, '');
      expect(doc.uploadedAt, isNull);
      expect(doc.formattedFileSize, '0 B');
      expect(doc.formattedUploadDate, 'Unknown date');
      expect(doc.fileIcon, Icons.insert_drive_file_rounded);
    });

    test('correctly formats file size across different magnitudes', () {
      expect(
        const VaultDocument(
          id: 1,
          originalFilename: 'test.txt',
          mimeType: 'text/plain',
          fileSizeBytes: 0,
          sha256: '',
          uploadedAt: null,
        ).formattedFileSize,
        '0 B',
      );

      expect(
        const VaultDocument(
          id: 1,
          originalFilename: 'test.txt',
          mimeType: 'text/plain',
          fileSizeBytes: 512,
          sha256: '',
          uploadedAt: null,
        ).formattedFileSize,
        '512 B',
      );

      expect(
        const VaultDocument(
          id: 1,
          originalFilename: 'test.txt',
          mimeType: 'text/plain',
          fileSizeBytes: 2048,
          sha256: '',
          uploadedAt: null,
        ).formattedFileSize,
        '2.0 KB',
      );

      expect(
        const VaultDocument(
          id: 1,
          originalFilename: 'test.txt',
          mimeType: 'text/plain',
          fileSizeBytes: 1048576 * 3,
          sha256: '',
          uploadedAt: null,
        ).formattedFileSize,
        '3.0 MB',
      );

      expect(
        const VaultDocument(
          id: 1,
          originalFilename: 'test.txt',
          mimeType: 'text/plain',
          fileSizeBytes: 1073741824 * 2,
          sha256: '',
          uploadedAt: null,
        ).formattedFileSize,
        '2.0 GB',
      );
    });

    test('maps file types and extensions to icons and colors accurately', () {
      const docx = VaultDocument(
        id: 1,
        originalFilename: 'report.docx',
        mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        fileSizeBytes: 50000,
        sha256: 'abc',
        uploadedAt: null,
      );
      expect(docx.fileExtension, 'DOCX');
      expect(docx.fileIcon, Icons.article_rounded);
      expect(docx.fileIconColor, const Color(0xFF1976D2));

      const png = VaultDocument(
        id: 2,
        originalFilename: 'id_card.png',
        mimeType: 'image/png',
        fileSizeBytes: 12000,
        sha256: 'def',
        uploadedAt: null,
      );
      expect(png.fileExtension, 'PNG');
      expect(png.fileIcon, Icons.image_rounded);
      expect(png.fileIconColor, const Color(0xFF388E3C));

      const other = VaultDocument(
        id: 3,
        originalFilename: 'archive.tar',
        mimeType: 'application/x-tar',
        fileSizeBytes: 90000,
        sha256: 'ghi',
        uploadedAt: null,
      );
      expect(other.fileExtension, 'TAR');
      expect(other.fileIcon, Icons.insert_drive_file_rounded);
    });

    test('serializes to JSON accurately', () {
      final uploadTime = DateTime.utc(2026, 9, 29, 10, 0, 0);
      final doc = VaultDocument(
        id: 105,
        originalFilename: 'certificate.pdf',
        mimeType: 'application/pdf',
        fileSizeBytes: 123456,
        sha256: 'deadbeef',
        uploadedAt: uploadTime,
      );

      final json = doc.toJson();

      expect(json['id'], 105);
      expect(json['original_filename'], 'certificate.pdf');
      expect(json['mime_type'], 'application/pdf');
      expect(json['file_size_bytes'], 123456);
      expect(json['sha256'], 'deadbeef');
      expect(json['uploaded_at'], '2026-09-29T10:00:00.000Z');
    });

    test('equality and hashCode match for identical properties', () {
      final uploadTime = DateTime.utc(2026, 9, 29, 10, 0, 0);
      final doc1 = VaultDocument(
        id: 1,
        originalFilename: 'a.pdf',
        mimeType: 'application/pdf',
        fileSizeBytes: 100,
        sha256: 'hash',
        uploadedAt: uploadTime,
      );
      final doc2 = VaultDocument(
        id: 1,
        originalFilename: 'a.pdf',
        mimeType: 'application/pdf',
        fileSizeBytes: 100,
        sha256: 'hash',
        uploadedAt: uploadTime,
      );
      final doc3 = VaultDocument(
        id: 2,
        originalFilename: 'b.pdf',
        mimeType: 'application/pdf',
        fileSizeBytes: 200,
        sha256: 'hash2',
        uploadedAt: uploadTime,
      );

      expect(doc1, equals(doc2));
      expect(doc1.hashCode, equals(doc2.hashCode));
      expect(doc1, isNot(equals(doc3)));
    });
  });
}
