import 'package:docproject/features/access_periods/models/access_period.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AccessPeriod Model', () {
    test('parses valid access period JSON and preserves UTC instants', () {
      final json = {
        'id': 15,
        'subgroup_id': 20,
        'group_id': 10,
        'college_id': 5,
        'starts_at': '2026-10-01T09:00:00.000Z',
        'ends_at': '2026-10-15T18:00:00.000Z',
        'created_by': 42,
        'created_at': '2026-09-30T12:00:00.000Z',
      };

      final period = AccessPeriod.fromJson(json);

      expect(period.id, 15);
      expect(period.subgroupId, 20);
      expect(period.groupId, 10);
      expect(period.collegeId, 5);
      expect(period.createdBy, 42);
      expect(
        period.startsAt.toUtc(),
        DateTime.parse('2026-10-01T09:00:00.000Z'),
      );
      expect(period.endsAt.toUtc(), DateTime.parse('2026-10-15T18:00:00.000Z'));
      expect(
        period.createdAt?.toUtc(),
        DateTime.parse('2026-09-30T12:00:00.000Z'),
      );
    });

    test('toJson produces expected structure and UTC ISO strings', () {
      final period = AccessPeriod(
        id: 15,
        subgroupId: 20,
        groupId: 10,
        collegeId: 5,
        startsAt: DateTime.parse('2026-10-01T09:00:00.000Z'),
        endsAt: DateTime.parse('2026-10-15T18:00:00.000Z'),
        createdBy: 42,
        createdAt: DateTime.parse('2026-09-30T12:00:00.000Z'),
      );

      final json = period.toJson();

      expect(json['id'], 15);
      expect(json['subgroup_id'], 20);
      expect(json['starts_at'], '2026-10-01T09:00:00.000Z');
      expect(json['ends_at'], '2026-10-15T18:00:00.000Z');
    });

    group('Half-open interval [starts_at, ends_at) status tests', () {
      final startsAt = DateTime.parse('2026-10-10T10:00:00.000Z');
      final endsAt = DateTime.parse('2026-10-20T10:00:00.000Z');

      final period = AccessPeriod(
        id: 1,
        subgroupId: 2,
        groupId: 3,
        collegeId: 4,
        startsAt: startsAt,
        endsAt: endsAt,
        createdBy: 5,
      );

      test('time strictly before starts_at is Upcoming', () {
        final before = DateTime.parse('2026-10-10T09:59:59.000Z');
        expect(period.getStatus(before), AccessPeriodStatus.upcoming);
        expect(period.isUpcoming(before), isTrue);
        expect(period.isActiveNow(before), isFalse);
        expect(period.isExpired(before), isFalse);
      });

      test('boundary: exactly at starts_at is Active', () {
        expect(period.getStatus(startsAt), AccessPeriodStatus.active);
        expect(period.isActiveNow(startsAt), isTrue);
        expect(period.isUpcoming(startsAt), isFalse);
        expect(period.isExpired(startsAt), isFalse);
      });

      test('time between starts_at and ends_at is Active', () {
        final middle = DateTime.parse('2026-10-15T12:00:00.000Z');
        expect(period.getStatus(middle), AccessPeriodStatus.active);
        expect(period.isActiveNow(middle), isTrue);
      });

      test('boundary: exactly at ends_at is Expired (half-open exclusion)', () {
        expect(period.getStatus(endsAt), AccessPeriodStatus.expired);
        expect(period.isActiveNow(endsAt), isFalse);
        expect(period.isExpired(endsAt), isTrue);
      });

      test('time strictly after ends_at is Expired', () {
        final after = DateTime.parse('2026-10-20T10:00:01.000Z');
        expect(period.getStatus(after), AccessPeriodStatus.expired);
        expect(period.isExpired(after), isTrue);
      });
    });
  });
}
