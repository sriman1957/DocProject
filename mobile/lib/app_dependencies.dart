import 'core/network/api_client.dart';
import 'core/storage/token_storage.dart';
import 'features/auth/auth_controller.dart';
import 'features/auth/services/auth_service.dart';
import 'features/groups/groups_controller.dart';
import 'features/groups/services/groups_service.dart';

class AppDependencies {
  AppDependencies({
    required this.apiClient,
    required this.tokenStorage,
    required this.authService,
    required this.authController,
    required this.groupsService,
    required this.groupsController,
  });

  final ApiClient apiClient;
  final TokenStorage tokenStorage;
  final AuthService authService;
  final AuthController authController;
  final GroupsService groupsService;
  final GroupsController groupsController;

  static AppDependencies create({
    ApiClient? apiClient,
    TokenStorage? tokenStorage,
    AuthService? authService,
    AuthController? authController,
    GroupsService? groupsService,
    GroupsController? groupsController,
  }) {
    final effectiveTokenStorage =
        tokenStorage ?? authController?.tokenStorage ?? TokenStorage();
    final effectiveApiClient =
        apiClient ??
        authService?.apiClient ??
        authController?.authService.apiClient ??
        ApiClient();
    final effectiveAuthService =
        authService ??
        authController?.authService ??
        AuthService(effectiveApiClient);
    final effectiveAuthController =
        authController ??
        AuthController(
          authService: effectiveAuthService,
          tokenStorage: effectiveTokenStorage,
        );
    final effectiveGroupsService =
        groupsService ??
        GroupsService(
          apiClient: effectiveApiClient,
          tokenStorage: effectiveTokenStorage,
        );
    final effectiveGroupsController =
        groupsController ??
        GroupsController(groupsService: effectiveGroupsService);

    return AppDependencies(
      apiClient: effectiveApiClient,
      tokenStorage: effectiveTokenStorage,
      authService: effectiveAuthService,
      authController: effectiveAuthController,
      groupsService: effectiveGroupsService,
      groupsController: effectiveGroupsController,
    );
  }

  void dispose() {
    apiClient.close();
    authController.dispose();
    groupsController.dispose();
  }
}
