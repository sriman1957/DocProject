import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../models/access_period.dart';

class AccessPeriodsService {
  const AccessPeriodsService({
    required this.apiClient,
    required this.tokenStorage,
  });

  final ApiClient apiClient;
  final TokenStorage tokenStorage;

  /// Lists collection access periods for an authorized subgroup.
  /// (Faculty assigned to subgroup or College Admin only).
  Future<List<AccessPeriod>> getSubgroupAccessPeriods({
    required int groupId,
    required int subgroupId,
  }) async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to view collection access periods.',
      );
    }

    final response = await apiClient.get(
      '/groups/$groupId/subgroups/$subgroupId/access-periods',
      accessToken: token,
    );

    if (response is List) {
      return response
          .map(
            (item) =>
                AccessPeriod.fromJson(Map<String, dynamic>.from(item as Map)),
          )
          .toList();
    }

    return const [];
  }

  /// Creates a new access period for a subgroup.
  /// (Faculty assigned to subgroup or College Admin only).
  Future<AccessPeriod> createAccessPeriod({
    required int groupId,
    required int subgroupId,
    required DateTime startsAt,
    required DateTime endsAt,
  }) async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to create an access period.',
      );
    }

    if (!endsAt.isAfter(startsAt)) {
      throw const ApiException(
        statusCode: 400,
        message: 'Closing time must be after opening time.',
      );
    }

    final response = await apiClient.post(
      '/groups/$groupId/subgroups/$subgroupId/access-periods',
      body: {
        'starts_at': startsAt.toUtc().toIso8601String(),
        'ends_at': endsAt.toUtc().toIso8601String(),
      },
      accessToken: token,
    );

    if (response is Map) {
      return AccessPeriod.fromJson(Map<String, dynamic>.from(response));
    }

    throw const ApiException(
      statusCode: 500,
      message: 'Unexpected server response while creating access period.',
    );
  }
}
