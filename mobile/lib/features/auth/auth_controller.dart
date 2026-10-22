import 'package:flutter/foundation.dart';

import '../../core/network/api_client.dart';
import '../../core/storage/token_storage.dart';
import 'models/auth_response.dart';
import 'models/user.dart';
import 'services/auth_service.dart';

sealed class AuthState {
  const AuthState();
}

class AuthInitial extends AuthState {
  const AuthInitial();
}

class AuthLoading extends AuthState {
  const AuthLoading();
}

class Unauthenticated extends AuthState {
  const Unauthenticated();
}

class Authenticated extends AuthState {
  const Authenticated(this.user);
  final User user;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Authenticated &&
          runtimeType == other.runtimeType &&
          user.id == other.user.id &&
          user.role == other.user.role;

  @override
  int get hashCode => user.id.hashCode ^ user.role.hashCode;
}

class AuthSessionCheckFailure extends AuthState {
  const AuthSessionCheckFailure({required this.message, this.canRetry = true});

  final String message;
  final bool canRetry;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AuthSessionCheckFailure &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry;

  @override
  int get hashCode => message.hashCode ^ canRetry.hashCode;
}

class AuthController extends ChangeNotifier {
  AuthController({
    required this.authService,
    required this.tokenStorage,
    AuthState initialState = const AuthInitial(),
  }) : _state = initialState;

  final AuthService authService;
  final TokenStorage tokenStorage;

  AuthState _state;
  AuthState get state => _state;

  User? get currentUser => switch (_state) {
    Authenticated(:final user) => user,
    _ => null,
  };

  bool get isAuthenticated => _state is Authenticated;

  bool _isRestoring = false;

  Future<void> restoreSession({bool force = false}) async {
    if (_isRestoring) return;
    if (!force && (_state is Authenticated || _state is Unauthenticated)) {
      return;
    }

    _isRestoring = true;
    _state = const AuthLoading();
    notifyListeners();

    try {
      final token = await tokenStorage.readAccessToken();
      if (token == null || token.trim().isEmpty) {
        _state = const Unauthenticated();
        return;
      }

      final validatedClaimsUser = await authService.getMe(accessToken: token);
      final cachedUser = await tokenStorage.readUser();

      final activeUser =
          (cachedUser != null && cachedUser.id == validatedClaimsUser.id)
          ? cachedUser.copyWith(
              role: validatedClaimsUser.role,
              collegeId: validatedClaimsUser.collegeId,
            )
          : validatedClaimsUser;

      _state = Authenticated(activeUser);
    } on ApiException catch (error) {
      if (error.statusCode == 401) {
        // Token was explicitly rejected by backend as invalid or expired
        await tokenStorage.deleteAccessToken();
        await tokenStorage.deleteUser();
        _state = const Unauthenticated();
      } else {
        // Network or temporary server failure; do NOT delete token
        _state = AuthSessionCheckFailure(
          message: error.message,
          canRetry: true,
        );
      }
    } catch (_) {
      // General or storage failure; retain token for retry
      _state = const AuthSessionCheckFailure(
        message: 'Unable to restore session. Please check your connection.',
        canRetry: true,
      );
    } finally {
      _isRestoring = false;
      notifyListeners();
    }
  }

  Future<AuthResponse> login({
    required String collegeCode,
    required String email,
    required String password,
  }) async {
    _state = const AuthLoading();
    notifyListeners();

    try {
      final response = await authService.login(
        collegeCode: collegeCode,
        email: email,
        password: password,
      );

      await tokenStorage.saveAccessToken(response.accessToken);
      await tokenStorage.saveUser(response.user);

      _state = Authenticated(response.user);
      return response;
    } catch (error) {
      _state = const Unauthenticated();
      rethrow;
    } finally {
      notifyListeners();
    }
  }

  Future<void> logout() async {
    try {
      await tokenStorage.deleteAccessToken();
      await tokenStorage.deleteUser();
    } finally {
      _state = const Unauthenticated();
      notifyListeners();
    }
  }

  void markUnauthenticated() {
    _state = const Unauthenticated();
    notifyListeners();
  }
}
