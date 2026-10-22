import 'dart:convert';

import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/documents/services/documents_service.dart';
import 'package:docproject/features/documents/services/file_picker_service.dart';
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

    test('uploadPersonalVaultDocument calls POST /documents with multipart file and returns VaultDocument', () async {
      final mockClient = MockClient.streaming((request, bodyStream) async {
        expect(request, isA<http.MultipartRequest>());
        final multipartRequest = request as http.MultipartRequest;
        expect(multipartRequest.method, 'POST');
        expect(multipartRequest.url.path, '/documents');
        expect(multipartRequest.headers['authorization'], 'Bearer valid-jwt');
        expect(multipartRequest.files.length, 1);
        expect(multipartRequest.files.first.field, 'file');
        expect(multipartRequest.files.first.filename, 'grade_sheet.pdf');

        final responseBody = jsonEncode({
          'id': 101,
          'original_filename': 'grade_sheet.pdf',
          'mime_type': 'application/pdf',
          'file_size_bytes': 2048,
          'sha256': 'abcdef1234567890',
          'uploaded_at': '2026-10-22T12:00:00Z',
        });

        return http.StreamedResponse(
          Stream.value(utf8.encode(responseBody)),
          201,
          headers: {'content-type': 'application/json'},
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'valid-jwt');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      const file = SelectedFile(
        name: 'grade_sheet.pdf',
        size: 2048,
        bytes: [37, 80, 68, 70],
        extension: 'pdf',
      );

      final doc = await service.uploadPersonalVaultDocument(file);

      expect(doc.id, 101);
      expect(doc.originalFilename, 'grade_sheet.pdf');
      expect(doc.fileSizeBytes, 2048);
    });

    test(
      'uploadPersonalVaultDocument throws 401 when token is missing',
      () async {
        final apiClient = ApiClient(baseUrl: 'http://localhost:8080');
        final tokenStorage = InMemoryTokenStorage(token: null);
        final service = DocumentsService(
          apiClient: apiClient,
          tokenStorage: tokenStorage,
        );

        const file = SelectedFile(
          name: 'test.pdf',
          size: 100,
          bytes: [1, 2, 3],
          extension: 'pdf',
        );

        expect(
          () => service.uploadPersonalVaultDocument(file),
          throwsA(
            isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401),
          ),
        );
      },
    );

    test('uploadPersonalVaultDocument throws 413 when backend rejects oversized file', () async {
      final mockClient = MockClient.streaming((request, bodyStream) async {
        return http.StreamedResponse(
          Stream.value(
            utf8.encode(jsonEncode({'error': 'file exceeds 20MB limit'})),
          ),
          413,
          headers: {'content-type': 'application/json'},
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'valid-jwt');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      const file = SelectedFile(
        name: 'huge.pdf',
        size: 25000000,
        bytes: [1, 2, 3],
        extension: 'pdf',
      );

      expect(
        () => service.uploadPersonalVaultDocument(file),
        throwsA(
          isA<ApiException>()
              .having((e) => e.statusCode, 'statusCode', 413)
              .having((e) => e.message, 'message', 'file exceeds 20MB limit'),
        ),
      );
    });

    test('submitDocumentToSubgroup calls POST /documents/{id}/copy with group_id and subgroup_id', () async {
      String? capturedPath;
      String? capturedMethod;
      String? capturedAuth;
      Map<String, dynamic>? capturedBody;

      final mockClient = MockClient((request) async {
        capturedPath = request.url.path;
        capturedMethod = request.method;
        capturedAuth = request.headers['Authorization'];
        capturedBody = jsonDecode(request.body) as Map<String, dynamic>;

        final responseBody = jsonEncode({
          'id': 202,
          'original_filename': 'project_report.pdf',
          'mime_type': 'application/pdf',
          'file_size_bytes': 4096,
          'sha256': 'fedcba0987654321',
          'group_id': 42,
          'subgroup_id': 7,
          'uploaded_at': '2026-10-22T13:00:00Z',
        });

        return http.Response(
          responseBody,
          201,
          headers: {'content-type': 'application/json'},
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'student-jwt');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      final result = await service.submitDocumentToSubgroup(
        documentId: 101,
        groupId: 42,
        subgroupId: 7,
      );

      expect(capturedPath, '/documents/101/copy');
      expect(capturedMethod, 'POST');
      expect(capturedAuth, 'Bearer student-jwt');
      expect(capturedBody, {'group_id': 42, 'subgroup_id': 7});
      expect(result.id, 202);
      expect(result.originalFilename, 'project_report.pdf');
      expect(result.fileSizeBytes, 4096);
    });

    test('submitDocumentToSubgroup throws 403 when access period is closed or unauthorized', () async {
      final mockClient = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'error': 'submission window is closed for this subgroup',
          }),
          403,
          headers: {'content-type': 'application/json'},
        );
      });

      final apiClient = ApiClient(
        baseUrl: 'http://localhost:8080',
        httpClient: mockClient,
      );
      final tokenStorage = InMemoryTokenStorage(token: 'student-jwt');
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.submitDocumentToSubgroup(
          documentId: 101,
          groupId: 42,
          subgroupId: 7,
        ),
        throwsA(
          isA<ApiException>()
              .having((e) => e.statusCode, 'statusCode', 403)
              .having(
                (e) => e.message,
                'message',
                'submission window is closed for this subgroup',
              ),
        ),
      );
    });

    test('submitDocumentToSubgroup throws 401 when token is missing', () async {
      final apiClient = ApiClient(baseUrl: 'http://localhost:8080');
      final tokenStorage = InMemoryTokenStorage(token: null);
      final service = DocumentsService(
        apiClient: apiClient,
        tokenStorage: tokenStorage,
      );

      expect(
        () => service.submitDocumentToSubgroup(
          documentId: 101,
          groupId: 42,
          subgroupId: 7,
        ),
        throwsA(
          isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401),
        ),
      );
    });
  });
}
