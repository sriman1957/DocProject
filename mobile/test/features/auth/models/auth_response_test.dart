import 'package:docproject/features/auth/models/auth_response.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AuthResponse', () {
    test('parses a successful login response', () {
      final response = AuthResponse.fromJson({
        'access_token': 'test-token',
        'user': {
          'id': 7,
          'college_id': 1,
          'full_name': 'Test Student',
          'email': 'student@example.com',
          'role': 'student',
          'faculty_code': 'FAC001',
        },
      });

      expect(response.accessToken, 'test-token');
      expect(response.user.id, 7);
      expect(response.user.collegeId, 1);
      expect(response.user.fullName, 'Test Student');
      expect(response.user.email, 'student@example.com');
      expect(response.user.role, 'student');
      expect(response.user.facultyCode, 'FAC001');
    });

    test('parses a response without optional faculty code', () {
      final response = AuthResponse.fromJson({
        'access_token': 'test-token',
        'user': {
          'id': 8,
          'college_id': 2,
          'full_name': 'Test Faculty',
          'email': 'faculty@example.com',
          'role': 'faculty',
        },
      });

      expect(response.user.facultyCode, isNull);
    });
  });
}
