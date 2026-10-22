import 'package:flutter/foundation.dart';

import '../../core/network/api_client.dart';
import 'models/vault_document.dart';
import 'services/documents_service.dart';

sealed class DocumentsState {
  const DocumentsState();
}

class DocumentsInitial extends DocumentsState {
  const DocumentsInitial();
}

class DocumentsLoading extends DocumentsState {
  const DocumentsLoading();
}

class DocumentsLoaded extends DocumentsState {
  const DocumentsLoaded(this.documents);

  final List<VaultDocument> documents;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentsLoaded &&
          runtimeType == other.runtimeType &&
          listEquals(documents, other.documents);

  @override
  int get hashCode => documents.hashCode;
}

class DocumentsEmpty extends DocumentsState {
  const DocumentsEmpty();

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentsEmpty && runtimeType == other.runtimeType;

  @override
  int get hashCode => runtimeType.hashCode;
}

class DocumentsError extends DocumentsState {
  const DocumentsError({
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
      other is DocumentsError &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry &&
          statusCode == other.statusCode;

  @override
  int get hashCode =>
      message.hashCode ^ canRetry.hashCode ^ statusCode.hashCode;
}

class DocumentsController extends ChangeNotifier {
  DocumentsController({
    required this.documentsService,
    DocumentsState initialState = const DocumentsInitial(),
  }) : _state = initialState;

  final DocumentsService documentsService;

  DocumentsState _state;
  DocumentsState get state => _state;

  List<VaultDocument> get documents => switch (_state) {
    DocumentsLoaded(:final documents) => documents,
    _ => const [],
  };

  bool _isLoading = false;
  bool _hasLoaded = false;

  /// Loads personal vault documents from the backend.
  Future<void> loadDocuments({bool force = false}) async {
    if (_isLoading) return;
    if (!force &&
        _hasLoaded &&
        (_state is DocumentsLoaded || _state is DocumentsEmpty)) {
      return;
    }

    _isLoading = true;
    _state = const DocumentsLoading();
    notifyListeners();

    try {
      final result = await documentsService.getPersonalVaultDocuments();
      _hasLoaded = true;
      if (result.isEmpty) {
        _state = const DocumentsEmpty();
      } else {
        _state = DocumentsLoaded(result);
      }
    } on ApiException catch (e) {
      _state = DocumentsError(message: e.message, statusCode: e.statusCode);
    } catch (_) {
      _state = const DocumentsError(
        message: 'Unable to load personal vault documents. Please check your connection and try again.',
      );
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  /// Forces a fresh reload of documents.
  Future<void> refresh() => loadDocuments(force: true);

  /// Looks up a loaded document by its ID.
  VaultDocument? getDocumentById(int id) {
    for (final doc in documents) {
      if (doc.id == id) return doc;
    }
    return null;
  }

  /// Resets the controller state (e.g., on sign out).
  void clear() {
    _state = const DocumentsInitial();
    _hasLoaded = false;
    _isLoading = false;
    notifyListeners();
  }
}
