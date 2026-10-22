import 'package:flutter/foundation.dart';

import '../../../core/network/api_client.dart';
import '../models/vault_document.dart';
import '../services/documents_service.dart';
import '../services/file_picker_service.dart';
import '../utils/file_validator.dart';

sealed class UploadState {
  const UploadState();
}

class UploadInitial extends UploadState {
  const UploadInitial();
}

class UploadFileSelected extends UploadState {
  const UploadFileSelected(this.file, {this.validationError});

  final SelectedFile file;
  final String? validationError;

  bool get isValid => validationError == null;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is UploadFileSelected &&
          runtimeType == other.runtimeType &&
          file.name == other.file.name &&
          file.size == other.file.size &&
          validationError == other.validationError;

  @override
  int get hashCode =>
      file.name.hashCode ^ file.size.hashCode ^ validationError.hashCode;
}

class UploadInProgress extends UploadState {
  const UploadInProgress({required this.filename});

  final String filename;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is UploadInProgress &&
          runtimeType == other.runtimeType &&
          filename == other.filename;

  @override
  int get hashCode => filename.hashCode;
}

class UploadSuccess extends UploadState {
  const UploadSuccess({required this.uploadedDocument, required this.message});

  final VaultDocument uploadedDocument;
  final String message;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is UploadSuccess &&
          runtimeType == other.runtimeType &&
          uploadedDocument.id == other.uploadedDocument.id &&
          message == other.message;

  @override
  int get hashCode => uploadedDocument.id.hashCode ^ message.hashCode;
}

class UploadFailure extends UploadState {
  const UploadFailure({
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
      other is UploadFailure &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry &&
          statusCode == other.statusCode;

  @override
  int get hashCode =>
      message.hashCode ^ canRetry.hashCode ^ statusCode.hashCode;
}

class DocumentUploadController extends ChangeNotifier {
  DocumentUploadController({
    required this.documentsService,
    this.filePickerService = const PlatformFilePickerService(),
    UploadState initialState = const UploadInitial(),
  }) : _state = initialState;

  final DocumentsService documentsService;
  final FilePickerService filePickerService;

  UploadState _state;
  UploadState get state => _state;

  SelectedFile? _currentFile;
  SelectedFile? get selectedFile => _currentFile;

  bool _isUploading = false;
  bool get isUploading => _isUploading;

  /// Prompts the user to pick a document file from storage.
  Future<void> pickFile() async {
    if (_isUploading) return;

    try {
      final file = await filePickerService.pickFile();
      if (file == null) {
        // User cancelled selection without error
        return;
      }

      _currentFile = file;
      final error = FileValidator.validate(file);
      _state = UploadFileSelected(file, validationError: error);
      notifyListeners();
    } catch (e) {
      _state = UploadFailure(
        message: e is Exception
            ? e.toString().replaceFirst('Exception: ', '')
            : 'Failed to select file.',
        canRetry: false,
      );
      notifyListeners();
    }
  }

  /// Clears the currently selected file.
  void clearFile() {
    if (_isUploading) return;
    _currentFile = null;
    _state = const UploadInitial();
    notifyListeners();
  }

  /// Sets a file directly (e.g. for testing).
  void setFile(SelectedFile file) {
    _currentFile = file;
    final error = FileValidator.validate(file);
    _state = UploadFileSelected(file, validationError: error);
    notifyListeners();
  }

  /// Uploads the selected file directly to the student's Personal Vault.
  Future<VaultDocument?> uploadToPersonalVault() async {
    if (_isUploading) return null;

    final file = selectedFile;
    if (file == null) {
      _state = const UploadFailure(
        message: 'No file selected. Please select a file first.',
        canRetry: false,
      );
      notifyListeners();
      return null;
    }

    final validationError = FileValidator.validate(file);
    if (validationError != null) {
      _state = UploadFileSelected(file, validationError: validationError);
      notifyListeners();
      return null;
    }

    _isUploading = true;
    _state = UploadInProgress(filename: file.name);
    notifyListeners();

    try {
      final doc = await documentsService.uploadPersonalVaultDocument(file);
      _currentFile = null;
      _state = UploadSuccess(
        uploadedDocument: doc,
        message:
            'Document "${doc.originalFilename}" uploaded successfully to Personal Vault.',
      );
      return doc;
    } on ApiException catch (e) {
      _state = UploadFailure(message: e.message, statusCode: e.statusCode);
      return null;
    } catch (_) {
      _state = const UploadFailure(
        message: 'Network error during upload. Please check your connection and try again.',
      );
      return null;
    } finally {
      _isUploading = false;
      notifyListeners();
    }
  }

  /// Submits a document to a subgroup:
  /// - If [existingVaultDocId] is provided, copies that vault document into the subgroup.
  /// - If [selectedFile] is present, uploads it to the vault first, then copies it to the subgroup.
  Future<VaultDocument?> submitToSubgroup({
    required int groupId,
    required int subgroupId,
    int? existingVaultDocId,
  }) async {
    if (_isUploading) return null;

    int docId;
    String docName;

    if (existingVaultDocId != null) {
      docId = existingVaultDocId;
      docName = 'Vault Document #$docId';
    } else {
      final file = selectedFile;
      if (file == null) {
        _state = const UploadFailure(
          message: 'Please select a document to submit.',
          canRetry: false,
        );
        notifyListeners();
        return null;
      }

      final validationError = FileValidator.validate(file);
      if (validationError != null) {
        _state = UploadFileSelected(file, validationError: validationError);
        notifyListeners();
        return null;
      }

      _isUploading = true;
      _state = UploadInProgress(filename: file.name);
      notifyListeners();

      try {
        final uploaded = await documentsService.uploadPersonalVaultDocument(
          file,
        );
        docId = uploaded.id;
        docName = uploaded.originalFilename;
      } on ApiException catch (e) {
        _isUploading = false;
        _state = UploadFailure(message: e.message, statusCode: e.statusCode);
        notifyListeners();
        return null;
      } catch (_) {
        _isUploading = false;
        _state = const UploadFailure(
          message: 'Network error during upload. Please try again.',
        );
        notifyListeners();
        return null;
      }
    }

    _isUploading = true;
    _state = UploadInProgress(filename: docName);
    notifyListeners();

    try {
      final result = await documentsService.submitDocumentToSubgroup(
        documentId: docId,
        groupId: groupId,
        subgroupId: subgroupId,
      );

      _currentFile = null;
      _state = UploadSuccess(
        uploadedDocument: result,
        message:
            'Document "${result.originalFilename}" submitted successfully to subgroup.',
      );
      return result;
    } on ApiException catch (e) {
      String message = e.message;
      if (e.statusCode == 403) {
        message = 'Submission denied. The collection window may be closed or you may not be authorized to submit to this subgroup.';
      }
      _state = UploadFailure(message: message, statusCode: e.statusCode);
      return null;
    } catch (_) {
      _state = const UploadFailure(
        message: 'Failed to submit document to subgroup. Please check your connection.',
      );
      return null;
    } finally {
      _isUploading = false;
      notifyListeners();
    }
  }

  /// Resets state back to initial.
  void reset() {
    _isUploading = false;
    _currentFile = null;
    _state = const UploadInitial();
    notifyListeners();
  }
}
