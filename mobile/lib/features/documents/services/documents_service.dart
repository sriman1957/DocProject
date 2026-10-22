import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../models/vault_document.dart';
import 'file_picker_service.dart';

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

  /// Uploads a file to the authenticated student's Personal Vault.
  Future<VaultDocument> uploadPersonalVaultDocument(SelectedFile file) async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to upload documents.',
      );
    }

    final response = await apiClient.postMultipart(
      '/documents',
      fieldName: 'file',
      filename: file.name,
      bytes: file.bytes,
      accessToken: token,
    );

    if (response is Map) {
      return VaultDocument.fromJson(Map<String, dynamic>.from(response));
    }

    throw const ApiException(
      statusCode: 500,
      message: 'Unexpected server response while uploading document.',
    );
  }

  /// Submits an existing Personal Vault document into an authorized subgroup.
  Future<VaultDocument> submitDocumentToSubgroup({
    required int documentId,
    required int groupId,
    required int subgroupId,
  }) async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to submit documents.',
      );
    }

    final response = await apiClient.post(
      '/documents/$documentId/copy',
      body: {'group_id': groupId, 'subgroup_id': subgroupId},
      accessToken: token,
    );

    if (response is Map) {
      return VaultDocument.fromJson(Map<String, dynamic>.from(response));
    }

    throw const ApiException(
      statusCode: 500,
      message: 'Unexpected server response while submitting document.',
    );
  }
}
