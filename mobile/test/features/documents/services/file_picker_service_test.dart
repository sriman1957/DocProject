import 'dart:io';

import 'package:docproject/features/documents/services/file_picker_service.dart';
import 'package:flutter_test/flutter_test.dart';

class MockFilePickerService implements FilePickerService {
  MockFilePickerService({this.fileToReturn, this.shouldThrow = false});

  final SelectedFile? fileToReturn;
  final bool shouldThrow;

  @override
  Future<SelectedFile?> pickFile() async {
    if (shouldThrow) {
      throw const FileSystemException('Cannot read file from device.');
    }
    return fileToReturn;
  }
}

void main() {
  group('SelectedFile', () {
    test('formats size correctly across various byte lengths', () {
      const zeroFile = SelectedFile(
        name: 'a.pdf',
        size: 0,
        bytes: [],
        extension: 'pdf',
      );
      expect(zeroFile.formattedSize, '0 B');

      const bytesFile = SelectedFile(
        name: 'b.pdf',
        size: 512,
        bytes: [1],
        extension: 'pdf',
      );
      expect(bytesFile.formattedSize, '512 B');

      const kbFile = SelectedFile(
        name: 'c.pdf',
        size: 2048,
        bytes: [1],
        extension: 'pdf',
      );
      expect(kbFile.formattedSize, '2.0 KB');

      const mbFile = SelectedFile(
        name: 'd.pdf',
        size: 5 * 1024 * 1024,
        bytes: [1],
        extension: 'pdf',
      );
      expect(mbFile.formattedSize, '5.0 MB');
    });

    test('preserves file name, path, extension, and bytes', () {
      const file = SelectedFile(
        name: 'certificate.pdf',
        size: 1024,
        bytes: [10, 20, 30],
        path: '/storage/emulated/0/Download/certificate.pdf',
        extension: 'pdf',
      );

      expect(file.name, 'certificate.pdf');
      expect(file.size, 1024);
      expect(file.bytes, [10, 20, 30]);
      expect(file.path, '/storage/emulated/0/Download/certificate.pdf');
      expect(file.extension, 'pdf');
    });
  });

  group('FilePickerService', () {
    test('returns SelectedFile when user chooses a file', () async {
      const expectedFile = SelectedFile(
        name: 'marksheet.pdf',
        size: 4096,
        bytes: [1, 2, 3],
        extension: 'pdf',
      );

      final service = MockFilePickerService(fileToReturn: expectedFile);
      final result = await service.pickFile();

      expect(result, isNotNull);
      expect(result!.name, 'marksheet.pdf');
      expect(result.size, 4096);
    });

    test('returns null when user cancels selection without error', () async {
      final service = MockFilePickerService(fileToReturn: null);
      final result = await service.pickFile();

      expect(result, isNull);
    });

    test('throws FileSystemException when file cannot be read', () async {
      final service = MockFilePickerService(shouldThrow: true);

      expect(() => service.pickFile(), throwsA(isA<FileSystemException>()));
    });
  });
}
