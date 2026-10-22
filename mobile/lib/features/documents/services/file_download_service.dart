import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';

abstract class FileDownloadService {
  Future<String?> saveFile({
    required String filename,
    required List<int> bytes,
    String mimeType = 'application/octet-stream',
  });
}

class PlatformFileDownloadService implements FileDownloadService {
  const PlatformFileDownloadService();

  @override
  Future<String?> saveFile({
    required String filename,
    required List<int> bytes,
    String mimeType = 'application/octet-stream',
  }) async {
    final uri = await FilePicker.saveFile(
      fileName: filename,
      bytes: Uint8List.fromList(bytes),
      mimeType: mimeType,
      dialogTitle: 'Save Document',
    );

    return uri?.toString();
  }
}
