import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../models/group.dart';

class GroupsService {
  GroupsService({required this.apiClient, required this.tokenStorage});

  final ApiClient apiClient;
  final TokenStorage tokenStorage;

  Future<List<Group>> getGroups() async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to view groups.',
      );
    }

    final response = await apiClient.get('/groups', accessToken: token);

    if (response is List) {
      return response
          .map((item) => Group.fromJson(Map<String, dynamic>.from(item as Map)))
          .toList();
    }

    return const [];
  }
}
