import 'package:docproject/features/notifications/models/notification_item.dart';
import 'package:docproject/features/notifications/widgets/notification_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('NotificationTile Widget Tests', () {
    final unreadNotification = NotificationItem(
      id: 55,
      recipientUserId: 1,
      actorUserId: 2,
      groupId: 3,
      type: 'document_submitted',
      message: 'Alex uploaded assignment.pdf',
      isRead: false,
      createdAt: DateTime.parse('2026-10-15T09:00:00Z'),
    );

    final readNotification = NotificationItem(
      id: 56,
      recipientUserId: 1,
      actorUserId: 2,
      groupId: 3,
      type: 'faculty_assigned',
      message: 'Assigned as reviewer for subgroup',
      isRead: true,
      createdAt: DateTime.parse('2026-10-15T10:00:00Z'),
    );

    testWidgets(
      'renders unread notification with distinct styling, dot, and mark-read button',
      (tester) async {
        var markedRead = false;

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: NotificationTile(
                notification: unreadNotification,
                onMarkAsRead: () {
                  markedRead = true;
                },
              ),
            ),
          ),
        );

        expect(find.text('Document Submission'), findsOneWidget);
        expect(find.text('Alex uploaded assignment.pdf'), findsOneWidget);
        expect(find.text('Mark as read'), findsOneWidget);

        await tester.tap(find.text('Mark as read'));
        await tester.pump();

        expect(markedRead, isTrue);
      },
    );

    testWidgets(
      'renders read notification without unread dot or mark-read button',
      (tester) async {
        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: NotificationTile(
                notification: readNotification,
                onMarkAsRead: () {},
              ),
            ),
          ),
        );

        expect(find.text('Faculty Assignment'), findsOneWidget);
        expect(find.text('Assigned as reviewer for subgroup'), findsOneWidget);
        expect(find.text('Mark as read'), findsNothing);
      },
    );
  });
}
