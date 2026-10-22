import 'dart:convert';

import 'package:docproject/app.dart';
import 'package:docproject/app_dependencies.dart';
import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/auth/screens/login_screen.dart';
import 'package:docproject/features/documents/screens/personal_vault_screen.dart';
import 'package:docproject/features/groups/screens/groups_listing_screen.dart';
import 'package:docproject/features/home/screens/student_dashboard_screen.dart';
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

void main() {
  group('E2E Student Workflow Integration Test', () {
    late InMemoryTokenStorage tokenStorage;

    const studentUser = {
      'id': 101,
      'college_id': 1,
      'full_name': 'Aarav Patel',
      'email': 'aarav@college.edu',
      'role': 'student',
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

    const mockVaultDocs = [
      {
        'id': 501,
        'original_filename': 'lab1_report.pdf',
        'mime_type': 'application/pdf',
        'file_size_bytes': 102400,
        'sha256': 'a1b2c3d4e5f6',
        'uploaded_at': '2026-10-01T10:00:00Z',
      },
    ];

    const mockNotifications = [
      {
        'id': 1,
        'recipient_user_id': 101,
        'actor_user_id': 2,
        'group_id': 10,
        'subgroup_id': 20,
        'document_id': null,
        'type': 'access_period_opened',
        'message': 'Submission window is open for Cloud Computing Lab',
        'is_read': false,
        'created_at': '2026-10-10T10:00:00Z',
      },
    ];

    setUp(() {
      tokenStorage = InMemoryTokenStorage(token: 'student-valid-token');
    });

    http.Response handleRequest(http.Request request) {
      final path = request.url.path;

      // GET /auth/me
      if (path == '/auth/me') {
        return http.Response(
          jsonEncode(studentUser),
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

      // GET /documents (Personal Vault)
      if (path == '/documents' && request.method == 'GET') {
        return http.Response(
          jsonEncode(mockVaultDocs),
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

      // PATCH /notifications/1/read
      if (path == '/notifications/1/read') {
        final updated = Map<String, dynamic>.from(mockNotifications.first);
        updated['is_read'] = true;
        return http.Response(
          jsonEncode(updated),
          200,
          headers: {'content-type': 'application/json'},
        );
      }

      return http.Response('not found', 404);
    }

    testWidgets(
      'executes complete student workflow through all tabs and actions',
      (tester) async {
        final mockClient = MockClient(
          (request) async => handleRequest(request),
        );
        final apiClient = ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        );

        await tester.pumpWidget(
          DocProjectApp(
            dependencies: AppDependencies.create(
              apiClient: apiClient,
              tokenStorage: tokenStorage,
            ),
          ),
        );

        // Session restoration
        await tester.pump();
        await tester.pumpAndSettle();

        // Step 1: Confirm Student Dashboard appears with student details
        expect(find.byType(StudentDashboardScreen), findsOneWidget);
        expect(find.text('Aarav Patel'), findsOneWidget);
        expect(find.text('Student'), findsWidgets);

        // Step 2: Open Groups tab
        final groupsTab = find.text('Groups');
        await tester.tap(groupsTab);
        await tester.pumpAndSettle();

        expect(find.byType(GroupsListingScreen), findsOneWidget);
        expect(find.text('B.Tech CSE Year 3'), findsOneWidget);

        // Step 3: Open Group Details and then Subgroups
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
        expect(find.text('Submit Document'), findsOneWidget);
        // Faculty review buttons must NOT be visible to student
        expect(
          find.byKey(const Key('review_submissions_button')),
          findsNothing,
        );
        expect(
          find.byKey(const Key('collection_windows_button')),
          findsNothing,
        );

        // Return back to tabs (pop SubgroupDetails, SubgroupsListing, GroupDetails)
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();

        // Step 5: Switch to Personal Vault tab
        final vaultTab = find.text('Vault');
        await tester.tap(vaultTab);
        await tester.pumpAndSettle();

        expect(find.byType(PersonalVaultScreen), findsOneWidget);
        expect(find.text('lab1_report.pdf'), findsOneWidget);
        expect(find.text('PDF'), findsOneWidget);

        // Step 6: Switch to Notifications / Alerts tab
        final alertsTab = find.text('Alerts');
        await tester.tap(alertsTab);
        await tester.pumpAndSettle();

        expect(find.byType(NotificationsScreen), findsOneWidget);
        expect(
          find.text('Submission window is open for Cloud Computing Lab'),
          findsOneWidget,
        );

        // Step 7: Logout
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

        // Verify token cleared and returned to Login Form
        expect(tokenStorage.token, isNull);
        expect(find.byType(LoginScreen), findsOneWidget);
        expect(find.widgetWithText(FilledButton, 'Sign in'), findsOneWidget);
      },
    );
  });
}
