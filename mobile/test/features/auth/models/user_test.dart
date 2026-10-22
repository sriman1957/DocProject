import 'package:docproject/features/auth/models/user.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('User and UserRole', () {
    test('UserRole.fromString correctly parses roles', () {
      expect(UserRole.fromString('student'), UserRole.student);
      expect(UserRole.fromString('STUDENT'), UserRole.student);
      expect(UserRole.fromString('faculty'), UserRole.faculty);
      expect(UserRole.fromString('FACULTY '), UserRole.faculty);
      expect(UserRole.fromString('college_admin'), UserRole.collegeAdmin);
      expect(UserRole.fromString('unknown_role'), UserRole.unknown);
      expect(UserRole.fromString(''), UserRole.unknown);
    });

    test('UserRole boolean helpers return expected values', () {
      expect(UserRole.student.isStudent, isTrue);
      expect(UserRole.student.isFaculty, isFalse);
      expect(UserRole.student.isSupported, isTrue);

      expect(UserRole.faculty.isFaculty, isTrue);
      expect(UserRole.faculty.isStudent, isFalse);
      expect(UserRole.faculty.isSupported, isTrue);

      expect(UserRole.collegeAdmin.isCollegeAdmin, isTrue);
      expect(UserRole.collegeAdmin.isSupported, isFalse);

      expect(UserRole.unknown.isSupported, isFalse);
    });

    test('User model parses role and exposes userRole', () {
      const student = User(
        id: 1,
        collegeId: 10,
        fullName: 'Student Name',
        email: 'student@example.com',
        role: 'student',
      );
      expect(student.userRole, UserRole.student);
      expect(student.userRole.isSupported, isTrue);

      const faculty = User(
        id: 2,
        collegeId: 10,
        fullName: 'Faculty Name',
        email: 'faculty@example.com',
        role: 'faculty',
        facultyCode: 'T014',
      );
      expect(faculty.userRole, UserRole.faculty);
      expect(faculty.userRole.isSupported, isTrue);
      expect(faculty.facultyCode, 'T014');

      const admin = User(
        id: 3,
        collegeId: 10,
        fullName: 'Admin Name',
        email: 'admin@example.com',
        role: 'college_admin',
      );
      expect(admin.userRole, UserRole.collegeAdmin);
      expect(admin.userRole.isSupported, isFalse);
    });
  });
}
