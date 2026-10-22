import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/auth/auth_controller.dart';
import 'package:docproject/features/auth/models/auth_response.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/auth/services/auth_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/testing.dart';

class FakeAuthService extends AuthService {
  FakeAuthService({
    this.loginResponse,
    this.loginError,
    this.getMeResponse,
    this.getMeError,
  }) : super(
         ApiClient(
           httpClient: MockClient((request) async {
             throw StateError('Unexpected HTTP request in unit test');
           }),
         ),
       );

  final AuthResponse? loginResponse;
  final Exception? loginError;
  final User? getMeResponse;
  final Exception? getMeError;

  @override
  Future<AuthResponse> login({
    required String collegeCode,
    required String email,
    required String password,
  }) async {
    if (loginError != null) {
      throw loginError!;
    }
    return loginResponse!;
  }

  @override
  Future<User> getMe({required String accessToken}) async {
    if (getMeError != null) {
      throw getMeError!;
    }
    return getMeResponse!;
  }
}

class FakeTokenStorage extends TokenStorage {
  String? token;
  User? user;
  bool deleted = false;
  bool throwOnDelete = false;

  @override
  Future<void> saveAccessToken(String token) async {
    this.token = token;
  }

  @override
  Future<String?> readAccessToken() async => token;

  @override
  Future<void> deleteAccessToken() async {
    if (throwOnDelete) {
      throw Exception('Disk/Keychain write error');
    }
    token = null;
    deleted = true;
  }

  @override
  Future<void> saveUser(User user) async {
    this.user = user;
  }

  @override
  Future<User?> readUser() async => user;

  @override
  Future<void> deleteUser() async {
    user = null;
  }

  @override
  Future<void> clearAll() async {
    await deleteAccessToken();
    await deleteUser();
  }
}

void main() {
  group('AuthController', () {
    const sampleUser = User(
      id: 1,
      collegeId: 10,
      fullName: 'Test Student',
      email: 'student@example.com',
      role: 'student',
    );

    const authResponse = AuthResponse(
      accessToken: 'test-token',
      user: sampleUser,
    );

    test('successful login saves the token, user, and transitions to Authenticated', () async {
      final authService = FakeAuthService(loginResponse: authResponse);
      final tokenStorage = FakeTokenStorage();

      final controller = AuthController(
        authService: authService,
        tokenStorage: tokenStorage,
      );

      final result = await controller.login(
        collegeCode: 'COLLEGE',
        email: 'student@example.com',
        password: 'password',
      );

      expect(result.accessToken, 'test-token');
      expect(result.user.fullName, 'Test Student');
      expect(tokenStorage.token, 'test-token');
      expect(tokenStorage.user?.fullName, 'Test Student');
      expect(controller.state, equals(const Authenticated(sampleUser)));
      expect(controller.isAuthenticated, isTrue);
      expect(controller.currentUser?.id, 1);
    });

    test(
      'failed login does not save a token or user and sets Unauthenticated',
      () async {
        final authService = FakeAuthService(
          loginError: Exception('invalid credentials'),
        );
        final tokenStorage = FakeTokenStorage();

        final controller = AuthController(
          authService: authService,
          tokenStorage: tokenStorage,
        );

        await expectLater(
          controller.login(
            collegeCode: 'COLLEGE',
            email: 'student@example.com',
            password: 'wrong-password',
          ),
          throwsException,
        );

        expect(tokenStorage.token, isNull);
        expect(tokenStorage.user, isNull);
        expect(controller.state, equals(const Unauthenticated()));
        expect(controller.isAuthenticated, isFalse);
      },
    );

    test(
      'logout deletes stored token and user and sets Unauthenticated',
      () async {
        final tokenStorage = FakeTokenStorage()
          ..token = 'test-token'
          ..user = sampleUser;

        final controller = AuthController(
          authService: FakeAuthService(loginResponse: authResponse),
          tokenStorage: tokenStorage,
          initialState: const Authenticated(sampleUser),
        );

        await controller.logout();

        expect(tokenStorage.token, isNull);
        expect(tokenStorage.user, isNull);
        expect(tokenStorage.deleted, isTrue);
        expect(controller.state, equals(const Unauthenticated()));
        expect(controller.currentUser, isNull);
      },
    );

    test(
      'logout clears in-memory session even if storage throws an exception',
      () async {
        final tokenStorage = FakeTokenStorage()
          ..token = 'test-token'
          ..user = sampleUser
          ..throwOnDelete = true;

        final controller = AuthController(
          authService: FakeAuthService(loginResponse: authResponse),
          tokenStorage: tokenStorage,
          initialState: const Authenticated(sampleUser),
        );

        try {
          await controller.logout();
        } catch (_) {}

        expect(controller.state, equals(const Unauthenticated()));
        expect(controller.currentUser, isNull);
      },
    );

    test('restoreSession with no saved token sets Unauthenticated', () async {
      final authService = FakeAuthService();
      final tokenStorage = FakeTokenStorage();

      final controller = AuthController(
        authService: authService,
        tokenStorage: tokenStorage,
      );

      await controller.restoreSession();

      expect(controller.state, equals(const Unauthenticated()));
      expect(controller.isAuthenticated, isFalse);
    });

    test('restoreSession with valid token validates against getMe and restores Authenticated session', () async {
      const claimsUser = User(
        id: 1,
        collegeId: 10,
        fullName: '',
        email: '',
        role: 'student',
      );

      final authService = FakeAuthService(getMeResponse: claimsUser);
      final tokenStorage = FakeTokenStorage()
        ..token = 'valid-jwt-token'
        ..user = sampleUser;

      final controller = AuthController(
        authService: authService,
        tokenStorage: tokenStorage,
      );

      await controller.restoreSession();

      expect(controller.state, equals(const Authenticated(sampleUser)));
      expect(controller.isAuthenticated, isTrue);
      expect(controller.currentUser?.fullName, 'Test Student');
    });

    test('restoreSession with 401 invalid/expired token deletes token and sets Unauthenticated', () async {
      final authService = FakeAuthService(
        getMeError: const ApiException(
          statusCode: 401,
          message: 'invalid or expired access token',
        ),
      );
      final tokenStorage = FakeTokenStorage()
        ..token = 'expired-token'
        ..user = sampleUser;

      final controller = AuthController(
        authService: authService,
        tokenStorage: tokenStorage,
      );

      await controller.restoreSession();

      expect(tokenStorage.token, isNull);
      expect(tokenStorage.user, isNull);
      expect(controller.state, equals(const Unauthenticated()));
      expect(controller.isAuthenticated, isFalse);
    });

    test('restoreSession with temporary network error does NOT delete token and sets AuthSessionCheckFailure', () async {
      final authService = FakeAuthService(
        getMeError: const ApiException(
          statusCode: 0,
          message: 'Unable to connect to the server. Check your connection.',
        ),
      );
      final tokenStorage = FakeTokenStorage()
        ..token = 'saved-valid-token'
        ..user = sampleUser;

      final controller = AuthController(
        authService: authService,
        tokenStorage: tokenStorage,
      );

      await controller.restoreSession();

      // Token and user MUST NOT be wiped on temporary network disconnect
      expect(tokenStorage.token, 'saved-valid-token');
      expect(tokenStorage.user, sampleUser);
      expect(
        controller.state,
        equals(
          const AuthSessionCheckFailure(
            message: 'Unable to connect to the server. Check your connection.',
            canRetry: true,
          ),
        ),
      );
    });

    test('markUnauthenticated forces transition to Unauthenticated', () {
      final controller = AuthController(
        authService: FakeAuthService(),
        tokenStorage: FakeTokenStorage(),
        initialState: const AuthSessionCheckFailure(
          message: 'Network error',
          canRetry: true,
        ),
      );

      controller.markUnauthenticated();

      expect(controller.state, equals(const Unauthenticated()));
    });
  });
}
