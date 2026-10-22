import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../models/notification_item.dart';

class NotificationsService {
  const NotificationsService({
    required this.apiClient,
    required this.tokenStorage,
  });

  final ApiClient apiClient;
  final TokenStorage tokenStorage;

  /// Retrieves notifications belonging to the authenticated user.
  Future<List<NotificationItem>> getNotifications() async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to view notifications.',
      );
    }

    final response = await apiClient.get('/notifications', accessToken: token);

    if (response is List) {
      return response
          .map(
            (item) => NotificationItem.fromJson(
              Map<String, dynamic>.from(item as Map),
            ),
          )
          .toList();
    }

    return const [];
  }

  /// Marks a specific notification as read.
  Future<NotificationItem> markAsRead(int notificationId) async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      throw const ApiException(
        statusCode: 401,
        message: 'Authentication required to update notification.',
      );
    }

    final response = await apiClient.patch(
      '/notifications/$notificationId/read',
      accessToken: token,
    );

    if (response is Map) {
      return NotificationItem.fromJson(Map<String, dynamic>.from(response));
    }

    throw const ApiException(
      statusCode: 500,
      message: 'Unexpected server response while updating notification.',
    );
  }

  /// Returns the unread notification count for the authenticated user.
  Future<int> getUnreadCount() async {
    final token = await tokenStorage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      return 0;
    }

    final response = await apiClient.get(
      '/notifications/unread-count',
      accessToken: token,
    );

    if (response is Map && response.containsKey('unread_count')) {
      return (response['unread_count'] as num?)?.toInt() ?? 0;
    }

    return 0;
  }
}
