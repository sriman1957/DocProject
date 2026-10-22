import 'package:docproject/features/documents/services/file_picker_service.dart';
import 'package:docproject/features/documents/utils/file_validator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FileValidator', () {
    test('accepts valid PDF file within 20 MB', () {
      const file = SelectedFile(
        name: 'sample.pdf',
        size: 1024 * 100,
        bytes: [1, 2, 3],
        extension: 'pdf',
      );

      final error = FileValidator.validate(file);
      expect(error, isNull);
    });

    test('accepts valid DOCX file within 20 MB', () {
      const file = SelectedFile(
        name: 'assignment.docx',
        size: 1024 * 500,
        bytes: [1, 2, 3],
        extension: 'docx',
      );

      final error = FileValidator.validate(file);
      expect(error, isNull);
    });

    test('accepts valid image files (PNG, JPG, JPEG)', () {
      for (final ext in ['png', 'jpg', 'jpeg']) {
        final file = SelectedFile(
          name: 'photo.$ext',
          size: 1024 * 50,
          bytes: [1, 2, 3],
          extension: ext,
        );

        final error = FileValidator.validate(file);
        expect(error, isNull, reason: 'Failed for extension $ext');
      }
    });

    test('handles case-insensitive extensions (.PDF, .DocX, .PNG)', () {
      const file = SelectedFile(
        name: 'REPORT.PDF',
        size: 2048,
        bytes: [1, 2, 3],
        extension: 'PDF',
      );

      final error = FileValidator.validate(file);
      expect(error, isNull);
    });

    test('rejects empty file when bytes are empty or size is 0', () {
      const emptyBytesFile = SelectedFile(
        name: 'empty.pdf',
        size: 1024,
        bytes: [],
        extension: 'pdf',
      );
      expect(FileValidator.validate(emptyBytesFile), contains('empty'));

      const zeroSizeFile = SelectedFile(
        name: 'empty.pdf',
        size: 0,
        bytes: [1, 2, 3],
        extension: 'pdf',
      );
      expect(FileValidator.validate(zeroSizeFile), contains('empty'));
    });

    test('rejects file larger than 20 MB (20 * 1024 * 1024 bytes)', () {
      const oversizedFile = SelectedFile(
        name: 'huge.pdf',
        size: (20 * 1024 * 1024) + 1,
        bytes: [1, 2, 3],
        extension: 'pdf',
      );

      final error = FileValidator.validate(oversizedFile);
      expect(error, isNotNull);
      expect(error, contains('20 MB'));
    });

    test('accepts file exactly at 20 MB boundary', () {
      const boundaryFile = SelectedFile(
        name: 'exact.pdf',
        size: 20 * 1024 * 1024,
        bytes: [1, 2, 3],
        extension: 'pdf',
      );

      final error = FileValidator.validate(boundaryFile);
      expect(error, isNull);
    });

    test(
      'rejects unsupported extensions (.exe, .zip, .mp4, .txt, no extension)',
      () {
        final unsupported = ['exe', 'zip', 'mp4', 'txt', 'sh', 'apk'];
        for (final ext in unsupported) {
          final file = SelectedFile(
            name: 'file.$ext',
            size: 1024,
            bytes: [1, 2, 3],
            extension: ext,
          );

          final error = FileValidator.validate(file);
          expect(error, isNotNull);
          expect(error, contains('Unsupported file type'));
        }

        const noExtFile = SelectedFile(
          name: 'unknown_file',
          size: 1024,
          bytes: [1, 2, 3],
          extension: null,
        );
        final error = FileValidator.validate(noExtFile);
        expect(error, isNotNull);
        expect(error, contains('Unsupported file type'));
      },
    );
  });
}
