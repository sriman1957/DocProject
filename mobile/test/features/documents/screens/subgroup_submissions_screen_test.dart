import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/documents/controllers/subgroup_submissions_controller.dart';
import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:docproject/features/documents/screens/subgroup_submissions_screen.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/services/file_download_service.dart';
import 'package:docproject/features/documents/widgets/subgroup_document_card.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/subgroups/models/subgroup.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

class FakeFileDownloadService implements FileDownloadService {
  FakeFileDownloadService({
    this.savedPath = '/storage/emulated/0/Download/test.pdf',
  });

  String? savedPath;
  int callCount = 0;
  Object? errorToThrow;

  @override
  Future<String?> saveFile({
    required String filename,
    required List<int> bytes,
    String mimeType = 'application/octet-stream',
  }) async {
    callCount++;
    if (errorToThrow != null) throw errorToThrow!;
    return savedPath;
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
  Duration? downloadDelay;

  int getSubmissionsCallCount = 0;
  int downloadCallCount = 0;
  Duration? submissionsDelay;

  @override
  Future<List<VaultDocument>> getSubgroupSubmissions({
    required int groupId,
    required int subgroupId,
  }) async {
    getSubmissionsCallCount++;
    if (submissionsDelay != null) {
      await Future<void>.delayed(submissionsDelay!);
    }
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
    if (downloadDelay != null) {
      await Future<void>.delayed(downloadDelay!);
    }
    if (downloadErrorToThrow != null) {
      throw downloadErrorToThrow!;
    }
    return downloadResponseToReturn ??
        http.Response.bytes(
          [1, 2, 3],
          200,
          headers: {
            'content-type': 'application/pdf',
            'content-disposition':
                'attachment; filename="document_$documentId.pdf"',
          },
        );
  }
}

void main() {
  const facultyUser = User(
    id: 5,
    collegeId: 10,
    fullName: 'Dr. Robert Smith',
    email: 'robert@college.edu',
    role: 'faculty',
    facultyCode: 'FAC101',
  );

  const studentUser = User(
    id: 12,
    collegeId: 10,
    fullName: 'Alice Student',
    email: 'alice@college.edu',
    role: 'student',
  );

  const unknownUser = User(
    id: 99,
    collegeId: 10,
    fullName: 'Guest User',
    email: 'guest@college.edu',
    role: 'guest',
  );

  const parentGroup = Group(
    id: 10,
    collegeId: 10,
    branchId: 1,
    name: 'Computer Science Dept',
    description: 'CS Academic Group',
    createdBy: 5,
  );

  const testSubgroup = Subgroup(
    id: 25,
    collegeId: 10,
    groupId: 10,
    name: 'Final Project Submissions',
    description: 'Upload your capstone thesis and code',
    createdBy: 5,
    isActive: true,
  );

  final sampleDocs = [
    VaultDocument(
      id: 101,
      originalFilename: 'capstone_thesis.pdf',
      mimeType: 'application/pdf',
      fileSizeBytes: 2048576,
      sha256:
          '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
      uploadedAt: DateTime.parse('2026-10-15T09:30:00.000Z'),
    ),
    VaultDocument(
      id: 102,
      originalFilename: 'architecture_diagram.png',
      mimeType: 'image/png',
      fileSizeBytes: 512000,
      sha256:
          '5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8',
      uploadedAt: DateTime.parse('2026-10-16T14:45:00.000Z'),
    ),
  ];

  Widget createWidget({
    required User user,
    required SubgroupSubmissionsController controller,
    bool autoLoad = true,
  }) {
    return MaterialApp(
      home: SubgroupSubmissionsScreen(
        user: user,
        parentGroup: parentGroup,
        subgroup: testSubgroup,
        controller: controller,
        autoLoad: autoLoad,
      ),
    );
  }

  group('SubgroupSubmissionsScreen Widget Tests', () {
    late FakeDocumentsService fakeDocumentsService;
    late FakeFileDownloadService fakeFileDownloadService;
    late SubgroupSubmissionsController controller;

    setUp(() {
      fakeDocumentsService = FakeDocumentsService();
      fakeFileDownloadService = FakeFileDownloadService();
      controller = SubgroupSubmissionsController(
        documentsService: fakeDocumentsService,
        groupId: 10,
        subgroupId: 25,
        fileDownloadService: fakeFileDownloadService,
      );
    });

    testWidgets('shows loading indicator initially', (tester) async {
      fakeDocumentsService.submissionsDelay = const Duration(milliseconds: 300);

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      // Advance one frame while request is in flight
      await tester.pump();

      expect(
        find.byKey(const Key('submissions_loading_indicator')),
        findsOneWidget,
      );

      // Settle pending timer
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pumpAndSettle();
    });

    testWidgets('displays submissions list with accurate metadata', (
      tester,
    ) async {
      fakeDocumentsService.submissionsToReturn = sampleDocs;

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      // Parent group banner
      expect(find.text('Parent Group: Computer Science Dept'), findsOneWidget);
      expect(
        find.text('Reviewing submissions for subgroup #25'),
        findsOneWidget,
      );

      // Check document cards
      expect(find.byType(SubgroupDocumentCard), findsNWidgets(2));
      expect(find.text('capstone_thesis.pdf'), findsOneWidget);
      expect(find.text('architecture_diagram.png'), findsOneWidget);

      // Check metadata: file types, sizes, sha256 badges
      expect(find.text('PDF'), findsOneWidget);
      expect(find.text('PNG'), findsOneWidget);
      expect(find.text('2.0 MB'), findsOneWidget);
      expect(find.text('500.0 KB'), findsOneWidget);
      expect(
        find.textContaining('SHA-256: 9f86d081884c7d65...'),
        findsOneWidget,
      );
      expect(find.text('#101'), findsOneWidget);
      expect(find.text('#102'), findsOneWidget);
    });

    testWidgets('displays empty state when subgroup has no submissions', (
      tester,
    ) async {
      fakeDocumentsService.submissionsToReturn = [];

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('submissions_empty_state')), findsOneWidget);
      expect(find.text('No Submissions Yet'), findsOneWidget);
      expect(
        find.text('No documents have been submitted to this subgroup yet.'),
        findsOneWidget,
      );
    });

    testWidgets('displays error state and retry button reloads submissions', (
      tester,
    ) async {
      fakeDocumentsService.submissionsErrorToThrow = const ApiException(
        statusCode: 500,
        message: 'Internal server failure',
      );

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('submissions_error_state')), findsOneWidget);
      expect(find.text('Unable to load submissions'), findsOneWidget);
      expect(find.text('Internal server failure'), findsOneWidget);

      // Now prepare success response and click Try Again
      fakeDocumentsService.submissionsErrorToThrow = null;
      fakeDocumentsService.submissionsToReturn = sampleDocs;

      final tryAgainBtn = find.text('Try Again');
      expect(tryAgainBtn, findsOneWidget);
      await tester.tap(tryAgainBtn);

      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.byType(SubgroupDocumentCard), findsNWidgets(2));
      expect(find.text('capstone_thesis.pdf'), findsOneWidget);
    });

    testWidgets(
      'tapping download triggers file download and shows success snackbar',
      (tester) async {
        fakeDocumentsService.submissionsToReturn = [sampleDocs.first];

        await tester.pumpWidget(
          createWidget(
            user: facultyUser,
            controller: controller,
            autoLoad: true,
          ),
        );

        await tester.pump();
        await tester.pumpAndSettle();

        final downloadBtn = find.byKey(const Key('download_button_101'));
        expect(downloadBtn, findsOneWidget);

        await tester.tap(downloadBtn);
        await tester.pump();
        await tester.pumpAndSettle();

        expect(fakeDocumentsService.downloadCallCount, 1);
        expect(fakeFileDownloadService.callCount, 1);

        // Verify success SnackBar
        expect(
          find.text('Saved "capstone_thesis.pdf" successfully.'),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'shows loading state on download button while download is in progress',
      (tester) async {
        fakeDocumentsService.submissionsToReturn = [sampleDocs.first];
        fakeDocumentsService.downloadDelay = const Duration(milliseconds: 500);

        await tester.pumpWidget(
          createWidget(
            user: facultyUser,
            controller: controller,
            autoLoad: true,
          ),
        );

        await tester.pump();
        await tester.pumpAndSettle();

        final downloadBtn = find.byKey(const Key('download_button_101'));
        await tester.tap(downloadBtn);
        await tester.pump(); // Advance 1 frame into download

        // Check downloading label and spinner inside button
        expect(find.text('Downloading...'), findsOneWidget);

        // Complete delay
        await tester.pump(const Duration(milliseconds: 500));
        await tester.pumpAndSettle();

        expect(find.text('Downloading...'), findsNothing);
        expect(find.text('Download'), findsOneWidget);
      },
    );

    testWidgets('shows error snackbar when download fails', (tester) async {
      fakeDocumentsService.submissionsToReturn = [sampleDocs.first];
      fakeDocumentsService.downloadErrorToThrow = const ApiException(
        statusCode: 404,
        message: 'File not found on storage server',
      );

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      final downloadBtn = find.byKey(const Key('download_button_101'));
      await tester.tap(downloadBtn);
      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text('File not found on storage server'), findsOneWidget);
    });

    testWidgets('student role is blocked with Faculty Access Only screen', (
      tester,
    ) async {
      await tester.pumpWidget(
        createWidget(user: studentUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text('Faculty Access Only'), findsOneWidget);
      expect(
        find.text(
          'Subgroup submissions are strictly restricted to authorized faculty members.',
        ),
        findsOneWidget,
      );
      // Submissions should not have loaded
      expect(fakeDocumentsService.getSubmissionsCallCount, 0);
      expect(find.byType(SubgroupDocumentCard), findsNothing);
    });

    testWidgets('unknown role is blocked with Faculty Access Only screen', (
      tester,
    ) async {
      await tester.pumpWidget(
        createWidget(user: unknownUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text('Faculty Access Only'), findsOneWidget);
      expect(fakeDocumentsService.getSubmissionsCallCount, 0);
      expect(find.byType(SubgroupDocumentCard), findsNothing);
    });

    testWidgets('refresh button in AppBar reloads submissions', (tester) async {
      fakeDocumentsService.submissionsToReturn = [sampleDocs.first];

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      expect(fakeDocumentsService.getSubmissionsCallCount, 1);

      final refreshBtn = find.byKey(const Key('refresh_submissions_button'));
      expect(refreshBtn, findsOneWidget);

      await tester.tap(refreshBtn);
      await tester.pump();
      await tester.pumpAndSettle();

      expect(fakeDocumentsService.getSubmissionsCallCount, 2);
    });
  });
}
