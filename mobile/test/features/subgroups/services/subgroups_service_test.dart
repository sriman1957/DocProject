import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/subgroups/services/subgroups_service.dart';
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

  group('SubgroupsService', () {
    test(
      'sends bearer token and parses list of subgroups for specific group',
      () async {
        await tokenStorage.saveAccessToken('secret-jwt-token');

        final client = MockClient((request) async {
          expect(request.url.path, '/groups/42/subgroups');
          expect(request.method, 'GET');
          expect(request.headers['Authorization'], 'Bearer secret-jwt-token');

          final body = jsonEncode([
            {
              'id': 1,
              'college_id': 7,
              'group_id': 42,
              'name': 'Semester 1 Marks',
              'description': 'Submit original grade sheets',
              'created_by': 10,
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

        final service = SubgroupsService(
          apiClient: ApiClient(httpClient: client),
          tokenStorage: tokenStorage,
        );

        final subgroups = await service.getSubgroups(42);

        expect(subgroups.length, 1);
        expect(subgroups.first.id, 1);
        expect(subgroups.first.groupId, 42);
        expect(subgroups.first.name, 'Semester 1 Marks');
        expect(subgroups.first.description, 'Submit original grade sheets');
      },
    );

    test('returns empty list when backend returns empty array', () async {
      await tokenStorage.saveAccessToken('valid-token');

      final client = MockClient((request) async {
        return http.Response(
          '[]',
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = SubgroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      final subgroups = await service.getSubgroups(42);

      expect(subgroups, isEmpty);
    });

    test('throws 400 ApiException when group ID is invalid (<= 0)', () async {
      final client = MockClient((request) async {
        throw StateError('Should not be called');
      });

      final service = SubgroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getSubgroups(0),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 400),
        ),
      );
    });

    test(
      'throws 401 ApiException when no token is present in storage',
      () async {
        final client = MockClient((request) async {
          throw StateError('Should not be called');
        });

        final service = SubgroupsService(
          apiClient: ApiClient(httpClient: client),
          tokenStorage: tokenStorage,
        );

        expect(
          () => service.getSubgroups(42),
          throwsA(
            isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401),
          ),
        );
      },
    );

    test('propagates 403 Forbidden ApiException from backend', () async {
      await tokenStorage.saveAccessToken('student-token');

      final client = MockClient((request) async {
        return http.Response(
          jsonEncode({'error': 'forbidden'}),
          403,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = SubgroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getSubgroups(42),
        throwsA(
          isA<ApiException>()
              .having((e) => e.statusCode, 'statusCode', 403)
              .having((e) => e.message, 'message', 'forbidden'),
        ),
      );
    });

    test('propagates 404 Group Not Found ApiException from backend', () async {
      await tokenStorage.saveAccessToken('valid-token');

      final client = MockClient((request) async {
        return http.Response(
          jsonEncode({'error': 'group not found'}),
          404,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = SubgroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getSubgroups(999),
        throwsA(
          isA<ApiException>()
              .having((e) => e.statusCode, 'statusCode', 404)
              .having((e) => e.message, 'message', 'group not found'),
        ),
      );
    });

    test('propagates network error message when connection fails', () async {
      await tokenStorage.saveAccessToken('any-token');

      final client = MockClient((request) async {
        throw http.ClientException('Connection dropped');
      });

      final service = SubgroupsService(
        apiClient: ApiClient(httpClient: client),
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getSubgroups(42),
        throwsA(
          isA<ApiException>().having(
            (e) => e.message,
            'message',
            contains('Unable to connect to the server'),
          ),
        ),
      );
    });
  });
}
