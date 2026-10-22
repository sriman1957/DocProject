import 'dart:io';

import 'package:file_picker/file_picker.dart';

class SelectedFile {
  const SelectedFile({
    required this.name,
    required this.size,
    required this.bytes,
    this.path,
    this.extension,
  });

  final String name;
  final int size;
  final List<int> bytes;
  final String? path;
  final String? extension;

  String get formattedSize {
    if (size <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    var s = size.toDouble();
    var idx = 0;
    while (s >= 1024 && idx < units.length - 1) {
      s /= 1024;
      idx++;
    }
    return idx == 0
        ? '${s.toInt()} B'
        : '${s.toStringAsFixed(1)} ${units[idx]}';
  }
}

abstract class FilePickerService {
  Future<SelectedFile?> pickFile();
}

class PlatformFilePickerService implements FilePickerService {
  const PlatformFilePickerService();

  @override
  Future<SelectedFile?> pickFile() async {
    final platformFile = await FilePicker.pickFile(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'docx', 'png', 'jpg', 'jpeg'],
    );

    if (platformFile == null) {
      return null;
    }

    final length = await platformFile.length() ?? 0;
    List<int>? bytes;
    try {
      bytes = await platformFile.readAsBytes();
    } catch (_) {
      // If readAsBytes fails, try fallback to path if available
      if (platformFile.path != null) {
        final file = File(platformFile.path!);
        if (await file.exists()) {
          bytes = await file.readAsBytes();
        }
      }
    }

    if (bytes == null || bytes.isEmpty) {
      throw const FileSystemException(
        'Selected file could not be read or is empty.',
      );
    }

    final ext =
        platformFile.extension?.toLowerCase() ??
        (platformFile.name.contains('.')
            ? platformFile.name.split('.').last.toLowerCase()
            : null);

    return SelectedFile(
      name: platformFile.name,
      size: length > 0 ? length : bytes.length,
      bytes: bytes,
      path: platformFile.path,
      extension: ext,
    );
  }
}
