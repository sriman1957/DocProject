import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/notifications/services/notifications_service.dart';
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
  group('NotificationsService', () {
    test(
      'getNotifications calls GET /notifications with Bearer token',
      () async {
        String? capturedPath;
        String? capturedAuth;

        final mockClient = MockClient((request) async {
          capturedPath = request.url.path;
          capturedAuth = request.headers['Authorization'];

          final body = jsonEncode([
            {
              'id': 1,
              'recipient_user_id': 10,
              'actor_user_id': 2,
              'group_id': 5,
              'subgroup_id': 12,
              'document_id': null,
              'type': 'access_period_opened',
              'message': 'Collection period is now open',
              'is_read': false,
              'created_at': '2026-10-10T10:00:00Z',
            },
          ]);
          return http.Response(
            body,
            200,
            headers: {'content-type': 'application/json'},
          );
        });

        final service = NotificationsService(
          apiClient: ApiClient(
            baseUrl: 'http://localhost',
            httpClient: mockClient,
          ),
          tokenStorage: InMemoryTokenStorage(token: 'user-token'),
        );

        final items = await service.getNotifications();

        expect(capturedPath, '/notifications');
        expect(capturedAuth, 'Bearer user-token');
        expect(items.length, 1);
        expect(items.first.id, 1);
        expect(items.first.isRead, isFalse);
      },
    );

    test('markAsRead calls PATCH /notifications/:id/read', () async {
      String? capturedPath;
      String? capturedMethod;

      final mockClient = MockClient((request) async {
        capturedPath = request.url.path;
        capturedMethod = request.method;

        final body = jsonEncode({
          'id': 7,
          'recipient_user_id': 10,
          'actor_user_id': 2,
          'group_id': 5,
          'type': 'document_submitted',
          'message': 'Document submitted',
          'is_read': true,
          'created_at': '2026-10-10T10:00:00Z',
        });
        return http.Response(
          body,
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final service = NotificationsService(
        apiClient: ApiClient(
          baseUrl: 'http://localhost',
          httpClient: mockClient,
        ),
        tokenStorage: InMemoryTokenStorage(token: 'user-token'),
      );

      final updated = await service.markAsRead(7);

      expect(capturedPath, '/notifications/7/read');
      expect(capturedMethod, 'PATCH');
      expect(updated.id, 7);
      expect(updated.isRead, isTrue);
    });

    test(
      'getUnreadCount calls GET /notifications/unread-count and returns count',
      () async {
        String? capturedPath;

        final mockClient = MockClient((request) async {
          capturedPath = request.url.path;
          return http.Response(
            jsonEncode({'unread_count': 4}),
            200,
            headers: {'content-type': 'application/json'},
          );
        });

        final service = NotificationsService(
          apiClient: ApiClient(
            baseUrl: 'http://localhost',
            httpClient: mockClient,
          ),
          tokenStorage: InMemoryTokenStorage(token: 'user-token'),
        );

        final count = await service.getUnreadCount();

        expect(capturedPath, '/notifications/unread-count');
        expect(count, 4);
      },
    );

    test('getNotifications throws 401 when unauthenticated', () async {
      final service = NotificationsService(
        apiClient: ApiClient(baseUrl: 'http://localhost'),
        tokenStorage: InMemoryTokenStorage(token: null),
      );

      expect(
        () => service.getNotifications(),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401),
        ),
      );
    });
  });
}
