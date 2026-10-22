import 'package:flutter/material.dart';

import '../../../core/widgets/unsupported_role_screen.dart';
import '../../access_periods/controllers/access_periods_controller.dart';
import '../../auth/auth_controller.dart';
import '../../auth/models/user.dart';
import '../../documents/controllers/document_upload_controller.dart';
import '../../documents/controllers/subgroup_submissions_controller.dart';
import '../../documents/documents_controller.dart';
import '../../groups/groups_controller.dart';
import '../../subgroups/subgroups_controller.dart';
import 'app_shell.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({
    required this.user,
    required this.authController,
    this.groupsController,
    this.subgroupsController,
    this.documentsController,
    this.uploadController,
    this.accessPeriodsController,
    this.submissionsController,
    super.key,
  });

  final User user;
  final AuthController authController;
  final GroupsController? groupsController;
  final SubgroupsController? subgroupsController;
  final DocumentsController? documentsController;
  final DocumentUploadController? uploadController;
  final AccessPeriodsController? accessPeriodsController;
  final SubgroupSubmissionsController? submissionsController;

  @override
  Widget build(BuildContext context) {
    if (!user.userRole.isSupported) {
      return UnsupportedRoleScreen(user: user, authController: authController);
    }

    return AppShell(
      user: user,
      authController: authController,
      groupsController: groupsController,
      subgroupsController: subgroupsController,
      documentsController: documentsController,
      uploadController: uploadController,
      accessPeriodsController: accessPeriodsController,
      submissionsController: submissionsController,
    );
  }
}
