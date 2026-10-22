import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/access_periods/controllers/access_periods_controller.dart';
import 'package:docproject/features/access_periods/models/access_period.dart';
import 'package:docproject/features/access_periods/screens/subgroup_access_periods_screen.dart';
import 'package:docproject/features/access_periods/services/access_periods_service.dart';
import 'package:docproject/features/auth/auth_controller.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/auth/services/auth_service.dart';
import 'package:docproject/features/documents/controllers/subgroup_submissions_controller.dart';
import 'package:docproject/features/documents/screens/subgroup_submissions_screen.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/services/file_download_service.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/subgroups/models/subgroup.dart';
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
  group('Security, Boundary, and Negative Integrations Test', () {
    const studentUser = User(
      id: 101,
      collegeId: 1,
      fullName: 'Aarav Patel',
      email: 'aarav@college.edu',
      role: 'student',
    );

    const parentGroup = Group(
      id: 10,
      collegeId: 1,
      branchId: 1,
      name: 'B.Tech CSE Year 3',
      description: 'Computer Science Department',
      createdBy: 2,
      isActive: true,
    );

    final subgroup = Subgroup(
      id: 20,
      collegeId: 1,
      groupId: 10,
      name: 'Cloud Computing Lab',
      description: 'Submit weekly assignment files',
      createdBy: 2,
      isActive: true,
      createdAt: '2026-09-01T00:00:00Z',
      updatedAt: '2026-09-01T00:00:00Z',
    );

    test(
      'access period half-open boundary conditions [starts_at, ends_at)',
      () {
        final startsAt = DateTime.utc(2026, 10, 1, 9, 0, 0);
        final endsAt = DateTime.utc(2026, 10, 15, 17, 0, 0);

        final period = AccessPeriod(
          id: 1,
          subgroupId: 20,
          groupId: 10,
          collegeId: 1,
          startsAt: startsAt,
          endsAt: endsAt,
          createdBy: 2,
        );

        // Boundary: Exactly 1 second before starts_at -> upcoming (closed)
        final justBefore = startsAt.subtract(const Duration(seconds: 1));
        expect(
          period.getStatus(justBefore),
          equals(AccessPeriodStatus.upcoming),
        );
        expect(period.isActiveNow(justBefore), isFalse);

        // Boundary: Exactly at starts_at -> active (open)
        expect(period.getStatus(startsAt), equals(AccessPeriodStatus.active));
        expect(period.isActiveNow(startsAt), isTrue);

        // Boundary: Inside interval -> active (open)
        final mid = startsAt.add(const Duration(days: 5));
        expect(period.getStatus(mid), equals(AccessPeriodStatus.active));
        expect(period.isActiveNow(mid), isTrue);

        // Boundary: Exactly 1 millisecond before ends_at -> active (open)
        final justBeforeEnd = endsAt.subtract(const Duration(milliseconds: 1));
        expect(
          period.getStatus(justBeforeEnd),
          equals(AccessPeriodStatus.active),
        );
        expect(period.isActiveNow(justBeforeEnd), isTrue);

        // Boundary: Exactly at ends_at -> expired (closed, half-open upper boundary)
        expect(period.getStatus(endsAt), equals(AccessPeriodStatus.expired));
        expect(period.isActiveNow(endsAt), isFalse);

        // Boundary: After ends_at -> expired (closed)
        final afterEnd = endsAt.add(const Duration(hours: 1));
        expect(period.getStatus(afterEnd), equals(AccessPeriodStatus.expired));
        expect(period.isActiveNow(afterEnd), isFalse);
      },
    );

    testWidgets(
      'student is blocked with 403 Forbidden from accessing collection periods management',
      (tester) async {
        final tokenStorage = InMemoryTokenStorage(token: 'student-token');
        final mockClient = MockClient((request) async {
          if (request.url.path.contains('/access-periods')) {
            return http.Response(
              jsonEncode({
                'error': 'Students are forbidden from viewing or managing collection periods',
              }),
              403,
              headers: {'content-type': 'application/json'},
            );
          }
          return http.Response('not found', 404);
        });

        final apiClient = ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        );
        final controller = AccessPeriodsController(
          accessPeriodsService: AccessPeriodsService(
            apiClient: apiClient,
            tokenStorage: tokenStorage,
          ),
          groupId: 10,
          subgroupId: 20,
        );

        await tester.pumpWidget(
          MaterialApp(
            home: SubgroupAccessPeriodsScreen(
              user: studentUser,
              parentGroup: parentGroup,
              subgroup: subgroup,
              controller: controller,
            ),
          ),
        );

        // Let controller finish loading
        await tester.pump();
        await tester.pumpAndSettle();

        // Verify access denied screen renders without exposing periods
        expect(find.text('Faculty Access Only'), findsOneWidget);
        expect(
          find.text(
            'Collection window management is strictly restricted to authorized faculty members.',
          ),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'student is blocked with 403 Forbidden from accessing faculty submission review screen',
      (tester) async {
        final tokenStorage = InMemoryTokenStorage(token: 'student-token');
        final mockClient = MockClient((request) async {
          return http.Response(
            jsonEncode({
              'error': 'Only faculty may review subgroup submissions',
            }),
            403,
            headers: {'content-type': 'application/json'},
          );
        });

        final apiClient = ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        );
        final controller = SubgroupSubmissionsController(
          documentsService: DocumentsService(
            apiClient: apiClient,
            tokenStorage: tokenStorage,
          ),
          groupId: 10,
          subgroupId: 20,
          fileDownloadService: FakeFileDownloadService(),
        );

        await tester.pumpWidget(
          MaterialApp(
            home: SubgroupSubmissionsScreen(
              user: studentUser,
              parentGroup: parentGroup,
              subgroup: subgroup,
              controller: controller,
            ),
          ),
        );

        await tester.pump();
        await tester.pumpAndSettle();

        // Verify access restricted message
        expect(find.text('Faculty Access Only'), findsOneWidget);
        expect(
          find.text(
            'Subgroup submissions are strictly restricted to authorized faculty members.',
          ),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'handles 409 conflict when creating overlapping access period',
      (tester) async {
        final tokenStorage = InMemoryTokenStorage(token: 'faculty-token');
        final mockClient = MockClient((request) async {
          if (request.method == 'POST' &&
              request.url.path.contains('/access-periods')) {
            return http.Response(
              jsonEncode({
                'error': 'An access period already overlaps with the specified window',
              }),
              409,
              headers: {'content-type': 'application/json'},
            );
          }
          if (request.method == 'GET' &&
              request.url.path.contains('/access-periods')) {
            return http.Response(
              jsonEncode([]),
              200,
              headers: {'content-type': 'application/json'},
            );
          }
          return http.Response('not found', 404);
        });

        final apiClient = ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        );
        final controller = AccessPeriodsController(
          accessPeriodsService: AccessPeriodsService(
            apiClient: apiClient,
            tokenStorage: tokenStorage,
          ),
          groupId: 10,
          subgroupId: 20,
        );

        await controller.loadAccessPeriods();
        expect(controller.state, isA<AccessPeriodsEmpty>());

        // Attempt to create overlapping period
        expect(
          () => controller.createAccessPeriod(
            startsAt: DateTime.utc(2026, 11, 1),
            endsAt: DateTime.utc(2026, 11, 10),
          ),
          throwsA(
            isA<ApiException>().having((e) => e.statusCode, 'statusCode', 409),
          ),
        );
      },
    );

    test('expired 401 token during session restore cleans stored token and sets Unauthenticated', () async {
      final tokenStorage = InMemoryTokenStorage(
        token: 'expired-token',
        user: studentUser,
      );

      final mockClient = MockClient((request) async {
        return http.Response(
          jsonEncode({'error': 'token expired'}),
          401,
          headers: {'content-type': 'application/json'},
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost',
        httpClient: mockClient,
      );
      final authService = AuthService(apiClient);
      final authController = AuthController(
        authService: authService,
        tokenStorage: tokenStorage,
      );

      await authController.restoreSession(force: true);

      // Verifies token deleted on authoritative 401
      expect(tokenStorage.token, isNull);
      expect(authController.state, isA<Unauthenticated>());
    });

    test(
      'network failure during session restore preserves token for retry',
      () async {
        final tokenStorage = InMemoryTokenStorage(
          token: 'valid-persisted-token',
          user: studentUser,
        );

        final mockClient = MockClient((request) async {
          throw http.ClientException('Network unreachable');
        });

        final apiClient = ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        );
        final authService = AuthService(apiClient);
        final authController = AuthController(
          authService: authService,
          tokenStorage: tokenStorage,
        );

        await authController.restoreSession(force: true);

        // Token MUST NOT be wiped on transient network failure
        expect(tokenStorage.token, equals('valid-persisted-token'));
        expect(authController.state, isA<AuthSessionCheckFailure>());
      },
    );
  });
}
