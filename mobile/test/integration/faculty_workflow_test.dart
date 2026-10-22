import 'dart:convert';

import 'package:docproject/app.dart';
import 'package:docproject/app_dependencies.dart';
import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/access_periods/controllers/access_periods_controller.dart';
import 'package:docproject/features/access_periods/screens/subgroup_access_periods_screen.dart';
import 'package:docproject/features/access_periods/services/access_periods_service.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/auth/screens/login_screen.dart';
import 'package:docproject/features/documents/controllers/subgroup_submissions_controller.dart';
import 'package:docproject/features/documents/screens/subgroup_submissions_screen.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/services/file_download_service.dart';
import 'package:docproject/features/groups/screens/groups_listing_screen.dart';
import 'package:docproject/features/home/screens/faculty_dashboard_screen.dart';
import 'package:docproject/features/notifications/screens/notifications_screen.dart';
import 'package:docproject/features/subgroups/screens/subgroup_details_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class InMemoryTokenStorage extends TokenStorage {
  InMemoryTokenStorage({this.token, this.user});
  String? token;
  User? user;

  @override
  Future<String?> readAccessToken() async => token;
  @override
  Future<void> saveAccessToken(String accessToken) async => token = accessToken;
  @override
  Future<void> deleteAccessToken() async => token = null;

  @override
  Future<User?> readUser() async => user;
  @override
  Future<void> saveUser(User user) async => this.user = user;
  @override
  Future<void> deleteUser() async => user = null;

  @override
  Future<void> clearAll() async {
    token = null;
    user = null;
  }
}

class FakeFileDownloadService implements FileDownloadService {
  @override
  Future<String?> saveFile({
    required String filename,
    required List<int> bytes,
    String mimeType = 'application/octet-stream',
  }) async {
    return '/fake/downloads/$filename';
  }
}

void main() {
  group('E2E Faculty Workflow Integration Test', () {
    late InMemoryTokenStorage tokenStorage;

    const facultyUser = {
      'id': 2,
      'college_id': 1,
      'full_name': 'Dr. Suresh Kumar',
      'email': 'suresh@college.edu',
      'role': 'faculty',
    };

    const mockGroup = {
      'id': 10,
      'college_id': 1,
      'branch_id': 1,
      'name': 'B.Tech CSE Year 3',
      'description': 'Computer Science Department 2024-2025',
      'created_by': 2,
    };

    const mockSubgroup = {
      'id': 20,
      'college_id': 1,
      'group_id': 10,
      'name': 'Cloud Computing Lab',
      'description': 'Submit weekly assignment files',
      'created_by': 2,
      'is_active': true,
      'created_at': '2026-09-01T00:00:00Z',
      'updated_at': '2026-09-01T00:00:00Z',
    };

    const mockAccessPeriods = [
      {
        'id': 1,
        'college_id': 1,
        'group_id': 10,
        'subgroup_id': 20,
        'starts_at': '2026-10-01T00:00:00Z',
        'ends_at': '2026-12-31T23:59:59Z',
        'created_by': 2,
        'created_at': '2026-10-01T00:00:00Z',
      },
    ];

    const mockSubmissions = [
      {
        'id': 501,
        'original_filename': 'lab1_report.pdf',
        'mime_type': 'application/pdf',
        'file_size_bytes': 102400,
        'sha256': 'abcdef123456',
        'uploaded_at': '2026-10-05T12:00:00Z',
      },
    ];

    const mockNotifications = [
      {
        'id': 1,
        'recipient_user_id': 2,
        'actor_user_id': 101,
        'group_id': 10,
        'subgroup_id': 20,
        'document_id': 501,
        'type': 'document_submitted',
        'message': 'Aarav Patel submitted a document in Cloud Computing Lab',
        'is_read': false,
        'created_at': '2026-10-10T10:00:00Z',
      },
    ];

    setUp(() {
      tokenStorage = InMemoryTokenStorage(token: 'faculty-valid-token');
    });

    http.Response handleRequest(http.Request request) {
      final path = request.url.path;

      // GET /auth/me
      if (path == '/auth/me') {
        return http.Response(
          jsonEncode(facultyUser),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      // GET /groups
      if (path == '/groups') {
        return http.Response(
          jsonEncode([mockGroup]),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      // GET /groups/10/subgroups
      if (path == '/groups/10/subgroups') {
        return http.Response(
          jsonEncode([mockSubgroup]),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      // GET /groups/10/subgroups/20/access-periods
      if (path == '/groups/10/subgroups/20/access-periods') {
        return http.Response(
          jsonEncode(mockAccessPeriods),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      // GET /groups/10/subgroups/20/documents
      if (path == '/groups/10/subgroups/20/documents') {
        return http.Response(
          jsonEncode(mockSubmissions),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      // GET /notifications
      if (path == '/notifications' && request.method == 'GET') {
        return http.Response(
          jsonEncode(mockNotifications),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      // GET /notifications/unread-count
      if (path == '/notifications/unread-count') {
        return http.Response(
          jsonEncode({'unread_count': 1}),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      return http.Response('not found', 404);
    }

    testWidgets(
      'executes complete faculty workflow through dashboard, groups, windows, review and logout',
      (tester) async {
        final mockClient = MockClient(
          (request) async => handleRequest(request),
        );
        final apiClient = ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        );

        final accessPeriodsController = AccessPeriodsController(
          accessPeriodsService: AccessPeriodsService(
            apiClient: apiClient,
            tokenStorage: tokenStorage,
          ),
          groupId: 10,
          subgroupId: 20,
        );

        final submissionsController = SubgroupSubmissionsController(
          documentsService: DocumentsService(
            apiClient: apiClient,
            tokenStorage: tokenStorage,
          ),
          groupId: 10,
          subgroupId: 20,
          fileDownloadService: FakeFileDownloadService(),
        );

        await tester.pumpWidget(
          DocProjectApp(
            dependencies: AppDependencies.create(
              apiClient: apiClient,
              tokenStorage: tokenStorage,
            ),
            accessPeriodsController: accessPeriodsController,
            submissionsController: submissionsController,
          ),
        );

        // Restore session
        await tester.pump();
        await tester.pumpAndSettle();

        // Step 1: Confirm Faculty Dashboard appears
        expect(find.byType(FacultyDashboardScreen), findsOneWidget);
        expect(find.text('Dr. Suresh Kumar'), findsOneWidget);
        expect(find.text('Faculty'), findsWidgets);

        // Step 2: Open Groups tab
        final groupsTab = find.text('Groups');
        await tester.tap(groupsTab);
        await tester.pumpAndSettle();

        expect(find.byType(GroupsListingScreen), findsOneWidget);
        expect(find.text('B.Tech CSE Year 3'), findsOneWidget);

        // Step 3: Open Group Details and navigate to Subgroups
        await tester.tap(find.text('B.Tech CSE Year 3'));
        await tester.pumpAndSettle();

        expect(find.text('View Collection Subgroups'), findsOneWidget);
        await tester.tap(find.text('View Collection Subgroups'));
        await tester.pumpAndSettle();

        expect(find.text('Cloud Computing Lab'), findsOneWidget);

        // Step 4: Open Subgroup Details
        await tester.tap(find.text('Cloud Computing Lab'));
        await tester.pumpAndSettle();

        expect(find.byType(SubgroupDetailsScreen), findsOneWidget);
        // Student submit button should NOT be visible to faculty
        expect(
          find.byKey(const Key('submit_subgroup_document_button')),
          findsNothing,
        );
        // Faculty buttons MUST be visible
        expect(
          find.byKey(const Key('collection_windows_button')),
          findsOneWidget,
        );
        expect(
          find.byKey(const Key('review_submissions_button')),
          findsOneWidget,
        );

        // Step 5: Open Collection Windows
        final windowsBtn = find.byKey(const Key('collection_windows_button'));
        await tester.ensureVisible(windowsBtn);
        await tester.pumpAndSettle();
        await tester.tap(windowsBtn);
        await tester.pumpAndSettle();

        expect(find.byType(SubgroupAccessPeriodsScreen), findsOneWidget);
        expect(find.text('Active (Open)'), findsOneWidget);
        expect(find.text('Window #1'), findsOneWidget);

        // Return back to Subgroup Details
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();

        // Step 6: Open Review Submissions
        final reviewBtn = find.byKey(const Key('review_submissions_button'));
        await tester.ensureVisible(reviewBtn);
        await tester.pumpAndSettle();
        await tester.tap(reviewBtn);
        await tester.pumpAndSettle();

        expect(find.byType(SubgroupSubmissionsScreen), findsOneWidget);
        expect(find.text('lab1_report.pdf'), findsOneWidget);

        // Return back to Subgroup Details
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();

        // Step 7: Return back to AppShell tabs
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();

        // Step 8: Switch to Notifications tab
        final alertsTab = find.text('Alerts');
        await tester.tap(alertsTab);
        await tester.pumpAndSettle();

        expect(find.byType(NotificationsScreen), findsOneWidget);
        expect(
          find.text('Aarav Patel submitted a document in Cloud Computing Lab'),
          findsOneWidget,
        );

        // Step 9: Logout
        final logoutBtn = find.byTooltip('Sign out');
        await tester.tap(logoutBtn);
        await tester.pumpAndSettle();

        expect(
          find.text('Are you sure you want to sign out of DocProject?'),
          findsOneWidget,
        );
        final confirmBtn = find.widgetWithText(FilledButton, 'Sign out');
        await tester.tap(confirmBtn);
        await tester.pumpAndSettle();

        // Step 10: Verify Token cleared and Login screen rendered
        expect(tokenStorage.token, isNull);
        expect(find.byType(LoginScreen), findsOneWidget);
        expect(find.widgetWithText(FilledButton, 'Sign in'), findsOneWidget);
      },
    );
  });
}
