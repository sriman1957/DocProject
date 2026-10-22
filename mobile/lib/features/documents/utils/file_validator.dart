import '../services/file_picker_service.dart';

class FileValidator {
  const FileValidator._();

  /// 20 MB maximum file size matching the Go backend MaxUploadSize constraint.
  static const int maxUploadSizeBytes = 20 * 1024 * 1024;

  /// Accepted document file extensions matching backend MIME validation.
  static const Set<String> allowedExtensions = {
    'pdf',
    'docx',
    'png',
    'jpg',
    'jpeg',
  };

  /// Validates [file] against the backend upload contract.
  /// Returns `null` if valid, or a descriptive error message if invalid.
  static String? validate(SelectedFile file) {
    if (file.size <= 0 || file.bytes.isEmpty) {
      return 'The selected file is empty. Please select a valid document.';
    }

    if (file.size > maxUploadSizeBytes ||
        file.bytes.length > maxUploadSizeBytes) {
      return 'File size exceeds the 20 MB limit. Please select a smaller file.';
    }

    final ext = file.extension?.toLowerCase().trim() ?? '';
    if (ext.isEmpty || !allowedExtensions.contains(ext)) {
      return 'Unsupported file type. Only PDF, DOCX, PNG, and JPEG documents are permitted.';
    }

    return null;
  }
}
