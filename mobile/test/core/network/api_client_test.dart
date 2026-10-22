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

    test('postMultipart sends multipart request with token, fields, and file bytes', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient.streaming((request, bodyStream) async {
          expect(request, isA<http.MultipartRequest>());
          final multipartRequest = request as http.MultipartRequest;
          expect(multipartRequest.method, 'POST');
          expect(multipartRequest.url.path, '/documents');
          expect(
            multipartRequest.headers['authorization'],
            'Bearer test-token',
          );
          expect(multipartRequest.fields['category'], 'assignment');
          expect(multipartRequest.files.length, 1);
          expect(multipartRequest.files.first.field, 'file');
          expect(multipartRequest.files.first.filename, 'test.pdf');

          final responseBody = jsonEncode({
            'id': 42,
            'original_filename': 'test.pdf',
          });
          return http.StreamedResponse(
            Stream.value(utf8.encode(responseBody)),
            201,
            headers: {'content-type': 'application/json'},
          );
        }),
      );

      final result = await client.postMultipart(
        '/documents',
        fieldName: 'file',
        filename: 'test.pdf',
        bytes: [1, 2, 3, 4],
        fields: {'category': 'assignment'},
        accessToken: 'test-token',
      );

      expect(result, {'id': 42, 'original_filename': 'test.pdf'});
      client.close();
    });

    test(
      'postMultipart throws ApiException on server error response',
      () async {
        final client = ApiClient(
          baseUrl: 'http://localhost:8080',
          httpClient: MockClient((request) async {
            return http.Response(
              jsonEncode({'error': 'file size exceeds 20MB'}),
              413,
              headers: {'content-type': 'application/json'},
            );
          }),
        );

        await expectLater(
          client.postMultipart(
            '/documents',
            fieldName: 'file',
            filename: 'large.pdf',
            bytes: [1, 2, 3],
            accessToken: 'test-token',
          ),
          throwsA(
            isA<ApiException>()
                .having((e) => e.statusCode, 'statusCode', 413)
                .having((e) => e.message, 'message', 'file size exceeds 20MB'),
          ),
        );

        client.close();
      },
    );

    test('postMultipart throws ApiException on network failure', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          throw http.ClientException('Network down');
        }),
      );

      await expectLater(
        client.postMultipart(
          '/documents',
          fieldName: 'file',
          filename: 'doc.pdf',
          bytes: [1],
        ),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 0),
        ),
      );

      client.close();
    });

    test(
      'getBytes sends GET request with bearer token and returns response bytes',
      () async {
        final client = ApiClient(
          baseUrl: 'http://localhost:8080',
          httpClient: MockClient((request) async {
            expect(request.method, 'GET');
            expect(
              request.url.path,
              '/groups/10/subgroups/20/documents/100/download',
            );
            expect(request.headers['authorization'], 'Bearer faculty-jwt');
            expect(request.headers['accept'], '*/*');

            return http.Response.bytes(
              [37, 80, 68, 70],
              200,
              headers: {
                'content-type': 'application/pdf',
                'content-disposition': 'attachment; filename="certificate.pdf"',
              },
            );
          }),
        );

        final response = await client.getBytes(
          '/groups/10/subgroups/20/documents/100/download',
          accessToken: 'faculty-jwt',
        );

        expect(response.statusCode, 200);
        expect(response.bodyBytes, [37, 80, 68, 70]);
        expect(response.headers['content-type'], 'application/pdf');
        expect(
          response.headers['content-disposition'],
          'attachment; filename="certificate.pdf"',
        );
        client.close();
      },
    );

    test('getBytes throws ApiException on server error response without returning error body as file', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          return http.Response(
            jsonEncode({'error': 'forbidden: only faculty can download'}),
            403,
            headers: {'content-type': 'application/json'},
          );
        }),
      );

      await expectLater(
        client.getBytes(
          '/groups/10/subgroups/20/documents/100/download',
          accessToken: 'student-token',
        ),
        throwsA(
          isA<ApiException>()
              .having((e) => e.statusCode, 'statusCode', 403)
              .having(
                (e) => e.message,
                'message',
                'forbidden: only faculty can download',
              ),
        ),
      );

      client.close();
    });

    test('getBytes throws ApiException on network failure', () async {
      final client = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: MockClient((request) async {
          throw http.ClientException('Connection aborted');
        }),
      );

      await expectLater(
        client.getBytes('/groups/10/subgroups/20/documents/100/download'),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 0),
        ),
      );

      client.close();
    });
  });
}
