import 'package:docproject/features/groups/models/group.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Group model', () {
    test('parses full backend JSON payload correctly', () {
      final json = {
        'id': 101,
        'college_id': 7,
        'branch_id': 3,
        'name': 'CSE 2024-2028',
        'description': 'Computer Science Department Batch',
        'created_by': 42,
        'is_active': true,
        'created_at': '2026-10-01 10:00:00',
        'updated_at': '2026-10-05 14:30:00',
      };

      final group = Group.fromJson(json);

      expect(group.id, 101);
      expect(group.collegeId, 7);
      expect(group.branchId, 3);
      expect(group.name, 'CSE 2024-2028');
      expect(group.description, 'Computer Science Department Batch');
      expect(group.createdBy, 42);
      expect(group.isActive, isTrue);
      expect(group.createdAt, '2026-10-01 10:00:00');
      expect(group.updatedAt, '2026-10-05 14:30:00');
    });

    test('handles null description and timestamps gracefully', () {
      final json = {
        'id': 102,
        'college_id': 7,
        'branch_id': 3,
        'name': 'ECE Batch B',
        'description': null,
        'created_by': 42,
        'is_active': false,
        'created_at': null,
        'updated_at': null,
      };

      final group = Group.fromJson(json);

      expect(group.id, 102);
      expect(group.name, 'ECE Batch B');
      expect(group.description, '');
      expect(group.isActive, isFalse);
      expect(group.createdAt, isNull);
      expect(group.updatedAt, isNull);
    });

    test('serializes to JSON correctly', () {
      const group = Group(
        id: 103,
        collegeId: 7,
        branchId: 3,
        name: 'Civil Engineering',
        description: 'Batch of 2025',
        createdBy: 12,
        isActive: true,
        createdAt: '2026-09-01 09:00:00',
      );

      final json = group.toJson();

      expect(json['id'], 103);
      expect(json['college_id'], 7);
      expect(json['branch_id'], 3);
      expect(json['name'], 'Civil Engineering');
      expect(json['description'], 'Batch of 2025');
      expect(json['created_by'], 12);
      expect(json['is_active'], isTrue);
      expect(json['created_at'], '2026-09-01 09:00:00');
    });

    test('equality and hashCode match for identical properties', () {
      const group1 = Group(
        id: 1,
        collegeId: 1,
        branchId: 1,
        name: 'Group 1',
        description: 'Desc',
        createdBy: 1,
      );
      const group2 = Group(
        id: 1,
        collegeId: 1,
        branchId: 1,
        name: 'Group 1',
        description: 'Desc',
        createdBy: 1,
      );
      const group3 = Group(
        id: 2,
        collegeId: 1,
        branchId: 1,
        name: 'Group 2',
        description: 'Desc',
        createdBy: 1,
      );

      expect(group1, equals(group2));
      expect(group1.hashCode, equals(group2.hashCode));
      expect(group1, isNot(equals(group3)));
    });
  });
}
