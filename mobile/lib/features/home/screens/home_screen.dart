import 'package:flutter/material.dart';

import '../../../core/widgets/unsupported_role_screen.dart';
import '../../auth/auth_controller.dart';
import '../../auth/models/user.dart';
import '../../groups/groups_controller.dart';
import 'app_shell.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({
    required this.user,
    required this.authController,
    this.groupsController,
    super.key,
  });

  final User user;
  final AuthController authController;
  final GroupsController? groupsController;

  @override
  Widget build(BuildContext context) {
    if (!user.userRole.isSupported) {
      return UnsupportedRoleScreen(user: user, authController: authController);
    }

    return AppShell(
      user: user,
      authController: authController,
      groupsController: groupsController,
    );
  }
}
