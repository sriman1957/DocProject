import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/features/auth/services/auth_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  group('AuthService', () {
    test('login sends credentials and parses the response', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          expect(request.method, 'POST');
          expect(request.url.path, '/auth/login');

          final body = jsonDecode(request.body) as Map<String, dynamic>;

          expect(body['college_code'], 'DOCPROISO');
          expect(body['email'], 'student@example.com');
          expect(body['password'], 'test-password');

          return http.Response(
            jsonEncode({
              'access_token': 'test-token',
              'user': {
                'id': 7,
                'college_id': 1,
                'full_name': 'Test Student',
                'email': 'student@example.com',
                'role': 'student',
              },
            }),
            200,
          );
        }),
      );

      final service = AuthService(client);

      final result = await service.login(
        collegeCode: ' DOCPROISO ',
        email: ' student@example.com ',
        password: 'test-password',
      );

      expect(result.accessToken, 'test-token');
      expect(result.user.fullName, 'Test Student');
      expect(result.user.role, 'student');

      client.close();
    });

    test('login propagates invalid credential errors', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          return http.Response(
            jsonEncode({'error': 'invalid credentials'}),
            401,
          );
        }),
      );

      final service = AuthService(client);

      await expectLater(
        service.login(
          collegeCode: 'DOCPROISO',
          email: 'student@example.com',
          password: 'wrong-password',
        ),
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

    test('getMe sends bearer token and parses user response', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          expect(request.method, 'GET');
          expect(request.url.path, '/auth/me');
          expect(request.headers['authorization'], 'Bearer valid-jwt-token');

          return http.Response(
            jsonEncode({'user_id': 7, 'college_id': 1, 'role': 'student'}),
            200,
          );
        }),
      );

      final service = AuthService(client);

      final user = await service.getMe(accessToken: 'valid-jwt-token');

      expect(user.id, 7);
      expect(user.collegeId, 1);
      expect(user.role, 'student');

      client.close();
    });

    test('getMe propagates 401 unauthorized error on expired token', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          return http.Response(
            jsonEncode({'error': 'invalid or expired access token'}),
            401,
          );
        }),
      );

      final service = AuthService(client);

      await expectLater(
        service.getMe(accessToken: 'expired-token'),
        throwsA(
          isA<ApiException>()
              .having((error) => error.statusCode, 'statusCode', 401)
              .having(
                (error) => error.message,
                'message',
                'invalid or expired access token',
              ),
        ),
      );

      client.close();
    });
  });
}
