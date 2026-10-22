import '../../../core/network/api_client.dart';
import '../models/auth_response.dart';
import '../models/user.dart';

class AuthService {
  AuthService(this._apiClient);

  final ApiClient _apiClient;
  ApiClient get apiClient => _apiClient;

  Future<AuthResponse> login({
    required String collegeCode,
    required String email,
    required String password,
  }) async {
    final response = await _apiClient.post(
      '/auth/login',
      body: {
        'college_code': collegeCode.trim(),
        'email': email.trim(),
        'password': password,
      },
    );

    if (response is! Map<String, dynamic>) {
      throw const FormatException('Invalid login response.');
    }

    return AuthResponse.fromJson(response);
  }

  Future<User> getMe({required String accessToken}) async {
    final response = await _apiClient.get('/auth/me', accessToken: accessToken);

    if (response is! Map<String, dynamic>) {
      throw const FormatException('Invalid /auth/me response.');
    }

    return User.fromJson(response);
  }
}
