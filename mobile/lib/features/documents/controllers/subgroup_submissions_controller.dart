import 'package:flutter/foundation.dart';

import '../../../core/network/api_client.dart';
import '../models/vault_document.dart';
import '../services/documents_service.dart';
import '../services/file_download_service.dart';

sealed class SubgroupSubmissionsState {
  const SubgroupSubmissionsState();
}

class SubmissionsInitial extends SubgroupSubmissionsState {
  const SubmissionsInitial();
}

class SubmissionsLoading extends SubgroupSubmissionsState {
  const SubmissionsLoading();
}

class SubmissionsLoaded extends SubgroupSubmissionsState {
  const SubmissionsLoaded(this.documents);

  final List<VaultDocument> documents;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SubmissionsLoaded &&
          runtimeType == other.runtimeType &&
          listEquals(documents, other.documents);

  @override
  int get hashCode => documents.hashCode;
}

class SubmissionsEmpty extends SubgroupSubmissionsState {
  const SubmissionsEmpty();
}

class SubmissionsError extends SubgroupSubmissionsState {
  const SubmissionsError({
    required this.message,
    this.canRetry = true,
    this.statusCode,
  });

  final String message;
  final bool canRetry;
  final int? statusCode;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SubmissionsError &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry &&
          statusCode == other.statusCode;

  @override
  int get hashCode =>
      message.hashCode ^ canRetry.hashCode ^ statusCode.hashCode;
}

class SubgroupSubmissionsController extends ChangeNotifier {
  SubgroupSubmissionsController({
    required this.documentsService,
    required this.groupId,
    required this.subgroupId,
    this.fileDownloadService = const PlatformFileDownloadService(),
    SubgroupSubmissionsState initialState = const SubmissionsInitial(),
  }) : _state = initialState;

  final DocumentsService documentsService;
  final int groupId;
  final int subgroupId;
  final FileDownloadService fileDownloadService;

  SubgroupSubmissionsState _state;
  SubgroupSubmissionsState get state => _state;

  final Set<int> _downloadingDocIds = <int>{};
  Set<int> get downloadingDocIds => Set.unmodifiable(_downloadingDocIds);

  bool isDownloading(int documentId) => _downloadingDocIds.contains(documentId);

  /// Loads documents submitted to the subgroup.
  Future<void> loadSubmissions({bool force = false}) async {
    if (!force &&
        (_state is SubmissionsLoading || _state is SubmissionsLoaded)) {
      return;
    }

    _state = const SubmissionsLoading();
    notifyListeners();

    try {
      final documents = await documentsService.getSubgroupSubmissions(
        groupId: groupId,
        subgroupId: subgroupId,
      );

      if (documents.isEmpty) {
        _state = const SubmissionsEmpty();
      } else {
        _state = SubmissionsLoaded(documents);
      }
    } on ApiException catch (e) {
      String message = e.message;
      if (e.statusCode == 403) {
        message = 'Access denied. You must be an authorized faculty member or admin for this group.';
      } else if (e.statusCode == 404) {
        message = 'Subgroup not found or inactive.';
      }
      _state = SubmissionsError(message: message, statusCode: e.statusCode);
    } catch (_) {
      _state = const SubmissionsError(
        message: 'Network error. Please check your connection and try again.',
      );
    } finally {
      notifyListeners();
    }
  }

  /// Forces a reload of submissions from the backend.
  Future<void> refresh() => loadSubmissions(force: true);

  /// Downloads the specified document and saves it using the platform file saver.
  /// Returns the saved file path/URI string if saved, null if cancelled or failed.
  Future<String?> downloadDocument(VaultDocument doc) async {
    if (isDownloading(doc.id)) {
      return null;
    }

    _downloadingDocIds.add(doc.id);
    notifyListeners();

    try {
      final response = await documentsService.downloadSubgroupDocument(
        groupId: groupId,
        subgroupId: subgroupId,
        documentId: doc.id,
      );

      String filename = doc.originalFilename;
      final contentDisposition = response.headers['content-disposition'];
      if (contentDisposition != null) {
        final match = RegExp(r'filename="?([^"]+)"?')
            .firstMatch(contentDisposition);
        if (match != null &&
            match.group(1) != null &&
            match.group(1)!.trim().isNotEmpty) {
          filename = match.group(1)!.trim();
        }
      }

      final mimeType = response.headers['content-type'] ?? doc.mimeType;

      final savedPath = await fileDownloadService.saveFile(
        filename: filename,
        bytes: response.bodyBytes,
        mimeType: mimeType,
      );

      return savedPath;
    } finally {
      _downloadingDocIds.remove(doc.id);
      notifyListeners();
    }
  }
}
