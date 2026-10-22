import 'core/network/api_client.dart';
import 'core/storage/token_storage.dart';
import 'features/auth/auth_controller.dart';
import 'features/auth/services/auth_service.dart';
import 'features/groups/groups_controller.dart';
import 'features/groups/services/groups_service.dart';
import 'features/subgroups/services/subgroups_service.dart';
import 'features/subgroups/subgroups_controller.dart';

class AppDependencies {
  AppDependencies({
    required this.apiClient,
    required this.tokenStorage,
    required this.authService,
    required this.authController,
    required this.groupsService,
    required this.groupsController,
    required this.subgroupsService,
    required this.subgroupsController,
  });

  final ApiClient apiClient;
  final TokenStorage tokenStorage;
  final AuthService authService;
  final AuthController authController;
  final GroupsService groupsService;
  final GroupsController groupsController;
  final SubgroupsService subgroupsService;
  final SubgroupsController subgroupsController;

  static AppDependencies create({
    ApiClient? apiClient,
    TokenStorage? tokenStorage,
    AuthService? authService,
    AuthController? authController,
    GroupsService? groupsService,
    GroupsController? groupsController,
    SubgroupsService? subgroupsService,
    SubgroupsController? subgroupsController,
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
    final effectiveSubgroupsService =
        subgroupsService ??
        SubgroupsService(
          apiClient: effectiveApiClient,
          tokenStorage: effectiveTokenStorage,
        );
    final effectiveSubgroupsController =
        subgroupsController ??
        SubgroupsController(subgroupsService: effectiveSubgroupsService);

    return AppDependencies(
      apiClient: effectiveApiClient,
      tokenStorage: effectiveTokenStorage,
      authService: effectiveAuthService,
      authController: effectiveAuthController,
      groupsService: effectiveGroupsService,
      groupsController: effectiveGroupsController,
      subgroupsService: effectiveSubgroupsService,
      subgroupsController: effectiveSubgroupsController,
    );
  }

  void dispose() {
    apiClient.close();
    authController.dispose();
    groupsController.dispose();
    subgroupsController.dispose();
  }
}
