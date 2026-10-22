import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/notifications/controllers/notifications_controller.dart';
import 'package:docproject/features/notifications/models/notification_item.dart';
import 'package:docproject/features/notifications/services/notifications_service.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeNotificationsService extends NotificationsService {
  FakeNotificationsService()
    : super(
        apiClient: ApiClient(baseUrl: 'http://localhost'),
        tokenStorage: TokenStorage(),
      );

  List<NotificationItem> notificationsToReturn = const [];
  int unreadCountToReturn = 0;
  NotificationItem? markedReadToReturn;
  Object? errorToThrow;
  int listCallCount = 0;
  int markReadCallCount = 0;
  int unreadCountCallCount = 0;

  @override
  Future<List<NotificationItem>> getNotifications() async {
    listCallCount++;
    if (errorToThrow != null) throw errorToThrow!;
    return notificationsToReturn;
  }

  @override
  Future<int> getUnreadCount() async {
    unreadCountCallCount++;
    if (errorToThrow != null) throw errorToThrow!;
    return unreadCountToReturn;
  }

  @override
  Future<NotificationItem> markAsRead(int notificationId) async {
    markReadCallCount++;
    if (errorToThrow != null) throw errorToThrow!;
    return markedReadToReturn ??
        NotificationItem(
          id: notificationId,
          recipientUserId: 1,
          actorUserId: 2,
          groupId: 3,
          type: 'general',
          message: 'Updated',
          isRead: true,
          createdAt: DateTime.now(),
        );
  }
}

void main() {
  group('NotificationsController', () {
    late FakeNotificationsService fakeService;
    late NotificationsController controller;

    final sampleNotification = NotificationItem(
      id: 10,
      recipientUserId: 1,
      actorUserId: 2,
      groupId: 3,
      type: 'document_submitted',
      message: 'New document submitted',
      isRead: false,
      createdAt: DateTime.now(),
    );

    setUp(() {
      fakeService = FakeNotificationsService();
      controller = NotificationsController(notificationsService: fakeService);
    });

    test('initial state is NotificationsInitial', () {
      expect(controller.state, isA<NotificationsInitial>());
      expect(controller.unreadCount, 0);
    });

    test('loadNotifications transitions to Loading then Loaded', () async {
      fakeService.notificationsToReturn = [sampleNotification];
      fakeService.unreadCountToReturn = 1;

      await controller.loadNotifications();

      expect(controller.state, isA<NotificationsLoaded>());
      final loaded = controller.state as NotificationsLoaded;
      expect(loaded.notifications.length, 1);
      expect(loaded.unreadCount, 1);
      expect(controller.unreadCount, 1);
    });

    test(
      'loadNotifications transitions to Empty when no notifications',
      () async {
        fakeService.notificationsToReturn = [];
        fakeService.unreadCountToReturn = 0;

        await controller.loadNotifications();

        expect(controller.state, isA<NotificationsEmpty>());
        expect(controller.unreadCount, 0);
      },
    );

    test('loadNotifications transitions to Error on failure', () async {
      fakeService.errorToThrow = const ApiException(
        statusCode: 500,
        message: 'Server error',
      );

      await controller.loadNotifications();

      expect(controller.state, isA<NotificationsError>());
    });

    test(
      'markAsRead updates item in list and decrements unread count',
      () async {
        fakeService.notificationsToReturn = [sampleNotification];
        fakeService.unreadCountToReturn = 1;

        await controller.loadNotifications();
        expect(controller.unreadCount, 1);

        await controller.markAsRead(10);

        expect(fakeService.markReadCallCount, 1);
        expect(controller.unreadCount, 0);
        final loaded = controller.state as NotificationsLoaded;
        expect(loaded.notifications.first.isRead, isTrue);
      },
    );

    test(
      'fetchUnreadCount updates cached unread count without state reset',
      () async {
        fakeService.unreadCountToReturn = 5;

        await controller.fetchUnreadCount();

        expect(controller.unreadCount, 5);
      },
    );
  });
}
