import 'package:docproject/features/notifications/models/notification_item.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('NotificationItem Model', () {
    test('parses full backend notification JSON schema correctly', () {
      final json = {
        'id': 101,
        'recipient_user_id': 42,
        'actor_user_id': 5,
        'group_id': 10,
        'subgroup_id': 20,
        'document_id': 300,
        'type': 'document_submitted',
        'message': 'John Doe submitted project_proposal.pdf',
        'is_read': false,
        'created_at': '2026-10-15T09:30:00.000Z',
      };

      final item = NotificationItem.fromJson(json);

      expect(item.id, 101);
      expect(item.recipientUserId, 42);
      expect(item.actorUserId, 5);
      expect(item.groupId, 10);
      expect(item.subgroupId, 20);
      expect(item.documentId, 300);
      expect(item.type, 'document_submitted');
      expect(item.message, 'John Doe submitted project_proposal.pdf');
      expect(item.isRead, isFalse);
      expect(item.icon, Icons.upload_file_rounded);
      expect(item.typeLabel, 'Document Submission');
      expect(item.formattedCreatedAt, isNotEmpty);
    });

    test('handles nullable subgroupId and documentId safely', () {
      final json = {
        'id': 102,
        'recipient_user_id': 42,
        'actor_user_id': 1,
        'group_id': 10,
        'subgroup_id': null,
        'document_id': null,
        'type': 'access_period_opened',
        'message': 'Collection window opened for your department',
        'is_read': true,
        'created_at': '2026-10-16T12:00:00.000Z',
      };

      final item = NotificationItem.fromJson(json);

      expect(item.id, 102);
      expect(item.subgroupId, isNull);
      expect(item.documentId, isNull);
      expect(item.isRead, isTrue);
      expect(item.icon, Icons.event_available_rounded);
      expect(item.typeLabel, 'Collection Window Opened');
    });

    test('copyWith updates fields while preserving others', () {
      final original = NotificationItem(
        id: 1,
        recipientUserId: 2,
        actorUserId: 3,
        groupId: 4,
        type: 'faculty_assigned',
        message: 'You have been assigned to review submissions',
        isRead: false,
        createdAt: DateTime.parse('2026-10-10T00:00:00Z'),
      );

      final updated = original.copyWith(isRead: true);

      expect(updated.id, original.id);
      expect(updated.isRead, isTrue);
      expect(original.isRead, isFalse);
    });

    test('toJson serializes according to backend expectations', () {
      final item = NotificationItem(
        id: 5,
        recipientUserId: 10,
        actorUserId: 12,
        groupId: 8,
        subgroupId: 15,
        documentId: 99,
        type: 'general',
        message: 'System alert',
        isRead: false,
        createdAt: DateTime.parse('2026-10-01T00:00:00.000Z'),
      );

      final json = item.toJson();

      expect(json['id'], 5);
      expect(json['recipient_user_id'], 10);
      expect(json['created_at'], '2026-10-01T00:00:00.000Z');
    });
  });
}
