import 'package:flutter/foundation.dart';

import '../../../core/network/api_client.dart';
import '../models/notification_item.dart';
import '../services/notifications_service.dart';

sealed class NotificationsState {
  const NotificationsState();
}

class NotificationsInitial extends NotificationsState {
  const NotificationsInitial();
}

class NotificationsLoading extends NotificationsState {
  const NotificationsLoading();
}

class NotificationsLoaded extends NotificationsState {
  const NotificationsLoaded({
    required this.notifications,
    required this.unreadCount,
  });

  final List<NotificationItem> notifications;
  final int unreadCount;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NotificationsLoaded &&
          runtimeType == other.runtimeType &&
          unreadCount == other.unreadCount &&
          listEquals(notifications, other.notifications);

  @override
  int get hashCode => notifications.hashCode ^ unreadCount.hashCode;
}

class NotificationsEmpty extends NotificationsState {
  const NotificationsEmpty();
}

class NotificationsError extends NotificationsState {
  const NotificationsError({
    required this.message,
    this.canRetry = true,
    this.statusCode,
  });

  final String message;
  final bool canRetry;
  final int? statusCode;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NotificationsError &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry &&
          statusCode == other.statusCode;

  @override
  int get hashCode =>
      message.hashCode ^ canRetry.hashCode ^ statusCode.hashCode;
}

class NotificationsController extends ChangeNotifier {
  NotificationsController({
    required this.notificationsService,
    NotificationsState initialState = const NotificationsInitial(),
  }) : _state = initialState;

  final NotificationsService notificationsService;

  NotificationsState _state;
  NotificationsState get state => _state;

  int _cachedUnreadCount = 0;
  int get unreadCount => _cachedUnreadCount;

  Future<void> loadNotifications({bool force = false}) async {
    if (!force &&
        (_state is NotificationsLoading || _state is NotificationsLoaded)) {
      return;
    }

    _state = const NotificationsLoading();
    notifyListeners();

    try {
      final notifications = await notificationsService.getNotifications();
      final unreadCount = await notificationsService.getUnreadCount();
      _cachedUnreadCount = unreadCount;

      if (notifications.isEmpty) {
        _state = const NotificationsEmpty();
      } else {
        _state = NotificationsLoaded(
          notifications: notifications,
          unreadCount: unreadCount,
        );
      }
    } on ApiException catch (e) {
      _state = NotificationsError(message: e.message, statusCode: e.statusCode);
    } catch (_) {
      _state = const NotificationsError(
        message: 'Network error. Please check your connection and try again.',
      );
    } finally {
      notifyListeners();
    }
  }

  Future<void> refresh() => loadNotifications(force: true);

  Future<void> fetchUnreadCount() async {
    try {
      _cachedUnreadCount = await notificationsService.getUnreadCount();
      notifyListeners();
    } catch (_) {
      // Ignored non-critical background count error
    }
  }

  Future<void> markAsRead(int notificationId) async {
    final currentState = _state;
    if (currentState is! NotificationsLoaded) return;

    try {
      final updated = await notificationsService.markAsRead(notificationId);

      final updatedList = currentState.notifications.map((item) {
        if (item.id == notificationId) {
          return updated;
        }
        return item;
      }).toList();

      final newUnreadCount = (_cachedUnreadCount > 0)
          ? _cachedUnreadCount - 1
          : 0;
      _cachedUnreadCount = newUnreadCount;

      _state = NotificationsLoaded(
        notifications: updatedList,
        unreadCount: newUnreadCount,
      );
      notifyListeners();
    } catch (_) {
      // Re-raise or notify so caller can inform user if needed
      rethrow;
    }
  }
}
