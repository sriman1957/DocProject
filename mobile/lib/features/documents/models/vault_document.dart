import 'package:flutter/material.dart';

class VaultDocument {
  const VaultDocument({
    required this.id,
    required this.originalFilename,
    required this.mimeType,
    required this.fileSizeBytes,
    required this.sha256,
    required this.uploadedAt,
  });

  final int id;
  final String originalFilename;
  final String mimeType;
  final int fileSizeBytes;
  final String sha256;
  final DateTime? uploadedAt;

  factory VaultDocument.fromJson(Map<String, dynamic> json) {
    DateTime? parsedUploadedAt;
    final rawUploadedAt = json['uploaded_at'];
    if (rawUploadedAt is String && rawUploadedAt.isNotEmpty) {
      parsedUploadedAt = DateTime.tryParse(rawUploadedAt);
    }

    return VaultDocument(
      id: (json['id'] as num?)?.toInt() ?? 0,
      originalFilename:
          json['original_filename'] as String? ?? 'Untitled Document',
      mimeType: json['mime_type'] as String? ?? 'application/octet-stream',
      fileSizeBytes: (json['file_size_bytes'] as num?)?.toInt() ?? 0,
      sha256: json['sha256'] as String? ?? '',
      uploadedAt: parsedUploadedAt,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'original_filename': originalFilename,
      'mime_type': mimeType,
      'file_size_bytes': fileSizeBytes,
      'sha256': sha256,
      if (uploadedAt != null) 'uploaded_at': uploadedAt!.toIso8601String(),
    };
  }

  /// Returns a human-readable file size representation.
  String get formattedFileSize {
    if (fileSizeBytes <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    var size = fileSizeBytes.toDouble();
    var unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return unitIndex == 0
        ? '${size.toInt()} B'
        : '${size.toStringAsFixed(1)} ${units[unitIndex]}';
  }

  /// Extracts file extension from original filename (e.g. "PDF", "DOCX").
  String get fileExtension {
    final dotIndex = originalFilename.lastIndexOf('.');
    if (dotIndex != -1 && dotIndex < originalFilename.length - 1) {
      return originalFilename.substring(dotIndex + 1).toUpperCase();
    }
    return '';
  }

  /// Formatted date string for display.
  String get formattedUploadDate {
    if (uploadedAt == null) return 'Unknown date';
    final dt = uploadedAt!.toLocal();
    final year = dt.year;
    final month = _monthName(dt.month);
    final day = dt.day.toString().padLeft(2, '0');
    return '$month $day, $year';
  }

  static String _monthName(int month) {
    const months = [
      'Jan',
      'Feb',
      'Mar',
      'Apr',
      'May',
      'Jun',
      'Jul',
      'Aug',
      'Sep',
      'Oct',
      'Nov',
      'Dec',
    ];
    if (month >= 1 && month <= 12) {
      return months[month - 1];
    }
    return '';
  }

  /// Icon representing the file type.
  IconData get fileIcon {
    final lowerMime = mimeType.toLowerCase();
    final ext = fileExtension.toLowerCase();

    if (lowerMime.contains('pdf') || ext == 'pdf') {
      return Icons.picture_as_pdf_rounded;
    }
    if (lowerMime.contains('word') ||
        lowerMime.contains('officedocument') ||
        ext == 'doc' ||
        ext == 'docx') {
      return Icons.article_rounded;
    }
    if (lowerMime.startsWith('image/') ||
        ext == 'png' ||
        ext == 'jpg' ||
        ext == 'jpeg') {
      return Icons.image_rounded;
    }
    return Icons.insert_drive_file_rounded;
  }

  /// Color accent representing the document category.
  Color get fileIconColor {
    final lowerMime = mimeType.toLowerCase();
    final ext = fileExtension.toLowerCase();

    if (lowerMime.contains('pdf') || ext == 'pdf') {
      return const Color(0xFFD32F2F); // Red
    }
    if (lowerMime.contains('word') ||
        lowerMime.contains('officedocument') ||
        ext == 'doc' ||
        ext == 'docx') {
      return const Color(0xFF1976D2); // Blue
    }
    if (lowerMime.startsWith('image/') ||
        ext == 'png' ||
        ext == 'jpg' ||
        ext == 'jpeg') {
      return const Color(0xFF388E3C); // Green
    }
    return const Color(0xFF5C6BC0); // Indigo
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is VaultDocument &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          originalFilename == other.originalFilename &&
          mimeType == other.mimeType &&
          fileSizeBytes == other.fileSizeBytes &&
          sha256 == other.sha256 &&
          uploadedAt == other.uploadedAt;

  @override
  int get hashCode =>
      id.hashCode ^
      originalFilename.hashCode ^
      mimeType.hashCode ^
      fileSizeBytes.hashCode ^
      sha256.hashCode ^
      uploadedAt.hashCode;
}
