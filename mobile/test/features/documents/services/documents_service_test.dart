import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class InMemoryTokenStorage extends TokenStorage {
  InMemoryTokenStorage({this.token});

  String? token;

  @override
  Future<String?> readAccessToken() async => token;

  @override
  Future<void> saveAccessToken(String accessToken) async {
    token = accessToken;
  }

  @override
  Future<void> clearAll() async {
    token = null;
  }
}

void main() {
  group('DocumentsService', () {
    test('calls GET /documents with Bearer token authentication', () async {
      String? capturedPath;
      String? capturedAuthHeader;

      final mockClient = MockClient((request) async {
        capturedPath = request.url.path;
        capturedAuthHeader = request.headers['Authorization'];

        final body = jsonEncode([
          {
            'id': 1,
            'original_filename': 'resume.pdf',
            'mime_type': 'application/pdf',
            'file_size_bytes': 102400,
            'sha256': '1234567890abcdef',
            'uploaded_at': '2026-09-29T10:00:00.000Z',
          },
        ]);

        return http.Response(
          body,
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'student-jwt-token');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      final documents = await service.getPersonalVaultDocuments();

      expect(capturedPath, '/documents');
      expect(capturedAuthHeader, 'Bearer student-jwt-token');
      expect(documents.length, 1);
      expect(documents.first.id, 1);
      expect(documents.first.originalFilename, 'resume.pdf');
    });

    test('returns empty list when backend returns empty array', () async {
      final mockClient = MockClient((request) async {
        return http.Response(
          '[]',
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'valid-token');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      final documents = await service.getPersonalVaultDocuments();

      expect(documents, isEmpty);
    });

    test('throws 401 ApiException when no token is stored', () async {
      final apiClient = ApiClient(baseUrl: 'http://localhost:8080');
      final tokenStorage = InMemoryTokenStorage(token: null);
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getPersonalVaultDocuments(),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401),
        ),
      );
    });

    test('throws 403 ApiException when backend returns forbidden', () async {
      final mockClient = MockClient((request) async {
        return http.Response(
          'forbidden: only students have a personal vault',
          403,
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'faculty-token');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getPersonalVaultDocuments(),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 403),
        ),
      );
    });

    test('throws 500 ApiException on internal server error', () async {
      final mockClient = MockClient((request) async {
        return http.Response('internal server error', 500);
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'student-token');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getPersonalVaultDocuments(),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 500),
        ),
      );
    });

    test('throws 0 ApiException on network failure', () async {
      final mockClient = MockClient((request) async {
        throw http.ClientException('Connection failed');
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'student-token');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.getPersonalVaultDocuments(),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 0),
        ),
      );
    });
  });
}
