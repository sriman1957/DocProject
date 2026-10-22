import 'user.dart';

class AuthResponse {
  const AuthResponse({required this.accessToken, required this.user});

  final String accessToken;
  final User user;

  factory AuthResponse.fromJson(Map<String, dynamic> json) {
    return AuthResponse(
      accessToken: json['access_token'] as String,
      user: User.fromJson(json['user'] as Map<String, dynamic>),
    );
  }
}
