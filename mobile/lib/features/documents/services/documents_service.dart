import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../models/vault_document.dart';

class DocumentsService {
  DocumentsService({required this.apiClient, required this.tokenStorage});

  final ApiClient apiClient;
  final TokenStorage tokenStorage;

  /// Fetches the authenticated student's personal vault documents.
  Future<List<VaultDocument>> getPersonalVaultDocuments() async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to access personal vault.',
      );
    }

    final response = await apiClient.get('/documents', accessToken: token);

    if (response is List) {
      return response
          .map(
            (item) =>
                VaultDocument.fromJson(Map<String, dynamic>.from(item as Map)),
          )
          .toList();
    }

    return const [];
  }
}
