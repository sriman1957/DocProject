import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/groups/services/groups_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class FakeTokenStorage extends TokenStorage {
  String? token;

  @override
  Future<String?> readAccessToken() async => token;

  @override
  Future<void> saveAccessToken(String value) async {
    token = value;
  }

  @override
  Future<void> deleteAccessToken() async {
    token = null;
  }
}

void main() {
  late FakeTokenStorage tokenStorage;

  setUp(() {
    tokenStorage = FakeTokenStorage();
  });

  group('GroupsService', () {
    test('sends bearer token and parses list of groups', () async {
      await tokenStorage.saveAccessToken('secret-jwt-token');

      final client = MockClient((request) async {
        expect(request.url.path, '/groups');
        expect(request.method, 'GET');
        expect(request.headers['Authorization'], 'Bearer secret-jwt-token');

        final body = jsonEncode([
          {
            'id': 100,
            'college_id': 7,
            'branch_id': 3,
            'name': 'IT 2024-2028',
            'description': 'IT department batch',
            'created_by': 42,
            'is_active': true,
            'created_at': '2026-10-01 10:00:00',
            'updated_at': '2026-10-01 10:00:00',
          },
        ]);

        return http.Response(
          body,
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = GroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      final groups = await service.getGroups();

      expect(groups.length, 1);
      expect(groups.first.id, 100);
      expect(groups.first.name, 'IT 2024-2028');
      expect(groups.first.branchId, 3);
      expect(groups.first.description, 'IT department batch');
    });

    test('returns empty list when backend response is empty array', () async {
      await tokenStorage.saveAccessToken('valid-token');

      final client = MockClient((request) async {
        return http.Response(
          '[]',
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = GroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      final groups = await service.getGroups();

      expect(groups, isEmpty);
    });

    test(
      'throws 401 ApiException when no token is present in storage',
      () async {
        final client = MockClient((request) async {
          throw StateError('Should not be called');
        });

        final service = GroupsService(
          apiClient: ApiClient(httpClient: client),
          tokenStorage: tokenStorage,
        );

        expect(
          () => service.getGroups(),
          throwsA(
            isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401),
          ),
        );
      },
    );

    test('propagates 403 Forbidden ApiException from backend', () async {
      await tokenStorage.saveAccessToken('unauthorized-token');

      final client = MockClient((request) async {
        return http.Response(
          jsonEncode({'error': 'forbidden'}),
          403,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = GroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getGroups(),
        throwsA(
          isA<ApiException>()
              .having((e) => e.statusCode, 'statusCode', 403)
              .having((e) => e.message, 'message', 'forbidden'),
        ),
      );
    });

    test(
      'propagates network error message when server is unreachable',
      () async {
        await tokenStorage.saveAccessToken('any-token');

        final client = MockClient((request) async {
          throw http.ClientException('Network down');
        });

        final service = GroupsService(
          apiClient: ApiClient(httpClient: client),
          tokenStorage: tokenStorage,
        );

        expect(
          () => service.getGroups(),
          throwsA(
            isA<ApiException>().having(
              (e) => e.message,
              'message',
              contains('Unable to connect to the server'),
            ),
          ),
        );
      },
    );
  });
}
