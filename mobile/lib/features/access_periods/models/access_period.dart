import 'package:flutter/material.dart';

enum AccessPeriodStatus {
  active,
  upcoming,
  expired;

  String get label {
    switch (this) {
      case AccessPeriodStatus.active:
        return 'Active (Open)';
      case AccessPeriodStatus.upcoming:
        return 'Upcoming';
      case AccessPeriodStatus.expired:
        return 'Expired (Closed)';
    }
  }

  Color get color {
    switch (this) {
      case AccessPeriodStatus.active:
        return Colors.green;
      case AccessPeriodStatus.upcoming:
        return Colors.blue;
      case AccessPeriodStatus.expired:
        return Colors.grey;
    }
  }

  IconData get icon {
    switch (this) {
      case AccessPeriodStatus.active:
        return Icons.lock_open_rounded;
      case AccessPeriodStatus.upcoming:
        return Icons.schedule_rounded;
      case AccessPeriodStatus.expired:
        return Icons.lock_clock_rounded;
    }
  }
}

class AccessPeriod {
  const AccessPeriod({
    required this.id,
    required this.subgroupId,
    required this.groupId,
    required this.collegeId,
    required this.startsAt,
    required this.endsAt,
    required this.createdBy,
    this.createdAt,
  });

  final int id;
  final int subgroupId;
  final int groupId;
  final int collegeId;
  final DateTime startsAt;
  final DateTime endsAt;
  final int createdBy;
  final DateTime? createdAt;

  factory AccessPeriod.fromJson(Map<String, dynamic> json) {
    DateTime parseTime(dynamic raw) {
      if (raw == null) return DateTime.now();
      if (raw is DateTime) return raw;
      return DateTime.tryParse(raw.toString())?.toLocal() ?? DateTime.now();
    }

    return AccessPeriod(
      id: (json['id'] as num?)?.toInt() ?? 0,
      subgroupId: (json['subgroup_id'] as num?)?.toInt() ?? 0,
      groupId: (json['group_id'] as num?)?.toInt() ?? 0,
      collegeId: (json['college_id'] as num?)?.toInt() ?? 0,
      startsAt: parseTime(json['starts_at']),
      endsAt: parseTime(json['ends_at']),
      createdBy: (json['created_by'] as num?)?.toInt() ?? 0,
      createdAt: json['created_at'] != null
          ? parseTime(json['created_at'])
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'subgroup_id': subgroupId,
      'group_id': groupId,
      'college_id': collegeId,
      'starts_at': startsAt.toUtc().toIso8601String(),
      'ends_at': endsAt.toUtc().toIso8601String(),
      'created_by': createdBy,
      if (createdAt != null) 'created_at': createdAt!.toUtc().toIso8601String(),
    };
  }

  /// Evaluates status based on half-open interval: [starts_at, ends_at).
  /// starts_at <= now < ends_at is ACTIVE.
  AccessPeriodStatus getStatus([DateTime? now]) {
    final current = now ?? DateTime.now();
    if (current.isBefore(startsAt)) {
      return AccessPeriodStatus.upcoming;
    } else if (current.isBefore(endsAt)) {
      return AccessPeriodStatus.active;
    } else {
      return AccessPeriodStatus.expired;
    }
  }

  bool isActiveNow([DateTime? now]) =>
      getStatus(now) == AccessPeriodStatus.active;

  bool isUpcoming([DateTime? now]) =>
      getStatus(now) == AccessPeriodStatus.upcoming;

  bool isExpired([DateTime? now]) =>
      getStatus(now) == AccessPeriodStatus.expired;

  String formattedDateTime(DateTime dt) {
    final local = dt.toLocal();
    final year = local.year.toString().padLeft(4, '0');
    final month = local.month.toString().padLeft(2, '0');
    final day = local.day.toString().padLeft(2, '0');
    final hour = local.hour.toString().padLeft(2, '0');
    final minute = local.minute.toString().padLeft(2, '0');
    return '$year-$month-$day $hour:$minute';
  }

  String get formattedStartsAt => formattedDateTime(startsAt);
  String get formattedEndsAt => formattedDateTime(endsAt);

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AccessPeriod &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          subgroupId == other.subgroupId &&
          groupId == other.groupId &&
          collegeId == other.collegeId &&
          startsAt == other.startsAt &&
          endsAt == other.endsAt &&
          createdBy == other.createdBy;

  @override
  int get hashCode =>
      id.hashCode ^
      subgroupId.hashCode ^
      groupId.hashCode ^
      collegeId.hashCode ^
      startsAt.hashCode ^
      endsAt.hashCode ^
      createdBy.hashCode;
}
