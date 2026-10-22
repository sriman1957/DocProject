import 'dart:io';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/documents/controllers/document_upload_controller.dart';
import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/services/file_picker_service.dart';
import 'package:flutter_test/flutter_test.dart';

class MockFilePickerService implements FilePickerService {
  MockFilePickerService({this.fileToReturn, this.errorToThrow});

  SelectedFile? fileToReturn;
  Object? errorToThrow;

  @override
  Future<SelectedFile?> pickFile() async {
    if (errorToThrow != null) throw errorToThrow!;
    return fileToReturn;
  }
}

class FakeDocumentsService extends DocumentsService {
  FakeDocumentsService()
    : super(
        apiClient: ApiClient(baseUrl: 'http://localhost'),
        tokenStorage: TokenStorage(),
      );

  VaultDocument? vaultDocToReturn;
  VaultDocument? subgroupDocToReturn;
  Object? errorToThrow;
  Object? uploadErrorToThrow;
  Object? submitErrorToThrow;
  int uploadCallCount = 0;
  int submitCallCount = 0;
  SelectedFile? capturedUploadFile;
  int? capturedDocId;
  int? capturedGroupId;
  int? capturedSubgroupId;

  @override
  Future<VaultDocument> uploadPersonalVaultDocument(SelectedFile file) async {
    uploadCallCount++;
    capturedUploadFile = file;
    final err = uploadErrorToThrow ?? errorToThrow;
    if (err != null) throw err;
    return vaultDocToReturn!;
  }

  @override
  Future<VaultDocument> submitDocumentToSubgroup({
    required int documentId,
    required int groupId,
    required int subgroupId,
  }) async {
    submitCallCount++;
    capturedDocId = documentId;
    capturedGroupId = groupId;
    capturedSubgroupId = subgroupId;
    final err = submitErrorToThrow ?? errorToThrow;
    if (err != null) throw err;
    return subgroupDocToReturn!;
  }
}

void main() {
  late FakeDocumentsService fakeDocumentsService;
  late MockFilePickerService mockPicker;
  late DocumentUploadController controller;

  const validFile = SelectedFile(
    name: 'test_doc.pdf',
    size: 2048,
    bytes: [1, 2, 3],
    extension: 'pdf',
  );

  final sampleUploadedDoc = VaultDocument(
    id: 10,
    originalFilename: 'test_doc.pdf',
    mimeType: 'application/pdf',
    fileSizeBytes: 2048,
    sha256: 'abc123',
    uploadedAt: DateTime.parse('2026-10-22T10:00:00Z'),
  );

  setUp(() {
    fakeDocumentsService = FakeDocumentsService();
    mockPicker = MockFilePickerService();
    controller = DocumentUploadController(
      documentsService: fakeDocumentsService,
      filePickerService: mockPicker,
    );
  });

  group('DocumentUploadController', () {
    test('starts with initial state and no selected file', () {
      expect(controller.state, isA<UploadInitial>());
      expect(controller.selectedFile, isNull);
      expect(controller.isUploading, isFalse);
    });

    test('pickFile selects valid file and transitions to UploadFileSelected without validation error', () async {
      mockPicker.fileToReturn = validFile;

      await controller.pickFile();

      expect(controller.state, isA<UploadFileSelected>());
      final selectedState = controller.state as UploadFileSelected;
      expect(selectedState.file.name, 'test_doc.pdf');
      expect(selectedState.isValid, isTrue);
      expect(selectedState.validationError, isNull);
      expect(controller.selectedFile, validFile);
    });

    test('pickFile cancellation leaves state unchanged', () async {
      mockPicker.fileToReturn = null;

      await controller.pickFile();

      expect(controller.state, isA<UploadInitial>());
      expect(controller.selectedFile, isNull);
    });

    test(
      'pickFile handles invalid file with validation error in state',
      () async {
        const invalidFile = SelectedFile(
          name: 'script.exe',
          size: 1024,
          bytes: [1, 2, 3],
          extension: 'exe',
        );
        mockPicker.fileToReturn = invalidFile;

        await controller.pickFile();

        expect(controller.state, isA<UploadFileSelected>());
        final selectedState = controller.state as UploadFileSelected;
        expect(selectedState.isValid, isFalse);
        expect(
          selectedState.validationError,
          contains('Unsupported file type'),
        );
      },
    );

    test('pickFile handles read failure with UploadFailure state and canRetry false', () async {
      mockPicker.errorToThrow = const FileSystemException(
        'Permission denied reading file',
      );

      await controller.pickFile();

      expect(controller.state, isA<UploadFailure>());
      final failureState = controller.state as UploadFailure;
      expect(failureState.canRetry, isFalse);
      expect(failureState.message, contains('Permission denied'));
    });

    test('clearFile resets state to UploadInitial', () {
      controller.setFile(validFile);
      expect(controller.selectedFile, isNotNull);

      controller.clearFile();

      expect(controller.state, isA<UploadInitial>());
      expect(controller.selectedFile, isNull);
    });

    test('uploadToPersonalVault fails when no file is selected', () async {
      final doc = await controller.uploadToPersonalVault();

      expect(doc, isNull);
      expect(controller.state, isA<UploadFailure>());
      expect(fakeDocumentsService.uploadCallCount, 0);
    });

    test(
      'uploadToPersonalVault fails when selected file has validation error',
      () async {
        const oversizedFile = SelectedFile(
          name: 'big.pdf',
          size: 25 * 1024 * 1024,
          bytes: [1, 2, 3],
          extension: 'pdf',
        );
        controller.setFile(oversizedFile);

        final doc = await controller.uploadToPersonalVault();

        expect(doc, isNull);
        expect(fakeDocumentsService.uploadCallCount, 0);
      },
    );

    test('uploadToPersonalVault transitions through InProgress to UploadSuccess on success', () async {
      fakeDocumentsService.vaultDocToReturn = sampleUploadedDoc;
      controller.setFile(validFile);

      final states = <UploadState>[];
      controller.addListener(() {
        states.add(controller.state);
      });

      final doc = await controller.uploadToPersonalVault();

      expect(doc, isNotNull);
      expect(doc!.id, 10);
      expect(states.any((s) => s is UploadInProgress), isTrue);
      expect(controller.state, isA<UploadSuccess>());
      final success = controller.state as UploadSuccess;
      expect(success.uploadedDocument.id, 10);
      expect(success.message, contains('uploaded successfully'));
      expect(controller.isUploading, isFalse);
    });

    test('uploadToPersonalVault handles backend ApiException (e.g. 413) with UploadFailure', () async {
      fakeDocumentsService.errorToThrow = const ApiException(
        statusCode: 413,
        message: 'File size exceeds maximum permitted 20 MB',
      );
      controller.setFile(validFile);

      final doc = await controller.uploadToPersonalVault();

      expect(doc, isNull);
      expect(controller.state, isA<UploadFailure>());
      final failure = controller.state as UploadFailure;
      expect(failure.statusCode, 413);
      expect(failure.message, 'File size exceeds maximum permitted 20 MB');
      expect(failure.canRetry, isTrue);
      expect(controller.isUploading, isFalse);
    });

    test(
      'uploadToPersonalVault handles network error without claiming success',
      () async {
        fakeDocumentsService.errorToThrow = Exception('Network lost');
        controller.setFile(validFile);

        final doc = await controller.uploadToPersonalVault();

        expect(doc, isNull);
        expect(controller.state, isA<UploadFailure>());
        final failure = controller.state as UploadFailure;
        expect(failure.message, contains('Network error'));
        expect(failure.canRetry, isTrue);
      },
    );

    test(
      'prevents duplicate upload submissions while an upload is in progress',
      () async {
        fakeDocumentsService.vaultDocToReturn = sampleUploadedDoc;
        controller.setFile(validFile);

        // Trigger first upload without waiting
        final future1 = controller.uploadToPersonalVault();
        // Second attempt while first is active
        final future2 = controller.uploadToPersonalVault();

        await Future.wait([future1, future2]);

        expect(fakeDocumentsService.uploadCallCount, 1);
      },
    );

    test('submitToSubgroup with new file uploads to vault and then submits to subgroup', () async {
      fakeDocumentsService.vaultDocToReturn = sampleUploadedDoc;
      fakeDocumentsService.subgroupDocToReturn = sampleUploadedDoc;
      controller.setFile(validFile);

      final doc = await controller.submitToSubgroup(groupId: 42, subgroupId: 7);

      expect(doc, isNotNull);
      expect(fakeDocumentsService.uploadCallCount, 1);
      expect(fakeDocumentsService.submitCallCount, 1);
      expect(fakeDocumentsService.capturedDocId, 10);
      expect(fakeDocumentsService.capturedGroupId, 42);
      expect(fakeDocumentsService.capturedSubgroupId, 7);
      expect(controller.state, isA<UploadSuccess>());
    });

    test('submitToSubgroup with existingVaultDocId directly copies document without vault upload', () async {
      fakeDocumentsService.subgroupDocToReturn = sampleUploadedDoc;

      final doc = await controller.submitToSubgroup(
        groupId: 42,
        subgroupId: 7,
        existingVaultDocId: 99,
      );

      expect(doc, isNotNull);
      expect(fakeDocumentsService.uploadCallCount, 0);
      expect(fakeDocumentsService.submitCallCount, 1);
      expect(fakeDocumentsService.capturedDocId, 99);
      expect(fakeDocumentsService.capturedGroupId, 42);
      expect(fakeDocumentsService.capturedSubgroupId, 7);
      expect(controller.state, isA<UploadSuccess>());
    });

    test('submitToSubgroup handles 403 Forbidden with clear access period error message', () async {
      fakeDocumentsService.vaultDocToReturn = sampleUploadedDoc;
      fakeDocumentsService.submitErrorToThrow = const ApiException(
        statusCode: 403,
        message: 'Forbidden',
      );
      controller.setFile(validFile);

      final doc = await controller.submitToSubgroup(groupId: 42, subgroupId: 7);

      expect(doc, isNull);
      expect(controller.state, isA<UploadFailure>());
      final failure = controller.state as UploadFailure;
      expect(failure.statusCode, 403);
      expect(failure.message, contains('collection window may be closed'));
    });

    test('reset clears state back to UploadInitial', () {
      controller.setFile(validFile);
      expect(controller.selectedFile, isNotNull);

      controller.reset();

      expect(controller.state, isA<UploadInitial>());
      expect(controller.selectedFile, isNull);
      expect(controller.isUploading, isFalse);
    });
  });
}
