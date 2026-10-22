import 'package:docproject/app.dart';
import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/core/widgets/unsupported_role_screen.dart';
import 'package:docproject/features/auth/auth_controller.dart';
import 'package:docproject/features/auth/models/auth_response.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/auth/screens/session_failure_screen.dart';
import 'package:docproject/features/auth/screens/startup_screen.dart';
import 'package:docproject/features/auth/services/auth_service.dart';
import 'package:docproject/features/documents/screens/personal_vault_screen.dart';
import 'package:docproject/features/home/screens/app_shell.dart';
import 'package:docproject/features/home/screens/faculty_dashboard_screen.dart';
import 'package:docproject/features/home/screens/student_dashboard_screen.dart';
import 'package:docproject/features/groups/screens/groups_listing_screen.dart';
import 'package:docproject/features/notifications/screens/notifications_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/testing.dart';

class FakeAuthService extends AuthService {
  FakeAuthService({
    this.loginResponse,
    this.loginError,
    this.loginDelay,
    this.getMeResponse,
    this.getMeError,
    this.getMeDelay,
  }) : super(
         ApiClient(
           httpClient: MockClient((request) async {
             throw StateError('Unexpected HTTP request in widget test');
           }),
         ),
       );

  final AuthResponse? loginResponse;
  final Exception? loginError;
  final Duration? loginDelay;

  final User? getMeResponse;
  final Exception? getMeError;
  final Duration? getMeDelay;

  int getMeCallCount = 0;

  @override
  Future<AuthResponse> login({
    required String collegeCode,
    required String email,
    required String password,
  }) async {
    if (loginDelay != null) {
      await Future<void>.delayed(loginDelay!);
    }
    if (loginError != null) {
      throw loginError!;
    }
    return loginResponse!;
  }

  @override
  Future<User> getMe({required String accessToken}) async {
    getMeCallCount++;
    if (getMeDelay != null) {
      await Future<void>.delayed(getMeDelay!);
    }
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

  @override
  Future<void> saveAccessToken(String token) async {
    this.token = token;
  }

  @override
  Future<String?> readAccessToken() async => token;

  @override
  Future<void> deleteAccessToken() async {
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
  const sampleStudent = User(
    id: 1,
    collegeId: 10,
    fullName: 'Test Student',
    email: 'student@college.edu',
    role: 'student',
  );

  const sampleFaculty = User(
    id: 2,
    collegeId: 10,
    fullName: 'Prof. Sharma',
    email: 'sharma@college.edu',
    role: 'faculty',
    facultyCode: 'CS101',
  );

  const sampleResponse = AuthResponse(
    accessToken: 'mock_jwt_token',
    user: sampleStudent,
  );

  group('Feature 1: Persistent Authentication & Navigation', () {
    testWidgets(
      'first launch with no saved token shows StartupScreen then resolves to login form',
      (WidgetTester tester) async {
        final authService = FakeAuthService(loginResponse: sampleResponse);
        final tokenStorage = FakeTokenStorage();
        final controller = AuthController(
          authService: authService,
          tokenStorage: tokenStorage,
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));

        expect(find.byType(StartupScreen), findsOneWidget);

        await tester.pumpAndSettle();

        expect(find.text('DocProject'), findsOneWidget);
        expect(find.text('Your college document vault'), findsOneWidget);
        expect(find.text('Sign in'), findsNWidgets(2));
        expect(find.byType(TextFormField), findsNWidgets(3));
      },
    );

    testWidgets(
      'startup with valid saved token validates against backend and navigates to AppShell',
      (WidgetTester tester) async {
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
          ..user = sampleStudent;

        final controller = AuthController(
          authService: authService,
          tokenStorage: tokenStorage,
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        expect(find.byType(AppShell), findsOneWidget);
        expect(find.text('Welcome, Test Student'), findsOneWidget);
        expect(find.text('student@college.edu'), findsWidgets);
      },
    );

    testWidgets(
      'startup with expired token (401) clears token and opens login form',
      (WidgetTester tester) async {
        final authService = FakeAuthService(
          getMeError: const ApiException(
            statusCode: 401,
            message: 'invalid or expired access token',
          ),
          loginResponse: sampleResponse,
        );
        final tokenStorage = FakeTokenStorage()
          ..token = 'expired-token'
          ..user = sampleStudent;

        final controller = AuthController(
          authService: authService,
          tokenStorage: tokenStorage,
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        expect(tokenStorage.token, isNull);
        expect(find.text('Sign in'), findsNWidgets(2));
      },
    );

    testWidgets(
      'temporary network failure during session restoration does not erase token and shows retry screen',
      (WidgetTester tester) async {
        final authService = FakeAuthService(
          getMeError: const ApiException(
            statusCode: 0,
            message: 'Unable to connect to the server. Check your connection.',
          ),
        );
        final tokenStorage = FakeTokenStorage()
          ..token = 'saved-token'
          ..user = sampleStudent;

        final controller = AuthController(
          authService: authService,
          tokenStorage: tokenStorage,
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        expect(tokenStorage.token, 'saved-token');
        expect(find.byType(SessionFailureScreen), findsOneWidget);
        expect(find.text('Connection Issue'), findsOneWidget);
        expect(
          find.text('Unable to connect to the server. Check your connection.'),
          findsOneWidget,
        );
        expect(find.text('Retry Connection'), findsOneWidget);
        expect(find.text('Log in with another account'), findsOneWidget);

        await tester.tap(find.text('Log in with another account'));
        await tester.pumpAndSettle();

        expect(find.text('Sign in'), findsNWidgets(2));
      },
    );

    testWidgets(
      'session restoration does not trigger repeatedly on widget rebuilds',
      (WidgetTester tester) async {
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
          ..user = sampleStudent;

        final controller = AuthController(
          authService: authService,
          tokenStorage: tokenStorage,
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        expect(authService.getMeCallCount, 1);

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        expect(authService.getMeCallCount, 1);
      },
    );

    testWidgets(
      'displays required field validation errors when submitting empty form',
      (WidgetTester tester) async {
        final controller = AuthController(
          authService: FakeAuthService(loginResponse: sampleResponse),
          tokenStorage: FakeTokenStorage(),
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        final submitButton = find.widgetWithText(FilledButton, 'Sign in');
        await tester.tap(submitButton);
        await tester.pumpAndSettle();

        expect(find.text('College code is required.'), findsOneWidget);
        expect(find.text('Email address is required.'), findsOneWidget);
        expect(find.text('Password is required.'), findsOneWidget);
      },
    );

    testWidgets('validates email address format', (WidgetTester tester) async {
      final controller = AuthController(
        authService: FakeAuthService(loginResponse: sampleResponse),
        tokenStorage: FakeTokenStorage(),
      );

      await tester.pumpWidget(DocProjectApp(authController: controller));
      await tester.pumpAndSettle();

      final fields = find.byType(TextFormField);
      await tester.enterText(fields.at(0), 'DOCPRO');
      await tester.enterText(fields.at(1), 'not-an-email');
      await tester.enterText(fields.at(2), 'mypassword');

      final submitButton = find.widgetWithText(FilledButton, 'Sign in');
      await tester.tap(submitButton);
      await tester.pumpAndSettle();

      expect(find.text('Enter a valid email address.'), findsOneWidget);
    });

    testWidgets('toggles password visibility when suffix icon is tapped', (
      WidgetTester tester,
    ) async {
      final controller = AuthController(
        authService: FakeAuthService(loginResponse: sampleResponse),
        tokenStorage: FakeTokenStorage(),
      );

      await tester.pumpWidget(DocProjectApp(authController: controller));
      await tester.pumpAndSettle();

      final toggleButton = find.byTooltip('Show password');
      expect(toggleButton, findsOneWidget);

      await tester.tap(toggleButton);
      await tester.pumpAndSettle();

      expect(find.byTooltip('Hide password'), findsOneWidget);
    });

    testWidgets('displays error message when login fails with ApiException', (
      WidgetTester tester,
    ) async {
      final controller = AuthController(
        authService: FakeAuthService(
          loginError: const ApiException(
            statusCode: 401,
            message: 'Invalid college code, email, or password.',
          ),
        ),
        tokenStorage: FakeTokenStorage(),
      );

      await tester.pumpWidget(DocProjectApp(authController: controller));
      await tester.pumpAndSettle();

      final fields = find.byType(TextFormField);
      await tester.enterText(fields.at(0), 'DOCPRO');
      await tester.enterText(fields.at(1), 'student@college.edu');
      await tester.enterText(fields.at(2), 'wrongpassword');

      final submitButton = find.widgetWithText(FilledButton, 'Sign in');
      await tester.tap(submitButton);
      await tester.pumpAndSettle();

      expect(
        find.text('Invalid college code, email, or password.'),
        findsOneWidget,
      );
      expect(find.byIcon(Icons.error_outline), findsOneWidget);
    });
  });

  group('Feature 2: Role-Based Dashboard and App Shell', () {
    testWidgets(
      'authenticated student receives student dashboard with branding, greeting, and action cards',
      (WidgetTester tester) async {
        final controller = AuthController(
          authService: FakeAuthService(loginResponse: sampleResponse),
          tokenStorage: FakeTokenStorage()
            ..token = 'test-token'
            ..user = sampleStudent,
          initialState: const Authenticated(sampleStudent),
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        expect(find.byType(AppShell), findsOneWidget);
        expect(find.byType(StudentDashboardScreen), findsOneWidget);
        expect(find.text('DocProject'), findsOneWidget);
        expect(find.text('Welcome, Test Student'), findsOneWidget);
        expect(find.text('Student'), findsWidgets); // badge and row
        expect(find.text('Personal Document Vault'), findsOneWidget);
        expect(find.text('Academic Groups'), findsOneWidget);
        expect(find.text('Notifications & Alerts'), findsOneWidget);
      },
    );

    testWidgets(
      'authenticated faculty receives faculty dashboard with greeting, faculty code, and management cards',
      (WidgetTester tester) async {
        final controller = AuthController(
          authService: FakeAuthService(),
          tokenStorage: FakeTokenStorage()
            ..token = 'faculty-token'
            ..user = sampleFaculty,
          initialState: const Authenticated(sampleFaculty),
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        expect(find.byType(AppShell), findsOneWidget);
        expect(find.byType(FacultyDashboardScreen), findsOneWidget);
        expect(find.text('DocProject'), findsOneWidget);
        expect(find.text('Welcome, Prof. Sharma'), findsOneWidget);
        expect(find.text('Faculty • CS101'), findsOneWidget);
        expect(find.text('Groups Management'), findsOneWidget);
        expect(find.text('Subgroup Activities'), findsOneWidget);
        expect(find.text('Activity Notifications'), findsOneWidget);
      },
    );

    testWidgets(
      'unknown or unsupported role displays UnsupportedRoleScreen with Sign out action',
      (WidgetTester tester) async {
        const unsupportedUser = User(
          id: 99,
          collegeId: 10,
          fullName: 'System Auditor',
          email: 'auditor@college.edu',
          role: 'auditor', // Unsupported on mobile
        );

        final tokenStorage = FakeTokenStorage()
          ..token = 'auditor-token'
          ..user = unsupportedUser;

        final controller = AuthController(
          authService: FakeAuthService(),
          tokenStorage: tokenStorage,
          initialState: const Authenticated(unsupportedUser),
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        // Must NOT render Student or Faculty Dashboard
        expect(find.byType(StudentDashboardScreen), findsNothing);
        expect(find.byType(FacultyDashboardScreen), findsNothing);
        expect(find.byType(AppShell), findsNothing);

        // Must render UnsupportedRoleScreen
        expect(find.byType(UnsupportedRoleScreen), findsOneWidget);
        expect(find.text('Access Unavailable'), findsOneWidget);
        expect(
          find.text(
            'Your account role (auditor) is not supported on this mobile application.',
          ),
          findsOneWidget,
        );

        // Sign out is accessible
        final signOutButton = find.widgetWithText(FilledButton, 'Sign out');
        expect(signOutButton, findsOneWidget);
        await tester.tap(signOutButton);
        await tester.pumpAndSettle();

        expect(find.text('Sign in'), findsNWidgets(2));
        expect(tokenStorage.token, isNull);
      },
    );

    testWidgets(
      'bottom navigation bar switches tabs and displays placeholder with honest coming-soon badge',
      (WidgetTester tester) async {
        final controller = AuthController(
          authService: FakeAuthService(),
          tokenStorage: FakeTokenStorage()
            ..token = 'test-token'
            ..user = sampleStudent,
          initialState: const Authenticated(sampleStudent),
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        // Initially on Dashboard
        expect(find.byType(StudentDashboardScreen), findsOneWidget);

        // Tap "Vault" navigation item in bottom bar
        final vaultTab = find.text('Vault');
        await tester.tap(vaultTab);
        await tester.pumpAndSettle();

        // Displays PersonalVaultScreen with title
        expect(find.byType(PersonalVaultScreen), findsOneWidget);
        expect(find.text('Personal Vault'), findsOneWidget);

        // Tap "Alerts" navigation item
        final alertsTab = find.text('Alerts');
        await tester.tap(alertsTab);
        await tester.pumpAndSettle();

        expect(find.byType(NotificationsScreen), findsOneWidget);
        expect(find.text('Notifications'), findsOneWidget);

        // Tap "Groups" navigation item
        final groupsTab = find.text('Groups');
        await tester.tap(groupsTab);
        await tester.pumpAndSettle();

        expect(find.byType(GroupsListingScreen), findsOneWidget);

        // Return to Dashboard tab
        final dashboardTab = find.text('Dashboard');
        await tester.tap(dashboardTab);
        await tester.pumpAndSettle();

        expect(find.byType(StudentDashboardScreen), findsOneWidget);
      },
    );

    testWidgets(
      'quick action card on dashboard navigates to corresponding tab destination',
      (WidgetTester tester) async {
        final controller = AuthController(
          authService: FakeAuthService(),
          tokenStorage: FakeTokenStorage()
            ..token = 'test-token'
            ..user = sampleStudent,
          initialState: const Authenticated(sampleStudent),
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        // Tap "Personal Document Vault" tile directly on dashboard
        final vaultActionCard = find.text('Personal Document Vault');
        await tester.tap(vaultActionCard);
        await tester.pumpAndSettle();

        // Switches to Vault tab
        expect(find.byType(PersonalVaultScreen), findsOneWidget);
        expect(find.text('Personal Vault'), findsOneWidget);
      },
    );

    testWidgets(
      'logout from app bar triggers confirmation dialog and redirects to login on confirm',
      (WidgetTester tester) async {
        final tokenStorage = FakeTokenStorage()
          ..token = 'test-token'
          ..user = sampleStudent;

        final controller = AuthController(
          authService: FakeAuthService(),
          tokenStorage: tokenStorage,
          initialState: const Authenticated(sampleStudent),
        );

        await tester.pumpWidget(DocProjectApp(authController: controller));
        await tester.pumpAndSettle();

        // Tap sign out icon in AppBar
        final logoutIcon = find.byTooltip('Sign out');
        await tester.tap(logoutIcon);
        await tester.pumpAndSettle();

        // Confirmation dialog appears
        expect(find.byType(AlertDialog), findsOneWidget);
        expect(
          find.text('Are you sure you want to sign out of DocProject?'),
          findsOneWidget,
        );

        // Cancel keeps user in AppShell
        await tester.tap(find.text('Cancel'));
        await tester.pumpAndSettle();
        expect(find.byType(AppShell), findsOneWidget);

        // Re-open and confirm
        await tester.tap(logoutIcon);
        await tester.pumpAndSettle();
        await tester.tap(find.widgetWithText(FilledButton, 'Sign out'));
        await tester.pumpAndSettle();

        // Navigates to LoginScreen and token is cleared
        expect(find.text('Sign in'), findsNWidgets(2));
        expect(tokenStorage.token, isNull);
        expect(find.byType(AppShell), findsNothing);
      },
    );
  });
}
