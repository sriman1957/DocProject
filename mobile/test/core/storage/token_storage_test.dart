import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage_platform_interface/flutter_secure_storage_platform_interface.dart';

import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/auth/models/user.dart';

class FakeSecureStoragePlatform extends FlutterSecureStoragePlatform {
  final Map<String, String> _values = {};

  @override
  Future<void> write({
    required String key,
    required String value,
    required Map<String, String> options,
  }) async {
    _values[key] = value;
  }

  @override
  Future<String?> read({
    required String key,
    required Map<String, String> options,
  }) async {
    return _values[key];
  }

  @override
  Future<bool> containsKey({
    required String key,
    required Map<String, String> options,
  }) async {
    return _values.containsKey(key);
  }

  @override
  Future<void> delete({
    required String key,
    required Map<String, String> options,
  }) async {
    _values.remove(key);
  }

  @override
  Future<Map<String, String>> readAll({
    required Map<String, String> options,
  }) async {
    return Map<String, String>.from(_values);
  }

  @override
  Future<void> deleteAll({required Map<String, String> options}) async {
    _values.clear();
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late FakeSecureStoragePlatform fakePlatform;
  late TokenStorage tokenStorage;

  setUp(() {
    fakePlatform = FakeSecureStoragePlatform();
    FlutterSecureStoragePlatform.instance = fakePlatform;
    tokenStorage = TokenStorage();
  });

  tearDown(() {
    FlutterSecureStoragePlatform.instance = MethodChannelFlutterSecureStorage();
  });

  group('TokenStorage', () {
    test('saves and reads an access token', () async {
      await tokenStorage.saveAccessToken('test-token');

      expect(await tokenStorage.readAccessToken(), 'test-token');
    });

    test('returns null when no token exists', () async {
      expect(await tokenStorage.readAccessToken(), isNull);
    });

    test('deletes the saved access token', () async {
      await tokenStorage.saveAccessToken('test-token');

      await tokenStorage.deleteAccessToken();

      expect(await tokenStorage.readAccessToken(), isNull);
    });

    test('rejects an empty access token', () async {
      await expectLater(
        tokenStorage.saveAccessToken('  '),
        throwsArgumentError,
      );
    });

    test('saves and reads cached user', () async {
      const user = User(
        id: 1,
        collegeId: 10,
        fullName: 'Test Student',
        email: 'student@example.com',
        role: 'student',
      );

      await tokenStorage.saveUser(user);
      final restored = await tokenStorage.readUser();

      expect(restored, isNotNull);
      expect(restored!.id, 1);
      expect(restored.fullName, 'Test Student');
      expect(restored.role, 'student');
    });

    test('deletes cached user', () async {
      const user = User(
        id: 1,
        collegeId: 10,
        fullName: 'Test Student',
        email: 'student@example.com',
        role: 'student',
      );

      await tokenStorage.saveUser(user);
      await tokenStorage.deleteUser();

      expect(await tokenStorage.readUser(), isNull);
    });

    test('clearAll deletes both token and user', () async {
      const user = User(
        id: 1,
        collegeId: 10,
        fullName: 'Test Student',
        email: 'student@example.com',
        role: 'student',
      );

      await tokenStorage.saveAccessToken('my-token');
      await tokenStorage.saveUser(user);

      await tokenStorage.clearAll();

      expect(await tokenStorage.readAccessToken(), isNull);
      expect(await tokenStorage.readUser(), isNull);
    });
  });
}
