import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/documents/documents_controller.dart';
import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeDocumentsService extends DocumentsService {
  FakeDocumentsService({this.documentsToReturn = const [], this.errorToThrow})
    : super(
        apiClient: ApiClient(baseUrl: 'http://localhost:8080'),
        tokenStorage: TokenStorage(),
      );

  List<VaultDocument> documentsToReturn;
  Object? errorToThrow;
  int callCount = 0;

  @override
  Future<List<VaultDocument>> getPersonalVaultDocuments() async {
    callCount++;
    if (errorToThrow != null) {
      throw errorToThrow!;
    }
    return documentsToReturn;
  }
}

void main() {
  group('DocumentsController', () {
    final sampleDocs = [
      const VaultDocument(
        id: 1,
        originalFilename: 'cert1.pdf',
        mimeType: 'application/pdf',
        fileSizeBytes: 1024,
        sha256: 'hash1',
        uploadedAt: null,
      ),
      const VaultDocument(
        id: 2,
        originalFilename: 'cert2.pdf',
        mimeType: 'application/pdf',
        fileSizeBytes: 2048,
        sha256: 'hash2',
        uploadedAt: null,
      ),
    ];

    test('initial state is DocumentsInitial', () {
      final service = FakeDocumentsService();
      final controller = DocumentsController(documentsService: service);

      expect(controller.state, isA<DocumentsInitial>());
      expect(controller.documents, isEmpty);
    });

    test(
      'loadDocuments transitions to Loading then Loaded when results exist',
      () async {
        final service = FakeDocumentsService(documentsToReturn: sampleDocs);
        final controller = DocumentsController(documentsService: service);

        final states = <DocumentsState>[];
        controller.addListener(() {
          states.add(controller.state);
        });

        await controller.loadDocuments();

        expect(states, [isA<DocumentsLoading>(), isA<DocumentsLoaded>()]);
        expect(controller.state, isA<DocumentsLoaded>());
        expect(controller.documents, sampleDocs);
        expect(service.callCount, 1);
      },
    );

    test('loadDocuments transitions to Empty when results are empty', () async {
      final service = FakeDocumentsService(documentsToReturn: const []);
      final controller = DocumentsController(documentsService: service);

      final states = <DocumentsState>[];
      controller.addListener(() {
        states.add(controller.state);
      });

      await controller.loadDocuments();

      expect(states, [isA<DocumentsLoading>(), isA<DocumentsEmpty>()]);
      expect(controller.state, isA<DocumentsEmpty>());
      expect(controller.documents, isEmpty);
    });

    test(
      'loadDocuments handles ApiException and transitions to Error',
      () async {
        final service = FakeDocumentsService(
          errorToThrow: const ApiException(
            statusCode: 403,
            message: 'Access denied: not a student.',
          ),
        );
        final controller = DocumentsController(documentsService: service);

        await controller.loadDocuments();

        expect(controller.state, isA<DocumentsError>());
        final errorState = controller.state as DocumentsError;
        expect(errorState.message, 'Access denied: not a student.');
        expect(errorState.statusCode, 403);
        expect(errorState.canRetry, isTrue);
      },
    );

    test('loadDocuments handles generic exception gracefully', () async {
      final service = FakeDocumentsService(
        errorToThrow: Exception('Disk failure'),
      );
      final controller = DocumentsController(documentsService: service);

      await controller.loadDocuments();

      expect(controller.state, isA<DocumentsError>());
      final errorState = controller.state as DocumentsError;
      expect(
        errorState.message,
        contains('Unable to load personal vault documents'),
      );
    });

    test('avoids duplicate network calls when already loaded', () async {
      final service = FakeDocumentsService(documentsToReturn: sampleDocs);
      final controller = DocumentsController(documentsService: service);

      await controller.loadDocuments();
      expect(service.callCount, 1);

      // Second call without force should be a no-op
      await controller.loadDocuments();
      expect(service.callCount, 1);
    });

    test('refresh forces network call even if already loaded', () async {
      final service = FakeDocumentsService(documentsToReturn: sampleDocs);
      final controller = DocumentsController(documentsService: service);

      await controller.loadDocuments();
      expect(service.callCount, 1);

      await controller.refresh();
      expect(service.callCount, 2);
    });

    test('getDocumentById returns matching document or null', () async {
      final service = FakeDocumentsService(documentsToReturn: sampleDocs);
      final controller = DocumentsController(documentsService: service);

      await controller.loadDocuments();

      expect(controller.getDocumentById(1)?.originalFilename, 'cert1.pdf');
      expect(controller.getDocumentById(2)?.originalFilename, 'cert2.pdf');
      expect(controller.getDocumentById(999), isNull);
    });

    test('clear resets controller to initial state', () async {
      final service = FakeDocumentsService(documentsToReturn: sampleDocs);
      final controller = DocumentsController(documentsService: service);

      await controller.loadDocuments();
      expect(controller.state, isA<DocumentsLoaded>());

      controller.clear();
      expect(controller.state, isA<DocumentsInitial>());
      expect(controller.documents, isEmpty);
    });
  });
}
