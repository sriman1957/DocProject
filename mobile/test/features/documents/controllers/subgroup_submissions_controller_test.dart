import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/documents/controllers/subgroup_submissions_controller.dart';
import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/services/file_download_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

class MockFileDownloadService implements FileDownloadService {
  MockFileDownloadService({
    this.savedPathToReturn = '/mock/downloads/test.pdf',
  });

  String? savedPathToReturn;
  String? capturedFilename;
  List<int>? capturedBytes;
  String? capturedMimeType;
  int callCount = 0;
  Object? errorToThrow;

  @override
  Future<String?> saveFile({
    required String filename,
    required List<int> bytes,
    String mimeType = 'application/octet-stream',
  }) async {
    callCount++;
    capturedFilename = filename;
    capturedBytes = bytes;
    capturedMimeType = mimeType;
    if (errorToThrow != null) throw errorToThrow!;
    return savedPathToReturn;
  }
}

class FakeDocumentsService extends DocumentsService {
  FakeDocumentsService()
    : super(
        apiClient: ApiClient(baseUrl: 'http://localhost:8080'),
        tokenStorage: TokenStorage(),
      );

  List<VaultDocument> submissionsToReturn = const [];
  http.Response? downloadResponseToReturn;
  Object? submissionsErrorToThrow;
  Object? downloadErrorToThrow;

  int getSubmissionsCallCount = 0;
  int downloadCallCount = 0;
  int? capturedGroupId;
  int? capturedSubgroupId;
  int? capturedDocumentId;

  @override
  Future<List<VaultDocument>> getSubgroupSubmissions({
    required int groupId,
    required int subgroupId,
  }) async {
    getSubmissionsCallCount++;
    capturedGroupId = groupId;
    capturedSubgroupId = subgroupId;

    if (submissionsErrorToThrow != null) {
      throw submissionsErrorToThrow!;
    }
    return submissionsToReturn;
  }

  @override
  Future<http.Response> downloadSubgroupDocument({
    required int groupId,
    required int subgroupId,
    required int documentId,
  }) async {
    downloadCallCount++;
    capturedGroupId = groupId;
    capturedSubgroupId = subgroupId;
    capturedDocumentId = documentId;

    if (downloadErrorToThrow != null) {
      throw downloadErrorToThrow!;
    }
    return downloadResponseToReturn!;
  }
}

void main() {
  group('SubgroupSubmissionsController', () {
    const testDoc = VaultDocument(
      id: 50,
      originalFilename: 'assignment1.pdf',
      mimeType: 'application/pdf',
      fileSizeBytes: 2048,
      sha256: 'fedcba9876543210',
      uploadedAt: null,
    );

    late FakeDocumentsService fakeDocumentsService;
    late MockFileDownloadService mockFileDownloadService;
    late SubgroupSubmissionsController controller;

    setUp(() {
      fakeDocumentsService = FakeDocumentsService();
      mockFileDownloadService = MockFileDownloadService();
      controller = SubgroupSubmissionsController(
        documentsService: fakeDocumentsService,
        groupId: 10,
        subgroupId: 20,
        fileDownloadService: mockFileDownloadService,
      );
    });

    test(
      'initial state is SubmissionsInitial and downloadingDocIds is empty',
      () {
        expect(controller.state, isA<SubmissionsInitial>());
        expect(controller.downloadingDocIds, isEmpty);
        expect(controller.isDownloading(50), isFalse);
      },
    );

    test(
      'loadSubmissions transitions to Loading then Loaded when results exist',
      () async {
        fakeDocumentsService.submissionsToReturn = [testDoc];

        final states = <SubgroupSubmissionsState>[];
        controller.addListener(() {
          states.add(controller.state);
        });

        await controller.loadSubmissions();

        expect(states.length, 2);
        expect(states[0], isA<SubmissionsLoading>());
        expect(states[1], isA<SubmissionsLoaded>());

        final loaded = controller.state as SubmissionsLoaded;
        expect(loaded.documents.length, 1);
        expect(loaded.documents.first.id, 50);
        expect(fakeDocumentsService.capturedGroupId, 10);
        expect(fakeDocumentsService.capturedSubgroupId, 20);
      },
    );

    test('loadSubmissions transitions to SubmissionsEmpty when empty list returned', () async {
      fakeDocumentsService.submissionsToReturn = [];

      final states = <SubgroupSubmissionsState>[];
      controller.addListener(() {
        states.add(controller.state);
      });

      await controller.loadSubmissions();

      expect(states.length, 2);
      expect(states[0], isA<SubmissionsLoading>());
      expect(states[1], isA<SubmissionsEmpty>());
    });

    test(
      'loadSubmissions transitions to SubmissionsError on 403 Forbidden',
      () async {
        fakeDocumentsService.submissionsErrorToThrow = const ApiException(
          statusCode: 403,
          message: 'Forbidden',
        );

        await controller.loadSubmissions();

        expect(controller.state, isA<SubmissionsError>());
        final error = controller.state as SubmissionsError;
        expect(error.statusCode, 403);
        expect(
          error.message,
          'Access denied. You must be an authorized faculty member or admin for this group.',
        );
      },
    );

    test(
      'loadSubmissions transitions to SubmissionsError on 404 Not Found',
      () async {
        fakeDocumentsService.submissionsErrorToThrow = const ApiException(
          statusCode: 404,
          message: 'Not found',
        );

        await controller.loadSubmissions();

        expect(controller.state, isA<SubmissionsError>());
        final error = controller.state as SubmissionsError;
        expect(error.statusCode, 404);
        expect(error.message, 'Subgroup not found or inactive.');
      },
    );

    test(
      'loadSubmissions transitions to SubmissionsError on network error',
      () async {
        fakeDocumentsService.submissionsErrorToThrow = Exception(
          'Network down',
        );

        await controller.loadSubmissions();

        expect(controller.state, isA<SubmissionsError>());
        final error = controller.state as SubmissionsError;
        expect(
          error.message,
          'Network error. Please check your connection and try again.',
        );
        expect(error.canRetry, isTrue);
      },
    );

    test('loadSubmissions does not trigger duplicate API calls if already loading or loaded', () async {
      fakeDocumentsService.submissionsToReturn = [testDoc];

      await controller.loadSubmissions();
      expect(fakeDocumentsService.getSubmissionsCallCount, 1);

      // Calling again without force should be a no-op
      await controller.loadSubmissions();
      expect(fakeDocumentsService.getSubmissionsCallCount, 1);
    });

    test('refresh() forces a reload even when already loaded', () async {
      fakeDocumentsService.submissionsToReturn = [testDoc];

      await controller.loadSubmissions();
      expect(fakeDocumentsService.getSubmissionsCallCount, 1);

      await controller.refresh();
      expect(fakeDocumentsService.getSubmissionsCallCount, 2);
    });

    group('downloadDocument', () {
      test('successfully downloads and invokes fileDownloadService with parsed Content-Disposition filename', () async {
        final mockBytes = [10, 20, 30, 40];
        fakeDocumentsService.downloadResponseToReturn = http.Response.bytes(
          mockBytes,
          200,
          headers: {
            'content-type': 'application/pdf',
            'content-disposition':
                'attachment; filename="custom_server_name.pdf"',
          },
        );

        final result = await controller.downloadDocument(testDoc);

        expect(result, '/mock/downloads/test.pdf');
        expect(fakeDocumentsService.capturedGroupId, 10);
        expect(fakeDocumentsService.capturedSubgroupId, 20);
        expect(fakeDocumentsService.capturedDocumentId, 50);

        expect(mockFileDownloadService.callCount, 1);
        expect(
          mockFileDownloadService.capturedFilename,
          'custom_server_name.pdf',
        );
        expect(mockFileDownloadService.capturedBytes, mockBytes);
        expect(mockFileDownloadService.capturedMimeType, 'application/pdf');

        // Document ID should be removed from downloading set when completed
        expect(controller.isDownloading(50), isFalse);
        expect(controller.downloadingDocIds, isEmpty);
      });

      test(
        'falls back to doc.originalFilename if Content-Disposition is absent',
        () async {
          final mockBytes = [50, 60];
          fakeDocumentsService.downloadResponseToReturn = http.Response.bytes(
            mockBytes,
            200,
            headers: {'content-type': 'application/pdf'},
          );

          await controller.downloadDocument(testDoc);

          expect(mockFileDownloadService.capturedFilename, 'assignment1.pdf');
        },
      );

      test('prevents duplicate concurrent downloads for the same document', () async {
        fakeDocumentsService.downloadResponseToReturn = http.Response.bytes([
          1,
          2,
        ], 200);

        // Start first download
        final firstFuture = controller.downloadDocument(testDoc);
        expect(controller.isDownloading(50), isTrue);

        // Second call while first is in progress should immediately return null
        final secondCall = await controller.downloadDocument(testDoc);
        expect(secondCall, isNull);

        await firstFuture;
        expect(controller.isDownloading(50), isFalse);
      });

      test(
        'cleans up downloading state and rethrows on download failure',
        () async {
          fakeDocumentsService.downloadErrorToThrow = const ApiException(
            statusCode: 403,
            message: 'Not authorized to download file',
          );

          await expectLater(
            controller.downloadDocument(testDoc),
            throwsA(
              isA<ApiException>().having(
                (e) => e.statusCode,
                'statusCode',
                403,
              ),
            ),
          );

          // Ensure downloading state was cleared despite error
          expect(controller.isDownloading(50), isFalse);
          expect(controller.downloadingDocIds, isEmpty);
        },
      );
    });
  });
}
