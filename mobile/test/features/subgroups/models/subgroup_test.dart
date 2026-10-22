import 'package:docproject/features/subgroups/models/subgroup.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Subgroup model', () {
    test('parses full backend JSON payload correctly', () {
      final json = {
        'id': 201,
        'college_id': 7,
        'group_id': 100,
        'name': 'Midterm Certificates',
        'description': 'Collection for midterm academic certificates',
        'created_by': 42,
        'is_active': true,
        'created_at': '2026-10-01 10:00:00',
        'updated_at': '2026-10-05 14:30:00',
      };

      final subgroup = Subgroup.fromJson(json);

      expect(subgroup.id, 201);
      expect(subgroup.collegeId, 7);
      expect(subgroup.groupId, 100);
      expect(subgroup.name, 'Midterm Certificates');
      expect(
        subgroup.description,
        'Collection for midterm academic certificates',
      );
      expect(subgroup.createdBy, 42);
      expect(subgroup.isActive, isTrue);
      expect(subgroup.createdAt, '2026-10-01 10:00:00');
      expect(subgroup.updatedAt, '2026-10-05 14:30:00');
    });

    test('handles null description and timestamps gracefully', () {
      final json = {
        'id': 202,
        'college_id': 7,
        'group_id': 100,
        'name': 'Internship Reports',
        'description': null,
        'created_by': 42,
        'is_active': false,
        'created_at': null,
        'updated_at': null,
      };

      final subgroup = Subgroup.fromJson(json);

      expect(subgroup.id, 202);
      expect(subgroup.name, 'Internship Reports');
      expect(subgroup.description, '');
      expect(subgroup.isActive, isFalse);
      expect(subgroup.createdAt, isNull);
      expect(subgroup.updatedAt, isNull);
    });

    test('serializes to JSON correctly', () {
      const subgroup = Subgroup(
        id: 203,
        collegeId: 7,
        groupId: 100,
        name: 'Sports Certificates',
        description: 'Sports tournament certificates',
        createdBy: 12,
        isActive: true,
        createdAt: '2026-09-01 09:00:00',
      );

      final json = subgroup.toJson();

      expect(json['id'], 203);
      expect(json['college_id'], 7);
      expect(json['group_id'], 100);
      expect(json['name'], 'Sports Certificates');
      expect(json['description'], 'Sports tournament certificates');
      expect(json['created_by'], 12);
      expect(json['is_active'], isTrue);
      expect(json['created_at'], '2026-09-01 09:00:00');
    });

    test('equality and hashCode match for identical properties', () {
      const sub1 = Subgroup(
        id: 1,
        collegeId: 1,
        groupId: 10,
        name: 'Subgroup 1',
        description: 'Desc',
        createdBy: 1,
      );
      const sub2 = Subgroup(
        id: 1,
        collegeId: 1,
        groupId: 10,
        name: 'Subgroup 1',
        description: 'Desc',
        createdBy: 1,
      );
      const sub3 = Subgroup(
        id: 2,
        collegeId: 1,
        groupId: 10,
        name: 'Subgroup 2',
        description: 'Desc',
        createdBy: 1,
      );

      expect(sub1, equals(sub2));
      expect(sub1.hashCode, equals(sub2.hashCode));
      expect(sub1, isNot(equals(sub3)));
    });
  });
}
