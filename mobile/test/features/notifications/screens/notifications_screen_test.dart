import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/notifications/controllers/notifications_controller.dart';
import 'package:docproject/features/notifications/models/notification_item.dart';
import 'package:docproject/features/notifications/screens/notifications_screen.dart';
import 'package:docproject/features/notifications/services/notifications_service.dart';
import 'package:docproject/features/notifications/widgets/notification_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeNotificationsService extends NotificationsService {
  FakeNotificationsService()
    : super(
        apiClient: ApiClient(baseUrl: 'http://localhost'),
        tokenStorage: TokenStorage(),
      );

  List<NotificationItem> notificationsToReturn = const [];
  int unreadCountToReturn = 0;
  Object? errorToThrow;
  Duration? delay;
  int listCallCount = 0;
  int markReadCallCount = 0;

  @override
  Future<List<NotificationItem>> getNotifications() async {
    listCallCount++;
    if (delay != null) await Future<void>.delayed(delay!);
    if (errorToThrow != null) throw errorToThrow!;
    return notificationsToReturn;
  }

  @override
  Future<int> getUnreadCount() async => unreadCountToReturn;

  @override
  Future<NotificationItem> markAsRead(int notificationId) async {
    markReadCallCount++;
    final match = notificationsToReturn.firstWhere(
      (n) => n.id == notificationId,
    );
    return match.copyWith(isRead: true);
  }
}

void main() {
  final sampleNotification = NotificationItem(
    id: 99,
    recipientUserId: 1,
    actorUserId: 2,
    groupId: 3,
    type: 'access_period_opened',
    message: 'Collection window opened for Project Submissions',
    isRead: false,
    createdAt: DateTime.now(),
  );

  group('NotificationsScreen', () {
    late FakeNotificationsService fakeService;
    late NotificationsController controller;

    setUp(() {
      fakeService = FakeNotificationsService();
      controller = NotificationsController(notificationsService: fakeService);
    });

    Widget createWidget({bool autoLoad = true}) {
      return MaterialApp(
        home: NotificationsScreen(controller: controller, autoLoad: autoLoad),
      );
    }

    testWidgets('shows loading indicator while loading', (tester) async {
      fakeService.delay = const Duration(milliseconds: 300);

      await tester.pumpWidget(createWidget());
      await tester.pump();

      expect(
        find.byKey(const Key('notifications_loading_indicator')),
        findsOneWidget,
      );

      await tester.pump(const Duration(milliseconds: 300));
      await tester.pumpAndSettle();
    });

    testWidgets(
      'renders list of notifications with unread count summary banner',
      (tester) async {
        fakeService.notificationsToReturn = [sampleNotification];
        fakeService.unreadCountToReturn = 1;

        await tester.pumpWidget(createWidget());
        await tester.pumpAndSettle();

        expect(find.text('Notifications'), findsOneWidget);
        expect(find.text('1 unread notification'), findsOneWidget);
        expect(find.byType(NotificationTile), findsOneWidget);
        expect(
          find.text('Collection window opened for Project Submissions'),
          findsOneWidget,
        );
        expect(find.text('Collection Window Opened'), findsOneWidget);
        expect(find.byKey(const Key('mark_read_button_99')), findsOneWidget);
      },
    );

    testWidgets('tapping mark as read marks notification as read', (
      tester,
    ) async {
      fakeService.notificationsToReturn = [sampleNotification];
      fakeService.unreadCountToReturn = 1;

      await tester.pumpWidget(createWidget());
      await tester.pumpAndSettle();

      final markReadBtn = find.byKey(const Key('mark_read_button_99'));
      expect(markReadBtn, findsOneWidget);

      await tester.tap(markReadBtn);
      await tester.pumpAndSettle();

      expect(fakeService.markReadCallCount, 1);
      // Once read, mark as read button disappears
      expect(find.byKey(const Key('mark_read_button_99')), findsNothing);
    });

    testWidgets('renders empty state when no notifications exist', (
      tester,
    ) async {
      fakeService.notificationsToReturn = [];
      fakeService.unreadCountToReturn = 0;

      await tester.pumpWidget(createWidget());
      await tester.pumpAndSettle();

      expect(
        find.byKey(const Key('notifications_empty_state')),
        findsOneWidget,
      );
      expect(find.text('No Notifications Yet'), findsOneWidget);
    });

    testWidgets('renders error state and retry button works', (tester) async {
      fakeService.errorToThrow = const ApiException(
        statusCode: 500,
        message: 'Server down',
      );

      await tester.pumpWidget(createWidget());
      await tester.pumpAndSettle();

      expect(
        find.byKey(const Key('notifications_error_state')),
        findsOneWidget,
      );
      expect(find.text('Server down'), findsOneWidget);

      fakeService.errorToThrow = null;
      fakeService.notificationsToReturn = [sampleNotification];

      await tester.tap(find.text('Try Again'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.byType(NotificationTile), findsOneWidget);
    });
  });
}
