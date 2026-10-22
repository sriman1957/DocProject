import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/documents/controllers/document_upload_controller.dart';
import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/services/file_picker_service.dart';
import 'package:docproject/features/documents/widgets/file_upload_sheet.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class MockFilePickerService implements FilePickerService {
  MockFilePickerService({this.fileToReturn});

  SelectedFile? fileToReturn;

  @override
  Future<SelectedFile?> pickFile() async => fileToReturn;
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
  int uploadCallCount = 0;
  int submitCallCount = 0;
  int? capturedGroupId;
  int? capturedSubgroupId;
  int? capturedDocId;

  @override
  Future<VaultDocument> uploadPersonalVaultDocument(SelectedFile file) async {
    uploadCallCount++;
    if (errorToThrow != null) throw errorToThrow!;
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
    if (errorToThrow != null) throw errorToThrow!;
    return subgroupDocToReturn!;
  }
}

void main() {
  late FakeDocumentsService fakeDocumentsService;
  late MockFilePickerService mockPicker;
  late DocumentUploadController controller;

  const validPdfFile = SelectedFile(
    name: 'thesis_proposal.pdf',
    size: 2048,
    bytes: [1, 2, 3],
    extension: 'pdf',
  );

  final sampleUploadedDoc = VaultDocument(
    id: 1,
    originalFilename: 'thesis_proposal.pdf',
    mimeType: 'application/pdf',
    fileSizeBytes: 2048,
    sha256: 'sha256hash',
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

  Widget buildSheet({
    UploadDestination destination = UploadDestination.personalVault,
    int? groupId,
    int? subgroupId,
    String? subgroupName,
    List<VaultDocument> vaultDocs = const [],
    VoidCallback? onSuccess,
  }) {
    return MaterialApp(
      home: Scaffold(
        body: FileUploadSheet(
          uploadController: controller,
          destination: destination,
          groupId: groupId,
          subgroupId: subgroupId,
          subgroupName: subgroupName,
          vaultDocuments: vaultDocs,
          onSuccess: onSuccess,
        ),
      ),
    );
  }

  group('FileUploadSheet', () {
    testWidgets(
      'renders initial upload sheet with tap to select prompt and disabled upload button',
      (tester) async {
        await tester.pumpWidget(buildSheet());

        expect(find.text('Upload to Personal Vault'), findsOneWidget);
        expect(find.text('Tap to select a document'), findsOneWidget);
        expect(
          find.text('Supported: PDF, DOCX, PNG, JPEG (Max 20 MB)'),
          findsOneWidget,
        );

        final uploadButton = tester.widget<FilledButton>(
          find.widgetWithText(FilledButton, 'Upload'),
        );
        expect(uploadButton.onPressed, isNull);
      },
    );

    testWidgets(
      'tapping select prompt picks a file and displays file name and size',
      (tester) async {
        mockPicker.fileToReturn = validPdfFile;

        await tester.pumpWidget(buildSheet());

        await tester.tap(find.text('Tap to select a document'));
        await tester.pumpAndSettle();

        expect(find.text('thesis_proposal.pdf'), findsOneWidget);
        expect(find.text('2.0 KB'), findsOneWidget);

        final uploadButton = tester.widget<FilledButton>(
          find.widgetWithText(FilledButton, 'Upload'),
        );
        expect(uploadButton.onPressed, isNotNull);
      },
    );

    testWidgets(
      'displays validation error and disables upload button for unsupported file',
      (tester) async {
        const invalidFile = SelectedFile(
          name: 'program.exe',
          size: 1024,
          bytes: [1, 2, 3],
          extension: 'exe',
        );
        mockPicker.fileToReturn = invalidFile;

        await tester.pumpWidget(buildSheet());

        await tester.tap(find.text('Tap to select a document'));
        await tester.pumpAndSettle();

        expect(find.text('program.exe'), findsOneWidget);
        expect(find.textContaining('Unsupported file type'), findsOneWidget);

        final uploadButton = tester.widget<FilledButton>(
          find.widgetWithText(FilledButton, 'Upload'),
        );
        expect(uploadButton.onPressed, isNull);
      },
    );

    testWidgets(
      'successful upload displays success view and calls onSuccess callback',
      (tester) async {
        fakeDocumentsService.vaultDocToReturn = sampleUploadedDoc;
        mockPicker.fileToReturn = validPdfFile;
        var successCalled = false;

        await tester.pumpWidget(
          buildSheet(onSuccess: () => successCalled = true),
        );

        await tester.tap(find.text('Tap to select a document'));
        await tester.pumpAndSettle();

        await tester.tap(find.widgetWithText(FilledButton, 'Upload'));
        await tester.pumpAndSettle();

        expect(find.text('Success!'), findsOneWidget);
        expect(find.textContaining('uploaded successfully'), findsOneWidget);
        expect(find.text('Done'), findsOneWidget);
        expect(successCalled, isTrue);
      },
    );

    testWidgets('upload failure displays error banner and allows retry', (
      tester,
    ) async {
      fakeDocumentsService.errorToThrow = const ApiException(
        statusCode: 413,
        message: 'File size exceeds maximum permitted 20 MB',
      );
      mockPicker.fileToReturn = validPdfFile;

      await tester.pumpWidget(buildSheet());

      await tester.tap(find.text('Tap to select a document'));
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(FilledButton, 'Upload'));
      await tester.pumpAndSettle();

      expect(
        find.text('File size exceeds maximum permitted 20 MB'),
        findsOneWidget,
      );

      // Upload button is re-enabled for retry
      final uploadButton = tester.widget<FilledButton>(
        find.widgetWithText(FilledButton, 'Upload'),
      );
      expect(uploadButton.onPressed, isNotNull);
    });

    testWidgets(
      'subgroup destination displays subgroup name and tabs when vault documents available',
      (tester) async {
        await tester.pumpWidget(
          buildSheet(
            destination: UploadDestination.subgroup,
            groupId: 10,
            subgroupId: 5,
            subgroupName: 'Semester 1 Marks',
            vaultDocs: [sampleUploadedDoc],
          ),
        );

        expect(find.text('Submit Document'), findsOneWidget);
        expect(find.text('Destination: Semester 1 Marks'), findsOneWidget);
        expect(find.text('From Device'), findsOneWidget);
        expect(find.text('From Vault'), findsOneWidget);
      },
    );

    testWidgets(
      'subgroup submission using existing vault document submits without device upload',
      (tester) async {
        fakeDocumentsService.subgroupDocToReturn = sampleUploadedDoc;

        await tester.pumpWidget(
          buildSheet(
            destination: UploadDestination.subgroup,
            groupId: 10,
            subgroupId: 5,
            subgroupName: 'Semester 1 Marks',
            vaultDocs: [sampleUploadedDoc],
          ),
        );

        // Switch to From Vault tab
        await tester.tap(find.text('From Vault'));
        await tester.pumpAndSettle();

        // Tap the vault document tile
        await tester.tap(find.text('thesis_proposal.pdf'));
        await tester.pumpAndSettle();

        // Submit button should be enabled
        await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
        await tester.pumpAndSettle();

        expect(fakeDocumentsService.uploadCallCount, 0);
        expect(fakeDocumentsService.submitCallCount, 1);
        expect(fakeDocumentsService.capturedGroupId, 10);
        expect(fakeDocumentsService.capturedSubgroupId, 5);
        expect(fakeDocumentsService.capturedDocId, 1);
        expect(find.text('Success!'), findsOneWidget);
      },
    );
  });
}
