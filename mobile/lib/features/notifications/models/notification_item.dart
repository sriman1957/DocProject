import 'package:flutter/material.dart';

class NotificationItem {
  const NotificationItem({
    required this.id,
    required this.recipientUserId,
    required this.actorUserId,
    required this.groupId,
    this.subgroupId,
    this.documentId,
    required this.type,
    required this.message,
    required this.isRead,
    required this.createdAt,
  });

  final int id;
  final int recipientUserId;
  final int actorUserId;
  final int groupId;
  final int? subgroupId;
  final int? documentId;
  final String type;
  final String message;
  final bool isRead;
  final DateTime createdAt;

  factory NotificationItem.fromJson(Map<String, dynamic> json) {
    DateTime parseTime(dynamic raw) {
      if (raw == null) return DateTime.now();
      if (raw is DateTime) return raw;
      return DateTime.tryParse(raw.toString())?.toLocal() ?? DateTime.now();
    }

    return NotificationItem(
      id: (json['id'] as num?)?.toInt() ?? 0,
      recipientUserId: (json['recipient_user_id'] as num?)?.toInt() ?? 0,
      actorUserId: (json['actor_user_id'] as num?)?.toInt() ?? 0,
      groupId: (json['group_id'] as num?)?.toInt() ?? 0,
      subgroupId: (json['subgroup_id'] as num?)?.toInt(),
      documentId: (json['document_id'] as num?)?.toInt(),
      type: json['type'] as String? ?? 'general',
      message: json['message'] as String? ?? '',
      isRead: json['is_read'] as bool? ?? false,
      createdAt: parseTime(json['created_at']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'recipient_user_id': recipientUserId,
      'actor_user_id': actorUserId,
      'group_id': groupId,
      'subgroup_id': subgroupId,
      'document_id': documentId,
      'type': type,
      'message': message,
      'is_read': isRead,
      'created_at': createdAt.toUtc().toIso8601String(),
    };
  }

  NotificationItem copyWith({bool? isRead, String? message}) {
    return NotificationItem(
      id: id,
      recipientUserId: recipientUserId,
      actorUserId: actorUserId,
      groupId: groupId,
      subgroupId: subgroupId,
      documentId: documentId,
      type: type,
      message: message ?? this.message,
      isRead: isRead ?? this.isRead,
      createdAt: createdAt,
    );
  }

  String get formattedCreatedAt {
    final local = createdAt.toLocal();
    final year = local.year.toString().padLeft(4, '0');
    final month = local.month.toString().padLeft(2, '0');
    final day = local.day.toString().padLeft(2, '0');
    final hour = local.hour.toString().padLeft(2, '0');
    final minute = local.minute.toString().padLeft(2, '0');
    return '$year-$month-$day $hour:$minute';
  }

  IconData get icon {
    switch (type) {
      case 'document_submitted':
        return Icons.upload_file_rounded;
      case 'access_period_opened':
        return Icons.event_available_rounded;
      case 'access_period_closed':
        return Icons.event_busy_rounded;
      case 'faculty_assigned':
        return Icons.person_add_rounded;
      default:
        return Icons.notifications_rounded;
    }
  }

  Color get iconColor {
    switch (type) {
      case 'document_submitted':
        return Colors.teal;
      case 'access_period_opened':
        return Colors.green;
      case 'access_period_closed':
        return Colors.orange;
      case 'faculty_assigned':
        return Colors.blue;
      default:
        return Colors.indigo;
    }
  }

  String get typeLabel {
    switch (type) {
      case 'document_submitted':
        return 'Document Submission';
      case 'access_period_opened':
        return 'Collection Window Opened';
      case 'access_period_closed':
        return 'Collection Window Closed';
      case 'faculty_assigned':
        return 'Faculty Assignment';
      default:
        return 'Notification';
    }
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NotificationItem &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          recipientUserId == other.recipientUserId &&
          actorUserId == other.actorUserId &&
          groupId == other.groupId &&
          subgroupId == other.subgroupId &&
          documentId == other.documentId &&
          type == other.type &&
          message == other.message &&
          isRead == other.isRead &&
          createdAt == other.createdAt;

  @override
  int get hashCode =>
      id.hashCode ^
      recipientUserId.hashCode ^
      actorUserId.hashCode ^
      groupId.hashCode ^
      subgroupId.hashCode ^
      documentId.hashCode ^
      type.hashCode ^
      message.hashCode ^
      isRead.hashCode ^
      createdAt.hashCode;
}
