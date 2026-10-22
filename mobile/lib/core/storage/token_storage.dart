import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../../features/auth/models/user.dart';

class TokenStorage {
  TokenStorage({FlutterSecureStorage? storage})
    : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  static const String _accessTokenKey = 'access_token';
  static const String _userDataKey = 'user_data';

  Future<void> saveAccessToken(String token) async {
    if (token.trim().isEmpty) {
      throw ArgumentError.value(
        token,
        'token',
        'Access token cannot be empty.',
      );
    }

    await _storage.write(key: _accessTokenKey, value: token);
  }

  Future<String?> readAccessToken() {
    return _storage.read(key: _accessTokenKey);
  }

  Future<void> deleteAccessToken() {
    return _storage.delete(key: _accessTokenKey);
  }

  Future<void> saveUser(User user) async {
    await _storage.write(key: _userDataKey, value: jsonEncode(user.toJson()));
  }

  Future<User?> readUser() async {
    final data = await _storage.read(key: _userDataKey);
    if (data == null || data.trim().isEmpty) return null;

    try {
      final decoded = jsonDecode(data);
      if (decoded is Map<String, dynamic>) {
        return User.fromJson(decoded);
      }
    } catch (_) {
      return null;
    }

    return null;
  }

  Future<void> deleteUser() {
    return _storage.delete(key: _userDataKey);
  }

  Future<void> clearAll() async {
    await deleteAccessToken();
    await deleteUser();
  }
}
