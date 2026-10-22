import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../models/subgroup.dart';

class SubgroupsService {
  SubgroupsService({required this.apiClient, required this.tokenStorage});

  final ApiClient apiClient;
  final TokenStorage tokenStorage;

  Future<List<Subgroup>> getSubgroups(int groupId) async {
    if (groupId <= 0) {
      throw const ApiException(
        statusCode: 400,
        message: 'Invalid group identifier.',
      );
    }

    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to view subgroups.',
      );
    }

    final response = await apiClient.get(
      '/groups/$groupId/subgroups',
      accessToken: token,
    );

    if (response is List) {
      return response
          .map(
            (item) => Subgroup.fromJson(Map<String, dynamic>.from(item as Map)),
          )
          .toList();
    }

    return const [];
  }
}
