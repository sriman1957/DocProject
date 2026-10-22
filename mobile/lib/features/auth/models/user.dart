enum UserRole {
  student,
  faculty,
  collegeAdmin,
  unknown;

  static UserRole fromString(String role) {
    return switch (role.toLowerCase().trim()) {
      'student' => UserRole.student,
      'faculty' => UserRole.faculty,
      'college_admin' => UserRole.collegeAdmin,
      _ => UserRole.unknown,
    };
  }

  bool get isStudent => this == UserRole.student;
  bool get isFaculty => this == UserRole.faculty;
  bool get isCollegeAdmin => this == UserRole.collegeAdmin;
  bool get isSupported => isStudent || isFaculty;
}

class User {
  const User({
    required this.id,
    required this.collegeId,
    required this.fullName,
    required this.email,
    required this.role,
    this.facultyCode,
  });

  final int id;
  final int collegeId;
  final String fullName;
  final String email;
  final String role;
  final String? facultyCode;

  UserRole get userRole => UserRole.fromString(role);

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: (json['id'] ?? json['user_id']) as int,
      collegeId: (json['college_id'] ?? json['collegeId']) as int,
      fullName: (json['full_name'] ?? '') as String,
      email: (json['email'] ?? '') as String,
      role: json['role'] as String,
      facultyCode: json['faculty_code'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'college_id': collegeId,
      'full_name': fullName,
      'email': email,
      'role': role,
      if (facultyCode != null) 'faculty_code': facultyCode,
    };
  }

  User copyWith({
    int? id,
    int? collegeId,
    String? fullName,
    String? email,
    String? role,
    String? facultyCode,
  }) {
    return User(
      id: id ?? this.id,
      collegeId: collegeId ?? this.collegeId,
      fullName: fullName ?? this.fullName,
      email: email ?? this.email,
      role: role ?? this.role,
      facultyCode: facultyCode ?? this.facultyCode,
    );
  }
}
