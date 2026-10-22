import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/documents/documents_controller.dart';
import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:docproject/features/documents/screens/personal_vault_screen.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/widgets/document_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeDocumentsService extends DocumentsService {
  FakeDocumentsService({
    this.documentsToReturn = const [],
    this.errorToThrow,
    this.delay,
  }) : super(
         apiClient: ApiClient(baseUrl: 'http://localhost:8080'),
         tokenStorage: TokenStorage(),
       );

  List<VaultDocument> documentsToReturn;
  Object? errorToThrow;
  Duration? delay;
  int callCount = 0;

  @override
  Future<List<VaultDocument>> getPersonalVaultDocuments() async {
    callCount++;
    if (delay != null) {
      await Future<void>.delayed(delay!);
    }
    if (errorToThrow != null) {
      throw errorToThrow!;
    }
    return documentsToReturn;
  }
}

void main() {
  const studentUser = User(
    id: 42,
    collegeId: 7,
    fullName: 'Aditi Sharma',
    email: 'aditi@college.edu',
    role: 'student',
  );

  Widget createTestWidget({
    required DocumentsController controller,
    bool autoLoad = true,
  }) {
    return MaterialApp(
      home: PersonalVaultScreen(
        user: studentUser,
        controller: controller,
        autoLoad: autoLoad,
      ),
    );
  }

  group('PersonalVaultScreen', () {
    testWidgets('displays loading indicator while documents are loading', (
      tester,
    ) async {
      final service = FakeDocumentsService(
        delay: const Duration(milliseconds: 300),
      );
      final controller = DocumentsController(documentsService: service);

      await tester.pumpWidget(createTestWidget(controller: controller));
      // First frame shows loading
      await tester.pump();

      expect(find.byKey(const Key('vault_loading_indicator')), findsOneWidget);

      await tester.pumpAndSettle();
    });

    testWidgets('displays list of documents with cards and metadata', (
      tester,
    ) async {
      final docs = [
        VaultDocument(
          id: 1,
          originalFilename: 'sem1_transcript.pdf',
          mimeType: 'application/pdf',
          fileSizeBytes: 204800,
          sha256: 'abcdef1234567890abcdef1234567890',
          uploadedAt: DateTime.utc(2026, 9, 29, 10, 0, 0),
        ),
        VaultDocument(
          id: 2,
          originalFilename: 'internship_offer.docx',
          mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
          fileSizeBytes: 512000,
          sha256: 'fedcba0987654321fedcba0987654321',
          uploadedAt: DateTime.utc(2026, 10, 1, 14, 30, 0),
        ),
      ];

      final service = FakeDocumentsService(documentsToReturn: docs);
      final controller = DocumentsController(documentsService: service);

      await tester.pumpWidget(createTestWidget(controller: controller));
      await tester.pumpAndSettle();

      expect(find.text('Personal Vault'), findsOneWidget);
      expect(find.text('Private & Confidential'), findsOneWidget);
      expect(find.text('2 files'), findsOneWidget);
      expect(find.byType(DocumentCard), findsNWidgets(2));
      expect(find.text('sem1_transcript.pdf'), findsOneWidget);
      expect(find.text('internship_offer.docx'), findsOneWidget);
      expect(find.text('200.0 KB'), findsOneWidget);
      expect(find.text('500.0 KB'), findsOneWidget);
      expect(find.text('PDF'), findsOneWidget);
      expect(find.text('DOCX'), findsOneWidget);
    });

    testWidgets('displays empty state when vault has no documents', (
      tester,
    ) async {
      final service = FakeDocumentsService(documentsToReturn: const []);
      final controller = DocumentsController(documentsService: service);

      await tester.pumpWidget(createTestWidget(controller: controller));
      await tester.pumpAndSettle();

      expect(find.text('Your Personal Vault is empty'), findsOneWidget);
      expect(
        find.text('No documents have been stored in your personal vault yet.'),
        findsOneWidget,
      );
      expect(find.byIcon(Icons.folder_open_rounded), findsOneWidget);
      expect(find.widgetWithText(OutlinedButton, 'Refresh'), findsOneWidget);
    });

    testWidgets('displays error state and retries on button tap', (
      tester,
    ) async {
      final service = FakeDocumentsService(
        errorToThrow: const ApiException(
          statusCode: 500,
          message: 'Internal server error while fetching documents.',
        ),
      );
      final controller = DocumentsController(documentsService: service);

      await tester.pumpWidget(createTestWidget(controller: controller));
      await tester.pumpAndSettle();

      expect(find.text('Unable to load vault'), findsOneWidget);
      expect(
        find.text('Internal server error while fetching documents.'),
        findsOneWidget,
      );

      final retryButton = find.widgetWithText(FilledButton, 'Try Again');
      expect(retryButton, findsOneWidget);

      // Change service to return documents on retry
      service.errorToThrow = null;
      service.documentsToReturn = [
        const VaultDocument(
          id: 5,
          originalFilename: 'cert.pdf',
          mimeType: 'application/pdf',
          fileSizeBytes: 1024,
          sha256: 'hash',
          uploadedAt: null,
        ),
      ];

      await tester.tap(retryButton);
      await tester.pumpAndSettle();

      expect(find.text('cert.pdf'), findsOneWidget);
      expect(find.byType(DocumentCard), findsOneWidget);
    });

    testWidgets('handles extremely long filenames without overflow', (
      tester,
    ) async {
      final docs = [
        const VaultDocument(
          id: 10,
          originalFilename: 'very_long_academic_document_name_final_version_approved_by_head_of_department_and_examination_controller_2026_with_all_signatures.pdf',
          mimeType: 'application/pdf',
          fileSizeBytes: 1048576,
          sha256: 'abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef',
          uploadedAt: null,
        ),
      ];

      final service = FakeDocumentsService(documentsToReturn: docs);
      final controller = DocumentsController(documentsService: service);

      await tester.pumpWidget(createTestWidget(controller: controller));
      await tester.pumpAndSettle();

      expect(find.byType(DocumentCard), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('app bar refresh button triggers vault reload', (tester) async {
      final service = FakeDocumentsService(documentsToReturn: const []);
      final controller = DocumentsController(documentsService: service);

      await tester.pumpWidget(createTestWidget(controller: controller));
      await tester.pumpAndSettle();

      expect(service.callCount, 1);

      final refreshIconButton = find.byTooltip('Refresh vault');
      expect(refreshIconButton, findsOneWidget);

      await tester.tap(refreshIconButton);
      await tester.pumpAndSettle();

      expect(service.callCount, 2);
    });
  });
}
