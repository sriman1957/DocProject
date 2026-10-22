import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/access_periods/services/access_periods_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class InMemoryTokenStorage extends TokenStorage {
  InMemoryTokenStorage({this.token});
  String? token;

  @override
  Future<String?> readAccessToken() async => token;
  @override
  Future<void> saveAccessToken(String accessToken) async => token = accessToken;
  @override
  Future<void> clearAll() async => token = null;
}

void main() {
  group('AccessPeriodsService', () {
    test('getSubgroupAccessPeriods calls GET with Bearer token', () async {
      String? capturedPath;
      String? capturedAuth;

      final mockClient = MockClient((request) async {
        capturedPath = request.url.path;
        capturedAuth = request.headers['Authorization'];

        final body = jsonEncode([
          {
            'id': 10,
            'subgroup_id': 25,
            'group_id': 5,
            'college_id': 1,
            'starts_at': '2026-10-01T00:00:00Z',
            'ends_at': '2026-10-10T23:59:59Z',
            'created_by': 8,
          },
        ]);
        return http.Response(
          body,
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = AccessPeriodsService(
        apiClient: ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        ),
        tokenStorage: InMemoryTokenStorage(token: 'faculty-token'),
      );

      final periods = await service.getSubgroupAccessPeriods(
        groupId: 5,
        subgroupId: 25,
      );

      expect(capturedPath, '/groups/5/subgroups/25/access-periods');
      expect(capturedAuth, 'Bearer faculty-token');
      expect(periods.length, 1);
      expect(periods.first.id, 10);
    });

    test(
      'getSubgroupAccessPeriods throws 403 when forbidden (e.g. student)',
      () async {
        final mockClient = MockClient((request) async {
          return http.Response('forbidden', 403);
        });

        final service = AccessPeriodsService(
          apiClient: ApiClient(
            baseUrl: 'http://localhost',
            httpClient: mockClient,
          ),
          tokenStorage: InMemoryTokenStorage(token: 'student-token'),
        );

        expect(
          () => service.getSubgroupAccessPeriods(groupId: 5, subgroupId: 25),
          throwsA(
            isA<ApiException>().having((e) => e.statusCode, 'statusCode', 403),
          ),
        );
      },
    );

    test('createAccessPeriod sends POST with UTC ISO timestamps', () async {
      String? capturedPath;
      String? capturedBody;

      final mockClient = MockClient((request) async {
        capturedPath = request.url.path;
        capturedBody = request.body;

        final response = {
          'id': 11,
          'subgroup_id': 25,
          'group_id': 5,
          'college_id': 1,
          'starts_at': '2026-11-01T00:00:00Z',
          'ends_at': '2026-11-10T23:59:59Z',
          'created_by': 8,
        };
        return http.Response(
          jsonEncode(response),
          201,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = AccessPeriodsService(
        apiClient: ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        ),
        tokenStorage: InMemoryTokenStorage(token: 'faculty-token'),
      );

      final start = DateTime.parse('2026-11-01T00:00:00Z');
      final end = DateTime.parse('2026-11-10T23:59:59Z');

      final period = await service.createAccessPeriod(
        groupId: 5,
        subgroupId: 25,
        startsAt: start,
        endsAt: end,
      );

      expect(capturedPath, '/groups/5/subgroups/25/access-periods');
      expect(capturedBody, contains('2026-11-01T00:00:00.000Z'));
      expect(capturedBody, contains('2026-11-10T23:59:59.000Z'));
      expect(period.id, 11);
    });

    test(
      'createAccessPeriod rejects client-side if endsAt is before startsAt',
      () async {
        final service = AccessPeriodsService(
          apiClient: ApiClient(baseUrl: 'http://localhost'),
          tokenStorage: InMemoryTokenStorage(token: 'faculty-token'),
        );

        final start = DateTime.parse('2026-11-10T00:00:00Z');
        final end = DateTime.parse('2026-11-01T00:00:00Z');

        expect(
          () => service.createAccessPeriod(
            groupId: 5,
            subgroupId: 25,
            startsAt: start,
            endsAt: end,
          ),
          throwsA(
            isA<ApiException>().having((e) => e.statusCode, 'statusCode', 400),
          ),
        );
      },
    );

    test(
      'createAccessPeriod throws 409 when backend reports overlap conflict',
      () async {
        final mockClient = MockClient((request) async {
          return http.Response(
            'access period overlaps an existing period',
            409,
          );
        });

        final service = AccessPeriodsService(
          apiClient: ApiClient(
            baseUrl: 'http://localhost',
            httpClient: mockClient,
          ),
          tokenStorage: InMemoryTokenStorage(token: 'faculty-token'),
        );

        expect(
          () => service.createAccessPeriod(
            groupId: 5,
            subgroupId: 25,
            startsAt: DateTime.parse('2026-11-01T00:00:00Z'),
            endsAt: DateTime.parse('2026-11-05T00:00:00Z'),
          ),
          throwsA(
            isA<ApiException>().having((e) => e.statusCode, 'statusCode', 409),
          ),
        );
      },
    );
  });
}
