import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  group('ApiClient', () {
    test('GET decodes a JSON response', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          expect(request.method, 'GET');
          expect(request.url.path, '/health/live');

          return http.Response(jsonEncode({'status': 'alive'}), 200);
        }),
      );

      final result = await client.get('/health/live');

      expect(result, {'status': 'alive'});
      client.close();
    });

    test('POST sends JSON and decodes the response', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          expect(request.method, 'POST');
          expect(request.headers['content-type'], 'application/json');

          expect(jsonDecode(request.body), {'email': 'student@example.com'});

          return http.Response(jsonEncode({'success': true}), 200);
        }),
      );

      final result = await client.post(
        '/example',
        body: {'email': 'student@example.com'},
      );

      expect(result, {'success': true});
      client.close();
    });

    test('protected requests include a bearer token', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          expect(request.headers['authorization'], 'Bearer test-token');

          return http.Response(jsonEncode({'user_id': 1}), 200);
        }),
      );

      final result = await client.get('/auth/me', accessToken: 'test-token');

      expect(result, {'user_id': 1});
      client.close();
    });

    test('API errors preserve the server error message', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          return http.Response(
            jsonEncode({'error': 'invalid credentials'}),
            401,
          );
        }),
      );

      await expectLater(
        client.post('/auth/login'),
        throwsA(
          isA<ApiException>()
              .having((error) => error.statusCode, 'statusCode', 401)
              .having(
                (error) => error.message,
                'message',
                'invalid credentials',
              ),
        ),
      );

      client.close();
    });

    test('empty successful responses return null', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          return http.Response('', 204);
        }),
      );

      final result = await client.delete('/example');

      expect(result, isNull);
      client.close();
    });
  });
}
